import { test, expect } from '../harness/fixtures';
import { seedInstance } from '../harness/seed';

test.describe('smoke', () => {
  test('fresh instance auto-opens the first-run settings dialog', async ({ page, backrest }) => {
    await page.goto(backrest.url);

    // A fresh backrest config has an empty instance id (auth is disabled by
    // default), so shouldShowSettings() is true and the SPA auto-opens the
    // Settings modal with an empty, editable instance-id field.
    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();
    const instanceId = dialog.getByTestId('settings-instance-id');
    await expect(instanceId).toBeVisible();
    await expect(instanceId).toHaveValue('');
    await expect(instanceId).toBeEditable();
  });

  test('seeded instance skips setup and shows the sidebar', async ({ page, backrest }) => {
    await seedInstance(backrest);

    await page.goto(backrest.url);

    // Sidebar is rendered once the app has loaded a usable config.
    await expect(page.getByTestId('sidebar-add-repo')).toBeVisible();
    await expect(page.getByTestId('sidebar-add-plan')).toBeVisible();

    // And no dialog auto-opens for a configured instance.
    await expect(page.getByRole('dialog')).toHaveCount(0);
  });
});
