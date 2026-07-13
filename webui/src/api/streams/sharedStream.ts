// Leader-elected shared stream. Backrest is usually served over http:// (h2c),
// where browsers use HTTP/1.1 and cap ~6 connections/origin across all tabs, so
// a long-lived server-stream per tab exhausts the pool. Here one tab (the leader,
// via Web Locks) holds the upstream stream and rebroadcasts messages to the rest
// over a BroadcastChannel; followers hold no upstream connection. Falls back to a
// per-tab stream when Web Locks/BroadcastChannel are unavailable (e.g. Chrome
// Android).

/** A tab hidden at least this long stops being eligible to lead. */
export const BACKGROUND_ELIGIBILITY_MS = 120_000;

const DEFAULT_BACKOFF_MS = 5_000;

export type StreamStatus = "live" | "reconnecting" | "offline";

export interface StreamSubscriber<T> {
  onMessage(msg: T): void;
  /** Fired whenever this subscriber (re)joins a live stream: on subscribe, on
   *  reconnect/leader-handoff, and on foreground return past the hidden
   *  threshold. The stream carries only deltas, so consumers (re)load their
   *  full initial state from the API here. */
  onConnectOrResync(): void;
  onStatus?(status: StreamStatus): void;
}

export interface SharedStreamOpts<T> {
  /** Used for both the lock name and the BroadcastChannel name. */
  name: string;
  connect: (signal: AbortSignal) => AsyncIterable<T>;
  encode: (msg: T) => Uint8Array;
  decode: (bytes: Uint8Array) => T;
  backoffMs?: number;
  /** How long a hidden tab stays eligible to lead. Overridable for tests. */
  hiddenEligibilityMs?: number;
}

export interface SharedStream<T> {
  /** First subscribe starts the stream, last unsubscribe tears it down. */
  subscribe(sub: StreamSubscriber<T>): () => void;
}

// Uint8Array is structured-cloneable, so encoded proto payloads pass through the
// BroadcastChannel unchanged.
type BusMsg =
  | { t: "m"; d: Uint8Array } // forwarded message
  | { t: "r" }; // leader (re)connected — reload from the API

const hasWebLocks = (): boolean =>
  typeof navigator !== "undefined" && "locks" in navigator;
const hasBroadcastChannel = (): boolean =>
  typeof BroadcastChannel !== "undefined";
const hasDocument = (): boolean => typeof document !== "undefined";

export function createSharedStream<T>(
  opts: SharedStreamOpts<T>,
): SharedStream<T> {
  return new SharedStreamImpl(opts);
}

class SharedStreamImpl<T> implements SharedStream<T> {
  private readonly backoffMs: number;
  private readonly shared: boolean;
  private readonly eligibility: Eligibility;

  private readonly subscribers = new Set<StreamSubscriber<T>>();

  // Non-null while running; its signal aborts the leader loop / election on the
  // last unsubscribe. Doubles as the started/stopped flag.
  private run: AbortController | null = null;

  private bc: BroadcastChannel | null = null;
  private status: StreamStatus | null = null;
  private hasBeenLive = false;

  constructor(private readonly opts: SharedStreamOpts<T>) {
    this.backoffMs = opts.backoffMs ?? DEFAULT_BACKOFF_MS;
    this.shared = hasWebLocks() && hasBroadcastChannel();
    this.eligibility = new Eligibility(
      opts.hiddenEligibilityMs ?? BACKGROUND_ELIGIBILITY_MS,
      () => this.deliverConnectOrResync(),
    );
  }

  subscribe(sub: StreamSubscriber<T>): () => void {
    this.subscribers.add(sub);
    if (!this.run) this.start();
    // A late joiner missed everything so far; have it load initial state now.
    this.fireConnectOrResync(sub);
    return () => {
      this.subscribers.delete(sub);
      if (this.subscribers.size === 0) this.stop();
    };
  }

