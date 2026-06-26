import { test, expect } from '@playwright/test';
import { SEED, resetDB } from '../support/seed';
import { loginAsAdmin, loginAsTranslator } from '../support/auth';

/**
 * Money statistics surfaced this cycle: the global current-month expenditure
 * banner (admin-only), the patient-list actual-paid total column, and the
 * patient-history actual-paid total + date-range filter.
 *
 * The seed gives patients[0]'s completed visit a single non-zero actual amount
 * (SEED.seededActualPaidTotal), today, so the exact computed totals are
 * assertable end-to-end: a broken SUM / range / scope would change the number.
 */

const BANNER = /本月病人總支出|This month's patient expenditure|ค่าใช้จ่ายผู้ป่วยเดือนนี้/i;
const ACTUAL_TOTAL = /實付總額|Actual paid total|ยอดชำระจริงรวม/i;
// e.g. 1500 → "NT$ 1,500" (allow optional space: antd Statistic omits it).
const total = SEED.seededActualPaidTotal.toLocaleString();
const NT = (n: string) => new RegExp(`NT\\$\\s*${n}`);

test.describe('money stats (banner + actual-paid totals)', () => {
  test.beforeEach(async ({ baseURL }) => {
    await resetDB(baseURL!);
  });

  test('admin banner shows the current-month expenditure total', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/admin/patients');
    // Scope the figure to the banner element (its label's parent), so this
    // proves the banner — not the table column — shows the computed total.
    const bannerLabel = page.getByText(BANNER).first();
    await expect(bannerLabel).toBeVisible();
    await expect(bannerLabel.locator('..')).toContainText(NT(total));
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

  test('patient list actual-paid column shows each patient computed total', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/admin/patients');
    await expect(page.locator('.ant-table-thead')).toContainText(ACTUAL_TOTAL);
    // patients[0] has the seeded paid visit; patients[1] (pending) totals 0.
    await expect(page.locator('tr', { hasText: SEED.patients[0].name })).toContainText(NT(total));
    await expect(page.locator('tr', { hasText: SEED.patients[1].name })).toContainText(NT('0'));
  });

  test('patient history shows the actual-paid total and a date-range filter', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/admin/patients');
    const row = page.locator('tr', { hasText: SEED.patients[0].name });
    await row.getByRole('button', { name: /History|歷史|ประวัติ/i }).click();

    // The history page surfaces the actual-paid total (antd Statistic) ...
    await expect(page.locator('body')).toContainText(ACTUAL_TOTAL);
    await expect(page.locator('.ant-statistic-content')).toContainText(NT(total));
    // ... and a RangePicker to narrow the history by date.
    await expect(page.locator('.ant-picker-range')).toBeVisible();
  });
});
