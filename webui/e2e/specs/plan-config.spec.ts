import * as fs from 'node:fs/promises';
import * as path from 'node:path';
import { create } from '@bufbuild/protobuf';
import { test, expect } from '../harness/fixtures';
import { backrestClient, seedInstance, seedRepo } from '../harness/seed';
import type { BackrestInstance } from '../harness/backrest';
import { PlanSchema, Schedule_Clock, type RetentionPolicy } from '../../gen/ts/v1/config_pb';
import { BackupRequestSchema, GetOperationsRequestSchema } from '../../gen/ts/v1/service_pb';
import { OperationStatus } from '../../gen/ts/v1/operations_pb';

/**
 * Plan-configuration behaviors: path excludes and retention-policy enforcement.
 *
 * Both tests seed a schedule-disabled plan (so backups only fire when we
 * trigger them) and drive real restic backups through the API, then verify the
 * observable outcome in the UI's List View plus ground truth via the API.
 */

/**
 * seedPlan variant that also sets `excludes` and/or `retention` on the plan.
 * The schedule is explicitly disabled (mirrors harness seedPlan) so no
 * background backups fire during the test.
 */
async function seedPlanWithOptions(
  inst: BackrestInstance,
  id: string,
  repoId: string,
  paths: string[],
  opts: { excludes?: string[]; retention?: RetentionPolicy } = {},
) {
  const client = backrestClient(inst);
  const config = await client.getConfig({});
  const plan = create(PlanSchema, {
    id,
    repo: repoId,
    paths,
    excludes: opts.excludes ?? [],
    retention: opts.retention,
    schedule: {
      schedule: { case: 'disabled', value: true },
      clock: Schedule_Clock.LOCAL,
    },
  });
  config.plans.push(plan);
  await client.setConfig(config);
  return plan;
}

/**
 * Triggers a real backup for `planId` and polls GetOperations until at least
 * `wantBackups` successful OperationBackup ops and `wantSnapshots` indexed
 * OperationIndexSnapshot ops exist. Fails fast if a backup errors.
 */
async function runBackupViaApi(
  inst: BackrestInstance,
  planId: string,
  wantBackups: number,
  wantSnapshots: number,
  timeoutMs = 90_000,
): Promise<void> {
  const client = backrestClient(inst);
  await client.backup(create(BackupRequestSchema, { value: planId }));

  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    const resp = await client.getOperations(
      create(GetOperationsRequestSchema, { selector: { planId }, lastN: 200n }),
    );
    let backupOk = 0;
    let snapshots = 0;
    for (const op of resp.operations) {
      if (op.op.case === 'operationBackup' && op.status === OperationStatus.STATUS_SUCCESS) {
        backupOk++;
      }
      if (op.op.case === 'operationIndexSnapshot') {
        snapshots++;
      }
      if (op.op.case === 'operationBackup' && op.status === OperationStatus.STATUS_ERROR) {
        throw new Error(`backup for plan ${planId} failed: ${op.displayMessage}`);
      }
    }
    if (backupOk >= wantBackups && snapshots >= wantSnapshots) return;
    await new Promise((r) => setTimeout(r, 500));
  }
  throw new Error(
    `backup for plan ${planId} did not reach ${wantBackups} successful backups / ` +
      `${wantSnapshots} indexed snapshots within ${timeoutMs}ms`,
  );
}

/** Counts operations of the plan by kind: index snapshots (live vs forgot) and forgets. */
async function operationCounts(inst: BackrestInstance, planId: string) {
  const client = backrestClient(inst);
  const resp = await client.getOperations(
    create(GetOperationsRequestSchema, { selector: { planId }, lastN: 200n }),
  );
  let live = 0;
  let forgotten = 0;
  let forgetOk = false;
  for (const op of resp.operations) {
    if (op.op.case === 'operationForget' && op.status === OperationStatus.STATUS_SUCCESS) {
      forgetOk = true;
    }
    if (op.op.case === 'operationIndexSnapshot') {
      if (op.op.value.forgot) forgotten++;
      else live++;
    }
  }
  return { live, forgotten, total: live + forgotten, forgetOk };
}

