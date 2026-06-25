# Project Memory — Translator Check-in System

## Architecture

- **Backend**: Go + Gin + GORM + PostgreSQL
- **Frontend**: React + TypeScript + Vite + Ant Design v5
- **Infra**: Docker Compose (postgres, jaeger, backend, frontend)
- **Tracing**: OpenTelemetry SDK → OTLP/gRPC → Jaeger

## Conventions

### TDD — Test-Driven Development（必須遵守）

**所有新功能或重構必須遵守 TDD 流程：**

1. **先寫測試（Red）**：在動任何 production code 前，先寫一個失敗的測試表達期望行為
2. **最小實作（Green）**：寫剛好夠讓測試過的程式碼，不過度設計
3. **重構（Refactor）**：紅綠循環後再清理結構

**規則：**
- 後端用 Go `testing` + `testify`（assert/require），DB 操作用 in-memory SQLite (`gorm.io/driver/sqlite` `:memory:`）作 fake
- 前端用 `vitest` + `@testing-library/react`
- service 層的商業邏輯、handler 的錯誤映射、純函式必須有測試
- 每個 PR / staged commit 結束時跑 `go test ./...`（後端）與 `npm test` + `npm run typecheck`（前端），不過不 merge
  - ⚠️ 前端型別檢查**必須**用 `npm run typecheck`（= `tsc -b`，走 project references 檢查 `src` 含測試檔）。直接 `npx tsc --noEmit` 因根 `tsconfig.json` 是 `files: []` + references，**不會檢查任何檔案**（等於沒檢查）；vitest 也不做型別檢查。型別破口只有 `tsc -b` / docker build 抓得到。
- 修 bug 時：先寫一個會 reproduce bug 的失敗測試，再修，避免再犯

**不需要 TDD 的部分：**
- 純 UI layout 調整（無邏輯）
- i18n 字串替換
- 設定檔 / docker / 文件

當不確定要不要寫測試時，**預設要寫**。

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

**Mandatory: Changelog file after every task**
After every staged commit, write a markdown file to `changelogs/`:
- Filename: `changelogs/YYYY-MM-DD_簡短描述.md`
- Content: date, commit hashes + messages, summary of what was done, affected modules, notes
- Commit the changelog file as the final commit of the session (or include in last layer)
- Use `/staged-commit` slash command which enforces this step
