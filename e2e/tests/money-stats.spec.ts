import { test, expect } from '@playwright/test';
import { SEED, resetDB } from '../support/seed';
import { loginAsAdmin, loginAsTranslator } from '../support/auth';

/**
 * Money statistics surfaced this cycle: the global current-month expenditure
 * banner (admin-only), the patient-list actual-paid total column, and the
 * patient-history actual-paid total + date-range filter.
 *
 * Seed visits have actual_amount = 0, so totals render as "NT$ 0"; these tests
 * assert the controls/labels are wired and role-gated, not specific amounts
 * (the arithmetic is covered by backend unit tests).
 */

const BANNER = /本月病人總支出|This month's patient expenditure|ค่าใช้จ่ายผู้ป่วยเดือนนี้/i;
const ACTUAL_TOTAL = /實付總額|Actual paid total|ยอดชำระจริงรวม/i;

test.describe('money stats (banner + actual-paid totals)', () => {
  test.beforeEach(async ({ baseURL }) => {
    await resetDB(baseURL!);
  });

  test('admin sees the current-month expenditure banner with an NT$ figure', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/admin/patients');
    await expect(page.locator('body')).toContainText(BANNER);
    // The banner renders an NT$ figure (seed actual amounts are 0 → "NT$ 0").
    await expect(page.getByText(/NT\$/).first()).toBeVisible();
  });

  test('translator does not see the admin expenditure banner', async ({ page }) => {
    await loginAsTranslator(page);
    // Land on the translator home and let the app shell + any fetches settle,
    // so the absence assertion isn't passing merely because the page is blank.
    await page.waitForURL((u) => !u.pathname.endsWith('/login'));
    await page.waitForLoadState('networkidle');
    await expect(page.getByRole('button', { name: /Logout|登出|ออกจากระบบ/i })).toBeVisible();
    await expect(page.locator('body')).not.toContainText(BANNER);
  });

  test('patient list shows an actual-paid total column', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/admin/patients');
    await expect(page.locator('.ant-table-thead')).toContainText(ACTUAL_TOTAL);
  });

  test('patient history shows actual-paid total and a date-range filter', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/admin/patients');
    const row = page.locator('tr', { hasText: SEED.patients[0].name });
    await row.getByRole('button', { name: /History|歷史|ประวัติ/i }).click();

    // The history page surfaces an actual-paid total (antd Statistic) ...
    await expect(page.locator('body')).toContainText(ACTUAL_TOTAL);
    // ... and a RangePicker to narrow the history by date.
    await expect(page.locator('.ant-picker-range')).toBeVisible();
  });
});
