import { test, expect } from '@playwright/test';
import { resetDB } from '../support/seed';
import { loginAsTranslator } from '../support/auth';

test.describe('makeup checkin', () => {
  test.beforeAll(async ({ baseURL }) => {
    await resetDB(baseURL!);
  });

  test('translator can navigate to makeup checkin page', async ({ page }) => {
    await loginAsTranslator(page);
    await page.goto('/makeup-checkin');
    await expect(page.locator('body')).toContainText(/Makeup|補打卡|ย้อนหลัง/i);
  });

  test.skip('translator can submit a makeup checkin with reason + selfie', async ({ page: _ }) => {
    // TODO: needs file upload fixture + geolocation mock. Defer until
    // upload helper is extracted.
  });
});