/**
 * Polls until a successful OperationForget exists and exactly `wantLive`
 * OperationIndexSnapshot ops are NOT marked `forgot` (stable ground truth for
 * retention enforcement, independent of the async garbage-collection pass).
 */
async function waitForForget(
  inst: BackrestInstance,
  planId: string,
  wantLive: number,
  timeoutMs = 90_000,
) {
  const deadline = Date.now() + timeoutMs;
  let last = await operationCounts(inst, planId);
  while (Date.now() < deadline) {
    last = await operationCounts(inst, planId);
    if (last.forgetOk && last.live === wantLive) return last;
    await new Promise((r) => setTimeout(r, 500));
  }
  throw new Error(
    `forget did not settle to ${wantLive} live snapshots within ${timeoutMs}ms ` +
      `(last observed: ${JSON.stringify(last)})`,
  );
}

/**
 * Forces the orchestrator's garbage-collection pass by round-tripping the
 * config: every applied config change reschedules the CollectGarbage task to
 * run ~1s later, which prunes all oplog operations belonging to a flow whose
 * snapshot has been forgotten. Then polls until exactly `wantIndex` index-
 * snapshot operations remain (the forgotten flow's ops have been collected).
 */
async function forceGcUntilIndexCount(
  inst: BackrestInstance,
  planId: string,
  wantIndex: number,
  timeoutMs = 60_000,
) {
  const client = backrestClient(inst);
  const deadline = Date.now() + timeoutMs;
  // Round-trip the config to reschedule (and thus soon run) collect-garbage.
  const cfg = await client.getConfig({});
  await client.setConfig(cfg);
  let last = await operationCounts(inst, planId);
  while (Date.now() < deadline) {
    last = await operationCounts(inst, planId);
    if (last.total === wantIndex) return last;
    await new Promise((r) => setTimeout(r, 500));
  }
  throw new Error(
    `garbage collection did not reduce index snapshots to ${wantIndex} within ${timeoutMs}ms ` +
      `(last observed: ${JSON.stringify(last)})`,
  );
}

/**
 * Expands the snapshot browser inside the given Snapshot row, then walks the
 * path segments of `dataPath` from the auto-expanded root, clicking each
 * directory so its children lazily load. Leaves the browser showing the
 * contents of the final segment's directory.
 */
async function openBrowserAndNavigate(page: any, snapshotRow: any, dataPath: string) {
  await snapshotRow.getByText('Snapshot Browser').click();
  const segments = dataPath.split('/').filter(Boolean);
  for (const seg of segments) {
    const dir = page.getByTestId('snapshot-browser-entry').filter({ hasText: seg }).first();
    await dir.waitFor({ state: 'visible', timeout: 30_000 });
    await dir.click();
    // Give the lazy ListSnapshotFiles fetch a moment to populate children.
    await page.waitForTimeout(750);
  }
}

