import { spawn, execFile, type ChildProcess } from 'node:child_process';
import { constants as fsConstants } from 'node:fs';
import * as fs from 'node:fs/promises';
import * as net from 'node:net';
import * as os from 'node:os';
import * as path from 'node:path';
import { promisify } from 'node:util';
import { getFreePort } from './ports';

const execFileAsync = promisify(execFile);

const SPAWN_ATTEMPTS = 3;
const READY_TIMEOUT_MS = 10_000;
const STOP_GRACE_MS = 5_000;

/**
 * A local, unprivileged, SFTP-only OpenSSH server for e2e tests.
 *
 * Design:
 * - Everything lives in one temp dir: an ssh-keygen-generated ed25519 host
 *   key, an ed25519 client keypair (pre-authorized via authorized_keys), a
 *   ready-to-use known_hosts file for the server's host key, and a `repos/`
 *   directory intended to hold restic repositories.
 * - sshd runs as the current user in the foreground (`sshd -D -e -f <config>`)
 *   on a free port on 127.0.0.1. `Subsystem sftp internal-sftp` serves SFTP
 *   in-process (no sftp-server binary needed), `StrictModes no` tolerates
 *   temp-dir permissions, and `PidFile none` avoids writing a pidfile.
 * - The sshd binary is resolved from PATH plus NixOS/system sbin locations
 *   (sshd must be spawned via an absolute path because it re-executes itself).
 *
 * Obtain one with `SftpServer.start()`; always `stop()` it in a finally block
 * or fixture teardown.
 */
export class SftpServer {
  /** Always 127.0.0.1. */
  readonly host = '127.0.0.1';
  readonly port: number;
  /** The user sshd authenticates (the user running the tests). */
  readonly user: string;
  /** Temp dir holding keys, config, logs, and the repos/ root. */
  readonly root: string;
  /** Directory intended to hold restic repos served over sftp (root/repos). */
  readonly reposDir: string;
  /** Private key pre-authorized in authorized_keys (no passphrase). */
  readonly clientKeyPath: string;
  /** known_hosts file containing the correct entry for this server. */
  readonly knownHostsPath: string;
  /** The file sshd reads authorized public keys from (appendable). */
  readonly authorizedKeysPath: string;
  /** Contents of the server host public key (authorized_keys format). */
  readonly hostPublicKey: string;

  private proc: ChildProcess;
  private logBuf: string[];
  private stopped = false;

  private constructor(opts: {
    port: number;
    user: string;
    root: string;
    proc: ChildProcess;
    logBuf: string[];
    hostPublicKey: string;
  }) {
    this.port = opts.port;
    this.user = opts.user;
    this.root = opts.root;
    this.reposDir = path.join(opts.root, 'repos');
    this.clientKeyPath = path.join(opts.root, 'client_key');
    this.knownHostsPath = path.join(opts.root, 'known_hosts');
    this.authorizedKeysPath = path.join(opts.root, 'authorized_keys');
    this.hostPublicKey = opts.hostPublicKey;
    this.proc = opts.proc;
    this.logBuf = opts.logBuf;
  }

