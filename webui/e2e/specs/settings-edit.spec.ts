import { test, expect } from '../harness/fixtures';
import { seedInstance } from '../harness/seed';

/**
 * Settings modal: reopening after first-run setup, immutability of the
 * instance id, and persistence of an edit across a reload.
 *
 * seedInstance(backrest) (default name "e2e-test") sets config.instance and
 * disables auth via the API directly, so on load shouldShowSettings() is
 * false (webui/src/state/configutil.ts) and no dialog auto-opens — the
 * Settings modal must be reopened explicitly from the sidebar's "Settings"
 * button (webui/src/app/App.tsx, SidebarContent).
 *
 * webui/src/features/settings/SettingsModal.tsx sets
 * `disabled={!!config.instance}` on the instance-id input, so once an
 * instance name exists it is permanently read-only through this modal.
 *
 * The chosen "harmless change" is adding an auth user while leaving
 * auth.disabled = true: toggling auth.disabled off is rejected client-side
 * unless a user already exists (`if (!newConfig.auth?.users &&
 * !newConfig.auth?.disabled) throw ...`), and editing the instance id is not
 * possible (disabled). Adding a user with auth still disabled is accepted and
 * has no side effect on how the rest of the suite reaches the app.
 */
const INSTANCE_NAME = 'e2e-test';

test.describe('settings edit', () => {
  test('reopened settings show the immutable instance id, and an edit persists across reload', async ({
    page,
    backrest,
  }) => {
    await seedInstance(backrest, INSTANCE_NAME);
    await page.goto(backrest.url);

    // Seeded instance: sidebar loads directly, no auto-opened dialog.
    await expect(page.getByTestId('sidebar-add-repo')).toBeVisible();
    await expect(page.getByRole('dialog')).toHaveCount(0);

    // Reopen Settings from the sidebar.
    await page.getByRole('button', { name: 'Settings' }).click();
    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();

    const instanceId = dialog.getByTestId('settings-instance-id');
    await expect(instanceId).toHaveValue(INSTANCE_NAME);
    await expect(instanceId).toBeDisabled();

    // No edits yet: the save bar is not dirty, so Save is disabled.
    await expect(dialog.getByTestId('settings-submit')).toBeDisabled();

    // Harmless change: add an auth user (auth stays disabled throughout).
    await dialog.getByRole('button', { name: 'Add user' }).click();
    await dialog.getByPlaceholder('Username', { exact: true }).fill('e2e-user');
    await dialog.getByPlaceholder('Password', { exact: true }).fill('e2e-password-123');

    await dialog.getByTestId('settings-submit').click();

    // Persistent state: the save bar flips from "unsaved changes" to "All
    // changes saved" once SetConfig resolves (dirty becomes false again).
    await expect(dialog.getByText('All changes saved')).toBeVisible();

    // The modal does not auto-close on save; it only arms a reload-on-close
    // flag (SettingsModal.tsx: setReloadOnCancel(true)). Closing it now
    // triggers window.location.reload().
    await dialog.getByRole('button', { name: 'Cancel', exact: true }).click();

    // Post-reload: seeded config is still valid (instance set, auth
    // disabled), so the sidebar loads directly with no auto-opened dialog.
    await expect(page.getByTestId('sidebar-add-repo')).toBeVisible();
    await expect(page.getByRole('dialog')).toHaveCount(0);

    // Reopen Settings and confirm both facts persisted.
    await page.getByRole('button', { name: 'Settings' }).click();
    const reopened = page.getByRole('dialog');
    await expect(reopened).toBeVisible();

    const reopenedInstanceId = reopened.getByTestId('settings-instance-id');
    await expect(reopenedInstanceId).toHaveValue(INSTANCE_NAME);
    await expect(reopenedInstanceId).toBeDisabled();

    await expect(reopened.getByPlaceholder('Username', { exact: true })).toHaveValue('e2e-user');
  });
});