  // --- lifecycle ----------------------------------------------------------

  private start() {
    const run = new AbortController();
    this.run = run;
    this.hasBeenLive = false;
    this.eligibility.start();

    if (this.shared) {
      this.bc = new BroadcastChannel(this.opts.name);
      this.bc.onmessage = (e) => this.handleBusMessage(e.data as BusMsg);
      void this.runElection(run.signal);
    } else {
      void this.leaderStreamLoop(run.signal);
    }
  }

  private stop() {
    this.run?.abort();
    this.run = null;
    this.status = null;

    // Unblocks anything waiting on eligibility / leadership so the election can
    // observe the abort and unwind.
    this.eligibility.stop();

    if (this.bc) {
      this.bc.onmessage = null;
      this.bc.close();
      this.bc = null;
    }
  }

  // --- leader election ----------------------------------------------------

  private async runElection(runSignal: AbortSignal) {
    while (!runSignal.aborted) {
      await this.eligibility.waitUntilEligible();
      if (runSignal.aborted) return;

      // One signal serves both roles: a Web Locks signal aborts a still-queued
      // request (so we leave the queue when we lose eligibility), and once the
      // lock is held the leader loop exits on the same signal, resolving the
      // callback and releasing the lock. stop() also aborts it via
      // eligibility.stop(), so teardown unwinds a held lock too.
      const signal = this.eligibility.ineligibleSignal();
      try {
        await navigator.locks.request(this.opts.name, { signal }, () =>
          this.leaderStreamLoop(signal),
        );
      } catch (err) {
        // AbortError just means we left the queue; otherwise log and re-contend
        // after a backoff so a persistently-rejecting lock API can't busy-loop.
        if ((err as Error)?.name !== "AbortError") {
          console.warn(`[sharedStream:${this.opts.name}] lock error`, err);
          await abortableDelay(this.backoffMs, runSignal);
        }
      }
    }
  }

  private async leaderStreamLoop(signal: AbortSignal) {
    while (!signal.aborted) {
      this.emitStatus("reconnecting");
      try {
        let first = true;
        for await (const msg of this.opts.connect(signal)) {
          if (first) {
            first = false;
            this.goLive();
          }
          this.deliverMessage(msg);
          // Encode only to feed the bus; the fallback path has no channel.
          if (this.bc) this.post({ t: "m", d: this.opts.encode(msg) });
        }
      } catch (err) {
        if (!signal.aborted) {
          console.warn(`[sharedStream:${this.opts.name}] stream error`, err);
        }
      }
      if (signal.aborted) break;
      this.emitStatus("offline");
      await abortableDelay(this.backoffMs, signal);
    }
  }

  // --- data plane ---------------------------------------------------------

  private handleBusMessage(msg: BusMsg) {
    switch (msg.t) {
      case "m":
        this.hasBeenLive = true;
        this.emitStatus("live");
        this.deliverMessage(this.opts.decode(msg.d));
        break;
      case "r":
        this.emitStatus("live");
        this.deliverConnectOrResync();
        break;
    }
  }

  // Leader (re)connected its upstream. On the first connect every subscriber
  // already loaded via subscribe, so only a reconnect/handoff is a real gap
  // worth reloading for.
  private goLive() {
    if (this.hasBeenLive) {
      this.post({ t: "r" });
      this.deliverConnectOrResync();
    }
    this.hasBeenLive = true;
    this.emitStatus("live");
  }

  private deliverMessage(msg: T) {
    for (const sub of this.subscribers) {
      try {
        sub.onMessage(msg);
      } catch (e) {
        console.warn(`[sharedStream:${this.opts.name}] onMessage threw`, e);
      }
    }
  }

  private deliverConnectOrResync() {
    for (const sub of this.subscribers) this.fireConnectOrResync(sub);
  }

