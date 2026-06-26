/**
 * E2E seed identities. These mirror the constants in
 * backend/internal/handler/test_reset_handler.go — keep in sync.
 *
 * The reset endpoint (POST /api/test/reset) wipes all tables and recreates
 * exactly this dataset. Any test that needs a clean slate calls resetDB()
 * before doing its setup.
 */
export const SEED = {
  password: 'Test1234!',
  admin: {
    email: 'admin@admin.com',
    name: 'E2E Admin',
  },
  translatorActive: {
    email: 'alice@translator.com',
    name: 'Alice (active)',
    phone: '0900-000-001',
  },
  translatorDisabled: {
    email: 'bob@translator.com',
    name: 'Bob (disabled)',
    phone: '0900-000-002',
  },
  patients: [
    { name: 'Patient Passport', idType: 'passport', idNumber: 'A123456' },
    { name: 'Patient HN', idType: 'hn', idNumber: 'HN001' },
    { name: 'Patient Unid', idType: 'unid', idNumber: 'UN-XYZ' },
  ],
  // Actual-paid amount on the seeded completed visit (patients[0], today's
  // schedule). Mirrors ActualAmount in test_reset_handler.go. It is the only
  // seeded actual amount, so it equals the current-month banner total and the
  // patients[0] list/history actual-paid total; patients[1]/[2] total 0.
  seededActualPaidTotal: 1500,
} as const;

/**
 * Call /api/test/reset to wipe the DB and re-seed. Idempotent.
 *
 * This is the *only* sanctioned way to put the system into a known state in
 * an E2E test — never reach into postgres directly from a spec.
 */
export async function resetDB(baseURL: string): Promise<void> {
  const res = await fetch(`${baseURL}/api/test/reset`, { method: 'POST' });
  if (!res.ok) {
    const body = await res.text();
    throw new Error(`reset failed: ${res.status} ${body}`);
  }
}
