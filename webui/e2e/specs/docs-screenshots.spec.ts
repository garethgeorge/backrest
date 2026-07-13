import * as fs from 'node:fs/promises';
import * as path from 'node:path';
import { create } from '@bufbuild/protobuf';
import type { Locator, Page } from '@playwright/test';
import { test, expect } from '../harness/fixtures';
import { backrestClient, seedInstance, seedRepo, seedPlan } from '../harness/seed';
import type { BackrestInstance } from '../harness/backrest';
import { BackupRequestSchema, GetOperationsRequestSchema } from '../../gen/ts/v1/service_pb';
import { OperationStatus } from '../../gen/ts/v1/operations_pb';

/**
 * Captures the screenshots embedded in the docs site (docs/src/public/screenshots).
 *
 * Not part of the regular e2e suite: it only runs when DOCS_SCREENSHOTS=1 is
 * set, because it produces image artifacts rather than assertions. To
 * regenerate the docs screenshots:
 *
 *   cd webui
 *   DOCS_SCREENSHOTS=1 pnpm exec playwright test docs-screenshots
 *
 * Output lands in DOCS_SCREENSHOTS_DIR (default: e2e/.cache/docs-screenshots);
 * review the images, then copy them into docs/src/public/screenshots/.
 */

const OUT_DIR =
  process.env.DOCS_SCREENSHOTS_DIR ?? path.join(__dirname, '..', '.cache', 'docs-screenshots');

test.skip(!process.env.DOCS_SCREENSHOTS, 'set DOCS_SCREENSHOTS=1 to capture docs screenshots');

test.use({
  viewport: { width: 1440, height: 900 },
  deviceScaleFactor: 2,
  colorScheme: 'light',
});

async function save(target: Page | Locator, name: string, clip?: Clip): Promise<void> {
  await fs.mkdir(OUT_DIR, { recursive: true });
  const file = path.join(OUT_DIR, name);
  if ('screenshot' in target && clip && isPage(target)) {
    await target.screenshot({ path: file, clip });
  } else {
    await (target as Locator).screenshot({ path: file });
  }
}

interface Clip {
  x: number;
  y: number;
  width: number;
  height: number;
}

function isPage(t: Page | Locator): t is Page {
  return typeof (t as Page).goto === 'function';
}

/**
 * Close-up of one section of a modal that has a section nav on the left:
 * clicks the nav entry (which scrolls that section into view), then clips the
 * modal's content pane (right of the nav column, between header and footer).
 */
async function scrollDialogToSection(
  page: Page,
  dialog: Locator,
  navText: string,
): Promise<Locator> {
  const nav = dialog.getByText(navText, { exact: true }).first();
  await nav.click({ timeout: 5_000 });
  await page.waitForTimeout(600); // allow the section scroll to settle
  return nav;
}

async function saveDialogSection(
  page: Page,
  dialog: Locator,
  navText: string,
  name: string,
): Promise<void> {
  const nav = await scrollDialogToSection(page, dialog, navText);
  const dlgBox = await dialog.boundingBox();
  const navBox = await nav.boundingBox();
  const cancelBox = await dialog
    .getByRole('button', { name: 'Cancel' })
    .first()
    .boundingBox();
  if (!dlgBox || !navBox || !cancelBox) {
    await save(dialog, name); // fallback: the whole dialog
    return;
  }
  const left = navBox.x + navBox.width + 12;
  const top = dlgBox.y + 64; // below the modal header bar
  const bottom = cancelBox.y - 16; // above the footer buttons
  await save(page, name, {
    x: left,
    y: top,
    width: dlgBox.x + dlgBox.width - left,
    height: bottom - top,
  });
}

/** Runs a real backup via RPC and waits for the indexed snapshot (mirrors restore.spec). */
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
      create(GetOperationsRequestSchema, { selector: { planId }, lastN: 100n }),
    );
    let backupOk = false;
    let indexed = false;
    for (const op of resp.operations) {
      if (op.op.case === 'operationBackup' && op.status === OperationStatus.STATUS_SUCCESS)
        backupOk = true;
      if (op.op.case === 'operationIndexSnapshot') indexed = true;
      if (op.op.case === 'operationBackup' && op.status === OperationStatus.STATUS_ERROR)
        throw new Error(`backup for ${planId} failed: ${op.displayMessage}`);
    }
    if (backupOk && indexed) return;
    await new Promise((r) => setTimeout(r, 500));
  }
  throw new Error(`backup for ${planId} did not complete within ${timeoutMs}ms`);
}

