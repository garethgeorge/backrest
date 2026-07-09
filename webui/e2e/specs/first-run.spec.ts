import { test, expect } from '../harness/fixtures';

test.describe('first run', () => {
  test('completes first-run setup with auth left disabled', async ({ page, backrest }) => {
    await page.goto(backrest.url);

    // Fresh instance: Settings auto-opens because config.instance is empty
    // (see smoke.spec.ts for the baseline assertion of this state).
    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();

    const instanceId = dialog.getByTestId('settings-instance-id');
    await expect(instanceId).toBeEditable();
    await instanceId.fill('e2e-instance');

    // A fresh backend config has auth.disabled=true, so the toggle (checked
    // = "Disable Authentication" is ON) should already reflect that. Only
    // flip it if the UI disagrees with that expectation.
    const disableAuthToggle = dialog.getByTestId('settings-disable-auth');
    const disableAuthCheckbox = disableAuthToggle.locator('input[type="checkbox"]');
    await expect(disableAuthCheckbox).toBeChecked();

    const submit = dialog.getByTestId('settings-submit');
    await expect(submit).toBeEnabled();
    await submit.click();

    // Persistent-state signal that the save succeeded: the instance id field
    // becomes immutable once config.instance is set, and the form is no
    // longer dirty so the submit button disables again.
    await expect(instanceId).toBeDisabled();
    await expect(submit).toBeDisabled();

    // Closing the modal after a successful save triggers a full page reload
    // (SettingsModal's handleCancel/reloadOnCancel behavior). Use the
    // footer "Cancel" button rather than the (untestid'd) header close icon.
    await dialog.getByRole('button', { name: 'Cancel' }).click();

    await expect(page.getByTestId('sidebar-add-plan')).toBeVisible();
    await expect(page.getByTestId('sidebar-add-repo')).toBeVisible();
    await expect(page.getByRole('dialog')).toHaveCount(0);

    // Reload explicitly: setup persisted server-side, so the Settings
    // dialog must not auto-open again.
    await page.reload();
    await expect(page.getByTestId('sidebar-add-plan')).toBeVisible();
    await expect(page.getByTestId('sidebar-add-repo')).toBeVisible();
    await expect(page.getByRole('dialog')).toHaveCount(0);
  });

  test('completes first-run setup with a user account and requires login', async ({
    page,
    backrest,
  }) => {
    await page.goto(backrest.url);

    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();

    const instanceId = dialog.getByTestId('settings-instance-id');
    await expect(instanceId).toBeEditable();
    await instanceId.fill('e2e-instance-auth');

    // Toggle OFF "Disable Authentication" (checked by default) to require a
    // login.
    const disableAuthToggle = dialog.getByTestId('settings-disable-auth');
    const disableAuthCheckbox = disableAuthToggle.locator('input[type="checkbox"]');
    await expect(disableAuthCheckbox).toBeChecked();
    await disableAuthToggle.click();
    await expect(disableAuthCheckbox).not.toBeChecked();

    // Add a user account (no testids on these controls; there are none in
    // the inventory, so fall back to role/placeholder).
    await dialog.getByRole('button', { name: 'Add user' }).click();
    await dialog.getByPlaceholder('Username').last().fill('e2e-user');
    await dialog.getByPlaceholder('Password').last().fill('e2e-password-12345');

    const submit = dialog.getByTestId('settings-submit');
    await expect(submit).toBeEnabled();
    await submit.click();

    // Persistent-state signal that the save succeeded (same as the other
    // test): instance id locks and the submit button is no longer dirty.
    await expect(instanceId).toBeDisabled();
    await expect(submit).toBeDisabled();

    // Saving does NOT itself force a re-auth check or reload (SettingsModal
    // only sets reloadOnCancel=true); closing the modal is what reloads the
    // page and causes AuthenticationBoundary to re-fetch config. With auth
    // now required and no token stored, that fetch comes back Unauthenticated
    // and the Login modal is shown instead of the sidebar.
    await dialog.getByRole('button', { name: 'Cancel' }).click();

    const loginDialog = page.getByRole('dialog');
    await expect(loginDialog).toBeVisible();
    const loginUsername = loginDialog.getByTestId('login-username');
    const loginPassword = loginDialog.getByTestId('login-password');
    await expect(loginUsername).toBeVisible();
    await expect(loginPassword).toBeVisible();

    await loginUsername.fill('e2e-user');
    await loginPassword.fill('e2e-password-12345');
    await loginDialog.getByTestId('login-submit').click();

    // Login succeeds (reloads the page, then loadConfig succeeds and finds
    // users configured): sidebar shows, no dialogs remain.
    await expect(page.getByTestId('sidebar-add-plan')).toBeVisible();
    await expect(page.getByTestId('sidebar-add-repo')).toBeVisible();
    await expect(page.getByRole('dialog')).toHaveCount(0);
  });
});
