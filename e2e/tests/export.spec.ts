import { test, expect } from '@playwright/test';
import { resetDB } from '../support/seed';
import { loginAsAdmin } from '../support/auth';

test.describe('export', () => {
  test.beforeAll(async ({ baseURL }) => {
    await resetDB(baseURL!);
  });

  test('admin can trigger Excel export download', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/admin/checkins');

    // Wait for the download dialog. The button text is i18n'd as
    // "Export Excel" / "匯出 Excel".
    const downloadPromise = page.waitForEvent('download');
    await page.getByRole('button', { name: /Export Excel|匯出 Excel|ส่งออก/i }).first().click();
    const dl = await downloadPromise;
    expect(dl.suggestedFilename()).toMatch(/\.xlsx$/);
  });

  test.skip('admin can trigger Google Sheet export', async ({ page: _ }) => {
    // Requires GOOGLE service account env. Skipped in default E2E run.
  });
});
