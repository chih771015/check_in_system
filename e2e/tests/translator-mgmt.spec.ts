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

    // The create modal has no placeholders — labels are rendered by antd
    // Form.Item. Scope all field lookups inside the modal to avoid hitting
    // table filter inputs in the background page.
    const modal = page.locator('.ant-modal').last();
    await modal.getByLabel(/Name|姓名|ชื่อ/i).fill(name);
    await modal.getByLabel(/Email|信箱|電子郵件|อีเมล/i).fill(email);
    await modal.getByLabel(/Phone|電話|โทร/i).fill('0911-test');
    // Input.Password's underlying input doesn't have role=textbox and no
    // placeholder — getByLabel works because antd wires the label.
    await modal.getByLabel(/Password|密碼|รหัสผ่าน/i).fill('Test1234!');

    await modal.getByRole('button', { name: /^(Create|新增|建立|สร้าง)$/i }).click();

    await expect(page.locator('table')).toContainText(name);
  });

  test('admin can disable an active translator', async ({ page }) => {
    const row = page.locator('tr', { hasText: SEED.translatorActive.name });
    await row.getByRole('button', { name: /Disable|停用|ปิด/i }).click();

    // Confirm dialog is .ant-modal-confirm; the .last() guards against any
    // stale confirm wrapper in the DOM.
    const confirm = page.locator('.ant-modal-confirm').last();
    // okText is t('common.confirm') → "Confirm" / "確認".
    await confirm.getByRole('button', { name: /^(Confirm|OK|確認|Disable|停用)$/i }).click();

    // After disable the row keeps the name but switches its status tag.
    await expect(row).toContainText(/Disabled|停用|ปิด/i);
  });
});