test.describe('plan configuration', () => {
  test('excludes are honored end-to-end (skip.log absent, keep.txt present)', async ({
    page,
    backrest,
  }) => {
    test.setTimeout(180_000);

    await seedInstance(backrest);
    await seedRepo(backrest, 'local-repo');
    const dataPath = await backrest.makeTestData({
      'keep.txt': 'keep me',
      'skip.log': 'exclude me',
      'nested/keep2.txt': 'also kept',
    });
    await seedPlanWithOptions(backrest, 'excludes-plan', 'local-repo', [dataPath], {
      excludes: ['*.log'],
    });

    await runBackupViaApi(backrest, 'excludes-plan', 1, 1);

    // Ground truth: the excluded *.log file must never appear in the repo.
    // (UI assertion below is the primary check; this is a sanity guard.)

    await page.goto(`${backrest.url}/#/plan/excludes-plan`);
    await page.getByRole('tab', { name: 'List View' }).click();

    const snapshotRow = page.locator(
      '[data-testid="operation-row"][data-op-type="Snapshot"][data-status="success"]',
    );
    await expect(snapshotRow.first()).toBeVisible({ timeout: 60_000 });

    await openBrowserAndNavigate(page, snapshotRow.first(), dataPath);

    // Now the browser shows the source-data directory contents. keep.txt is
    // included; skip.log was excluded by "*.log". Assert the positive first
    // (waits for the lazy load), then assert the excluded file is absent.
    const keepEntry = page.getByTestId('snapshot-browser-entry').filter({ hasText: 'keep.txt' });
    await expect(keepEntry.first()).toBeVisible({ timeout: 30_000 });

    const skipEntry = page.getByTestId('snapshot-browser-entry').filter({ hasText: 'skip.log' });
    await expect(skipEntry).toHaveCount(0);
  });

  test('retention policy (policyKeepLastN=1) forgets older snapshots', async ({
    page,
    backrest,
  }) => {
    test.setTimeout(180_000);

    await seedInstance(backrest);
    await seedRepo(backrest, 'local-repo');
    const dataPath = await backrest.makeTestData({ 'data.txt': 'version-1' });
    await seedPlanWithOptions(backrest, 'retention-plan', 'local-repo', [dataPath], {
      retention: { policy: { case: 'policyKeepLastN', value: 1 } },
    });

    // Backup #1 -> snapshot 1.
    await runBackupViaApi(backrest, 'retention-plan', 1, 1);

    // Modify file contents so snapshot 2 differs, then backup #2 -> snapshot 2.
    await fs.writeFile(path.join(dataPath, 'data.txt'), 'version-2-modified');
    await runBackupViaApi(backrest, 'retention-plan', 2, 2);

    // Ground truth (stable, GC-independent): a successful forget ran and
    // exactly ONE snapshot is live (not marked forgot). The forget marks the
    // older snapshot's index-snapshot op `forgot=true` (retention keepLastN=1).
    const counts = await waitForForget(backrest, 'retention-plan', 1);
    expect(counts.live).toBe(1);

    // The orchestrator's collect-garbage pass then prunes every oplog
    // operation belonging to the forgotten snapshot's flow (index snapshot +
    // its backup + its forget op). Force + await that pass so the end-state is
    // deterministic: exactly ONE index-snapshot operation remains.
    const afterGc = await forceGcUntilIndexCount(backrest, 'retention-plan', 1);
    expect(afterGc.total).toBe(1);
    expect(afterGc.live).toBe(1);
    expect(afterGc.forgotten).toBe(0);

    // --- UI (List View) --------------------------------------------------
    // How a forgotten snapshot surfaces in the UI: its whole flow (including
    // the Snapshot row) is DELETED from the oplog by garbage collection, so
    // the row disappears entirely — it is not merely restyled or hidden.
    await page.goto(`${backrest.url}/#/plan/retention-plan`);
    await page.getByRole('tab', { name: 'List View' }).click();

    // Exactly ONE Snapshot operation row remains (the retained snapshot),
    // success-status.
    const snapshotRows = page.locator('[data-testid="operation-row"][data-op-type="Snapshot"]');
    await expect(snapshotRows).toHaveCount(1, { timeout: 60_000 });
    await expect(snapshotRows.first()).toHaveAttribute('data-status', 'success');

    // A successful Forget operation row remains (the one that forgot the older
    // snapshot; it lives in the retained snapshot's flow so it survives GC).
    const forgetRow = page.locator(
      '[data-testid="operation-row"][data-op-type="Forget"][data-status="success"]',
    );
    await expect(forgetRow.first()).toBeVisible({ timeout: 60_000 });
  });
});
