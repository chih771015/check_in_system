# Translator Check-in System

A full-stack web application for a translation agency. Administrators schedule on-site
interpreter assignments at hospitals; interpreters check in on location with **GPS +
photo verification**. Built end to end with a test-driven workflow, distributed tracing,
and a containerized deployment.

> **Tech:** React 19 · TypeScript · Vite · Ant Design 5 · Go (Gin) · GORM · PostgreSQL ·
> JWT · Docker Compose · OpenTelemetry → Jaeger · Vitest · Playwright · GitHub Actions

---

## Why this project

It models a real operational problem — proving that an interpreter was physically present
for a paid assignment — and solves it with the kind of concerns a production system needs:
authentication and role-based access, an auditable record, recurring scheduling,
report export, notifications, and observability. The goal was not a toy CRUD app but a
realistic vertical slice of a fintech-style operations product.

---

## Architecture

```
                    ┌──────────────────────────────────────────────┐
   Browser  ──────▶ │  nginx  (serves SPA, proxies /api)            │
   (React SPA)      └───────────────┬──────────────────────────────┘
                                    │  /api/*
                                    ▼
        ┌───────────────────────────────────────────────────────────┐
        │  Go / Gin API                                             │
        │                                                          │
        │  Handler  ──▶  Service  ──▶  Repository  ──▶  GORM        │
        │  (HTTP,        (business     (data access)   (ORM)        │
        │   DTO map)      logic)                                    │
        │                                                          │
        │  Middleware: JWTAuth ▶ RequirePasswordChanged ▶ RoleReq  │
        │  Cron: scheduled exports · reminders · photo cleanup     │
        └───────┬───────────────────────────────┬──────────────────┘
                │                               │ OTLP/gRPC (spans)
                ▼                               ▼
        ┌──────────────┐                 ┌──────────────┐
        │ PostgreSQL   │                 │  Jaeger      │
        └──────────────┘                 └──────────────┘
```

**Layered backend.** Each request flows `Handler → Service → Repository → GORM`. Handlers
only deal with HTTP concerns and DTO mapping; business rules live in services; data access
is isolated in repositories. This keeps services unit-testable against an in-memory database
with no HTTP or real Postgres involved.

**Context propagation for tracing.** Every repository exposes a `WithCtx(ctx)` method and
every service method takes `ctx context.Context` as its first argument, so a single trace
spans the HTTP request, the business logic, and every SQL query.

---

## Features

**Accounts & access**
- Admin-managed interpreter accounts; forced password change on first login
- JWT (HS256) authentication with role-based authorization (`admin` / `translator`)

**Scheduling**
- Single and recurring assignments (e.g. every Mon/Wed/Fri until a end date)
- Recurring instances are independent rows grouped by a `recurrence_group_id`, so a single
  occurrence can be edited or deleted without touching the rest
- Filtering by interpreter, date range, and location

**Check-in**
- On-arrival and on-leave check-in with **GPS coordinates** (reverse-geocoded to an address)
  and **selfie + environment photo**
- Makeup check-in (with reason) when an interpreter forgets at the time
- Guard rails: no duplicate check-in, no leave-before-arrival

**Admin & reporting**
- Dashboard of today's attendance; per-record detail with photos and map location
- Excel export (`excelize`) and Google Sheet export
- Scheduled monthly exports emailed automatically (cron)
- Audit log of administrative actions

**Notifications**
- LINE / Telegram reminders for upcoming and missed check-ins; email reminders

**Observability**
- Distributed tracing across HTTP → service → SQL via OpenTelemetry, exported to Jaeger
- PII scrubbing: bound query parameters and sensitive headers are stripped from spans;
  span names use the route template (not raw URLs) to keep cardinality low

---

## Tech stack

| Layer        | Choices |
|--------------|---------|
| Frontend     | React 19 (hooks), TypeScript, Vite, Ant Design 5, axios, i18next, zustand |
| Backend      | Go, Gin, GORM, golang-jwt v5, bcrypt, robfig/cron, excelize |
| Database     | PostgreSQL 16 (GORM AutoMigrate); SQLite in-memory for tests |
| Tracing      | OpenTelemetry SDK → OTLP/gRPC → Jaeger |
| Infra        | Docker Compose, nginx |
| Tests        | Go `testing` + `testify`; Vitest + Testing Library; Playwright (E2E) |
| CI           | GitHub Actions (build, vet, `go test -race` + coverage, type-check, lint, unit tests) |

---

## Testing

Test-driven throughout. New behavior starts with a failing test; bug fixes start with a
test that reproduces the bug.

- **Backend** — Go `testing` + `testify`, with an **in-memory SQLite** database standing in
  for Postgres so service and handler logic is tested without external dependencies. CI runs
  the suite with the **race detector** and a coverage report.
- **Frontend** — Vitest + React Testing Library across components, stores, API layer, hooks,
  utilities and i18n.
- **End-to-end** — Playwright specs driving the real UI against a dedicated Docker stack. A
  test-only reset endpoint is compiled in **only** under the `e2e` build tag **and** an
  `ENABLE_TEST_RESET` env flag, so it can never exist in a production binary.

---

## Running locally

```bash
# Full stack (Postgres + Jaeger + backend + frontend)
cd docker
docker compose up --build
```

| Service        | URL |
|----------------|-----|
| Frontend (SPA) | http://localhost:3000 |
| API            | http://localhost:8080/api |
| Jaeger UI      | http://localhost:16686 |

A default admin account is seeded on first boot. To reset a password without the UI:

```bash
docker exec thai-backend ./server -reset-password admin@admin.com "NewPass123"
```

### Backend / frontend individually

```bash
# Backend
cd backend && go run ./cmd/server

# Frontend
cd frontend && npm install && npm run dev
```

---

## Notable design decisions

- **Tracing must never take down the API.** If the Jaeger collector is unreachable, the app
  still boots; spans are buffered and retried in the background. The E2E stack disables the
  exporter entirely via `OTEL_TRACES_EXPORTER=none`.
- **PII never leaves the process in a span.** The GORM tracing plugin is configured with
  `WithoutQueryVariables`, and a middleware scrubs sensitive headers/fields after the span
  is created.
- **Recurring schedules are materialized, not virtual.** Generating concrete rows (grouped
  by id) keeps per-occurrence editing/deletion simple and the check-in model uniform.
- **The test reset endpoint is build-tag + env gated** so production builds physically cannot
  contain it.

---

## Project layout

```
backend/
  cmd/server/        # entrypoint, routing, cron wiring
  internal/
    handler/         # HTTP handlers (DTO in/out)
    service/         # business logic (ctx-first methods)
    repository/      # data access (WithCtx pattern)
    middleware/      # JWT auth, role checks, password-change gate
    model/           # GORM models
    dto/             # request/response shapes
    tracing/         # OpenTelemetry setup
    config/          # env configuration
frontend/
  src/
    api/             # axios client + per-resource API modules
    stores/          # zustand state
    pages/           # admin/ and translator/ screens
    components/ hooks/ utils/ i18n/
docker/              # Dockerfiles, compose files, nginx config
e2e/                 # Playwright specs
```
