# 測試規格與使用說明（Testing Spec）

> 上層：[ARCHITECTURE_SPEC.md](ARCHITECTURE_SPEC.md)
> 專案測試規約（TDD）見 [CLAUDE.md](CLAUDE.md)；詳細案例見 [TEST-PLAN.md](TEST-PLAN.md) / [TEST-CASES.md](TEST-CASES.md)。

## 1. 測試金字塔

```
        ┌───────────────┐
        │   E2E (少)     │  Playwright，真實瀏覽器 + docker stack
        ├───────────────┤
        │  前端單元 (中) │  vitest + @testing-library/react
        ├───────────────┤
        │  後端單元 (多) │  go testing + testify + in-memory SQLite
        └───────────────┘
```
原則（TDD，CLAUDE.md）：**先寫失敗測試 → 最小實作 → 重構**。修 bug 先寫能重現的測試。商業邏輯、error mapping、權限、正規化「預設要寫」。純 UI/i18n/設定檔不需要。

## 2. 後端測試（Go）

**怎麼測**：service / handler / mapper 用 `testing` + `testify`（assert/require）。DB 操作用 **in-memory SQLite**（`gorm.io/driver/sqlite` `:memory:`）當 fake，不連真 postgres。

**怎麼跑**
```bash
cd backend
go test ./...                         # 全部
go test ./internal/service/... -run TestCheckin -v
go test ./... -count=1 -race -coverprofile=coverage.out   # CI 用（含 race detector）
go tool cover -func=coverage.out | tail -5                # 覆蓋率摘要
```

**測什麼（重點縫）**
- service 商業邏輯：lockout、打卡守衛、週期展開、多病人驗證、診斷狀態、id 正規化（見各 [service spec](backend/internal/service/SERVICE_SPEC.md)）。
- handler/error_mapper：sentinel error → HTTP code 對照齊全（`error_mapper_test.go`）。
- 純函式：`expandRecurrenceDates`、`normalizeIDNumber`、`getCheckinStatus`。

**為什麼用 SQLite fake**：快、無外部依賴、可平行；代價是少數 raw SQL（診斷總覽、病人歷史的 join、date 格式）在 SQLite 與 postgres 行為略不同，這些測試特別處理 date 的 `T` trim（見 [DIAGNOSIS_SERVICE_SPEC](backend/internal/service/DIAGNOSIS_SERVICE_SPEC.md) §6）。

**外部服務怎麼測**：GeocodingService 用 `SetBaseURL` 注入假伺服器；Export/Notification/Mail 避免實際外呼或檢查「未設定時回錯」路徑。

## 3. 前端測試（TypeScript）

**怎麼測**：`vitest` + `@testing-library/react`。元件以 props 注入 API 函式（預設真實），測試時注 mock。

**怎麼跑**
```bash
cd frontend
npm test                  # vitest（CI 跑這個）
npm test -- --watch       # watch 模式
npx tsc --noEmit          # 型別檢查
npm run lint              # ESLint
```

**測什麼**
- 頁面：Login（401 顯示 toast 不導向）、AdminManagement、ChangePassword。
- 元件：DiagnosisUploadModal（≤3 張）、NoShowModal（reason 必填）、PatientPicker（debounce）、SchedulePatientListEditor、MapLink。
- 純邏輯：`api/client`（`unwrapResponse`、`mapErrorResponse` 的 401/403 分流，刻意 export 供測）、`utils/apiError`、`utils/schedulePatient`、i18n。
- 細節見各 [frontend spec](frontend/src/FRONTEND_SPEC.md)。

## 4. E2E 測試（Playwright）
完整說明見 [E2E_SPEC](e2e/E2E_SPEC.md)。最短路徑：
```bash
cd e2e && ./run-e2e.sh --down
```
打獨立 docker stack（[DOCKER_SPEC §4](docker/DOCKER_SPEC.md)），用 `/api/test/reset` 在每個 spec 前回到固定 seed。

## 5. CI（.github/workflows/test.yml）
觸發：push 到 `main`/`master`/`feature/**`、PR 到 `main`/`master`。**兩個平行 job**：

| job | 步驟 |
|-----|------|
| **Backend (Go 1.26)** | `go mod download` → `go build ./...` → `go vet ./...` → `go test ./... -count=1 -race -coverprofile` → 覆蓋率摘要 |
| **Frontend (Node 20)** | `npm ci` → `tsc --noEmit` → `npm run lint --if-present` → `npm test` → `npm run build` |

> E2E 目前**不在 CI**（需 docker stack）；屬本機 / 手動驗收。要進 CI 需在 workflow 起 e2e compose。

## 6. 提交前自我檢查（對齊 CLAUDE.md）
```bash
cd backend  && go test ./...     # 後端全綠
cd frontend && npm test          # 前端全綠
# 動到流程時：cd e2e && ./run-e2e.sh --down
```
每個 staged commit 結束跑後端 + 前端測試；不過不 merge。

## 7. 已知限制
- E2E 未自動化進 CI。
- 後端整合層（真 postgres 行為、FK 級聯）主要靠 E2E 覆蓋，單元層用 SQLite 近似。
- 無前端 e2e 覆蓋率統計。