  static async start(): Promise<SftpServer> {
    const sshdPath = await resolveBinary('sshd');
    const sshKeygenPath = await resolveBinary('ssh-keygen');
    const user = os.userInfo().username;

    const root = await fs.mkdtemp(path.join(os.tmpdir(), 'backrest-e2e-sftpd-'));
    try {
      const hostKeyPath = path.join(root, 'host_key');
      const clientKeyPath = path.join(root, 'client_key');
      await execFileAsync(sshKeygenPath, ['-q', '-t', 'ed25519', '-N', '', '-f', hostKeyPath]);
      await execFileAsync(sshKeygenPath, ['-q', '-t', 'ed25519', '-N', '', '-f', clientKeyPath]);

      const hostPublicKey = (await fs.readFile(hostKeyPath + '.pub', 'utf8')).trim();
      const clientPublicKey = (await fs.readFile(clientKeyPath + '.pub', 'utf8')).trim();

      const authorizedKeysPath = path.join(root, 'authorized_keys');
      await fs.writeFile(authorizedKeysPath, clientPublicKey + '\n', { mode: 0o600 });

      await fs.mkdir(path.join(root, 'repos'), { recursive: true });

      let lastError: unknown;
      for (let attempt = 1; attempt <= SPAWN_ATTEMPTS; attempt++) {
        const port = await getFreePort();

        const configPath = path.join(root, 'sshd_config');
        await fs.writeFile(
          configPath,
          [
            `Port ${port}`,
            'ListenAddress 127.0.0.1',
            `HostKey ${hostKeyPath}`,
            'PidFile none',
            'PubkeyAuthentication yes',
            'PasswordAuthentication no',
            'KbdInteractiveAuthentication no',
            'UsePAM no',
            'StrictModes no',
            `AuthorizedKeysFile ${authorizedKeysPath}`,
            'Subsystem sftp internal-sftp',
            'LogLevel VERBOSE',
            '',
          ].join('\n'),
        );

        const logBuf: string[] = [];
        // -D: no daemonize, -e: log to stderr. sshd must be invoked with an
        // absolute path (it re-executes itself for each connection).
        const proc = spawn(sshdPath, ['-D', '-e', '-f', configPath], {
          stdio: ['ignore', 'pipe', 'pipe'],
        });
        proc.stdout!.on('data', (d) => logBuf.push(d.toString()));
        proc.stderr!.on('data', (d) => logBuf.push(d.toString()));
        let exited = false;
        proc.once('exit', () => {
          exited = true;
        });

        const ready = await waitForSshBanner(port, READY_TIMEOUT_MS, () => exited);
        if (ready) {
          // known_hosts entry for the non-default port form ssh expects.
          await fs.writeFile(
            path.join(root, 'known_hosts'),
            `[127.0.0.1]:${port} ${hostPublicKey}\n`,
          );
          return new SftpServer({ port, user, root, proc, logBuf, hostPublicKey });
        }

        // Lost the port race or failed to start: kill and retry on a new port.
        lastError = new Error(
          `sshd did not become ready on 127.0.0.1:${port} ` +
            `(attempt ${attempt}/${SPAWN_ATTEMPTS}, exited=${exited})\n--- sshd logs ---\n` +
            logBuf.join(''),
        );
        if (!exited) {
          proc.kill('SIGKILL');
          await waitForExit(proc, STOP_GRACE_MS);
        }
      }
      throw lastError;
    } catch (err) {
      await fs.rm(root, { recursive: true, force: true }).catch(() => {});
      throw err;
    }
  }

  /** Appends a public key (authorized_keys format) to authorized_keys. */
  async authorizeKey(publicKey: string): Promise<void> {
    await fs.appendFile(this.authorizedKeysPath, publicKey.trim() + '\n');
  }

  /** All captured sshd stderr/stdout so far. */
  logs(): string {
    return this.logBuf.join('');
  }

  /** SIGTERM (escalating to SIGKILL), then removes the temp dir. */
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
    await fs.rm(this.root, { recursive: true, force: true }).catch(() => {});
  }
}

/**
 * Resolves an executable from PATH plus common system sbin locations (on
 * NixOS, sshd lives at /run/current-system/sw/bin which is normally on PATH
 * inside the dev shell; sbin fallbacks cover conventional Linux distros).
 */
async function resolveBinary(name: string): Promise<string> {
  const dirs = (process.env.PATH ?? '').split(path.delimiter).filter(Boolean);
  dirs.push('/run/current-system/sw/bin', '/usr/sbin', '/sbin', '/usr/local/sbin');
  for (const dir of dirs) {
    const candidate = path.join(dir, name);
    try {
      await fs.access(candidate, fsConstants.X_OK);
      return candidate;
    } catch {
      // keep looking
    }
  }
  throw new Error(
    `${name} not found on PATH or in system sbin directories; ` +
      'an OpenSSH server is required for the sftp e2e tests',
  );
}

/** Polls the port until an SSH identification banner ("SSH-...") is served. */
function waitForSshBanner(
  port: number,
  timeoutMs: number,
  hasExited: () => boolean,
): Promise<boolean> {
  const deadline = Date.now() + timeoutMs;
  return (async () => {
    while (Date.now() < deadline && !hasExited()) {
      const ok = await new Promise<boolean>((resolve) => {
        const sock = net.connect({ host: '127.0.0.1', port });
        let settled = false;
        const finish = (value: boolean) => {
          if (settled) return;
          settled = true;
          sock.destroy();
          resolve(value);
        };
        sock.setTimeout(2_000, () => finish(false));
        sock.once('error', () => finish(false));
        sock.once('data', (data) => finish(data.toString().startsWith('SSH-')));
      });
      if (ok) return true;
      await sleep(100);
    }
    return false;
  })();
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
