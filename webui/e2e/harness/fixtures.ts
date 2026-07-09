import { test as base, expect, type Page } from '@playwright/test';
import { BackrestInstance } from './backrest';

interface BackrestFixtures {
  /** A fresh backrest instance (empty config, own data dir + port) per test. */
  backrest: BackrestInstance;
}

export const test = base.extend<BackrestFixtures>({
  backrest: async ({}, use, testInfo) => {
    const instance = await BackrestInstance.start();
    try {
      await use(instance);
    } finally {
      if (testInfo.status !== testInfo.expectedStatus) {
        await testInfo.attach('backrest-logs', {
          body: instance.logs(),
          contentType: 'text/plain',
        });
      }
      await instance.stop();
    }
  },
});

export { expect };

/**
 * Navigate to a hash route and force a real reload so the SPA re-fetches its
 * config/state from the server, rather than relying on whatever it cached on
 * its last full page load.
 *
 * The app is a HashRouter. If the page has already loaded once and you then
 * `page.goto(url + hashPath)` to a new route, that's a same-document
 * navigation (only `location.hash` changes) — no full reload happens, so the
 * SPA keeps serving from its previously-fetched config. That's a problem when
 * the route depends on state seeded through the API *after* the page's
 * initial load (e.g. a plan added via `seedPlan` while the app was already
 * open on some other page): the cached config predates the seed, and the
 * page renders a "not found" state even though the server has the right
 * data.
 *
 * `gotoFresh` works around this by navigating to the hash URL and then
 * forcing a full reload, guaranteeing the SPA re-fetches config against
 * current server state. It's safe to call even when the page hasn't loaded
 * anything yet — the extra reload just costs a little time.
 *
 * Not needed (and not used) by specs that seed all their state before the
 * page's first `page.goto` — only by specs that seed *after* an earlier
 * navigation in the same test.
 */
export async function gotoFresh(
  page: Page,
  backrest: BackrestInstance,
  hashPath: string,
): Promise<void> {
  await page.goto(`${backrest.url}${hashPath}`);
  await page.reload();
}
