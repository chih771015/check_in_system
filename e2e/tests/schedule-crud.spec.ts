import { test, expect } from '@playwright/test';
import { resetDB } from '../support/seed';
import { loginAsAdmin } from '../support/auth';

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

  test.skip('admin can create a new schedule', async ({ page: _ }) => {
    // The Add Schedule modal is a 7-field form (translator + date + start +
    // end + location + recurrence + per-patient subform) without stable
    // test-ids. Robust E2E selectors here will need a frontend pass to add
    // data-testid hooks; deferred until then.
  });
});
