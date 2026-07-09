import { execFile, spawn } from 'node:child_process';
import * as fs from 'node:fs/promises';
import * as path from 'node:path';
import { promisify } from 'node:util';
import { getFreePort } from './harness/ports';

const execFileAsync = promisify(execFile);

const WEBUI_DIR = path.resolve(__dirname, '..');
const CACHE_DIR = path.join(WEBUI_DIR, 'e2e', '.cache');
const DIST_MARKER = path.join(CACHE_DIR, 'webui-dist-marker.json');
const BACKREST_BIN = path.join(CACHE_DIR, 'backrest');
const RESTIC_DATA_DIR = path.join(CACHE_DIR, 'restic-data');

export default async function globalSetup() {
  await fs.mkdir(CACHE_DIR, { recursive: true });

  await buildWebuiDist();

  // The backrest binary embeds webui/dist (//go:embed in webuinix.go), so this
  // must come after the dist build.
  const binPath = await buildBackrest();
  process.env.E2E_BACKREST_BIN = binPath;

  // Worker processes inherit process.env set here.
  await provisionRestic(binPath);
}

/**
 * Builds webui/dist (embedded into the backrest binary) unless
 * E2E_SKIP_WEBUI_BUILD=1 or an mtime marker says dist is already fresh.
 */
async function buildWebuiDist(): Promise<void> {
  const distIndex = path.join(WEBUI_DIR, 'dist', 'index.html');

  if (process.env.E2E_SKIP_WEBUI_BUILD === '1') {
    if (!(await exists(distIndex))) {
      throw new Error(
        'E2E_SKIP_WEBUI_BUILD=1 but webui/dist does not exist; run `pnpm run build` once first',
      );
    }
    console.log('[e2e setup] E2E_SKIP_WEBUI_BUILD=1, skipping webui build');
    return;
  }

  const newestMtimeMs = await newestSourceMtime();
  const marker = await readJson<{ newestMtimeMs: number }>(DIST_MARKER);
  if (marker?.newestMtimeMs === newestMtimeMs && (await exists(distIndex))) {
    console.log('[e2e setup] webui dist is fresh, skipping build');
    return;
  }

  console.log('[e2e setup] building webui dist (pnpm run build)...');
  await execFileAsync('pnpm', ['run', 'build'], {
    cwd: WEBUI_DIR,
    maxBuffer: 64 * 1024 * 1024,
  });
  // Recompute: the build may itself touch tracked inputs (e.g. src/paraglide).
  await fs.writeFile(DIST_MARKER, JSON.stringify({ newestMtimeMs: await newestSourceMtime() }));
}

/** Newest mtime across the inputs that feed `pnpm run build`. */
async function newestSourceMtime(): Promise<number> {
  const roots = [
    'src',
    'gen',
    'assets',
    'messages',
    'index.html',
    'vite.config.ts',
    'package.json',
  ].map((p) => path.join(WEBUI_DIR, p));

  let newest = 0;
  const walk = async (p: string): Promise<void> => {
    let stat;
    try {
      stat = await fs.stat(p);
    } catch {
      return;
    }
    if (stat.isDirectory()) {
      const entries = await fs.readdir(p);
      await Promise.all(entries.map((e) => walk(path.join(p, e))));
    } else if (stat.mtimeMs > newest) {
      newest = stat.mtimeMs;
    }
  };
  await Promise.all(roots.map(walk));
  return newest;
}

/** Builds the backrest binary, or uses $E2E_BACKREST_BIN if provided. */
async function buildBackrest(): Promise<string> {
  if (process.env.E2E_BACKREST_BIN) {
    const provided = path.resolve(WEBUI_DIR, process.env.E2E_BACKREST_BIN);
    if (!(await exists(provided))) {
      throw new Error(`E2E_BACKREST_BIN=${provided} does not exist`);
    }
    console.log(`[e2e setup] using provided backrest binary: ${provided}`);
    return provided;
  }

  console.log('[e2e setup] building backrest binary (go build)...');
  await execFileAsync('go', ['build', '-o', BACKREST_BIN, '../cmd/backrest'], {
    cwd: WEBUI_DIR,
    maxBuffer: 64 * 1024 * 1024,
  });
  return BACKREST_BIN;
}

/**
 * Ensures a restic binary is available and exported as
 * BACKREST_RESTIC_COMMAND so per-test instances never download restic
 * themselves. If not already provided (env var or cached from a previous
 * run), boots a throwaway backrest instance: on startup backrest downloads
 * the pinned restic version into its data dir.
 */
async function provisionRestic(binPath: string): Promise<void> {
  if (process.env.BACKREST_RESTIC_COMMAND) {
    console.log(`[e2e setup] using BACKREST_RESTIC_COMMAND=${process.env.BACKREST_RESTIC_COMMAND}`);
    return;
  }

  const resticPath = path.join(RESTIC_DATA_DIR, 'restic');
  if (await isExecutable(resticPath)) {
    process.env.BACKREST_RESTIC_COMMAND = resticPath;
    console.log(`[e2e setup] using cached restic: ${resticPath}`);
    return;
  }

  console.log(
    '[e2e setup] provisioning restic via a throwaway backrest instance (downloads on first run)...',
  );
  await fs.mkdir(RESTIC_DATA_DIR, { recursive: true });
  const port = await getFreePort();
  const logs: string[] = [];
  const proc = spawn(
    binPath,
    [
      '-data-dir',
      RESTIC_DATA_DIR,
      '-config-file',
      path.join(RESTIC_DATA_DIR, 'config.json'),
      '-bind-address',
      `127.0.0.1:${port}`,
    ],
    { stdio: ['ignore', 'pipe', 'pipe'] },
  );
  proc.stdout!.on('data', (d) => logs.push(d.toString()));
  proc.stderr!.on('data', (d) => logs.push(d.toString()));
  let exited = false;
  proc.once('exit', () => {
    exited = true;
  });

  try {
    const deadline = Date.now() + 240_000;
    while (!(await isExecutable(resticPath))) {
      if (exited) {
        throw new Error(
          `throwaway backrest exited before installing restic\n--- logs ---\n${logs.join('')}`,
        );
      }
      if (Date.now() > deadline) {
        throw new Error(
          `timed out waiting for restic to be installed at ${resticPath}\n--- logs ---\n${logs.join('')}`,
        );
      }
      await new Promise((r) => setTimeout(r, 500));
    }
  } finally {
    if (!exited) {
      proc.kill('SIGTERM');
      await new Promise<void>((resolve) => {
        const t = setTimeout(() => {
          proc.kill('SIGKILL');
          resolve();
        }, 5_000);
        proc.once('exit', () => {
          clearTimeout(t);
          resolve();
        });
      });
    }
  }

  process.env.BACKREST_RESTIC_COMMAND = resticPath;
  console.log(`[e2e setup] restic installed at ${resticPath}`);
}

async function exists(p: string): Promise<boolean> {
  try {
    await fs.access(p);
    return true;
  } catch {
    return false;
  }
}

async function isExecutable(p: string): Promise<boolean> {
  try {
    await fs.access(p, (await import('node:fs')).constants.X_OK);
    return true;
  } catch {
    return false;
  }
}

async function readJson<T>(p: string): Promise<T | undefined> {
  try {
    return JSON.parse(await fs.readFile(p, 'utf8')) as T;
  } catch {
    return undefined;
  }
}
