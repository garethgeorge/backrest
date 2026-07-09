import * as fs from 'node:fs/promises';
import * as os from 'node:os';
import * as path from 'node:path';
import type { Locator } from '@playwright/test';
import { test as harnessTest, expect } from '../harness/fixtures';
import { BackrestInstance } from '../harness/backrest';
import { seedInstance, seedPlan } from '../harness/seed';
import { SftpServer } from '../harness/sftpserver';

/**
 * SFTP-backed repository e2e coverage.
 *
 * Each test gets a private, unprivileged OpenSSH server (harness/sftpserver.ts:
 * internal-sftp on a free localhost port, key-only auth). The happy-path test
 * drives the full Add Repo UI flow for an sftp repo, including the modal's
 * "Setup SSH Key" helper (the SetupSftp RPC: generates a keypair under the
 * instance's <dataDir>/.backrest-ssh and ssh-keyscans the host into a
 * known_hosts file there), then proves restic data physically landed in the
 * sftp-served directory.
 *
 * URI shape: restic's scp-style form `sftp:user@host:/abs/path` cannot carry a
 * port, so the modal's "SFTP Port" field is used; it contributes `-p <port>`
 * to the `--option=sftp.args='...'` flag the modal composes (along with
 * -oBatchMode=yes, -i <identity>, -oUserKnownHostsFile=<path>). Server-side,
 * sanitizeRepoFlags (internal/api/backresthandler.go) strips the double quotes
 * the UI wraps paths in — restic's --option CSV parser rejects bare `"` — and
 * shlex splitting in NewRepoOrchestrator strips the wrapping single quotes, so
 * restic receives a clean `sftp.args=...` value.
 *
 * The backrest instance for the happy path runs with HOME pointed at a
 * test-owned temp dir so any ~/.ssh or restic-cache writes stay hermetic.
 */

const PASSWORD = 'test-password-12345';

interface SftpFixtures {
  /** A fresh sftp-only sshd for this test. */
  sftp: SftpServer;
  /**
   * A backrest instance whose HOME is redirected to a temp dir (hermetic
   * ~/.ssh + restic cache). Used instead of `backrest` when the test performs
   * real ssh traffic.
   */
  sftpBackrest: BackrestInstance;
}

const test = harnessTest.extend<SftpFixtures>({
  sftp: async ({}, use, testInfo) => {
    const server = await SftpServer.start();
    try {
      await use(server);
    } finally {
      if (testInfo.status !== testInfo.expectedStatus) {
        await testInfo.attach('sshd-logs', {
          body: server.logs(),
          contentType: 'text/plain',
        });
      }
      await server.stop();
    }
  },
  sftpBackrest: async ({}, use, testInfo) => {
    const home = await fs.mkdtemp(path.join(os.tmpdir(), 'backrest-e2e-home-'));
    const instance = await BackrestInstance.start({ env: { HOME: home } });
    try {
      await use(instance);
    } finally {
      if (testInfo.status !== testInfo.expectedStatus) {
        await testInfo.attach('backrest-logs', {
          body: instance.logs(),
          contentType: 'text/plain',
        });
      }
      await instance.stop();
      await fs.rm(home, { recursive: true, force: true }).catch(() => {});
    }
  },
});

/**
 * The SFTP section's inputs carry data-testids (added in AddRepoModal.tsx).
 * The port field is a Chakra NumberInput whose testid lands on the wrapping
 * Root element, so reach through to the nested <input>.
 */
const sftpPortInput = (dialog: Locator): Locator =>
  dialog.getByTestId('add-repo-sftp-port').locator('input');
const identityInputOf = (dialog: Locator) => dialog.getByTestId('add-repo-sftp-identity');
const knownHostsInputOf = (dialog: Locator) => dialog.getByTestId('add-repo-sftp-known-hosts');

