import * as fs from 'node:fs/promises';
import * as path from 'node:path';
import { create } from '@bufbuild/protobuf';
import { test, expect } from '../harness/fixtures';
import { backrestClient, seedInstance, seedRepo, seedPlan } from '../harness/seed';
import type { BackrestInstance } from '../harness/backrest';
import { BackupRequestSchema, GetOperationsRequestSchema } from '../../gen/ts/v1/service_pb';
import { OperationStatus } from '../../gen/ts/v1/operations_pb';

/**
 * Triggers a real backup for `planId` via the Backup RPC, then polls
 * GetOperations until the flow has both a successful OperationBackup and an
 * indexed OperationIndexSnapshot (the snapshot the UI browses/restores from).
 * Returns the snapshot id it observed.
 */
async function runBackupViaApi(
  inst: BackrestInstance,
  planId: string,
  timeoutMs = 60_000,
): Promise<string> {
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

    let backupOk = false;
    let snapshotId: string | undefined;
    for (const op of resp.operations) {
      if (op.op.case === 'operationBackup' && op.status === OperationStatus.STATUS_SUCCESS) {
        backupOk = true;
      }
      if (op.op.case === 'operationIndexSnapshot') {
        snapshotId = op.op.value.snapshot?.id;
      }
      // If the backup errored, fail fast with a useful message.
      if (op.op.case === 'operationBackup' && op.status === OperationStatus.STATUS_ERROR) {
        throw new Error(`backup for plan ${planId} failed: ${op.displayMessage}`);
      }
    }

    if (backupOk && snapshotId) {
      return snapshotId;
    }
    await new Promise((r) => setTimeout(r, 500));
  }
  throw new Error(
    `backup for plan ${planId} did not complete (with an indexed snapshot) within ${timeoutMs}ms`,
  );
}

/** Recursively finds the first file named `name` under `dir`, or null. */
async function findFile(dir: string, name: string): Promise<string | null> {
  let entries: import('node:fs').Dirent[];
  try {
    entries = await fs.readdir(dir, { withFileTypes: true });
  } catch {
    return null;
  }
  for (const e of entries) {
    const full = path.join(dir, e.name);
    if (e.isDirectory()) {
      const found = await findFile(full, name);
      if (found) return found;
    } else if (e.isFile() && e.name === name) {
      return full;
    }
  }
  return null;
}

test.describe('restore', () => {
  test('backs up a plan then restores a file from its snapshot', async ({ page, backrest }) => {
    test.setTimeout(180_000);

    // --- Node-side setup: seed config + run a real backup. -----------------
    await seedInstance(backrest);
    await seedRepo(backrest, 'local-repo');
    const dataPath = await backrest.makeTestData({
      'hello.txt': 'restore me',
      'nested/deep.txt': 'deep content',
    });
    await seedPlan(backrest, 'my-plan', 'local-repo', [dataPath]);

    const snapshotId = await runBackupViaApi(backrest, 'my-plan');
    expect(snapshotId).toBeTruthy();

    // --- Browser: open the plan, find the snapshot, browse + restore. ------
    await page.goto(`${backrest.url}/#/plan/my-plan`);

    // The tree tab needs a flow selected to reveal rows; the list tab renders
    // OperationRows directly, which is the surface we drive.
    await page.getByRole('tab', { name: 'List View' }).click();

    const snapshotRow = page.locator('[data-testid="operation-row"][data-op-type="Snapshot"]');
    await expect(snapshotRow).toBeVisible({ timeout: 30_000 });

    // Expand the lazily-mounted snapshot browser accordion within that row.
    await snapshotRow.getByText('Snapshot Browser').click();

    // The browser auto-loads and auto-expands the root ("/"). Navigate down to
    // hello.txt. restic paths are absolute, so the file lives under the source
    // data dir's full path; expand each directory segment until the file shows
    // (in case ls returns one directory level at a time).
    const fileLoc = page.getByTestId('snapshot-browser-entry').filter({ hasText: 'hello.txt' });
    const segments = dataPath.split('/').filter(Boolean);
    for (const seg of segments) {
      if ((await fileLoc.count()) > 0) break;
      const dir = page.getByTestId('snapshot-browser-entry').filter({ hasText: seg }).first();
      await dir.waitFor({ state: 'visible', timeout: 20_000 });
      await dir.click();
      // Give the lazy ListSnapshotFiles fetch a moment to populate children.
      await page.waitForTimeout(750);
    }
    await expect(fileLoc.first()).toBeVisible({ timeout: 20_000 });

    // Open the file's row menu and trigger "Restore to path".
    await fileLoc.first().getByRole('button').click();
    await page.getByTestId('snapshot-restore').click();

    // Restore dialog: set a fresh target dir under the instance data dir.
    const restoreTarget = path.join(backrest.dataDir, 'restore-target');
    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();
    const targetInput = dialog.locator('input').first();
    await targetInput.click();
    await targetInput.fill(restoreTarget);
    // Close any open autocomplete popup without dismissing the dialog.
    await page.keyboard.press('Escape');
    await expect(dialog).toBeVisible();

    // ConfirmButton: first click arms it, second click confirms.
    await dialog.getByRole('button', { name: 'Restore' }).click();
    await dialog.getByRole('button', { name: 'Confirm Restore?' }).click();

    // --- Assert: a Restore operation row reaches success. ------------------
    const restoreRow = page.locator(
      '[data-testid="operation-row"][data-op-type="Restore"][data-status="success"]',
    );
    await expect(restoreRow).toBeVisible({ timeout: 60_000 });

    // --- Node-side fs check: the restored file exists with expected content.
    const restored = await findFile(restoreTarget, 'hello.txt');
    expect(restored, `hello.txt not found under ${restoreTarget}`).toBeTruthy();
    expect(await fs.readFile(restored!, 'utf8')).toBe('restore me');
  });
});
