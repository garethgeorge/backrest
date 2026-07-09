import * as fs from 'node:fs/promises';
import * as path from 'node:path';
import { create } from '@bufbuild/protobuf';
import { test, expect } from '../harness/fixtures';
import { backrestClient, seedInstance, seedPlan } from '../harness/seed';
import type { BackrestInstance } from '../harness/backrest';
import { BackupRequestSchema, GetOperationsRequestSchema } from '../../gen/ts/v1/service_pb';
import { OperationStatus } from '../../gen/ts/v1/operations_pb';

/**
 * Exercises a non-local restic backend: an rclone-backed repository.
 *
 * restic's rclone backend spawns `rclone serve restic --stdio <remote>:<path>`
 * from the restic process, so the backrest instance under test must have
 * `rclone` on its PATH (the nix dev shell provides it; backrest inherits the
 * spawning process's env). We use the zero-config "on-the-fly" local remote:
 * URI `rclone::local:<abs path>` — the leading colon in `:local:` makes it an
 * unconfigured on-the-fly remote, so no rclone.conf is required. Verified
 * empirically that the cached restic can `init` such a repo. Because the
 * `local` backend just writes to the filesystem, the restic repository
 * structure ends up directly at `<abs path>` and can be asserted Node-side.
 */

const PASSWORD = 'test-password-12345';
const REPO_NAME = 'rclone-repo';
const PLAN_ID = 'rclone-plan';

/**
 * Triggers a real backup for `planId` via the Backup RPC and polls
 * GetOperations until an OperationBackup reaches STATUS_SUCCESS. Fails fast if
 * the backup errors. Mirrors the helper pattern in restore.spec.ts.
 */
async function runBackupViaApi(
  inst: BackrestInstance,
  planId: string,
  timeoutMs = 90_000,
): Promise<void> {
  const client = backrestClient(inst);
  await client.backup(create(BackupRequestSchema, { value: planId }));

  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    const resp = await client.getOperations(
      create(GetOperationsRequestSchema, {
        selector: { planId },
        lastN: 100n,
      }),
    );
    for (const op of resp.operations) {
      if (op.op.case === 'operationBackup' && op.status === OperationStatus.STATUS_SUCCESS) {
        return;
      }
      if (op.op.case === 'operationBackup' && op.status === OperationStatus.STATUS_ERROR) {
        throw new Error(`backup for plan ${planId} failed: ${op.displayMessage}`);
      }
    }
    await new Promise((r) => setTimeout(r, 500));
  }
  throw new Error(`backup for plan ${planId} did not succeed within ${timeoutMs}ms`);
}

test.describe('rclone-backed repo', () => {
  test('adds an rclone repo through the UI, backs up, and is usable end-to-end', async ({
    page,
    backrest,
  }) => {
    // rclone init + test + backup each spawn a `rclone serve restic` process;
    // give the whole flow a generous budget on a possibly-loaded machine.
    test.setTimeout(420_000);

    await seedInstance(backrest);

    // On-the-fly local rclone remote. The restic structure lands directly at
    // repoDir (the `local` backend is a plain filesystem writer).
    const repoDir = path.join(backrest.dataDir, 'rclone-store');
    const uri = `rclone::local:${repoDir}`;

    await page.goto(backrest.url);

    // --- Add Repo UI ------------------------------------------------------
    await page.getByTestId('sidebar-add-repo').click();

    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();

    await dialog.getByTestId('add-repo-name').fill(REPO_NAME);
    await dialog.getByTestId('add-repo-uri').fill(uri);
    // The URI field is a combobox that opens a suggestions popover on input;
    // close it so it doesn't sit on top of the fields/buttons below it.
    await page.keyboard.press('Escape');
    await dialog.getByTestId('add-repo-password').fill(PASSWORD);

    // --- Test Configuration against the rclone URI, before submitting -----
    // messages/en.json: add_repo_modal_test_success_new = "Connected
    // successfully to {uri}. No existing repo found at this location, a new
    // one will be initialized" — the "new repo" variant, since nothing exists
    // at this rclone path yet. Testing spawns `rclone serve restic`, so allow
    // a wide timeout.
    await dialog.getByTestId('add-repo-test-config').click();
    await expect(page.getByText(`Connected successfully to ${uri}`)).toBeVisible({
      timeout: 60_000,
    });

    // Testing configuration must not have created the repo yet.
    await expect(page.getByTestId(`sidebar-item-repo-${REPO_NAME}`)).toHaveCount(0);
    await expect(dialog).toBeVisible();

    // --- Submit: initializes the repo through rclone ----------------------
    await dialog.getByTestId('add-repo-submit').click();

    const sidebarItem = page.getByTestId(`sidebar-item-repo-${REPO_NAME}`);
    // restic init through rclone + AddRepo's GUID lookup can take a while.
    await expect(sidebarItem).toBeVisible({ timeout: 60_000 });
    await expect(page.getByRole('dialog')).toHaveCount(0);

    // --- Prove it's usable: seed a plan + run a real backup via the API ---
    const dataPath = await backrest.makeTestData({
      'hello.txt': 'rclone backup me',
      'nested/deep.txt': 'deep rclone content',
    });
    await seedPlan(backrest, PLAN_ID, REPO_NAME, [dataPath]);

    await runBackupViaApi(backrest, PLAN_ID);

    // --- Assert a successful Backup operation-row in the UI ---------------
    // Assert via the repo view (the repo is in the live sidebar config, unlike
    // the API-seeded plan, whose config the already-loaded SPA hasn't fetched).
    // The oplog subscription + on-mount GetOperations surface the historical
    // Backup operation. The list tab renders OperationRows directly (the tree
    // tab needs a flow selected to reveal rows).
    await sidebarItem.click();
    await expect(page).toHaveURL(/#\/repo\//);
    await expect(page.getByRole('heading', { name: REPO_NAME })).toBeVisible();
    await page.getByRole('tab', { name: 'List View' }).click();

    await expect(
      page.locator('[data-testid="operation-row"][data-op-type="Backup"][data-status="success"]'),
    ).toBeVisible({ timeout: 60_000 });

    // --- Node-side: the restic repo structure exists at the rclone path ---
    const configStat = await fs.stat(path.join(repoDir, 'config'));
    expect(configStat.isFile()).toBe(true);
    const dataStat = await fs.stat(path.join(repoDir, 'data'));
    expect(dataStat.isDirectory()).toBe(true);
  });
});