test.describe('sftp-backed repository', () => {
  test('adds an sftp repo through the UI (Setup SSH Key flow) and backs up to it', async ({
    page,
    sftp,
    sftpBackrest: backrest,
  }) => {
    await seedInstance(backrest);

    const repoDir = path.join(sftp.reposDir, 'repo1');
    const uri = `sftp:${sftp.user}@${sftp.host}:${repoDir}`;

    await page.goto(backrest.url);
    await page.getByTestId('sidebar-add-repo').click();
    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();

    await dialog.getByTestId('add-repo-name').fill('sftp-repo');
    await dialog.getByTestId('add-repo-uri').fill(uri);
    // Dismiss the URI autocomplete popover (if it opened) by refocusing the
    // name field. Do NOT press Escape here: for sftp: URIs the popover often
    // has no suggestions and never opens, and Escape then closes the dialog.
    await dialog.getByTestId('add-repo-name').click();
    await dialog.getByTestId('add-repo-password').fill(PASSWORD);

    // The SFTP helper section appears for sftp: URIs. Set the port first so
    // the Setup SSH Key helper keyscans the right endpoint (it falls back to
    // 22 otherwise, since the scp-style URI carries no port).
    await sftpPortInput(dialog).fill(String(sftp.port));

    // "Setup SSH Key" = the SetupSftp RPC: generates an ed25519 keypair under
    // <dataDir>/.backrest-ssh/ and ssh-keyscans the host key into
    // <dataDir>/.backrest-ssh/known_hosts, then fills both path fields.
    await dialog.getByText('Setup SSH Key (Optional)').click();
    await dialog.getByTestId('add-repo-sftp-generate-key').click();

    await expect(dialog.getByText('Key Generated Successfully!')).toBeVisible({
      timeout: 15_000,
    });
    // The keyscan against our live sshd must have succeeded — no warning box.
    await expect(dialog.getByText('Host key scan failed')).toHaveCount(0);

    const identityInput = identityInputOf(dialog);
    await expect(identityInput).toHaveValue(/\.backrest-ssh[\\/]id_ed25519_/);
    await expect(knownHostsInputOf(dialog)).toHaveValue(/\.backrest-ssh[\\/]known_hosts$/);

    // Complete the loop a real user performs out-of-band: authorize the
    // generated public key on the server.
    const generatedKeyPath = await identityInput.inputValue();
    const generatedPublicKey = await fs.readFile(generatedKeyPath + '.pub', 'utf8');
    await sftp.authorizeKey(generatedPublicKey);

    // Test Configuration performs a real `restic cat config` over sftp; the
    // repo does not exist yet, so the "new repo" variant must appear.
    await dialog.getByTestId('add-repo-test-config').click();
    await expect(page.getByText(`Connected successfully to ${uri}`)).toBeVisible({
      timeout: 30_000,
    });
    await expect(dialog).toBeVisible();

    // Submit initializes the repo over sftp (restic init + cat config).
    await dialog.getByTestId('add-repo-submit').click();
    await expect(page.getByTestId('sidebar-item-repo-sftp-repo')).toBeVisible({
      timeout: 30_000,
    });
    await expect(page.getByRole('dialog')).toHaveCount(0);

    // Restic data must physically exist in the sftp-served directory.
    await expect(async () => {
      const stat = await fs.stat(path.join(repoDir, 'config'));
      expect(stat.isFile()).toBe(true);
    }).toPass({ timeout: 10_000 });
    for (const sub of ['data', 'keys', 'snapshots']) {
      expect((await fs.stat(path.join(repoDir, sub))).isDirectory()).toBe(true);
    }

    // Back up a small dataset to the sftp repo (plan seeded via RPC with the
    // schedule disabled; backup triggered through the UI).
    const dataPath = await backrest.makeTestData({
      'hello.txt': 'hello over sftp',
      'nested/world.txt': 'nested content',
    });
    await seedPlan(backrest, 'sftp-plan', 'sftp-repo', [dataPath]);

    // The page is already loaded, so a hash-only goto is a same-document
    // navigation and the SPA would keep its pre-seedPlan config; reload to
    // pick up the RPC-seeded plan.
    await page.goto(`${backrest.url}/#/plan/sftp-plan`);
    await page.reload();
    await expect(page.getByTestId('plan-backup-now')).toBeVisible();
    await page.getByTestId('plan-backup-now').click();
    await page.getByRole('tab', { name: 'List View' }).click();

    await expect(
      page.locator('[data-testid="operation-row"][data-op-type="Backup"][data-status="success"]'),
    ).toBeVisible({ timeout: 60_000 });
    const snapshotRow = page
      .locator('[data-testid="operation-row"][data-op-type="Snapshot"]')
      .first();
    await expect(snapshotRow).toBeVisible({ timeout: 60_000 });
    await expect(snapshotRow).toHaveAttribute('data-status', 'success', { timeout: 60_000 });

    // The backup's blobs and snapshot record landed under the sftp root.
    const dataEntries = await fs.readdir(path.join(repoDir, 'data'));
    expect(dataEntries.length).toBeGreaterThan(0);
    const snapshotEntries = await fs.readdir(path.join(repoDir, 'snapshots'));
    expect(snapshotEntries.length).toBe(1);
  });

  test('unreachable sftp port: Test Configuration surfaces an error and the dialog stays open', async ({
    page,
    backrest,
  }) => {
    await seedInstance(backrest);

    // Port 1 is privileged, so no test process can ever bind it: ssh fails
    // fast and deterministically with connection refused. (An ephemeral
    // "free" port is racy here — a concurrently starting test server could
    // bind it, accept the TCP connection, and leave ssh hanging for an SSH
    // banner that never comes.)
    const deadPort = 1;
    const uri = `sftp:${os.userInfo().username}@127.0.0.1:/tmp/backrest-e2e-no-such-repo`;

    await page.goto(backrest.url);
    await page.getByTestId('sidebar-add-repo').click();
    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();

    await dialog.getByTestId('add-repo-name').fill('sftp-bad');
    await dialog.getByTestId('add-repo-uri').fill(uri);
    // See the happy-path test: refocus instead of Escape to avoid closing the
    // dialog when the autocomplete popover never opened.
    await dialog.getByTestId('add-repo-name').click();
    await dialog.getByTestId('add-repo-password').fill(PASSWORD);
    await sftpPortInput(dialog).fill(String(deadPort));

    await dialog.getByTestId('add-repo-test-config').click();

    // en.json add_repo_modal_test_error = "Check error: ".
    await expect(page.getByText(/Check error/).first()).toBeVisible({ timeout: 45_000 });
    await expect(dialog).toBeVisible();
    await expect(page.getByTestId('sidebar-item-repo-sftp-bad')).toHaveCount(0);
  });

  test('wrong host key: Test Configuration surfaces the friendly host-key flow, not a generic error', async ({
    page,
    backrest,
    sftp,
  }) => {
    await seedInstance(backrest);

    // known_hosts with a mismatched key for exactly this host:port — ssh must
    // hard-fail host key verification even though the identity is valid.
    const clientPub = (await fs.readFile(sftp.clientKeyPath + '.pub', 'utf8')).trim();
    const bogusKnownHosts = path.join(sftp.root, 'bogus_known_hosts');
    await fs.writeFile(bogusKnownHosts, `[${sftp.host}]:${sftp.port} ${clientPub}\n`);

    const repoDir = path.join(sftp.reposDir, 'repo-badhostkey');
    const uri = `sftp:${sftp.user}@${sftp.host}:${repoDir}`;

    await page.goto(backrest.url);
    await page.getByTestId('sidebar-add-repo').click();
    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();

    await dialog.getByTestId('add-repo-name').fill('sftp-badkey');
    await dialog.getByTestId('add-repo-uri').fill(uri);
    // See the happy-path test: refocus instead of Escape to avoid closing the
    // dialog when the autocomplete popover never opened.
    await dialog.getByTestId('add-repo-name').click();
    await dialog.getByTestId('add-repo-password').fill(PASSWORD);
    await sftpPortInput(dialog).fill(String(sftp.port));
    await identityInputOf(dialog).fill(sftp.clientKeyPath);
    await knownHostsInputOf(dialog).fill(bogusKnownHosts);

    await dialog.getByTestId('add-repo-test-config').click();

    // The backend now recognizes the ssh host-key verification failure and the
    // UI raises the friendly "Unknown SFTP Host Key" confirmation instead of a
    // generic "Check error" toast.
    await expect(page.getByText('Unknown SFTP Host Key')).toBeVisible({ timeout: 45_000 });
    await expect(page.getByText(/Check error/)).toHaveCount(0);
    await expect(page.getByTestId('sidebar-item-repo-sftp-badkey')).toHaveCount(0);

    // Nothing was created on the server side.
    await expect(fs.stat(repoDir)).rejects.toThrow();
  });

  test('URL-style sftp URI: Setup SSH Key parses host/port and prefills the port field', async ({
    page,
    sftp,
    sftpBackrest: backrest,
  }) => {
    await seedInstance(backrest);

    // URL-style restic form carries the port in the authority. The modal's
    // Setup SSH Key helper must parse host+port out of it (previously it parsed
    // an empty host for this shape) and keyscan the right endpoint.
    const repoDir = path.join(sftp.reposDir, 'repo-urlstyle');
    const uri = `sftp://${sftp.user}@${sftp.host}:${sftp.port}${repoDir}`;

    await page.goto(backrest.url);
    await page.getByTestId('sidebar-add-repo').click();
    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();

    await dialog.getByTestId('add-repo-name').fill('sftp-urlstyle');
    await dialog.getByTestId('add-repo-uri').fill(uri);
    await dialog.getByTestId('add-repo-name').click();
    await dialog.getByTestId('add-repo-password').fill(PASSWORD);

    // Deliberately do NOT set the SFTP Port field: the port must be parsed from
    // the URL-style URI and prefilled by the helper.
    await dialog.getByText('Setup SSH Key (Optional)').click();
    await dialog.getByTestId('add-repo-sftp-generate-key').click();

    await expect(dialog.getByText('Key Generated Successfully!')).toBeVisible({
      timeout: 15_000,
    });
    // A successful keyscan (no warning box) proves the parsed host+port were
    // correct — an empty host would have failed the scan.
    await expect(dialog.getByText('Host key scan failed')).toHaveCount(0);

    // The port carried by the URL prefilled the SFTP Port field.
    await expect(sftpPortInput(dialog)).toHaveValue(String(sftp.port));
    await expect(identityInputOf(dialog)).toHaveValue(/\.backrest-ssh[\\/]id_ed25519_/);
    await expect(knownHostsInputOf(dialog)).toHaveValue(/\.backrest-ssh[\\/]known_hosts$/);
  });
});
