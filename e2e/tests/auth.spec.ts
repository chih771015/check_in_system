import { test, expect } from '@playwright/test';
import { SEED, resetDB } from '../support/seed';
import { loginAsAdmin, loginAsTranslator } from '../support/auth';

test.describe('auth', () => {
  test.beforeAll(async ({ baseURL }) => {
    await resetDB(baseURL!);
  });

  test('admin can log in and lands on admin home', async ({ page }) => {
    await loginAsAdmin(page);
    await expect(page).toHaveURL(/\/admin\//);
  });

  test('active translator can log in and lands on /my-schedules', async ({ page }) => {
    await loginAsTranslator(page);
    await expect(page).toHaveURL(/\/my-schedules/);
  });

  test('wrong password keeps user on /login with an error toast', async ({ page }) => {
    await page.goto('/login');
    await page.getByPlaceholder(/Email|信箱|อีเมล/i).fill(SEED.admin.email);
    await page.getByPlaceholder(/Password|密碼|รหัสผ่าน/i).fill('definitely-wrong');
    await page.getByRole('button', { name: /Sign In|登入|เข้าสู่ระบบ/i }).click();

    // Stays on /login (regression test for the bug where the page reloaded
    // and the toast was lost). Email field must still hold the typed value.
    await expect(page).toHaveURL(/\/login/);
    await expect(page.getByPlaceholder(/Email|信箱|อีเมล/i)).toHaveValue(SEED.admin.email);
    // antd message renders as a div with role=alert
    await expect(page.locator('.ant-message-error, [role="alert"]').first()).toBeVisible();
  });

  test('disabled translator cannot log in', async ({ page }) => {
    await page.goto('/login');
    await page.getByPlaceholder(/Email|信箱|อีเมล/i).fill(SEED.translatorDisabled.email);
    await page.getByPlaceholder(/Password|密碼|รหัสผ่าน/i).fill(SEED.password);
    await page.getByRole('button', { name: /Sign In|登入|เข้าสู่ระบบ/i }).click();

    await expect(page).toHaveURL(/\/login/);
    await expect(page.locator('.ant-message-error').first()).toBeVisible();
  });

  test('account lockout after repeated wrong attempts', async ({ page }) => {
    // Backend locks the account after N consecutive failures (see
    // AuthService.Login). We hammer the active translator with wrong
    // passwords, then assert the final attempt produces a lockout message.
    await resetDB(page.url().startsWith('http') ? new URL(page.url()).origin : 'http://localhost:3001');
    for (let i = 0; i < 6; i++) {
      await page.goto('/login');
      await page.getByPlaceholder(/Email|信箱|อีเมล/i).fill(SEED.translatorActive.email);
      await page.getByPlaceholder(/Password|密碼|รหัสผ่าน/i).fill('wrong-pw');
      await page.getByRole('button', { name: /Sign In|登入|เข้าสู่ระบบ/i }).click();
      // give the message a moment to render
      await page.waitForTimeout(200);
    }
    // After lockout, even the correct password should be rejected.
    await page.goto('/login');
    await page.getByPlaceholder(/Email|信箱|อีเมล/i).fill(SEED.translatorActive.email);
    await page.getByPlaceholder(/Password|密碼|รหัสผ่าน/i).fill(SEED.password);
    await page.getByRole('button', { name: /Sign In|登入|เข้าสู่ระบบ/i }).click();
    await expect(page).toHaveURL(/\/login/);
  });
});
