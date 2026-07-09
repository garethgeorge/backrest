import { test, expect } from '../harness/fixtures';
import { seedInstance } from '../harness/seed';

/**
 * CUJ2: add the first repository through the UI.
 *
 * Note on the "index/stats ops stream in live" expectation: AddRepo schedules
 * a one-off index-snapshots task automatically (internal/api/backresthandler.go,
 * `s.orchestrator.ScheduleTask(tasks.NewOneoffIndexSnapshotsTask(...))`), but
 * that task's ProtoOp is nil and indexSnapshotsHelper only creates one
 * OperationIndexSnapshot row *per restic snapshot found*
 * (internal/orchestrator/tasks/taskindexsnapshots.go). A brand-new local repo
 * has zero snapshots, so this never produces an operation-row on its own.
 * `DoRepoTaskRequest_Task.STATS` (the repo view's "Compute Stats" action,
 * internal/orchestrator/tasks/taskstats.go) unconditionally creates one
 * OperationStats row regardless of snapshot count, so tests below trigger it
 * explicitly via the UI to reliably observe a successful operation-row.
 */

const PASSWORD = 'test-password-12345';

test.describe('add repo (CUJ2)', () => {
  test('adds a repository through the UI and it becomes usable', async ({ page, backrest }) => {
    await seedInstance(backrest);
    await page.goto(backrest.url);

    await page.getByTestId('sidebar-add-repo').click();

    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();

    await dialog.getByTestId('add-repo-name').fill('my-repo');
    await dialog.getByTestId('add-repo-uri').fill(backrest.repoPath('my-repo'));
    // The URI field is a combobox that opens a suggestions popover on input;
    // close it so it doesn't sit on top of the fields/buttons below it.
    await page.keyboard.press('Escape');
    await dialog.getByTestId('add-repo-password').fill(PASSWORD);

    await dialog.getByTestId('add-repo-submit').click();

    // restic init + AddRepo's GUID lookup can take a few seconds.
    const sidebarItem = page.getByTestId('sidebar-item-repo-my-repo');
    await expect(sidebarItem).toBeVisible({ timeout: 30_000 });
    await expect(page.getByRole('dialog')).toHaveCount(0);

    await sidebarItem.click();
    await expect(page).toHaveURL(/#\/repo\//);

    // Repo view rendered.
    await expect(page.getByRole('heading', { name: 'my-repo' })).toBeVisible();

    // Switch to the list view (default tab is the tree view, which doesn't
    // render operation-row elements) before triggering an operation so the
    // subscription is active and catches the pending -> success transition.
    await page.getByRole('tab', { name: 'List View' }).click();

    // Trigger a repo-level Stats task via the "More actions" menu — see note
    // above for why this (rather than waiting passively) is required to get
    // a success operation-row for a freshly-initialized, snapshot-less repo.
    await page.getByRole('button', { name: 'More actions' }).click();
    await page.getByRole('menuitem', { name: 'Compute Stats' }).click();

    await expect(
      page.locator('[data-testid="operation-row"][data-status="success"]').first(),
    ).toBeVisible({ timeout: 60_000 });
  });

  test('Test Configuration reports a new repo, then submit adds it', async ({ page, backrest }) => {
    await seedInstance(backrest);
    await page.goto(backrest.url);

    await page.getByTestId('sidebar-add-repo').click();

    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();

    const uri = backrest.repoPath('my-repo');
    await dialog.getByTestId('add-repo-name').fill('my-repo');
    await dialog.getByTestId('add-repo-uri').fill(uri);
    await page.keyboard.press('Escape');
    await dialog.getByTestId('add-repo-password').fill(PASSWORD);

    await dialog.getByTestId('add-repo-test-config').click();

    // messages/en.json: add_repo_modal_test_success_new = "Connected
    // successfully to {uri}. No existing repo found at this location, a new
    // one will be initialized" — the "new repo" variant, since nothing exists
    // at this path yet.
    const toast = page.getByText(`Connected successfully to ${uri}`);
    await expect(toast).toBeVisible({ timeout: 15_000 });

    // Testing configuration must not have created the repo yet.
    await expect(page.getByTestId('sidebar-item-repo-my-repo')).toHaveCount(0);
    await expect(dialog).toBeVisible();

    // Toasts render top-end, clear of the dialog footer, so Submit is
    // clickable while the toast is still showing.
    await dialog.getByTestId('add-repo-submit').click();

    await expect(page.getByTestId('sidebar-item-repo-my-repo')).toBeVisible({ timeout: 30_000 });
    await expect(page.getByRole('dialog')).toHaveCount(0);
  });
});