  private fireConnectOrResync(sub: StreamSubscriber<T>) {
    try {
      sub.onConnectOrResync();
    } catch (e) {
      console.warn(`[sharedStream:${this.opts.name}] onConnectOrResync threw`, e);
    }
  }

  private emitStatus(status: StreamStatus) {
    if (this.status === status) return;
    this.status = status;
    for (const sub of this.subscribers) sub.onStatus?.(status);
  }

  private post(msg: BusMsg) {
    this.bc?.postMessage(msg);
  }

}

// Owns tab visibility and the hidden-eligibility timer, exposing the two things
// leader election needs: a promise that resolves while eligible, and a one-shot
// signal that aborts when eligibility is lost. A tab hidden past the threshold
// stops being eligible to lead so a foreground tab can take over.
class Eligibility {
  private eligible = true;
  private hiddenTimer: ReturnType<typeof setTimeout> | null = null;
  private readonly waiters: Array<() => void> = [];
  private readonly pending = new Set<AbortController>();
  private readonly onVisibilityChange = () => this.handleVisibilityChange();

  constructor(
    private readonly hiddenMs: number,
    // Fired on each ineligible -> eligible transition (never on the initial
    // eligible state), letting the owner run a catch-up after being throttled.
    private readonly onRegainEligible: () => void,
  ) {}

  start() {
    if (hasDocument()) {
      this.eligible = !document.hidden; // hidden-but-recent is still eligible
      if (document.hidden) this.armHiddenTimer();
      document.addEventListener("visibilitychange", this.onVisibilityChange);
    } else {
      this.eligible = true;
    }
  }

  stop() {
    if (hasDocument()) {
      document.removeEventListener("visibilitychange", this.onVisibilityChange);
    }
    this.clearHiddenTimer();
    // Resolve any eligibility wait and abort any in-flight leadership signal so
    // the election can unwind.
    this.eligible = true;
    this.drainWaiters();
    this.abortPending();
  }

  // Resolves immediately if eligible, else on the next transition to eligible
  // (or on stop).
  waitUntilEligible(): Promise<void> {
    if (this.eligible) return Promise.resolve();
    return new Promise((resolve) => this.waiters.push(resolve));
  }

  // A fresh one-shot signal that aborts when eligibility is next lost (or on
  // stop).
  ineligibleSignal(): AbortSignal {
    const ac = new AbortController();
    this.pending.add(ac);
    return ac.signal;
  }

  private handleVisibilityChange() {
    if (document.hidden) {
      this.armHiddenTimer();
      return;
    }
    this.clearHiddenTimer();
    this.setEligible(true);
  }

  private setEligible(next: boolean) {
    if (this.eligible === next) return;
    this.eligible = next;
    if (next) {
      this.drainWaiters();
      this.onRegainEligible();
    } else {
      this.abortPending();
    }
  }

  private armHiddenTimer() {
    this.clearHiddenTimer();
    this.hiddenTimer = setTimeout(() => this.setEligible(false), this.hiddenMs);
  }

  private clearHiddenTimer() {
    if (this.hiddenTimer !== null) {
      clearTimeout(this.hiddenTimer);
      this.hiddenTimer = null;
    }
  }

  private drainWaiters() {
    const waiters = this.waiters.splice(0);
    for (const w of waiters) w();
  }

  private abortPending() {
    for (const ac of this.pending) ac.abort();
    this.pending.clear();
  }
}

/** A setTimeout that also resolves early if `signal` aborts. */
function abortableDelay(ms: number, signal: AbortSignal): Promise<void> {
  return new Promise((resolve) => {
    if (signal.aborted) return resolve();
    const cleanup = () => {
      clearTimeout(timer);
      signal.removeEventListener("abort", onAbort);
    };
    const onAbort = () => {
      cleanup();
      resolve();
    };
    const timer = setTimeout(() => {
      cleanup();
      resolve();
    }, ms);
    signal.addEventListener("abort", onAbort, { once: true });
  });
}
