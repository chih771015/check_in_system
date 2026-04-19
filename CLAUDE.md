# Project Memory — Translator Check-in System

## Architecture

- **Backend**: Go + Gin + GORM + PostgreSQL
- **Frontend**: React + TypeScript + Vite + Ant Design v5
- **Infra**: Docker Compose (postgres, jaeger, backend, frontend)
- **Tracing**: OpenTelemetry SDK → OTLP/gRPC → Jaeger

## Conventions

### Commit Style
- Prefix: `feat(backend):`, `feat(frontend):`, `feat:`, `fix:`, `docs:`
- Language: Chinese commit messages (this is a Taiwanese team project)
- Always include `Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>`

### Context Propagation Pattern (Tracing)
Every new repository must include a `WithCtx` method:
```go
func (r *XxxRepository) WithCtx(ctx context.Context) *XxxRepository {
    return &XxxRepository{db: r.db.WithContext(ctx)}
}
```
Every service method that is called from an HTTP handler must accept `ctx context.Context` as the first parameter and use `repo.WithCtx(ctx)` internally. Handlers pass `c.Request.Context()`.

### Staged Commit Workflow
When making large changes across multiple files, organize commits by logical layer:
1. Models / Config / DTOs / Middleware (infrastructure)
2. New services (independent modules)
3. Repository + Service + Handler wiring (integration)
4. Infrastructure (Docker, tracing, CI)
5. Frontend UI
6. Documentation / Reports

Each commit message should include a Chinese summary and a bullet list of what changed and why.
