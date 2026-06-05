import { test, expect } from '@playwright/test';
import { resetDB } from '../support/seed';
import { loginAsTranslator } from '../support/auth';
import { selfieFile } from '../support/files';

/**
 * Translator-side day flow. Seed has a "today" schedule for Alice with
 * 2 patients (1 completed + photo, 1 pending).
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
});

/**
 * Full arrive checkin through the real UI: geolocation is granted at the
 * context level and the selfie is uploaded from an in-memory fixture. This
 * exercises useGeolocation → /checkin/:id/arrive → POST /api/checkins.
 *
 * We navigate to the checkin route directly using the seed's stable schedule
 * id (TRUNCATE ... RESTART IDENTITY makes the single seeded schedule id=1).
 * Going through the /my-schedules action button would be time-of-day
 * dependent: the seed schedule is 09:00–12:00 "today", so after noon the UI
 * shows a Makeup button instead of Arrive.
 */
const SEED_SCHEDULE_ID = 1;

test.describe('translator arrive checkin', () => {
  test.use({
    geolocation: { latitude: 13.7563, longitude: 100.5018 },
    permissions: ['geolocation'],
  });

  test.beforeEach(async ({ baseURL }) => {
    await resetDB(baseURL!);
  });

  test('translator can perform arrive checkin on today\'s schedule', async ({ page }) => {
    await loginAsTranslator(page);
    await page.goto(`/checkin/${SEED_SCHEDULE_ID}/arrive`);

    // The schedule detail card renders once getMySchedules resolves.
    await expect(page.locator('body')).toContainText('E2E Clinic, Bangkok');

    // Upload the selfie (the only file input on the page).
    await page.locator('input[type="file"]').setInputFiles(selfieFile());

    // GPS is granted at context level, so useGeolocation resolves to success
    // and enables the submit button. Wait for that, then submit.
    const submit = page.getByRole('button', { name: /Submit Check-in|送出/i });
    await expect(submit).toBeEnabled({ timeout: 10_000 });
    await submit.click();

    // A successful POST /api/checkins navigates back to the schedule list.
    await expect(page).toHaveURL(/\/my-schedules/);
  });
});
