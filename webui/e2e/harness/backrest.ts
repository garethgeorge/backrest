import { spawn, type ChildProcess } from 'node:child_process';
import * as fs from 'node:fs/promises';
import * as os from 'node:os';
import * as path from 'node:path';
import { getFreePort } from './ports';

export interface BackrestStartOptions {
  /** Path to the backrest binary. Defaults to $E2E_BACKREST_BIN (set by global-setup). */
  binPath?: string;
  /** Extra environment variables for the backrest process. */
  env?: Record<string, string>;
  /** How long to wait for the HTTP server to respond, per spawn attempt. */
  readinessTimeoutMs?: number;
}

const SPAWN_ATTEMPTS = 3;
const STOP_GRACE_MS = 5_000;

/**
 * A backrest server process with its own temporary data directory and config
 * file, listening on a dedicated free port. One instance per test; obtain one
 * through the `backrest` fixture in ./fixtures.ts.
 */
export class BackrestInstance {
  /** Base URL of the instance, e.g. "http://127.0.0.1:41234". */
  readonly url: string;
  /** Temporary directory holding all state for this instance. */
  readonly dataDir: string;
  /** Path of the instance's config.json. */
  readonly configFile: string;

  private proc: ChildProcess;
  private logBuf: string[] = [];
  private stopped = false;

  private constructor(
    url: string,
    dataDir: string,
    configFile: string,
    proc: ChildProcess,
    logBuf: string[],
  ) {
    this.url = url;
    this.dataDir = dataDir;
    this.configFile = configFile;
    this.proc = proc;
    this.logBuf = logBuf;
  }

  static async start(opts?: BackrestStartOptions): Promise<BackrestInstance> {
    const binPath = opts?.binPath ?? process.env.E2E_BACKREST_BIN;
    if (!binPath) {
      throw new Error(
        'backrest binary path unknown: set E2E_BACKREST_BIN or run via the playwright config (global-setup builds it)',
      );
    }
    const readinessTimeoutMs = opts?.readinessTimeoutMs ?? 15_000;

    const dataDir = await fs.mkdtemp(path.join(os.tmpdir(), 'backrest-e2e-'));
    const configFile = path.join(dataDir, 'config.json');

    let lastError: unknown;
    for (let attempt = 1; attempt <= SPAWN_ATTEMPTS; attempt++) {
      const port = await getFreePort();
      const addr = `127.0.0.1:${port}`;
      const url = `http://${addr}`;
      const logBuf: string[] = [];

      const proc = spawn(
        binPath,
        ['-data-dir', dataDir, '-config-file', configFile, '-bind-address', addr],
        {
          env: { ...process.env, ...opts?.env },
          stdio: ['ignore', 'pipe', 'pipe'],
        },
      );
      proc.stdout!.on('data', (d) => logBuf.push(d.toString()));
      proc.stderr!.on('data', (d) => logBuf.push(d.toString()));

      let exited = false;
      proc.once('exit', () => {
        exited = true;
      });

      // Poll readiness: GET / until HTTP 200 (mirrors test/e2e/first_run_test.go).
      const deadline = Date.now() + readinessTimeoutMs;
      let ready = false;
      while (Date.now() < deadline && !exited) {
        try {
          const resp = await fetch(url + '/');
          if (resp.status === 200) {
            ready = true;
            break;
          }
        } catch {
          // not listening yet
        }
        await sleep(150);
      }

      if (ready) {
        return new BackrestInstance(url, dataDir, configFile, proc, logBuf);
      }

      // Early exit (e.g. lost the bind race for the port) or readiness timeout:
      // kill and retry on a fresh port.
      lastError = new Error(
        `backrest did not become ready on ${addr} (attempt ${attempt}/${SPAWN_ATTEMPTS}, ` +
          `exited=${exited})\n--- logs ---\n${logBuf.join('')}`,
      );
      if (!exited) {
        proc.kill('SIGKILL');
        await waitForExit(proc, STOP_GRACE_MS);
      }
    }

    await fs.rm(dataDir, { recursive: true, force: true }).catch(() => {});
    throw lastError;
  }

  /** SIGTERM, escalating to SIGKILL after 5s; then removes the temp data dir. */
  async stop(): Promise<void> {
    if (this.stopped) return;
    this.stopped = true;
    if (this.proc.exitCode === null && this.proc.signalCode === null) {
      this.proc.kill('SIGTERM');
      const exited = await waitForExit(this.proc, STOP_GRACE_MS);
      if (!exited) {
        this.proc.kill('SIGKILL');
        await waitForExit(this.proc, STOP_GRACE_MS);
      }
    }
    await fs.rm(this.dataDir, { recursive: true, force: true }).catch(() => {});
  }

  /** All captured stdout + stderr so far. */
  logs(): string {
    return this.logBuf.join('');
  }

  /**
   * Writes test files under <dataDir>/source-data. Keys are paths relative to
   * source-data (subdirectories are created); values are file contents.
   * Returns the absolute path of the source-data directory — use it as a
   * backup path in plans.
   */
  async makeTestData(files: Record<string, string>): Promise<string> {
    const root = path.join(this.dataDir, 'source-data');
    await fs.mkdir(root, { recursive: true });
    for (const [rel, content] of Object.entries(files)) {
      const abs = path.join(root, rel);
      await fs.mkdir(path.dirname(abs), { recursive: true });
      await fs.writeFile(abs, content);
    }
    return root;
  }

  /**
   * Absolute path for a local restic repository named `name` under this
   * instance's data dir. The directory is not created; `restic init` (via
   * AddRepo) will create it.
   */
  repoPath(name: string): string {
    return path.join(this.dataDir, 'repos', name);
  }
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function waitForExit(proc: ChildProcess, timeoutMs: number): Promise<boolean> {
  if (proc.exitCode !== null || proc.signalCode !== null) {
    return Promise.resolve(true);
  }
  return new Promise((resolve) => {
    const timer = setTimeout(() => resolve(false), timeoutMs);
    proc.once('exit', () => {
      clearTimeout(timer);
      resolve(true);
    });
  });
}
