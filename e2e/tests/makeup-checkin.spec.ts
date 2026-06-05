import { test, expect } from '@playwright/test';
import { resetDB } from '../support/seed';
import { loginAsTranslator } from '../support/auth';
import { selfieFile } from '../support/files';

/**
 * Makeup checkin lives at `/makeup/:scheduleId/:type`. On a phone it is reached
 * via an action button on /my-schedules once a schedule is past its end time.
 * We drive the page directly using the seed's stable schedule id (id=1 via
 * TRUNCATE ... RESTART IDENTITY) so the test is independent of time of day —
 * the page works standalone and exercises the same POST /api/checkins/makeup
 * flow (reason + selfie + GPS).
 */
const SEED_SCHEDULE_ID = 1;

test.describe('makeup checkin', () => {
  test.use({
    geolocation: { latitude: 13.7563, longitude: 100.5018 },
    permissions: ['geolocation'],
  });

  test.beforeEach(async ({ baseURL }) => {
    await resetDB(baseURL!);
  });

  test('translator submits a makeup checkin', async ({ page }) => {
    await loginAsTranslator(page);
    await page.goto(`/makeup/${SEED_SCHEDULE_ID}/arrive`);

    // The schedule detail card renders once getMySchedules resolves.
    await expect(page.locator('body')).toContainText('E2E Clinic, Bangkok');

    // Makeup reason is required by the page before submit will fire.
    await page.getByPlaceholder(/Makeup Reason|補.*原因|เหตุผล/i).fill('forgot to check in on arrival');

    await page.locator('input[type="file"]').setInputFiles(selfieFile());

    const submit = page.getByRole('button', { name: /Submit Check-in|送出/i });
    await expect(submit).toBeEnabled({ timeout: 10_000 });
    await submit.click();

    // On success the page navigates back to the schedule list.
    await expect(page).toHaveURL(/\/my-schedules/);
  });
});
