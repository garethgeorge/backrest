import {
  afterEach,
  beforeEach,
  describe,
  expect,
  it,
  vi,
  type Mock,
} from "vitest";
import { createSharedStream, type StreamSubscriber } from "./sharedStream";

// ---------------------------------------------------------------------------
// Mocks: a shared in-process registry lets multiple SharedStream instances in a
// single test stand in for multiple browser tabs of the same origin.
// ---------------------------------------------------------------------------

// --- BroadcastChannel: delivers to sibling channels of the same name, never to
//     the sender (matching real semantics). ---
const bcRegistry = new Map<string, Set<MockBroadcastChannel>>();

class MockBroadcastChannel {
  onmessage: ((ev: { data: unknown }) => void) | null = null;
  constructor(public readonly name: string) {
    if (!bcRegistry.has(name)) bcRegistry.set(name, new Set());
    bcRegistry.get(name)!.add(this);
  }
  postMessage(data: unknown) {
    for (const ch of bcRegistry.get(this.name) ?? []) {
      if (ch !== this && ch.onmessage) {
        const handler = ch.onmessage;
        queueMicrotask(() => handler({ data }));
      }
    }
  }
  close() {
    bcRegistry.get(this.name)?.delete(this);
  }
}

// --- Web Locks: a single exclusive holder per name, FIFO queue, signal aborts a
//     still-queued request (matching real semantics). ---
interface QueuedLock {
  cb: () => Promise<unknown>;
  resolve: (v: unknown) => void;
  reject: (e: unknown) => void;
  signal?: AbortSignal;
}

class MockLockManager {
  private held = new Set<string>();
  private queues = new Map<string, QueuedLock[]>();

  request(
    name: string,
    opts: { signal?: AbortSignal },
    cb: () => Promise<unknown>,
  ): Promise<unknown> {
    return new Promise((resolve, reject) => {
      const entry: QueuedLock = { cb, resolve, reject, signal: opts.signal };
      if (opts.signal?.aborted) {
        reject(new DOMException("aborted", "AbortError"));
        return;
      }
      opts.signal?.addEventListener("abort", () => {
        const q = this.queues.get(name);
        if (q) {
          const i = q.indexOf(entry);
          if (i >= 0) {
            q.splice(i, 1);
            reject(new DOMException("aborted", "AbortError"));
          }
        }
      });
      if (!this.held.has(name)) {
        void this.grant(name, entry);
      } else {
        if (!this.queues.has(name)) this.queues.set(name, []);
        this.queues.get(name)!.push(entry);
      }
    });
  }

  private async grant(name: string, entry: QueuedLock) {
    this.held.add(name);
    try {
      entry.resolve(await entry.cb());
    } catch (e) {
      entry.reject(e);
    } finally {
      this.held.delete(name);
      const q = this.queues.get(name);
      while (q && q.length) {
        const next = q.shift()!;
        if (!next.signal?.aborted) {
          void this.grant(name, next);
          break;
        }
      }
    }
  }
}

// A never-resolving stream body that unwinds when its signal aborts — stands in
// for a long-lived server-stream that stays open.
const openUntilAborted = (signal: AbortSignal): Promise<void> =>
  new Promise((resolve) => {
    if (signal.aborted) return resolve();
    signal.addEventListener("abort", () => resolve(), { once: true });
  });

// Trivial codec over a { id } message.
type Msg = { id: number };
const encode = (m: Msg) => new TextEncoder().encode(JSON.stringify(m));
const decode = (b: Uint8Array): Msg => JSON.parse(new TextDecoder().decode(b));

const spySubscriber = (): StreamSubscriber<Msg> & {
  onMessage: Mock;
  onConnectOrResync: Mock;
} => ({
  onMessage: vi.fn(),
  onConnectOrResync: vi.fn(),
});

beforeEach(() => {
  bcRegistry.clear();
  vi.stubGlobal("BroadcastChannel", MockBroadcastChannel);
});

afterEach(() => {
  vi.unstubAllGlobals();
  // Remove any navigator.locks we defined.
  if ("locks" in navigator) {
    delete (navigator as unknown as { locks?: unknown }).locks;
  }
});

const installLocks = () => {
  Object.defineProperty(navigator, "locks", {
    value: new MockLockManager(),
    configurable: true,
    writable: true,
  });
};

