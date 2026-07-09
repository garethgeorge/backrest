import * as fs from 'node:fs/promises';
import * as os from 'node:os';
import * as path from 'node:path';
import { execFile } from 'node:child_process';
import { promisify } from 'node:util';
import { create } from '@bufbuild/protobuf';
import { test, expect } from '../harness/fixtures';
import { backrestClient, seedInstance, seedRepo, seedPlan } from '../harness/seed';
import type { BackrestInstance } from '../harness/backrest';
import type { Download } from '@playwright/test';
import {
  BackupRequestSchema,
  GetOperationsRequestSchema,
  ListSnapshotsRequestSchema,
} from '../../gen/ts/v1/service_pb';
import { OperationStatus } from '../../gen/ts/v1/operations_pb';

const execFileAsync = promisify(execFile);

/**
 * Triggers a real backup for `planId` via the Backup RPC, then polls
 * GetOperations until the flow has both a successful OperationBackup and an
 * indexed OperationIndexSnapshot. Returns the newest indexed snapshot id it
 * observed (operations come back oldest-first, so the last one wins).
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

    let backupCount = 0;
    let snapshotCount = 0;
    let snapshotId: string | undefined;
    for (const op of resp.operations) {
      if (op.op.case === 'operationBackup' && op.status === OperationStatus.STATUS_SUCCESS) {
        backupCount++;
      }
      if (op.op.case === 'operationIndexSnapshot') {
        snapshotCount++;
        snapshotId = op.op.value.snapshot?.id;
      }
      if (op.op.case === 'operationBackup' && op.status === OperationStatus.STATUS_ERROR) {
        throw new Error(`backup for plan ${planId} failed: ${op.displayMessage}`);
      }
    }

    // Only settle once the newly-triggered backup has produced its snapshot.
    if (backupCount >= 1 && snapshotCount >= 1 && snapshotId) {
      return snapshotId;
    }
    await new Promise((r) => setTimeout(r, 500));
  }
  throw new Error(
    `backup for plan ${planId} did not complete (with an indexed snapshot) within ${timeoutMs}ms`,
  );
}

/** Snapshot ids sorted oldest -> newest, as reported by the ListSnapshots RPC. */
async function listSnapshotIds(inst: BackrestInstance, repoId: string, planId: string) {
  const client = backrestClient(inst);
  const list = await client.listSnapshots(create(ListSnapshotsRequestSchema, { repoId, planId }));
  return [...list.snapshots].sort((a, b) => Number(a.unixTimeMs - b.unixTimeMs)).map((s) => s.id);
}

/**
 * Navigates the (already-mounted, auto-expanding-from-root) Snapshot Browser
 * accordion down to `fileName`, expanding each absolute-path directory segment
 * of `dataPath` until the file row shows up. Mirrors restore.spec.ts.
 */
async function browseToFile(page: any, dataPath: string, fileName: string) {
  const fileLoc = page.getByTestId('snapshot-browser-entry').filter({ hasText: fileName });
  const segments = dataPath.split('/').filter(Boolean);
  for (const seg of segments) {
    if ((await fileLoc.count()) > 0) break;
    const dir = page.getByTestId('snapshot-browser-entry').filter({ hasText: seg }).first();
    await dir.waitFor({ state: 'visible', timeout: 20_000 });
    await dir.click();
    await page.waitForTimeout(750);
  }
  await expect(fileLoc.first()).toBeVisible({ timeout: 20_000 });
  return fileLoc;
}

