import { resetDB } from './support/seed';

/**
 * Single one-time reset before the suite starts. Per-spec resets happen via
 * beforeAll() hooks in each file that needs isolation.
 *
 * Also probes the stack to fail fast with a useful message if the user
 * forgot to run `npm run stack:up`.
 */
export default async function globalSetup() {
  const baseURL = process.env.E2E_BASE_URL ?? 'http://localhost:3001';
  try {
    await resetDB(baseURL);
  } catch (err) {
    console.error('\n[globalSetup] could not reach /api/test/reset on', baseURL);
    console.error('[globalSetup] is the e2e stack running? `npm run stack:up`');
    throw err;
  }
}
