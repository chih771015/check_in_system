# 更動報告 — 2026-06-01：E2E 測試框架（Playwright）

本次 session 加入完整的端對端測試框架。後端、infra、E2E workspace 全部就位，11 個 spec file 含 happy / 錯誤情境，總共約 25 個 cases（部分 `test.skip` 標記待後續補完）。

## Commits

預定一次 staged commit（本檔案為最後一層）。

## 變更摘要

### 後端：build-tag 隔離的 reset endpoint

- 新增 `internal/handler/test_reset_handler.go`（`//go:build e2e`）
  - `RegisterTestResetRoutes(r, db, uploadDir)` 註冊 `POST /api/test/reset`
  - 三層防護：build tag `e2e` + env `ENABLE_TEST_RESET=true` + `GIN_MODE != release`
  - reset 行為：TRUNCATE 全表（FK 安全順序）→ 清空 upload dir → 重 seed
  - seed 內容：1 admin + 2 translators (active/disabled) + 3 patients (各 idType) + 1 含 completed sp + photo 的歷史排班
- 新增 `internal/handler/test_reset_stub.go`（`//go:build !e2e`）
  - 同名 no-op 函式，讓 main.go 可無條件呼叫
- `cmd/server/main.go` 增加 `handler.RegisterTestResetRoutes(r, db, cfg.UploadDir)` 呼叫
- 兩種 build (`go build` / `go build -tags e2e`) + `go vet` 皆通過
- 既有 backend tests 全綠（未改 production 行為）

### Infra：獨立 E2E stack

- 新增 `docker/Dockerfile.backend.e2e`：唯一差別是 `go build -tags e2e`
- 新增 `docker/docker-compose.e2e.yml`：
  - 獨立 postgres volume (`postgres_e2e_data`) + 獨立 uploads volume
  - postgres : 55432 / backend : 8081 / frontend : 3001（dev stack 不衝突）
  - 不跑 jaeger
  - env 注入 `ENABLE_TEST_RESET=true` + `GIN_MODE=debug`
  - 套用 `ADMIN_DEFAULT_PASSWORD=Test1234!` 與 seed 一致
- 共用 `docker/nginx.conf`（已有 `/api/` + `/uploads/` proxy）

### E2E workspace

```
e2e/
├── package.json              # scripts: stack:up / stack:down / test / test:ui / reset
├── playwright.config.ts      # baseURL + chromium-desktop + mobile-chrome projects
├── tsconfig.json
├── global-setup.ts           # 一次性 resetDB() + 失敗時提示 stack:up
├── support/
│   ├── seed.ts               # SEED 常數 + resetDB() helper
│   └── auth.ts               # loginAsAdmin / loginAsTranslator via 真實 form
├── tests/
│   ├── auth.spec.ts                # 5 case：admin/translator/wrong pw/disabled/lockout
│   ├── schedule-crud.spec.ts       # 2 case：create / delete
│   ├── schedule-validation.spec.ts # 2 placeholder (待 test-id)
│   ├── translator-checkin.spec.ts  # 2 case + 1 placeholder
│   ├── diagnosis-flow.spec.ts      # 2 case
│   ├── makeup-checkin.spec.ts      # 1 case + 1 placeholder
│   ├── patient-history.spec.ts     # 2 case
│   ├── translator-mgmt.spec.ts     # 3 case
│   ├── i18n.spec.ts                # 1 case（locale 切換不爆）
│   ├── responsive.spec.ts          # 1 case (mobile-chrome project)
│   ├── export.spec.ts              # 1 case + 1 placeholder
│   └── errors.spec.ts              # 2 case
└── README.md                # 操作指南 + 安全模型說明
```

- 預設 serial 跑（`workers: 1` + `fullyParallel: false`），因為共享 DB 經 reset 隔離
- `tsc --noEmit` 通過

### 設計決策摘要

| 議題 | 結論 |
|---|---|
| reset 機制 | Option A: build-tag e2e 的 endpoint（vs psql + seed CLI）→ 安全模型最強 |
| reset 觸發 | `beforeAll` per spec，需嚴格隔離的 spec 用 `beforeEach` |
| 測試身分 | 固定 seed 帳號（admin / alice / bob），新增實體用 `Date.now()` 後綴避撞 |
| 環境隔離 | 獨立 docker-compose stack + 獨立 volume，dev 永遠不會被擦 |
| Lockout 測試 | 連續錯 6 次 → 第 7 次正確密碼仍應被拒 |

### 待補（已標 `test.skip`）

- `schedule-validation.spec.ts`：需先在 form 加 `data-testid` 才能穩定選元素
- `translator-checkin.spec.ts`：需 "today's schedule" 第二種 seed 變體
- `makeup-checkin.spec.ts`：需 file upload helper + geolocation mock
- `export.spec.ts`：Google Sheet 需 service account env

## 安全模型

reset endpoint **不可能**意外進 production binary：
1. Build tag：production `Dockerfile.backend` 不傳 `-tags e2e` → handler code 根本沒編進去
2. Env flag：`ENABLE_TEST_RESET=true` 才註冊路由
3. `GIN_MODE=release` 直接拒絕註冊

三層任一失守都還有下一層。

## 操作指南

```bash
# 首次
cd e2e
npm install
npx playwright install chromium

# 跑測試
npm run stack:up        # docker-compose up -d --build
npm test                # playwright test
npm run test:ui         # UI mode debug

# 清乾淨
npm run stack:down      # docker-compose down -v
```

## 影響檔案範圍

**後端**
- `internal/handler/test_reset_handler.go`（新，build tag e2e）
- `internal/handler/test_reset_stub.go`（新，build tag !e2e）
- `cmd/server/main.go`：加 `handler.RegisterTestResetRoutes(r, db, cfg.UploadDir)` 呼叫

**Infra**
- `docker/Dockerfile.backend.e2e`（新）
- `docker/docker-compose.e2e.yml`（新）

**E2E**
- 整個 `e2e/` 目錄（新）

## 後續

- 跑一次完整 suite，補 selector / timing 微調
- 補 `data-testid` 到表單關鍵 input → 讓 `test.skip` 的 schedule-validation 可開
- Husky pre-push（下次 session）
- 視需要把 E2E 加進 GitHub Actions（會多 5-10 分鐘 CI 時間）
