import { create } from '@bufbuild/protobuf';
import { test, expect } from '../harness/fixtures';
import { backrestClient, seedInstance, seedRepo, seedPlan } from '../harness/seed';
import type { BackrestInstance } from '../harness/backrest';
import { BackupRequestSchema, GetOperationsRequestSchema } from '../../gen/ts/v1/service_pb';
import { OperationStatus } from '../../gen/ts/v1/operations_pb';

/**
 * Run Command (webui/src/features/operations/RunCommandModal.tsx): an
 * "advanced users" feature exposed from the repo view's "More actions" menu
 * that lets a user type an arbitrary restic subcommand (e.g. "snapshots") and
 * streams its stdout/stderr into the UI via a LogView embedded in the
 * resulting operation-row's "Command Output" accordion.
 *
 * Wiring confirmed by reading the source (see final report for details):
 * - RepoView.tsx: "More actions" (IconButton aria-label) -> menuitem
 *   "Run Command" (value="run-command") dynamically imports and opens
 *   RunCommandModal.
 * - RunCommandModal.tsx: a FormModal (role=dialog) titled
 *   "Run Command in repo <repoId>" with one text input (placeholder "Run a
 *   restic command e.g. 'help' to print help text") and an "Execute" button;
 *   below it, an OperationListView filtered to operationRunCommand ops for
 *   this repo (planId "_system_").
 * - OperationRow.tsx: a "Run Command" row's body has a "Command Output"
 *   accordion section wrapping <LogView logref={run.outputLogref} />; it's
 *   auto-expanded whenever outputSizeBytes < 64KiB (true for both commands
 *   here), so no extra click is needed to reveal it. LogView renders each
 *   streamed line as its own <pre>, with no data-testid.
 * - Verified locally against the pinned e2e restic binary: `restic snapshots`
 *   with one snapshot prints a summary line "1 snapshots"; `restic
 *   not-a-real-command` prints `unknown command "not-a-real-command" for
 *   "restic"` to stderr and exits 1. Both stdout+stderr are wired to the same
 *   log writer (pkg/restic/restic.go GenericCommand/handleOutput), so restic's
 *   real diagnostic text lands in the Command Output log -- the row's
 *   `displayMessage` banner only gets the wrapping Go error
 *   (`command "...": exit status 1`), not restic's stderr text.
 */

/** Runs a backup for `planId` via the Backup RPC and waits for it to succeed. */
async function runBackupViaApi(
  inst: BackrestInstance,
  planId: string,
  timeoutMs = 60_000,
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
  throw new Error(`backup for plan ${planId} did not complete within ${timeoutMs}ms`);
}

/**
 * Opens the Run Command modal from the repo view's "More actions" menu.
 * Assumes the repo view is already on screen.
 */
async function openRunCommandModal(page: import('@playwright/test').Page) {
  await page.getByRole('button', { name: 'More actions' }).click();
  await page.getByRole('menuitem', { name: 'Run Command' }).click();
  const dialog = page.getByRole('dialog');
  await expect(dialog).toBeVisible({ timeout: 15_000 });
  return dialog;
}

test.describe('run command', () => {
  test('runs a real restic command and streams its output', async ({ page, backrest }) => {
    test.setTimeout(120_000);

    // --- Seed a repo with one real snapshot so `snapshots` has evidence to
    // print. ---------------------------------------------------------------
    await seedInstance(backrest);
    await seedRepo(backrest, 'local-repo');
    const dataPath = await backrest.makeTestData({ 'hello.txt': 'hello' });
    await seedPlan(backrest, 'my-plan', 'local-repo', [dataPath]);
    await runBackupViaApi(backrest, 'my-plan');

    // --- Drive the UI. -------------------------------------------------
    await page.goto(`${backrest.url}/#/repo/local-repo`);
    await expect(page.getByRole('heading', { name: 'local-repo' })).toBeVisible({
      timeout: 15_000,
    });

    const dialog = await openRunCommandModal(page);
    await expect(dialog.getByText('Run Command in repo local-repo')).toBeVisible();

    const commandInput = dialog.getByRole('textbox');
    await commandInput.fill('snapshots');
    await dialog.getByRole('button', { name: 'Execute' }).click();

    // The modal embeds an OperationListView filtered to Run Command ops for
    // this repo; the RunCommand RPC itself runs the task synchronously
    // server-side, but the UI still finds out about it via a subscription, so
    // wait generously.
    const runRow = dialog.locator('[data-testid="operation-row"][data-op-type="Run Command"]');
    await expect(runRow).toBeVisible({ timeout: 30_000 });
    await expect(runRow).toHaveAttribute('data-status', 'success', { timeout: 30_000 });

    // The "Command Output" accordion is auto-expanded for small output
    // (<64KiB) -- assert on restic's real streamed stdout, not just the
    // generic status.
    await expect(runRow).toContainText('Command Output');
    await expect(runRow).toContainText(/\d+ snapshots?/, { timeout: 15_000 });
  });

  test('surfaces a failing command as an error with the real failure reason', async ({
    page,
    backrest,
  }) => {
    test.setTimeout(60_000);

    await seedInstance(backrest);
    await seedRepo(backrest, 'local-repo');

    await page.goto(`${backrest.url}/#/repo/local-repo`);
    await expect(page.getByRole('heading', { name: 'local-repo' })).toBeVisible({
      timeout: 15_000,
    });

    const dialog = await openRunCommandModal(page);

    const commandInput = dialog.getByRole('textbox');
    await commandInput.fill('not-a-real-command');
    await dialog.getByRole('button', { name: 'Execute' }).click();

    const runRow = dialog.locator('[data-testid="operation-row"][data-op-type="Run Command"]');
    await expect(runRow).toBeVisible({ timeout: 30_000 });
    await expect(runRow).toHaveAttribute('data-status', 'error', { timeout: 30_000 });

    // Row-level displayMessage banner: wraps the Go exec error, not restic's
    // stderr (internal/orchestrator/tasks/taskruncommand.go wraps as
    // `command %q: %w`).
    await expect(runRow).toContainText('command "not-a-real-command"');

    // The real restic diagnostic text is only in the streamed Command Output
    // log (both stdout and stderr are wired to the same log writer -- see
    // pkg/restic/restic.go handleOutput/GenericCommand).
    await expect(runRow).toContainText('Command Output');
    await expect(runRow).toContainText(/unknown command "not-a-real-command"/, {
      timeout: 15_000,
    });
  });
});