describe("sharedStream — shared (leader/follower) mode", () => {
  beforeEach(() => installLocks());

  it("elects a single leader; followers receive forwarded messages and never open their own upstream", async () => {
    const connectA = vi.fn((signal: AbortSignal) =>
      (async function* () {
        yield { id: 1 };
        await openUntilAborted(signal);
      })(),
    );
    const connectB = vi.fn((signal: AbortSignal) =>
      (async function* () {
        yield { id: 99 };
        await openUntilAborted(signal);
      })(),
    );

    const streamA = createSharedStream<Msg>({
      name: "t1",
      connect: connectA,
      encode,
      decode,
    });
    const streamB = createSharedStream<Msg>({
      name: "t1",
      connect: connectB,
      encode,
      decode,
    });

    const subA = spySubscriber();
    const subB = spySubscriber();
    // A subscribes first, so A wins the lock and leads.
    const disposeA = streamA.subscribe(subA);
    await Promise.resolve();
    const disposeB = streamB.subscribe(subB);

    // Follower B receives the leader's message over the bus.
    await vi.waitFor(() =>
      expect(subB.onMessage).toHaveBeenCalledWith({ id: 1 }),
    );
    // Leader A delivered it locally too.
    expect(subA.onMessage).toHaveBeenCalledWith({ id: 1 });
    // B never opened its own upstream stream.
    expect(connectB).not.toHaveBeenCalled();
    expect(connectA).toHaveBeenCalledTimes(1);

    disposeA();
    disposeB();
  });

  it("fails over to another tab when the leader stops", async () => {
    const connectA = vi.fn((signal: AbortSignal) =>
      (async function* () {
        yield { id: 1 };
        await openUntilAborted(signal);
      })(),
    );
    const connectB = vi.fn((signal: AbortSignal) =>
      (async function* () {
        yield { id: 2 };
        await openUntilAborted(signal);
      })(),
    );

    const streamA = createSharedStream<Msg>({
      name: "t2",
      connect: connectA,
      encode,
      decode,
    });
    const streamB = createSharedStream<Msg>({
      name: "t2",
      connect: connectB,
      encode,
      decode,
    });

    const disposeA = streamA.subscribe(spySubscriber());
    await Promise.resolve();
    const subB = spySubscriber();
    streamB.subscribe(subB);

    await vi.waitFor(() => expect(connectA).toHaveBeenCalledTimes(1));
    expect(connectB).not.toHaveBeenCalled();

    // Leader leaves — B should acquire the lock and open its own upstream.
    disposeA();
    await vi.waitFor(() => expect(connectB).toHaveBeenCalledTimes(1));
    await vi.waitFor(() =>
      expect(subB.onMessage).toHaveBeenCalledWith({ id: 2 }),
    );
  });

  it("tells a late-joining follower to reload itself instead of replaying state", async () => {
    const idle = (signal: AbortSignal) =>
      (async function* () {
        await openUntilAborted(signal);
      })();

    const connectA = vi.fn(idle);
    const streamA = createSharedStream<Msg>({
      name: "t5",
      connect: connectA,
      encode,
      decode,
    });
    const streamB = createSharedStream<Msg>({
      name: "t5",
      connect: vi.fn(idle),
      encode,
      decode,
    });

    streamA.subscribe(spySubscriber());
    // Wait until A is the leader (its upstream opened) before B joins.
    await vi.waitFor(() => expect(connectA).toHaveBeenCalledTimes(1));

    const subB = spySubscriber();
    streamB.subscribe(subB);

    // B is told to (re)load from the API on join; the stream carries no
    // snapshot, so B receives no forwarded messages from the idle leader.
    expect(subB.onConnectOrResync).toHaveBeenCalledTimes(1);
    await Promise.resolve();
    expect(subB.onMessage).not.toHaveBeenCalled();
  });
});

describe("sharedStream — fallback mode (no Web Locks)", () => {
  // navigator.locks is absent here → each tab runs its own upstream stream.

  it("delivers messages and fires onConnectOrResync on subscribe and each (re)connect", async () => {
    let call = 0;
    const connect = vi.fn((signal: AbortSignal) => {
      call++;
      const n = call;
      return (async function* () {
        if (n === 1) {
          yield { id: 1 };
          return; // stream ends → triggers a reconnect
        }
        yield { id: 2 };
        await openUntilAborted(signal);
      })();
    });

    const stream = createSharedStream<Msg>({
      name: "t3",
      connect,
      encode,
      decode,
      backoffMs: 5,
    });

    const sub = spySubscriber();
    const dispose = stream.subscribe(sub);

    await vi.waitFor(() =>
      expect(sub.onMessage).toHaveBeenCalledWith({ id: 2 }),
    );
    expect(sub.onMessage).toHaveBeenCalledWith({ id: 1 });
    // Once on subscribe, then again on the reconnect gap — the first connect is
    // already covered by the subscribe-time load.
    expect(sub.onConnectOrResync).toHaveBeenCalledTimes(2);

    dispose();
  });

  it("fires onConnectOrResync when a backgrounded tab (hidden past the threshold) returns to the foreground", async () => {
    let hidden = false;
    Object.defineProperty(document, "hidden", {
      configurable: true,
      get: () => hidden,
    });

    const connect = vi.fn((signal: AbortSignal) =>
      (async function* () {
        yield { id: 1 };
        await openUntilAborted(signal);
      })(),
    );
    const stream = createSharedStream<Msg>({
      name: "t4",
      connect,
      encode,
      decode,
      hiddenEligibilityMs: 20,
    });

    const sub = spySubscriber();
    const dispose = stream.subscribe(sub);
    await vi.waitFor(() =>
      expect(sub.onMessage).toHaveBeenCalledWith({ id: 1 }),
    );
    const before = sub.onConnectOrResync.mock.calls.length;

    // Hidden past the threshold, then visible again → catch-up reload.
    hidden = true;
    document.dispatchEvent(new Event("visibilitychange"));
    await new Promise((r) => setTimeout(r, 40));
    hidden = false;
    document.dispatchEvent(new Event("visibilitychange"));

    await vi.waitFor(() =>
      expect(sub.onConnectOrResync.mock.calls.length).toBe(before + 1),
    );

    dispose();
    delete (document as unknown as { hidden?: unknown }).hidden;
  });
});
