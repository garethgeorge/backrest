import { test, expect } from '../harness/fixtures';
import type { Page } from '@playwright/test';
import type { BackrestInstance } from '../harness/backrest';

const USERNAME = 'admin';
const PASSWORD = 'testpass123';

/**
 * Drives the first-run Settings modal to enable authentication and create a
 * single user, then returns once the app has reloaded into the Login gate.
 *
 * A fresh backrest config has an empty instance id and `auth.disabled = true`
 * (see internal/config NewDefaultConfig), so on load the SPA auto-opens the
 * Settings modal. We name the instance, flip the disable-auth toggle OFF to
 * *enable* auth, add a user, and save. handleOk() hashes the password and
 * SetConfigs while the server still has auth disabled, so the save itself is
 * unauthenticated and succeeds. The modal does not reload on save; it sets a
 * reload-on-close flag, so closing it triggers window.location.reload(). After
 * the reload GetConfig returns Unauthenticated and the LoginModal appears.
 */
async function enableAuthAndReachLogin(page: Page, backrest: BackrestInstance) {
  await page.goto(backrest.url);

  // First-run Settings modal auto-opens (portal → role=dialog).
  const settings = page.getByRole('dialog');
  await expect(settings.getByTestId('settings-instance-id')).toBeVisible();

  await settings.getByTestId('settings-instance-id').fill('auth-e2e');

  // Default config has auth disabled (toggle ON); flip it to enable auth.
  await settings.getByTestId('settings-disable-auth').click();

  // Add a user via the settings UI and fill its credentials. The row uses
  // placeholder-labelled inputs (no dedicated testids).
  await settings.getByRole('button', { name: 'Add user' }).click();
  await settings.getByPlaceholder('Username', { exact: true }).fill(USERNAME);
  await settings.getByPlaceholder('Password', { exact: true }).fill(PASSWORD);

  // Save. handleOk hashes the password then SetConfigs; the save bar flips to
  // "All changes saved" (dirty=false) once it resolves.
  await settings.getByTestId('settings-submit').click();
  await expect(settings.getByText('All changes saved')).toBeVisible();

  // Close the modal → reloadOnCancel triggers window.location.reload().
  await settings.getByRole('button', { name: 'Cancel', exact: true }).click();

  // After reload the API answers Unauthenticated and the Login gate appears.
  await expect(page.getByTestId('login-username')).toBeVisible();
}

test.describe('auth login', () => {
  test('enabling auth via first-run settings surfaces the login gate', async ({
    page,
    backrest,
  }) => {
    await enableAuthAndReachLogin(page, backrest);

    // The login gate is present and the app has not slipped through to the app
    // content (the add-repo sidebar action is not reachable while gated).
    const login = page.getByRole('dialog');
    await expect(login.getByTestId('login-username')).toBeVisible();
    await expect(login.getByTestId('login-password')).toBeVisible();
    await expect(login.getByTestId('login-submit')).toBeVisible();
  });

  test('rejects wrong credentials, then accepts the correct password', async ({
    page,
    backrest,
  }) => {
    await enableAuthAndReachLogin(page, backrest);

    const login = page.getByRole('dialog');

    // Wrong password: login() rejects, the modal shows an error and does NOT
    // reload. The login gate must still be present afterward.
    await login.getByTestId('login-username').fill(USERNAME);
    await login.getByTestId('login-password').fill('wrongpass');
    await login.getByTestId('login-submit').click();

    // Give the failed request time to resolve, then assert we are still gated:
    // the submit button re-enables (loading cleared) and the app content is
    // still unreachable.
    await expect(login.getByTestId('login-submit')).toBeEnabled();
    await expect(login.getByTestId('login-username')).toBeVisible();
    await expect(page.getByTestId('sidebar-add-repo')).toHaveCount(0);

    // Correct password: on success LoginModal stores the token and reloads the
    // page after 500ms. Assert the post-reload state (sidebar), not the toast.
    await login.getByTestId('login-username').fill(USERNAME);
    await login.getByTestId('login-password').fill(PASSWORD);
    await login.getByTestId('login-submit').click();

    await expect(page.getByTestId('sidebar-add-repo')).toBeVisible();
    // No dialog should remain once authenticated.
    await expect(page.getByRole('dialog')).toHaveCount(0);
  });

  test('logout returns to the login gate', async ({ page, backrest }) => {
    await enableAuthAndReachLogin(page, backrest);

    // Log in successfully.
    const login = page.getByRole('dialog');
    await login.getByTestId('login-username').fill(USERNAME);
    await login.getByTestId('login-password').fill(PASSWORD);
    await login.getByTestId('login-submit').click();
    await expect(page.getByTestId('sidebar-add-repo')).toBeVisible();

    // The header Logout control clears the token and reloads.
    await page.getByRole('button', { name: 'Logout' }).click();

    // Reload with an empty token → GetConfig Unauthenticated → Login gate.
    await expect(page.getByTestId('login-username')).toBeVisible();
    await expect(page.getByTestId('sidebar-add-repo')).toHaveCount(0);
  });
});
