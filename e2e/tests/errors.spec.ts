import { test, expect } from '@playwright/test';
import { SEED, resetDB } from '../support/seed';

/**
 * Catch-all for regression scenarios that don't fit a single feature:
 *   - 401 on protected route → redirected to /login (no infinite reload)
 *   - Direct visit to /admin/* without login → /login
 */

test.describe('errors / guards', () => {
  test.beforeAll(async ({ baseURL }) => {
    await resetDB(baseURL!);
  });

  test('unauthenticated visit to /admin redirects to /login', async ({ page }) => {
    await page.goto('/admin/translators');
    await expect(page).toHaveURL(/\/login/);
  });

  test('login form does NOT reload page on wrong password (regression)', async ({ page }) => {
    // This is the bug that motivated this whole session. The fix was to
    // skip the global 401 redirect when the failing request was /auth/login.
    await page.goto('/login');
    const emailInput = page.getByPlaceholder(/Email|信箱|อีเมล/i);
    await emailInput.fill(SEED.admin.email);
    await page.getByPlaceholder(/Password|密碼/i).fill('xxx');

    // Capture navigations — if the page reloaded we'd see a 'load' event.
    let reloaded = false;
    page.once('load', () => { reloaded = true; });

    await page.getByRole('button', { name: /Sign In|登入/i }).click();
    await page.waitForTimeout(500);

    expect(reloaded).toBe(false);
    // Form values preserved (proves no remount/reload)
    await expect(emailInput).toHaveValue(SEED.admin.email);
  });
});
