import { test as base, expect } from '@playwright/test';
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
