import { test, expect } from '@playwright/test';
import { SEED, resetDB } from '../support/seed';
import { loginAsAdmin } from '../support/auth';

test.describe('schedule CRUD', () => {
  test.beforeEach(async ({ page, baseURL }) => {
    await resetDB(baseURL!);
    await loginAsAdmin(page);
    await page.goto('/admin/schedules');
  });

  test('admin can create a new schedule', async ({ page }) => {
    await page.getByRole('button', { name: /Add Schedule|新增排班|เพิ่ม/i }).click();

    // Translator select — first translator (alice)
    const translatorSelect = page.locator('.ant-select').first();
    await translatorSelect.click();
    await page.getByText(SEED.translatorActive.name).click();

    // Date — set a future date via the date input
    const tomorrow = new Date();
    tomorrow.setDate(tomorrow.getDate() + 1);
    const dateStr = tomorrow.toISOString().slice(0, 10);
    await page.locator('input[placeholder*="date"], input[placeholder*="日期"]').first().fill(dateStr);
    await page.keyboard.press('Enter');

    // Overall start / end
    await page.locator('input[placeholder*="Start" i], input[placeholder*="開始" i]').first().fill('09:00');
    await page.keyboard.press('Tab');
    await page.locator('input[placeholder*="End" i], input[placeholder*="結束" i]').first().fill('11:00');

    // Location
    await page.locator('textarea, input[placeholder*="Location" i], input[placeholder*="地點" i]').first().fill('E2E Test Clinic');

    // Add one patient — open select + pick first patient
    await page.getByRole('button', { name: /Add Patient|新增病人|เพิ่มผู้ป่วย/i }).click();

    // Submit
    await page.getByRole('button', { name: /^(Create|建立|สร้าง|送出|OK)$/i }).first().click();

    // Confirm appears in the table
    await expect(page.locator('table')).toContainText('E2E Test Clinic');
  });

  test('admin can delete a schedule', async ({ page }) => {
    // The seeded historical schedule (yesterday) should be visible after
    // expanding the date filter. Click the delete button on the seeded row.
    const seededRow = page.locator('tr', { hasText: 'E2E Clinic, Bangkok' });
    await expect(seededRow).toBeVisible();
    await seededRow.getByRole('button', { name: /Delete|刪除|ลบ/i }).click();
    // antd modal.confirm
    await page.getByRole('button', { name: /^(OK|Delete|確認|刪除)$/i }).last().click();
    await expect(page.locator('table')).not.toContainText('E2E Clinic, Bangkok');
  });
});
