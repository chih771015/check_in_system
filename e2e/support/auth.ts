import type { Page } from '@playwright/test';
import { SEED } from './seed';

/**
 * Log in via the actual login form. Returns once the dashboard is visible.
 *
 * Why through the UI instead of injecting a JWT? E2E should exercise the real
 * auth pipeline (interceptor, redirect, route guards). The cost is ~1 second
 * per test; acceptable given small suite size.
 */
async function loginViaUI(page: Page, email: string, password: string): Promise<void> {
  await page.goto('/login');
  // Login form uses placeholders, not labels.
  await page.getByPlaceholder(/Email|信箱|อีเมล/i).fill(email);
  await page.getByPlaceholder(/Password|密碼|รหัสผ่าน/i).fill(password);
  await page.getByRole('button', { name: /Sign In|登入|เข้าสู่ระบบ/i }).click();
  // Wait for the URL to leave /login — successful auth redirects to either
  // /change-password, /admin/*, or /my-schedules depending on role.
  await page.waitForURL((url) => !url.pathname.endsWith('/login'));
}

export async function loginAsAdmin(page: Page): Promise<void> {
  await loginViaUI(page, SEED.admin.email, SEED.password);
}

export async function loginAsTranslator(page: Page): Promise<void> {
  await loginViaUI(page, SEED.translatorActive.email, SEED.password);
}
