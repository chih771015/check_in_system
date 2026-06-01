import { test } from '@playwright/test';
import { resetDB } from '../support/seed';

test.describe('makeup checkin', () => {
  test.beforeAll(async ({ baseURL }) => {
    await resetDB(baseURL!);
  });

  // Makeup checkin lives at `/makeup/:scheduleId/:type` and is reached via
  // an action on the my-schedules page, not a direct nav link. There's no
  // standalone "Makeup Checkin" page to test in isolation.
  //
  // A full flow needs:
  //   - geolocation permission
  //   - selfie file fixture
  //   - "makeup reason" text
  // Deferred until we add a shared upload helper.
  test.skip('translator submits a makeup checkin via my-schedules action', async ({ page: _ }) => {
    // TODO: implement once upload + geolocation fixtures land.
  });
});