/** Demo files that read naturally in a snapshot-browser screenshot. */
const DEMO_FILES: Record<string, string> = {
  'notes/meeting-notes.md': '# Meeting notes\n\n- ship the docs\n',
  'notes/ideas.md': '# Ideas\n',
  'projects/report-2026.txt': 'Quarterly report draft.\n',
  'projects/budget.csv': 'item,cost\nbackups,0\n',
  'recipes.txt': 'pancakes: flour, eggs, milk\n',
};

test.describe('docs screenshots', () => {
  test('first-run settings modal', async ({ page, backrest }) => {
    // No seeding: the initial-setup Settings modal opens on first load.
    await page.goto(backrest.url);
    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();
    await dialog.getByTestId('settings-instance-id').fill('my-backrest');
    await save(dialog, 'settings-view.png');
  });

  test('add repo, sftp setup, add plan, discord hook modals', async ({ page, backrest }) => {
    // Tall viewport so full-height modals (and the hook card at the bottom of
    // the Add Plan dialog) fit without clipping.
    await page.setViewportSize({ width: 1440, height: 1400 });
    await seedInstance(backrest, 'my-backrest');
    await page.goto(backrest.url);

    // --- Add Repo modal, filled out for a local repository. ----------------
    // (The 'mydrive' repo is seeded via the API only *after* this shot, so the
    // name typed here does not trigger the duplicate-name validation error.)
    await page.getByTestId('sidebar-add-repo').click();
    let dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();
    await dialog.getByTestId('add-repo-name').fill('mydrive');
    await dialog.getByTestId('add-repo-uri').fill('/mnt/backup-drive/backrest-repo');
    await dialog.getByTestId('add-repo-name').click(); // dismiss URI autocomplete
    await dialog.getByTestId('add-repo-password').fill('correct-horse-battery-staple');
    await save(dialog, 'add-repo-view.png');

    // Close-up: the repo Scheduling section (prune + check policies).
    try {
      await saveDialogSection(page, dialog, 'Scheduling', 'repo-policies.png');
    } catch {
      /* layout drift: skip the close-up, the full modal shot still exists */
    }

    // --- Same modal with an SFTP URI: the SFTP config + key helper. --------
    await scrollDialogToSection(page, dialog, 'Connection').catch(() => {});
    await dialog.getByTestId('add-repo-uri').fill('sftp://backup@nas.local:22/srv/backrest-repo');
    await dialog.getByTestId('add-repo-name').click();
    await dialog.getByText('Setup SSH Key (Optional)').click();
    await save(dialog, 'sftp-repo-setup.png');
    await page.keyboard.press('Escape');
    await expect(page.getByRole('dialog')).toHaveCount(0);

    // --- Add Plan modal, filled out, plus a Discord notification hook. -----
    await seedRepo(backrest, 'mydrive');
    await page.reload();
    await page.getByTestId('sidebar-add-plan').click();
    dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();
    await dialog.getByTestId('add-plan-name').fill('mydrive-documents');
    await dialog.getByTestId('add-plan-repo-select').click();
    await page.getByRole('option', { name: 'mydrive', exact: true }).click();
    await dialog.getByTestId('add-plan-path-add').click();
    await dialog.getByTestId('add-plan-path-input').last().fill('/home/alice/Documents');
    await dialog.getByTestId('add-plan-name').click(); // dismiss path autocomplete
    await save(dialog, 'add-plan-view.png');

    // Close-ups: the schedule and retention sections.
    try {
      await saveDialogSection(page, dialog, 'Schedule', 'schedule-form.png');
      await saveDialogSection(page, dialog, 'Retention', 'retention-policy.png');
    } catch {
      /* skip close-ups */
    }

    // Discord hook: add it, pick conditions, fill a placeholder webhook URL.
    await dialog.getByTestId('hooks-add').click();
    await page.getByRole('menuitem', { name: 'Discord', exact: true }).click();
    await dialog.getByText('Runs when...').click();
    await page.getByRole('option', { name: /CONDITION_ANY_ERROR/ }).click();
    await page.getByRole('option', { name: /CONDITION_SNAPSHOT_SUCCESS/ }).click();
    await dialog.getByTestId('add-plan-name').click(); // dismiss the multi-select
    // Fill the webhook URL field: the first text input below the conditions box.
    const hookCondBox = await dialog.getByTestId('hook-conditions').boundingBox();
    if (hookCondBox) {
      const hookInputs = dialog.locator('input');
      const count = await hookInputs.count();
      for (let i = 0; i < count; i++) {
        const box = await hookInputs.nth(i).boundingBox();
        if (box && box.y > hookCondBox.y) {
          await hookInputs
            .nth(i)
            .fill('https://discord.com/api/webhooks/1234567890/example-token');
          break;
        }
      }
    }
    // Close-up of the configured Discord hook card. The Add Plan modal has no
    // "Hooks" nav entry (hooks live under "Advanced"), so frame the card
    // itself: content pane right of the nav column, vertically around the
    // hook's conditions select.
    try {
      const navBox = await dialog.getByText('Details', { exact: true }).first().boundingBox();
      const dlgBox = await dialog.boundingBox();
      const cond = dialog.getByTestId('hook-conditions');
      // Scroll to the hook card's LAST field (the template textarea) so the
      // whole card is inside the dialog's visible scroll area, then measure.
      const lastField = dialog.locator('textarea').last();
      await lastField.scrollIntoViewIfNeeded().catch(() => cond.scrollIntoViewIfNeeded());
      await page.waitForTimeout(300);
      const condBox = await cond.boundingBox();
      const lastBox = await lastField.boundingBox().catch(() => null);
      const cancelBox = await dialog
        .getByRole('button', { name: 'Cancel' })
        .first()
        .boundingBox();
      if (!navBox || !dlgBox || !condBox || !cancelBox) throw new Error('no boxes');
      const left = navBox.x + navBox.width + 12;
      const top = Math.max(dlgBox.y + 64, condBox.y - 120);
      const cardBottom = lastBox ? lastBox.y + lastBox.height + 24 : condBox.y + 380;
      const bottom = Math.min(cancelBox.y - 16, cardBottom);
      await save(page, 'discord-hook.png', {
        x: left,
        y: top,
        width: dlgBox.x + dlgBox.width - left,
        height: bottom - top,
      });
    } catch {
      await save(dialog, 'discord-hook.png');
    }
    await page.keyboard.press('Escape');
  });

  test('dashboard, operations, snapshot browser, restore', async ({ page, backrest }) => {
    test.setTimeout(300_000);

    // --- Seed: instance, repo, plan over demo files; run two real backups. -
    await seedInstance(backrest, 'my-backrest');
    await seedRepo(backrest, 'mydrive');
    // Prefer a reader-friendly path over the harness tmp dir — it shows up in
    // the snapshot browser, restore dialog, and plan config screenshots.
    let dataPath = '/tmp/demo/home/alice/Documents';
    try {
      for (const [rel, content] of Object.entries(DEMO_FILES)) {
        const abs = path.join(dataPath, rel);
        await fs.mkdir(path.dirname(abs), { recursive: true });
        await fs.writeFile(abs, content);
      }
    } catch {
      dataPath = await backrest.makeTestData(DEMO_FILES);
    }
    await seedPlan(backrest, 'mydrive-documents', 'mydrive', [dataPath]);
    await runBackupViaApi(backrest, 'mydrive-documents');
    await fs.writeFile(path.join(dataPath, 'notes', 'todo.md'), '# Todo\n- test restores\n');
    await runBackupViaApi(backrest, 'mydrive-documents');

    // --- Summary dashboard. -------------------------------------------------
    await page.goto(backrest.url);
    await expect(page.getByTestId('sidebar-item-plan-mydrive-documents')).toBeVisible();
    await page.waitForTimeout(2_000); // let dashboard cards/charts settle
    await save(page, 'summary-dashboard.png');

    // --- List view (fresh load, so no hidden tree-tab panels linger). --------
    await page.goto(`${backrest.url}/#/plan/mydrive-documents`);
    await page.waitForTimeout(1_500);
    await page.getByRole('tab', { name: 'List View' }).click();
    const rows = page.locator('[data-testid="operation-row"]');
    await expect(rows.first()).toBeVisible({ timeout: 30_000 });

    // Diagnostic: record what rows exist and their geometry, so a failure of
    // any selector below is explainable from OUT_DIR/debug-rows.json.
    const rowDebug = await page.evaluate(() =>
      Array.from(document.querySelectorAll('[data-testid="operation-row"]')).map((el) => ({
        opType: el.getAttribute('data-op-type'),
        status: el.getAttribute('data-status'),
        rect: el.getBoundingClientRect().toJSON(),
      })),
    );
    await fs.mkdir(OUT_DIR, { recursive: true });
    await fs.writeFile(path.join(OUT_DIR, 'debug-rows.json'), JSON.stringify(rowDebug, null, 2));

    const backupRow = rows.filter({ hasText: '- Backup' }).first();
    try {
      await save(backupRow, 'backup-operation.png');
    } catch {
      /* no standalone backup row in this layout: skip */
    }

    // --- Snapshot browser: expand down to the demo files. --------------------
    // Click the accordion trigger by its accessible role; if Playwright's
    // actionability check stalls, dispatch the click directly.
    const browserTrigger = page.getByRole('button', { name: 'Snapshot Browser' }).first();
    await expect(browserTrigger).toBeAttached({ timeout: 15_000 });
    try {
      await browserTrigger.click({ timeout: 5_000 });
    } catch {
      await browserTrigger.dispatchEvent('click');
    }
    const snapshotRow = rows.filter({ hasText: 'Snapshot Browser' }).first();
    const fileLoc = page.getByTestId('snapshot-browser-entry').filter({ hasText: 'recipes.txt' });
    for (const seg of dataPath.split('/').filter(Boolean)) {
      if ((await fileLoc.count()) > 0) break;
      const dir = page.getByTestId('snapshot-browser-entry').filter({ hasText: seg }).first();
      await dir.waitFor({ state: 'visible', timeout: 20_000 });
      await dir.click();
      await page.waitForTimeout(750);
    }
    await expect(fileLoc.first()).toBeVisible({ timeout: 20_000 });
    // Also expand a subdirectory so the shot shows files at two levels.
    const notesDir = page.getByTestId('snapshot-browser-entry').filter({ hasText: 'notes' });
    if ((await notesDir.count()) > 0) {
      await notesDir.first().click();
      await page.waitForTimeout(750);
    }
    try {
      await save(snapshotRow, 'snapshot-browser.png');
    } catch {
      await save(page, 'snapshot-browser.png');
    }

    // --- Restore dialog (not submitted with these values). -------------------
    await fileLoc.first().getByRole('button').click();
    await page.getByTestId('snapshot-restore').click();
    const restoreDialog = page.getByRole('dialog');
    await expect(restoreDialog).toBeVisible();
    await save(restoreDialog, 'restore-dialog.png');

    // --- Run the restore for real; capture the completed operation. ----------
    // Prefer a reader-friendly destination (it is displayed in the operation
    // details); clear leftovers from prior runs since restore refuses to
    // overwrite an existing directory.
    let restoreTarget = '/tmp/demo/home/alice/restored-files';
    try {
      await fs.rm(restoreTarget, { recursive: true, force: true });
    } catch {
      restoreTarget = path.join(backrest.dataDir, 'restore-target');
    }
    const targetInput = restoreDialog.locator('input').first();
    await targetInput.click();
    await targetInput.fill(restoreTarget);
    await targetInput.blur();
    await restoreDialog.getByRole('button', { name: 'Restore' }).click();
    await restoreDialog.getByRole('button', { name: 'Confirm Restore?' }).click();
    const restoreRow = page.locator(
      '[data-testid="operation-row"][data-op-type="Restore"][data-status="success"]',
    );
    await expect(restoreRow).toBeVisible({ timeout: 60_000 });
    await save(restoreRow, 'restore-progress.png');

    // --- Plan view: operation tree (fresh load; default tab). ----------------
    await page.reload();
    await page.waitForTimeout(1_500);
    try {
      await page
        .getByRole('treeitem', { name: /Backup .*ID:/ })
        .first()
        .click({ timeout: 5_000 });
      await page.waitForTimeout(1_000);
    } catch {
      /* tree layout drift: capture unselected state */
    }
    await save(page, 'tree-view-for-restore-article.png');

    // --- Repo view: header with Index Snapshots + the maintenance menu open. -
    await page.goto(`${backrest.url}/#/repo/mydrive`);
    await expect(page.getByRole('heading', { name: 'mydrive' })).toBeVisible();
    await page.getByRole('button', { name: 'More actions' }).click();
    await page.waitForTimeout(500);
    await save(page, 'index-snapshots-btn.png', { x: 0, y: 0, width: 1440, height: 420 });
    await page.keyboard.press('Escape');

    // --- Repo stats panel (best effort; needs a stats op). -------------------
    try {
      await page.getByRole('button', { name: 'More actions' }).click({ timeout: 5_000 });
      await page.getByRole('menuitem', { name: /Stats/ }).click({ timeout: 5_000 });
      await page.waitForTimeout(8_000); // small repo: stats completes quickly
      await page.getByTestId('view-tab-stats').click();
      await page.waitForTimeout(2_000);
      await save(page, 'stats-panel.png');
    } catch {
      /* stats view unavailable: skip */
    }
  });
});
