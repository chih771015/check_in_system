import { test, expect } from '@playwright/test';
import { resetDB } from '../support/seed';
import { loginAsAdmin } from '../support/auth';
import { pickOption, fillPicker } from '../support/antd';

test.describe('schedule CRUD', () => {
  test.beforeEach(async ({ page, baseURL }) => {
    await resetDB(baseURL!);
    await loginAsAdmin(page);
    await page.goto('/admin/schedules');
  });

  test('seeded schedule appears in the list', async ({ page }) => {
    // The seed creates one schedule for Alice today at "E2E Clinic, Bangkok".
    await expect(page.locator('table')).toContainText('E2E Clinic, Bangkok');
  });

  test('admin can delete a schedule', async ({ page }) => {
    const seededRow = page.locator('tr', { hasText: 'E2E Clinic, Bangkok' });
    await expect(seededRow).toBeVisible();
    await seededRow.getByRole('button', { name: /Delete|刪除|ลบ/i }).click();

    // antd modal.confirm puts its buttons inside .ant-modal-confirm. Scope
    // the OK click there so we don't match the row's Delete button (which
    // is what page.getByRole('button', { name: 'Delete' }).last() would
    // otherwise pick up, getting intercepted by the modal overlay).
    const confirm = page.locator('.ant-modal-confirm').last();
    // Button text is "Confirm" in en, "確認" in zh-TW (frontend uses
    // okText: t('common.confirm')). The narrower "Delete" / "刪除" also
    // matches in case a future refactor switches to common.delete.
    await confirm.getByRole('button', { name: /^(Confirm|OK|Delete|確認|刪除)$/i }).click();

    await expect(page.locator('table')).not.toContainText('E2E Clinic, Bangkok');
  });

  test('admin can create a new schedule', async ({ page }) => {
    await page.getByRole('button', { name: /Add Schedule|新增排班|เพิ่ม/i }).click();

    // antd Form.Item assigns the field `name` as the control id, but BOTH the
    // create and edit modals use those names, so #endTime etc. are duplicated.
    // Scope every field to the open modal. For a Select we click the
    // .ant-select container (the inner #id input is zero-width until typed).
    const modal = page.locator('.ant-modal:visible');
    await pickOption(page, modal.locator('.ant-select:has(#translatorId) .ant-select-selector'), 'Alice');
    await fillPicker(modal.locator('#date'), '2026-12-10');
    await fillPicker(modal.locator('#startTime'), '09:00');
    await fillPicker(modal.locator('#endTime'), '12:00');
    await modal.locator('#location').fill('E2E Created Clinic');

    // The patient list editor renders each row inside a Card; its PatientPicker
    // is the only Select nested under .ant-card in the modal. The blank row's
    // start/end default to the overall window (09:00–12:00), so just picking a
    // patient yields a valid slot.
    await pickOption(page, modal.locator('.ant-card .ant-select-selector'), 'Patient Passport');

    await page.getByRole('button', { name: /^(Create|建立|สร้าง)$/i }).click();

    // The new schedule shows up in the data table (scope to .ant-table-tbody to
    // avoid matching a lingering DatePicker calendar table).
    await expect(page.locator('.ant-table-tbody')).toContainText('E2E Created Clinic');
  });
});
