import { test, expect } from '@playwright/test';
import { resetDB } from '../support/seed';
import { loginAsTranslator } from '../support/auth';

/**
 * Translator-side day flow. The seeded historical schedule lives on
 * yesterday, so we can't actually check in on it (backend rejects past
 * dates). These tests cover the UI surface that doesn't require a
 * present-day schedule.
 *
 * For a full arrive→leave flow we'd need a "create schedule for today as
 * admin then switch user" sequence, deferred to a later iteration.
 */

test.describe('translator pages', () => {
  test.beforeAll(async ({ baseURL }) => {
    await resetDB(baseURL!);
  });

  test('translator sees their schedule list', async ({ page }) => {
    await loginAsTranslator(page);
    await page.goto('/my-schedules');
    // Seeded schedule on yesterday should appear somewhere on the page.
    await expect(page.locator('body')).toContainText('E2E Clinic, Bangkok');
  });

  test('translator sees the checkin page', async ({ page }) => {
    await loginAsTranslator(page);
    await page.goto('/checkin');
    await expect(page.locator('body')).toContainText(/Checkin|打卡|เช็คอิน/i);
  });

  test.skip('translator can perform arrive checkin on today\'s schedule', async ({ page: _ }) => {
    // TODO: needs an admin-created "today" schedule seed. Either extend the
    // reset endpoint or add a second seed flavour for "today's flow".
  });
});