test.describe('snapshot lifecycle', () => {
  test('downloads a file from a snapshot via the Snapshot Browser', async ({ page, backrest }) => {
    test.setTimeout(180_000);

    // --- Setup: seed config + run one real backup. -------------------------
    await seedInstance(backrest);
    await seedRepo(backrest, 'local-repo');
    const dataPath = await backrest.makeTestData({
      'hello.txt': 'download me',
      'nested/deep.txt': 'deep content',
    });
    await seedPlan(backrest, 'my-plan', 'local-repo', [dataPath]);

    const snapshotId = await runBackupViaApi(backrest, 'my-plan');
    expect(snapshotId).toBeTruthy();

    // --- Browser: open the plan, browse the snapshot to hello.txt. ---------
    await page.goto(`${backrest.url}/#/plan/my-plan`);
    await page.getByRole('tab', { name: 'List View' }).click();

    const snapshotRow = page.locator('[data-testid="operation-row"][data-op-type="Snapshot"]');
    await expect(snapshotRow).toBeVisible({ timeout: 30_000 });
    await snapshotRow.getByText('Snapshot Browser').click();

    const fileLoc = await browseToFile(page, dataPath, 'hello.txt');

    // Open the file's row menu (FiMoreHorizontal "..." button).
    await fileLoc.first().getByRole('button').click();

    // The Download menu item has no data-testid (see report); match by role/text.
    const downloadItem = page.getByRole('menuitem', { name: 'Download' });
    await expect(downloadItem).toBeVisible({ timeout: 10_000 });

    // The UI downloads via getDownloadURL() then window.open(url, "_blank"), so
    // the browser download can be reported either on the opener page or on a
    // short-lived popup. Register both listeners BEFORE clicking so nothing is
    // missed, then race them against a timeout.
    const context = page.context();
    const downloadPromise = new Promise<Download>((resolve) => {
      page.once('download', resolve);
      context.once('page', (popup) => {
        popup.once('download', resolve);
      });
    });

    await downloadItem.click();

    const download = await Promise.race<Download>([
      downloadPromise,
      new Promise<Download>((_, reject) =>
        setTimeout(() => reject(new Error('no browser download event within 30s')), 30_000),
      ),
    ]);

    // --- Assert on the delivered bytes. ------------------------------------
    // The /download handler dumps a single regular file as its raw contents
    // (restic dump); it only wraps output in a tar/zip when dumping a directory.
    // So hello.txt arrives verbatim with filename "hello.txt". Handle the
    // archive case defensively anyway.
    const suggested = download.suggestedFilename();
    const saveDir = await fs.mkdtemp(path.join(os.tmpdir(), 'backrest-dl-'));
    const savedPath = path.join(saveDir, suggested);
    await download.saveAs(savedPath);

    const stat = await fs.stat(savedPath);
    expect(stat.size).toBeGreaterThan(0);

    let content: string;
    if (suggested.endsWith('.tar')) {
      const extractDir = path.join(saveDir, 'extracted');
      await fs.mkdir(extractDir, { recursive: true });
      await execFileAsync('tar', ['-xf', savedPath, '-C', extractDir]);
      // Find hello.txt inside the extracted tree.
      const walk = async (dir: string): Promise<string | null> => {
        for (const e of await fs.readdir(dir, { withFileTypes: true })) {
          const full = path.join(dir, e.name);
          if (e.isDirectory()) {
            const found = await walk(full);
            if (found) return found;
          } else if (e.name === 'hello.txt') {
            return full;
          }
        }
        return null;
      };
      const found = await walk(extractDir);
      expect(found, 'hello.txt not found inside downloaded tar').toBeTruthy();
      content = await fs.readFile(found!, 'utf8');
    } else {
      // Content-Disposition sets filename to the (absolute) restic file path;
      // Chromium sanitizes the slashes to underscores, so the suggested name is
      // e.g. "_tmp_..._source-data_hello.txt". Assert on the basename tail.
      expect(suggested).toMatch(/hello\.txt$/);
      content = await fs.readFile(savedPath, 'utf8');
    }

    expect(content).toBe('download me');
  });

  test('forgets a snapshot from the Tree View and it disappears', async ({ page, backrest }) => {
    test.setTimeout(180_000);

    // --- Setup: two real backups so forgetting one leaves the repo sane. ---
    await seedInstance(backrest);
    await seedRepo(backrest, 'local-repo');
    const dataPath = await backrest.makeTestData({
      'hello.txt': 'download me',
    });
    await seedPlan(backrest, 'my-plan', 'local-repo', [dataPath]);

    await runBackupViaApi(backrest, 'my-plan');
    // Change the source so the second backup indexes a distinct snapshot.
    await fs.writeFile(path.join(dataPath, 'second.txt'), 'second backup');
    await runBackupViaApi(backrest, 'my-plan');

    const idsBefore = await listSnapshotIds(backrest, 'local-repo', 'my-plan');
    expect(idsBefore.length).toBe(2);
    const [olderId, newerId] = idsBefore;

    // --- Browser: open the plan (defaults to Tree View). -------------------
    await page.goto(`${backrest.url}/#/plan/my-plan`);
    await page.getByRole('tab', { name: 'Tree View' }).click();

    // Backup/snapshot flows render as tree leaves whose text starts with
    // "Backup <date>" (the flow's display type is its first op — the backup —
    // via displayInfoForFlow, and formatTime starts with a digit). Branch nodes
    // are month/day labels; the post-forget "Forget" leaf won't match either.
    const snapshotLeaves = page.getByRole('treeitem').filter({ hasText: /^Backup \d/ });
    await expect(snapshotLeaves).toHaveCount(2, { timeout: 30_000 });

    // Tree leaves are sorted newest-first, so the last one is the older
    // snapshot. Selecting it reveals its BackupView panel on the right.
    await snapshotLeaves.last().click();

    // The forget control lives in OperationTreeView's BackupView panel: a
    // destructive ConfirmButton labelled "Forget (Destructive)" that arms to
    // "Confirm forget?" on first click.
    const forgetBtn = page.getByRole('button', { name: 'Forget (Destructive)' });
    await expect(forgetBtn).toBeVisible({ timeout: 20_000 });
    await forgetBtn.click();
    await page.getByRole('button', { name: 'Confirm forget?' }).click();

    // --- Assert (no reload): the forgotten snapshot flow disappears. -------
    // A single-snapshot forget marks the OperationIndexSnapshot forgot=true and
    // DELETES its own transient OperationForget (see taskforgetsnapshot.go), so
    // — unlike a retention-policy forget — no persistent "Forget" row remains.
    // The observable UI signal is the flow aggregator hiding the forgotten flow,
    // so the older Backup leaf drops out of the tree (2 -> 1).
    await expect(snapshotLeaves).toHaveCount(1, { timeout: 60_000 });

    // --- Ground truth via the API: exactly the older snapshot is gone. -----
    const deadline = Date.now() + 60_000;
    let idsAfter = await listSnapshotIds(backrest, 'local-repo', 'my-plan');
    while (idsAfter.length !== 1 && Date.now() < deadline) {
      await new Promise((r) => setTimeout(r, 500));
      idsAfter = await listSnapshotIds(backrest, 'local-repo', 'my-plan');
    }
    expect(idsAfter).toEqual([newerId]);
    expect(idsAfter).not.toContain(olderId);
  });
});
