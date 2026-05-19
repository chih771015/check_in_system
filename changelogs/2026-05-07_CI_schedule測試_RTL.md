# 更動報告 — 2026-05-07：CI Pipeline + ScheduleService 測試 + RTL 互動測試

## 背景

完成測試生態三件套：CI 自動化、業務核心 ScheduleService 測試、前端 RTL 互動測試。

## 完成項目

### 1. CI Pipeline（`.github/workflows/test.yml`）

`push` 與 `pull_request` 觸發，兩個 job 並行：

**Backend job**：
- `actions/setup-go@v5`（go 1.26）+ go module cache
- `go mod download` → `go build ./...` → `go vet ./...` → `go test ./... -race -coverprofile`
- 印出 coverage summary

**Frontend job**：
- `actions/setup-node@v4`（node 20）+ npm cache
- `npm ci` → `npx tsc --noEmit` → `npm run lint --if-present` → `npm test` → `npm run build`

之後每次 push / PR 都會自動跑，紅了會擋 merge。

### 2. ScheduleService 測試（21 cases）

`backend/internal/service/schedule_service_test.go`：

**expandRecurrenceDates 純函式（5）**：
- daily 連續日期
- weekly:1,3,5 跨週展開
- monthly:31 在 2 月自動 clamp 到 28，4 月 clamp 到 30
- 未知 rule → error
- weekly 範圍外的星期值（9） → error

**Create（7）**：
- 單筆成功 + checkinStatus=none
- TranslatorID 不存在 → ErrTranslatorNotFound
- 對象 role 是 admin 而非 translator → ErrNotATranslator
- 日期格式錯 → ErrInvalidDateFormat
- recurrenceRule 設了但缺 recurrenceUntil → ErrRecurrenceUntilReq
- recurrenceUntil 早於 date → ErrRecurrenceBeforeStart
- daily 重複生成多筆，且共享同一 RecurrenceGroupID

**Update（2）**：
- 排班不存在 → ErrScheduleNotFound
- 只傳 Location 一個欄位 → 其他欄位保留

**Delete（2）**：
- 不存在 → ErrScheduleNotFound
- 刪除排班時連帶刪 checkins（FK 處理）

**DeleteRecurrenceGroup（2）**：
- 無 group_id 的單筆排班 → fallback 為 single delete（count=1）
- 4 天 daily group → 一次刪 4 筆

**List CheckinStatus 優先序（1）**：
- arrive+leave → completed；只 arrive → arrived；無 → none

**BatchImport（2）**：
- 混合 5 行：upstream error / 日期錯 / translator 不存在 / 2 筆成功 → 2 success + 3 failed，失敗行號保留
- admin 用戶 ID 被視為 "translator not found"

### 3. RTL 互動測試（6 cases）

`frontend/src/pages/__tests__/Login.test.tsx`：

- 預設英文 placeholder 渲染（"Email" / "Password" / "Sign In"）
- 切到 zh-TW 後顯示中文 placeholder
- admin 登入成功 → `navigate('/admin/translators')`
- mustChangePW=true → `navigate('/change-password')`
- translator 登入成功 → `navigate('/my-schedules')`
- 登入失敗 → 顯示翻譯後的 toast "Invalid email or password"

**技巧**：
- `vi.mock('../../api/auth')` 假造 login function
- `vi.mock('react-router-dom')` 攔截 useNavigate
- `userEvent.setup({ delay: null })` 取消每字延遲，否則 antd Form 跑很慢
- `vitest.config.ts` testTimeout 提高到 15s（antd + i18n 在 happy-dom 初始化偏慢）

## 測試現況

| 範圍 | Test files | Test cases |
|------|-----------|-----------|
| 後端 dto | 1 | 2 |
| 後端 handler | 1 | 27 |
| 後端 service | 4 | 49 |
| **後端小計** | **6** | **78** |
| 前端 i18n | 1 | 6 |
| 前端 api/client | 1 | 10 |
| 前端 pages/Login | 1 | 6 |
| **前端小計** | **3** | **22** |
| **總計** | **9** | **100** |

`go test ./...` 全綠；`npm test` 全綠。CI 設定齊全。

## 影響範圍

| 檔案 | 變動 |
|------|------|
| `.github/workflows/test.yml` | 新（CI）|
| `backend/internal/service/schedule_service_test.go` | 新（21 cases）|
| `frontend/vitest.config.ts` | 加 testTimeout |
| `frontend/src/pages/__tests__/Login.test.tsx` | 新（6 cases）|

## 仍未涵蓋

- 後端：checkin_service / export_service（外部 API 依賴多）
- 後端：handler 層 HTTP integration test（gin testserver）
- 前端：其他頁面 RTL（AdminMgmt 刪除流程、PatientMgmt 重複驗證、ScheduleMgmt 表單）
- 前端：authStore Provider 測試

但 CI 已啟動，TDD 規約在 CLAUDE.md，後續新功能會自動進測試流程。
