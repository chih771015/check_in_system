import { test, expect } from '@playwright/test';
import { resetDB } from '../support/seed';
import { loginAsTranslator } from '../support/auth';

/**
 * Translator-side day flow. Seed has a "today" schedule for Alice with
 * 2 patients (1 completed + photo, 1 pending).
 *
 * Full arrive→leave checkin is gated on geolocation + camera permissions
 * which Playwright can grant but we'd also need a file fixture for the
 * selfie. Deferred — for now we just confirm the data wires through.
 */

test.describe('translator pages', () => {
  test.beforeAll(async ({ baseURL }) => {
    await resetDB(baseURL!);
  });

  test('translator sees their schedule list', async ({ page }) => {
    await loginAsTranslator(page);
    await page.goto('/my-schedules');
    // Today's seeded schedule shows up without toggling "Show History".
    await expect(page.locator('body')).toContainText('E2E Clinic, Bangkok');
  });

  test('my-checkins page renders', async ({ page }) => {
    await loginAsTranslator(page);
    await page.goto('/my-checkins');
    // Page heading / nav matches one of the locale strings.
    await expect(page.locator('body')).toContainText(/My Check[- ]?ins|打卡紀錄|เช็คอิน/i);
  });

  test.skip('translator can perform arrive checkin on today\'s schedule', async ({ page: _ }) => {
    // TODO: needs geolocation context + selfie file fixture. The action
    // button on /my-schedules navigates to /checkin/:scheduleId/:type.
  });
});
