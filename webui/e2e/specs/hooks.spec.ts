import * as fs from 'node:fs/promises';
import * as path from 'node:path';
import { test, expect } from '../harness/fixtures';
import { backrestClient, seedInstance, seedRepo } from '../harness/seed';
import type { Page } from '@playwright/test';

/**
 * Hooks e2e: verify that a plan-level COMMAND hook wired to CONDITION_SNAPSHOT_SUCCESS
 * runs when a backup succeeds.
 *
 * The plan (including its hook) is built entirely through the Add Plan dialog's
 * HooksFormList UI — "Add Hook" menu -> Command, condition select, command
 * textarea. Backups are triggered with the "Backup now" button and observed on
 * the List View, mirroring add-plan-backup.spec.ts.
 *
 * Backend note (internal/hook): a COMMAND hook runs its script in `sh` (stdin),
 * so `echo done > <file>` writes a marker file, and `exit 1` fails the hook. A
 * failing hook uses the default on-error policy ON_ERROR_IGNORE, which is
 * NON-halting (internal/hook/hook.go applyHookErrorPolicy + IsHaltingError):
 * the Run Hook operation records STATUS_ERROR (orchestrator.updateOperationStatus)
 * while the parent Backup operation still succeeds.
 */

/**
 * Drives the Add Plan dialog to create a plan with a single COMMAND hook whose
 * only condition is CONDITION_SNAPSHOT_SUCCESS and whose script is `command`.
 * All interaction is via roles/testids against the real UI.
 */
async function createPlanWithCommandHookViaUI(
  page: Page,
  opts: { name: string; repo: string; dataPath: string; command: string },
): Promise<void> {
  await page.getByTestId('sidebar-add-plan').click();
  const dialog = page.getByRole('dialog');
  await expect(dialog).toBeVisible();

  // Name.
  await dialog.getByTestId('add-plan-name').fill(opts.name);

  // Repository (Chakra select; options render in a page-level portal).
  await dialog.getByTestId('add-plan-repo-select').click();
  await page.getByRole('option', { name: opts.repo, exact: true }).click();
  await expect(dialog.getByTestId('add-plan-repo-select')).toContainText(opts.repo);

  // Backup path.
  await dialog.getByTestId('add-plan-path-add').click();
  const pathInput = dialog.getByTestId('add-plan-path-input').last();
  await pathInput.fill(opts.dataPath);
  // Refocus the name field to dismiss the path autocomplete popup.
  await dialog.getByTestId('add-plan-name').click();
  await expect(pathInput).toHaveValue(opts.dataPath);

  // --- Hook: open the "Add Hook" menu and pick the Command hook type. -------
  await dialog.getByRole('button', { name: 'Add Hook' }).click();
  await page.getByRole('menuitem', { name: 'Command', exact: true }).click();

  // The new hook card shows a conditions select (placeholder "Runs when...").
  // Open it and choose CONDITION_SNAPSHOT_SUCCESS (options render in a portal;
  // the option label is the enum name followed by its description).
  await dialog.getByText('Runs when...').click();
  await page.getByRole('option', { name: /CONDITION_SNAPSHOT_SUCCESS/ }).click();
  // It's a multi-select, so the popup stays open; click the name field to
  // dismiss it (safer than Escape, which can bubble up to close the dialog).
  await dialog.getByTestId('add-plan-name').click();

  // Fill the command textarea (the only <textarea> in this dialog: the hook
  // script). It starts populated with a default echo template.
  const commandBox = dialog.locator('textarea').last();
  await expect(commandBox).toBeVisible();
  await commandBox.fill(opts.command);
  await expect(commandBox).toHaveValue(opts.command);

  // Submit -> dialog closes, plan appears in the sidebar.
  await dialog.getByTestId('add-plan-submit').click();
  await expect(page.getByRole('dialog')).toHaveCount(0);
  await expect(page.getByTestId(`sidebar-item-plan-${opts.name}`)).toBeVisible();
}

