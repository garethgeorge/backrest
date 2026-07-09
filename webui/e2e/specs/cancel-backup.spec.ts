import { randomBytes } from 'node:crypto';
import * as fs from 'node:fs/promises';
import * as path from 'node:path';
import { create } from '@bufbuild/protobuf';
import { PlanSchema, Schedule_Clock } from '../../gen/ts/v1/config_pb';
import { test, expect } from '../harness/fixtures';
import { backrestClient, seedInstance, seedRepo } from '../harness/seed';

/**
 * Cancel a running backup from the UI.
 *
 * Seeds an instance + local restic repo + plan via the API (schedule
 * disabled), lays down a large incompressible dataset so the backup runs for
 * a long time, triggers "Backup Now", cancels the in-progress Backup
 * operation-row through its row menu, and asserts the row reaches the
 * terminal status backrest records for a cancelled running backup.
 *
 * Terminal-status note (from internal/orchestrator/orchestrator.go): the
 * Cancel RPC applies STATUS_USER_CANCELLED ("cancelled") directly to
 * operations still *queued*. For an operation already running it records the
 * requested status in taskCancelStatus and cancels the task's context; restic
 * is killed, the task returns an error, and updateOperationStatus sees the
 * user-cancel marker and records STATUS_USER_CANCELLED ("cancelled") with a
 * display message noting "task was cancelled by user request". Only
 * shutdown-style context cancellations (no user cancel request) still record
 * STATUS_ERROR.
 */

// Incompressible dataset: 8 x 40 MiB = 320 MiB of crypto-random bytes.
const FILE_COUNT = 8;
const FILE_SIZE = 40 * 1024 * 1024;

// Belt-and-braces slowdown so the backup cannot finish before we cancel even
// on a fast idle machine: restic --limit-upload is in KiB/s, and applies to
// local backends too (the rate limiter wraps the backend layer). 8 MiB/s
// against ~320 MiB of unchunkable random data keeps the backup running for
// roughly 40s — a comfortable cancellation window.
const BACKUP_FLAGS = ['--limit-upload=8192'];

/** seedPlan + backup_flags (harness seedPlan doesn't expose flags). */
async function seedThrottledPlan(
  backrest: Parameters<typeof seedInstance>[0],
  id: string,
  repoId: string,
  paths: string[],
): Promise<void> {
  const client = backrestClient(backrest);
  const config = await client.getConfig({});
  config.plans.push(
    create(PlanSchema, {
      id,
      repo: repoId,
      paths,
      backupFlags: BACKUP_FLAGS,
      schedule: {
        schedule: { case: 'disabled', value: true },
        clock: Schedule_Clock.LOCAL,
      },
    }),
  );
  await client.setConfig(config);
}

test.describe('cancel a running backup', () => {
  test('backup can be cancelled from the plan view and stays cancelled', async ({
    page,
    backrest,
  }) => {
    test.setTimeout(300_000); // large dataset + shared machine: be generous.

    // 1. Seed instance + repo via the API, and write the large dataset.
    await seedInstance(backrest);
    await seedRepo(backrest, 'local-repo');

    const dataDir = path.join(backrest.dataDir, 'source-data');
    await fs.mkdir(dataDir, { recursive: true });
    const genStart = Date.now();
    for (let i = 0; i < FILE_COUNT; i++) {
      await fs.writeFile(path.join(dataDir, `random-${i}.bin`), randomBytes(FILE_SIZE));
    }
    console.log(
      `dataset: ${FILE_COUNT} x ${FILE_SIZE / 1024 / 1024} MiB written in ${Date.now() - genStart}ms`,
    );

    // Seeded via API so the schedule is disabled: exactly one Backup row will
    // ever exist (no permanently-"pending" scheduled row).
    await seedThrottledPlan(backrest, 'my-plan', 'local-repo', [dataDir]);

    // 2. Open the plan view and switch to List View (operation rows render
    //    directly there; the default Tree View only shows rows after
    //    selecting a node).
    await page.goto(`${backrest.url}/#/plan/my-plan`);
    await expect(page.getByTestId('plan-backup-now')).toBeVisible();
    await page.getByRole('tab', { name: 'List View' }).click();

    // Trigger the backup.
    await page.getByTestId('plan-backup-now').click();

    // 3. The (single) Backup operation-row reaches "in progress".
    const backupRow = page.locator('[data-testid="operation-row"][data-op-type="Backup"]');
    await expect(backupRow).toHaveCount(1, { timeout: 60_000 });
    await expect(backupRow).toHaveAttribute('data-status', 'in progress', {
      timeout: 60_000,
    });
    const inProgressAt = Date.now();

    // 4. Cancel via the row's actions menu. The cancel entry is a two-step
    //    ConfirmMenuItem: first click flips it to "Confirm Cancel?", second
    //    click actually calls backrestService.cancel (menu items render in a
    //    portal, so query at page level).
    await backupRow.getByRole('button', { name: 'Actions' }).click();
    await page.getByRole('menuitem', { name: 'Cancel Operation' }).click();
    await page.getByRole('menuitem', { name: 'Confirm Cancel?' }).click();

    // 5. Without reloading: the row leaves "in progress" and reaches the
    //    terminal status for a user-cancelled running backup — "cancelled"
    //    (STATUS_USER_CANCELLED; see header comment).
    await expect(backupRow).not.toHaveAttribute('data-status', 'in progress', {
      timeout: 120_000,
    });
    await expect(backupRow).toHaveAttribute('data-status', 'cancelled');
    console.log(
      `cancelled: terminal status reached ${Date.now() - inProgressAt}ms after in-progress`,
    );

    // The recorded display message attributes the interruption to the user's
    // cancel request.
    await expect(backupRow).toContainText('cancelled by user request');

    // No snapshot success evidence for this run: no indexed-snapshot
    // operation row appears, and the backup row itself never shows success.
    await expect(
      page.locator('[data-testid="operation-row"][data-op-type="Snapshot"]'),
    ).toHaveCount(0);
    await expect(backupRow).toHaveAttribute('data-status', 'cancelled');

    // 6. Durability + UI sanity: reload the page; the plan view still
    //    renders and the cancelled backup row persists.
    await page.reload();
    await expect(page.getByTestId('plan-backup-now')).toBeVisible();
    await page.getByRole('tab', { name: 'List View' }).click();

    const rowAfterReload = page.locator('[data-testid="operation-row"][data-op-type="Backup"]');
    await expect(rowAfterReload).toHaveCount(1, { timeout: 30_000 });
    await expect(rowAfterReload).toHaveAttribute('data-status', 'cancelled');
    await expect(rowAfterReload).toContainText('cancelled by user request');
    await expect(
      page.locator('[data-testid="operation-row"][data-op-type="Snapshot"]'),
    ).toHaveCount(0);
  });
});
