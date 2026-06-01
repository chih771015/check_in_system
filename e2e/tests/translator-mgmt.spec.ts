import { test, expect } from '@playwright/test';
import { SEED, resetDB } from '../support/seed';
import { loginAsAdmin } from '../support/auth';

test.describe('translator management', () => {
  test.beforeEach(async ({ page, baseURL }) => {
    await resetDB(baseURL!);
    await loginAsAdmin(page);
    await page.goto('/admin/translators');
  });

  test('seeded translators appear in the list', async ({ page }) => {
    await expect(page.locator('table')).toContainText(SEED.translatorActive.name);
    await expect(page.locator('table')).toContainText(SEED.translatorDisabled.name);
  });

  test('admin can create a new translator (unique email per run)', async ({ page }) => {
    const runID = Date.now();
    const email = `new-${runID}@translator.local`;
    const name = `New Translator ${runID}`;

    await page.getByRole('button', { name: /Add Translator|新增翻譯員|เพิ่ม/i }).click();
    await page.getByRole('textbox', { name: /Name|姓名|ชื่อ/i }).fill(name);
    await page.getByRole('textbox', { name: /Email|信箱|อีเมล/i }).fill(email);
    await page.getByRole('textbox', { name: /Phone|電話|โทร/i }).fill('0911-test');
    await page.getByPlaceholder(/Password|密碼|รหัสผ่าน/i).fill('Test1234!');
    await page.getByRole('button', { name: /^(Create|新增|建立|สร้าง)$/i }).click();

    await expect(page.locator('table')).toContainText(name);
  });

  test('admin can disable an active translator', async ({ page }) => {
    const row = page.locator('tr', { hasText: SEED.translatorActive.name });
    await row.getByRole('button', { name: /Disable|停用|ปิด/i }).click();
    // modal.confirm
    await page.getByRole('button', { name: /^(OK|確認|Disable|停用)$/i }).last().click();
    // Active tag becomes disabled
    await expect(row).toContainText(/Disabled|停用|ปิด/i);
  });
});