test.describe('backup hooks', () => {
  test('COMMAND hook on snapshot success writes a marker file and reports success', async ({
    page,
    backrest,
  }) => {
    test.setTimeout(180_000);

    await seedInstance(backrest);
    await seedRepo(backrest, 'local-repo');
    const dataPath = await backrest.makeTestData({
      'hello.txt': 'hook me',
      'nested/world.txt': 'nested content',
    });

    // Marker file the hook writes on success. No `{{ }}` template syntax, so the
    // backend's Go-template render pass leaves the command untouched.
    const marker = path.join(backrest.dataDir, 'hook-ran.txt');

    await page.goto(backrest.url);
    await expect(page.getByTestId('sidebar-add-plan')).toBeVisible();
    await expect(page.getByTestId('sidebar-item-repo-local-repo')).toBeVisible();

    await createPlanWithCommandHookViaUI(page, {
      name: 'hook-plan',
      repo: 'local-repo',
      dataPath,
      command: `echo done > ${marker}`,
    });

    // Confirm the hook was actually persisted to the config (guards against a
    // silently-dropped UI interaction).
    const cfg = await backrestClient(backrest).getConfig({});
    const plan = cfg.plans.find((p) => p.id === 'hook-plan');
    expect(plan?.hooks.length, 'plan should have exactly one hook').toBe(1);

    // Run the backup from the plan view.
    await page.getByTestId('sidebar-item-plan-hook-plan').click();
    await expect(page).toHaveURL(/#\/plan\/hook-plan$/);
    await page.getByTestId('plan-backup-now').click();
    await page.getByRole('tab', { name: 'List View' }).click();

    // The backup itself succeeds. (A future-dated *scheduled* backup row also
    // exists and stays pending, so scope by data-status.)
    const backupRow = page.locator(
      '[data-testid="operation-row"][data-op-type="Backup"][data-status="success"]',
    );
    await expect(backupRow).toBeVisible({ timeout: 90_000 });

    // (a) The Run Hook operation reaches success. In List View, hook operations
    // render nested under their parent Backup row inside a "Hooks Triggered"
    // accordion. A *successful* hook's accordion is collapsed and its rows are
    // not mounted until it is expanded, so open it first, then assert on the
    // nested Run Hook row (attribute-based: the row stays in the DOM once
    // mounted, even while the accordion animates).
    const hooksTrigger = backupRow.getByText('Hooks Triggered', { exact: true });
    await expect(hooksTrigger).toBeVisible({ timeout: 90_000 });
    await hooksTrigger.click();

    const hookRow = backupRow.locator('[data-testid="operation-row"][data-op-type="Run Hook"]');
    await expect(hookRow).toHaveAttribute('data-status', 'success', { timeout: 90_000 });

    // (b) Node-side: the marker file exists with the expected content.
    await expect
      .poll(
        async () => {
          try {
            return (await fs.readFile(marker, 'utf8')).trim();
          } catch {
            return null;
          }
        },
        { timeout: 30_000 },
      )
      .toBe('done');
  });

  test('a failing COMMAND hook errors the Run Hook op but the backup still succeeds', async ({
    page,
    backrest,
  }) => {
    test.setTimeout(180_000);

    await seedInstance(backrest);
    await seedRepo(backrest, 'local-repo');
    const dataPath = await backrest.makeTestData({
      'hello.txt': 'hook me',
    });

    await page.goto(backrest.url);
    await expect(page.getByTestId('sidebar-add-plan')).toBeVisible();
    await expect(page.getByTestId('sidebar-item-repo-local-repo')).toBeVisible();

    await createPlanWithCommandHookViaUI(page, {
      name: 'hook-plan',
      repo: 'local-repo',
      dataPath,
      command: 'exit 1',
    });

    await page.getByTestId('sidebar-item-plan-hook-plan').click();
    await expect(page).toHaveURL(/#\/plan\/hook-plan$/);
    await page.getByTestId('plan-backup-now').click();
    await page.getByRole('tab', { name: 'List View' }).click();

    // Default on-error policy (ON_ERROR_IGNORE) is non-halting: the backup
    // completes successfully...
    const backupRow = page.locator(
      '[data-testid="operation-row"][data-op-type="Backup"][data-status="success"]',
    );
    await expect(backupRow).toBeVisible({ timeout: 90_000 });

    // ...while the Run Hook operation surfaces the non-zero exit as an error.
    // A failed hook's "Hooks Triggered" accordion is expanded by default, so its
    // nested Run Hook row is mounted without any extra interaction. (The row can
    // read as "hidden" mid-animation, so assert on its attribute rather than
    // visibility.)
    const hookRow = backupRow.locator('[data-testid="operation-row"][data-op-type="Run Hook"]');
    await expect(hookRow).toHaveAttribute('data-status', 'error', { timeout: 90_000 });
  });
});
