import { test, expect } from '../harness/fixtures';
import { seedInstance } from '../harness/seed';

// Login, logout and refresh all reload the tab in place. In v1.14.x a
// same-tab reload in a secure context (https, localhost, 127.0.0.1) pegged the
// page's CPU in WebKit and the reloaded page never loaded (#1306, #1402).
test.describe('page reload', () => {
  test('the app loads again after repeated same-tab reloads', async ({ page, backrest }) => {
    await seedInstance(backrest);

    await page.goto(backrest.url);
    await expect(page.getByTestId('sidebar-add-repo')).toBeVisible();

    for (let i = 0; i < 3; i++) {
      await page.reload({ timeout: 30_000 });
      await expect(page.getByTestId('sidebar-add-repo')).toBeVisible();
    }
  });
});
