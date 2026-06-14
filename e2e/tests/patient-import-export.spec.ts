import { test, expect, type APIRequestContext } from '@playwright/test';
import { SEED, resetDB } from '../support/seed';

/**
 * Patient xlsx import / export / template (admin). Driven at the API level: we
 * export the seeded patients to xlsx, then re-import the same file to prove the
 * round-trip works and duplicates are skipped (not re-created).
 */

async function loginAdmin(request: APIRequestContext): Promise<string> {
  const res = await request.post('/api/auth/login', {
    data: { email: SEED.admin.email, password: SEED.password },
  });
  expect(res.ok()).toBeTruthy();
  return (await res.json()).token as string;
}

const XLSX_MIME = 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet';

test.describe('patient import / export', () => {
  test.beforeEach(async ({ baseURL }) => {
    await resetDB(baseURL!);
  });

  test('export → re-import round-trip skips duplicates; template downloads', async ({ request }) => {
    const headers = { Authorization: `Bearer ${await loginAdmin(request)}` };

    // Template downloads as a real xlsx.
    const tpl = await request.get('/api/admin/export/patients-template', { headers });
    expect(tpl.ok()).toBeTruthy();
    expect(tpl.headers()['content-type']).toContain(XLSX_MIME);

    // Export the seeded patients (3 from the seed).
    const exp = await request.get('/api/admin/export/patients', { headers });
    expect(exp.ok()).toBeTruthy();
    const xlsx = await exp.body();
    expect(xlsx.length).toBeGreaterThan(0);

    // Re-import the exact same file → every row is a duplicate → 0 created.
    const imp = await request.post('/api/admin/patients/import', {
      headers,
      multipart: { file: { name: 'patients.xlsx', mimeType: XLSX_MIME, buffer: xlsx } },
    });
    expect(imp.ok()).toBeTruthy();
    const result = await imp.json();
    expect(result.created).toBe(0);
    expect(result.skipped).toBe(SEED.patients.length);
    expect(result.errors.length).toBe(SEED.patients.length);
  });

  test('importing a non-xlsx file is rejected', async ({ request }) => {
    const headers = { Authorization: `Bearer ${await loginAdmin(request)}` };
    const res = await request.post('/api/admin/patients/import', {
      headers,
      multipart: { file: { name: 'junk.xlsx', mimeType: XLSX_MIME, buffer: Buffer.from('not a real xlsx') } },
    });
    expect(res.status()).toBe(400);
    expect((await res.json()).code).toBe('INVALID_EXCEL');
  });
});
