import { test, expect, type APIRequestContext } from '@playwright/test';
import { SEED, resetDB } from '../support/seed';
import { selfieFile } from '../support/files';

/**
 * Diagnosis photo manage flow — delete & re-add (the 2026-06-11 feature).
 *
 * Driven at the API level rather than through MySchedules UI: the manage
 * buttons only render once the translator has an `arrived` check-in, which
 * needs geolocation mocking. The endpoints themselves are the new surface, so
 * we exercise them end-to-end through the real server + DB. The modal UI is
 * covered by frontend unit tests (DiagnosisUploadModal.test.tsx).
 */

async function login(request: APIRequestContext, email: string, password: string): Promise<string> {
  const res = await request.post('/api/auth/login', { data: { email, password } });
  expect(res.ok(), `login ${email} should succeed`).toBeTruthy();
  return (await res.json()).token as string;
}

function photoPart(name: string) {
  const f = selfieFile(name);
  return { name: f.name, mimeType: f.mimeType, buffer: f.buffer };
}

test.describe('diagnosis photo manage', () => {
  test.beforeEach(async ({ baseURL }) => {
    await resetDB(baseURL!);
  });

  test('translator can upload, re-add, delete, and the slot reverts to pending when emptied', async ({ request }) => {
    const token = await login(request, SEED.translatorActive.email, SEED.password);
    const headers = { Authorization: `Bearer ${token}` };

    // Locate Alice's seeded pending SchedulePatient (the 2nd patient today).
    const sched = (await (await request.get('/api/schedules', { headers })).json()).data as Array<{
      patients?: Array<{ id: number; status: string }>;
    }>;
    const pending = sched.flatMap((s) => s.patients ?? []).find((p) => p.status === 'pending');
    expect(pending, 'seed should contain a pending patient').toBeTruthy();
    const spId = pending!.id;

    const listPhotos = async () =>
      (await (await request.get(`/api/checkins/diagnosis/photos?schedulePatientId=${spId}`, { headers })).json())
        .photos as Array<{ id: number; photoUrl: string }>;
    const statusOf = async () => {
      const data = (await (await request.get('/api/schedules', { headers })).json()).data as Array<{
        patients?: Array<{ id: number; status: string }>;
      }>;
      return data.flatMap((s) => s.patients ?? []).find((p) => p.id === spId)?.status;
    };

    // 1) Upload a single photo — only one chosen, exactly the case that used to
    //    trap the translator. Slot flips to completed.
    const up1 = await request.post('/api/checkins/diagnosis', {
      headers,
      multipart: { schedulePatientId: String(spId), photo: photoPart('d1.jpg') },
    });
    expect(up1.ok()).toBeTruthy();
    let photos = await listPhotos();
    expect(photos).toHaveLength(1);
    expect(await statusOf()).toBe('completed');
    const firstId = photos[0].id;

    // 2) Re-add another photo afterwards (the new capability).
    const up2 = await request.post('/api/checkins/diagnosis', {
      headers,
      multipart: { schedulePatientId: String(spId), photo: photoPart('d2.jpg') },
    });
    expect(up2.ok()).toBeTruthy();
    expect(await listPhotos()).toHaveLength(2);

    // 3) Delete one — one remains, so the slot stays completed.
    const del1 = await request.delete(`/api/checkins/diagnosis/photos/${firstId}`, { headers });
    expect(del1.ok()).toBeTruthy();
    photos = await listPhotos();
    expect(photos).toHaveLength(1);
    expect(await statusOf()).toBe('completed');

    // 4) Delete the last one — slot reverts to pending so it's actionable again.
    const del2 = await request.delete(`/api/checkins/diagnosis/photos/${photos[0].id}`, { headers });
    expect(del2.ok()).toBeTruthy();
    expect(await listPhotos()).toHaveLength(0);
    expect(await statusOf()).toBe('pending');
  });

  test('marking no-show clears any photos that were uploaded by mistake', async ({ request }) => {
    const token = await login(request, SEED.translatorActive.email, SEED.password);
    const headers = { Authorization: `Bearer ${token}` };

    const sched = (await (await request.get('/api/schedules', { headers })).json()).data as Array<{
      patients?: Array<{ id: number; status: string }>;
    }>;
    const spId = sched.flatMap((s) => s.patients ?? []).find((p) => p.status === 'pending')!.id;

    // Upload (slot→completed), then change mind and mark no-show.
    await request.post('/api/checkins/diagnosis', {
      headers,
      multipart: { schedulePatientId: String(spId), photo: photoPart('oops.jpg') },
    });
    const ns = await request.post('/api/checkins/no-show', {
      headers,
      data: { schedulePatientId: spId, reason: 'patient did not show' },
    });
    expect(ns.ok()).toBeTruthy();

    // Photos are gone; slot is no_show.
    const photos = (await (await request.get(`/api/checkins/diagnosis/photos?schedulePatientId=${spId}`, { headers })).json()).photos;
    expect(photos).toHaveLength(0);
  });

  test('deleting a non-existent photo returns 404', async ({ request }) => {
    const token = await login(request, SEED.translatorActive.email, SEED.password);
    const res = await request.delete('/api/checkins/diagnosis/photos/999999', {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(res.status()).toBe(404);
    expect((await res.json()).code).toBe('DIAGNOSIS_PHOTO_NOT_FOUND');
  });
});
