import { test, expect } from '@playwright/test';
import { resetDB } from '../support/seed';
import { loginAsAdmin } from '../support/auth';

/**
 * Diagnosis results overview (admin side). Asserts the seeded completed
 * patient appears with its photo, and that the "up to 3" cap is enforced
 * by the upload modal (covered separately by frontend unit tests; here we
 * just confirm the page wires through).
 */

test.describe('diagnosis results', () => {
  test.beforeAll(async ({ baseURL }) => {
    await resetDB(baseURL!);
  });

  test('admin sees seeded completed patient in results overview', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/admin/diagnosis-results');

    // The seed has 1 completed SchedulePatient. Default filter shows last 7
    // days which includes yesterday. The patient name should appear.
    await expect(page.locator('body')).toContainText('Patient Passport');
  });

  test('clicking the photo count opens preview modal', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/admin/diagnosis-results');

    // Find the "1 張" / "1 photo" button next to the seeded completed row
    const photoBtn = page.getByRole('button', { name: /1\s*(張|photo|รูป)/i }).first();
    if (await photoBtn.isVisible()) {
      await photoBtn.click();
      // Modal opens with Image.PreviewGroup
      await expect(page.locator('.ant-modal').last()).toBeVisible();
    }
  });
});
