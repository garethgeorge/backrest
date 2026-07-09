import { test, expect } from '../harness/fixtures';
import { seedInstance, seedRepo } from '../harness/seed';

/**
 * Error-path coverage for the Add Repo and Add Plan dialogs: a wrong-password
 * rejection against a real repo, required-field validation, and the
 * paths-required rule for plans. In all three cases the primary assertion is
 * persistent state (the dialog stays open, nothing new appears in the
 * sidebar); the validation/error text is also asserted per the task brief,
 * even though it renders in a toast (see "Toast placement" note below).
 *
 * Toast placement: all of these errors surface via
 * webui/src/components/common/Alerts.tsx (`alerts.error` -> Chakra
 * `toaster`), which renders in its own top-level Portal
 * (webui/src/components/ui/toaster.tsx), not inside the dialog's DOM subtree.
 * So error text is asserted at the page level (`page.getByText(...)`), not
 * scoped to `dialog`. This is called out because the task brief allows
 * "error/validation text within the dialog" for these tests, but the current
 * implementation never renders these particular errors inline in the form —
 * they are only ever shown as toasts. Flagged as a gap below.
 */

test.describe('error paths', () => {
  test('wrong password against an already-initialized repo is rejected and not added', async ({
    page,
    backrest,
  }) => {
    await seedInstance(backrest);
    await seedRepo(backrest, 'existing-repo');
    await page.goto(backrest.url);
    await expect(page.getByTestId('sidebar-item-repo-existing-repo')).toBeVisible();

    await page.getByTestId('sidebar-add-repo').click();
    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();

    await dialog.getByTestId('add-repo-name').fill('bad-pw');
    await dialog.getByTestId('add-repo-uri').fill(backrest.repoPath('existing-repo'));
    // Close the URI autocomplete popover so it doesn't overlap other fields.
    await page.keyboard.press('Escape');
    await dialog.getByTestId('add-repo-password').fill('totally-wrong-password');

    await dialog.getByTestId('add-repo-submit').click();

    // internal/api/backresthandler.go AddRepo calls RepoOrchestrator.Init
    // synchronously, before the repo is appended to config. Against an
    // already-initialized repo dir with the wrong password, restic's
    // decrypt/unlock fails, `restic init` then also fails because a config
    // already exists on disk, and the handler returns
    // fmt.Errorf("failed to init repo: %w", err) (pkg/restic/restic.go's
    // errAlreadyInitialized = "repo already initialized"). This rejects the
    // AddRepo RPC before any config mutation, so the repo is never persisted.
    await expect(page.getByText(/failed to init repo/i)).toBeVisible({
      timeout: 15_000,
    });

    await expect(dialog).toBeVisible();
    await expect(page.getByTestId('sidebar-item-repo-bad-pw')).toHaveCount(0);
  });

  test('submitting Add Repo with everything empty shows required-field validation and adds nothing', async ({
    page,
    backrest,
  }) => {
    await seedInstance(backrest);
    await page.goto(backrest.url);

    await page.getByTestId('sidebar-add-repo').click();
    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();

    // Everything left blank; submit immediately. AddRepoModal's
    // validateLocal() checks fields in order and throws on the first
    // failure, so only the repo-name error surfaces from this click.
    await dialog.getByTestId('add-repo-submit').click();

    // en.json: add_repo_modal_error_repo_name_required = "Please input repo name"
    await expect(page.getByText(/Please input repo name/i)).toBeVisible();
    await expect(dialog).toBeVisible();
    await expect(page.locator('[data-testid^="sidebar-item-repo-"]')).toHaveCount(0);

    // Fill only the name so the *next* validation rule (URI) fires,
    // demonstrating the second add_repo_modal_error_* key as well.
    await dialog.getByTestId('add-repo-name').fill('temp-repo');
    await dialog.getByTestId('add-repo-submit').click();

    // en.json: add_repo_modal_error_uri_required = "Please input repo URI"
    await expect(page.getByText(/Please input repo URI/i)).toBeVisible();
    await expect(dialog).toBeVisible();
    await expect(page.locator('[data-testid^="sidebar-item-repo-"]')).toHaveCount(0);
  });

  test('submitting Add Plan with no paths surfaces the paths-required validation and adds nothing', async ({
    page,
    backrest,
  }) => {
    await seedInstance(backrest);
    await seedRepo(backrest, 'existing-repo');
    await page.goto(backrest.url);

    await page.getByTestId('sidebar-add-plan').click();
    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();

    await dialog.getByTestId('add-plan-name').fill('plan-no-paths');

    await dialog.getByTestId('add-plan-repo-select').click();
    await page.getByRole('option', { name: 'existing-repo', exact: true }).click();
    await expect(dialog.getByTestId('add-plan-repo-select')).toContainText('existing-repo');

    // Paths list is left empty; submit.
    await dialog.getByTestId('add-plan-submit').click();

    // GAP (flagged): en.json's `add_plan_modal_validation_paths_required`
    // ("Please enter at least one path to backup") is not referenced anywhere
    // in AddPlanModal.tsx's client-side handleOk() validation — it appears to
    // be unused/vestigial. The actual paths-required rule lives server-side
    // in internal/config/validate.go's validatePlan() ("at least one path is
    // required (unless backup_flags supplies paths, e.g. --files-from or
    // --stdin-from-command)"), enforced by the SetConfig RPC
    // (internal/api/backresthandler.go) and returned as a
    // connect.CodeInvalidArgument error, which the modal surfaces via a toast.
    await expect(page.getByText(/at least one path is required/i)).toBeVisible({
      timeout: 15_000,
    });

    await expect(dialog).toBeVisible();
    await expect(page.getByTestId('sidebar-item-plan-plan-no-paths')).toHaveCount(0);
  });
});
