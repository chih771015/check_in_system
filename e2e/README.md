# E2E — Playwright

End-to-end browser tests for the translator check-in system.

## Architecture

- Tests target an isolated docker-compose stack (`docker/docker-compose.e2e.yml`,
  project name `thai-e2e`) so the dev DB / uploads are never touched.
- Backend is built with `-tags e2e` so the **/api/test/reset** endpoint is
  available. That endpoint is gated by three layers (build tag + env flag +
  GIN_MODE check) — see `backend/internal/handler/test_reset_handler.go`.
- Each spec calls `resetDB()` in `beforeAll` (or `beforeEach` when isolation
  matters), which truncates the DB and reseeds a deterministic dataset.

## Seed dataset

| Identity | Email | Password | Role | Status |
|---|---|---|---|---|
| Admin | `admin@admin.com` | `Test1234!` | admin | active |
| Alice | `alice@translator.com` | `Test1234!` | translator | active |
| Bob | `bob@translator.com` | `Test1234!` | translator | disabled |

Plus 3 patients (passport / hn / unid) and 1 historical schedule for Alice
on yesterday's date with 2 SchedulePatients (1 completed + 1 photo, 1 pending).

## Running

```bash
# from repo root, first time only:
cd e2e
npm install
npx playwright install chromium

# bring the stack up (one-shot)
npm run stack:up

# run all specs against the stack
npm test

# debug with UI mode
npm run test:ui

# tear down when done
npm run stack:down
```

## Spec layout

| File | Coverage |
|---|---|
| `auth.spec.ts` | Login (admin / translator / wrong pw / disabled / lockout) |
| `schedule-crud.spec.ts` | Admin create + delete schedule |
| `schedule-validation.spec.ts` | clamp / validate patient times (placeholder, needs test-ids) |
| `translator-checkin.spec.ts` | Translator schedule list + checkin page render |
| `diagnosis-flow.spec.ts` | Admin sees completed patient + opens photo modal |
| `makeup-checkin.spec.ts` | Makeup checkin page render (deeper flow is TODO) |
| `patient-history.spec.ts` | Patient history page + Image.PreviewGroup wiring |
| `translator-mgmt.spec.ts` | Create / disable translator |
| `i18n.spec.ts` | Locale switching does not crash |
| `responsive.spec.ts` | Mobile sidebar auto-collapse (mobile-chrome project) |
| `export.spec.ts` | Excel export download |
| `errors.spec.ts` | Auth redirect + no-reload-on-login regression |

`test.skip` markers point to flows that need richer fixtures (today-dated
schedules, image upload helpers, geolocation mocks) — pick those up
incrementally.

## Adding new specs

1. Put the file under `tests/`.
2. Call `resetDB(baseURL!)` in `beforeAll` if the test needs the seed state.
   Otherwise the file inherits state from whatever ran before (which is
   non-deterministic — don't rely on it).
3. Prefer role / placeholder / text locators over CSS classes; antd class
   names occasionally change between versions.

## Safety

The reset endpoint cannot accidentally exist in a production binary:
1. Build tag `e2e` is required — the production `Dockerfile.backend` does
   not pass it, so the handler code is not compiled in.
2. `ENABLE_TEST_RESET=true` env flag — refuses to register otherwise.
3. `GIN_MODE != release` — refuses to register in release mode.

All three layers must be intentionally enabled. The e2e docker-compose
sets layers 1 and 2; layer 3 is the final backstop.
