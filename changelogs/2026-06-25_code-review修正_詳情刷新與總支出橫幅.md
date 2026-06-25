# 更動報告 — Code Review 修正：詳情 Modal 刷新 + 當月總支出橫幅不刷新

- 日期：2026-06-25
- 分支：`feature/local-expose`
- 來源：本分支 5 項需求改動的 code review（高強度）發現的兩條問題

## Commits

- `fix(frontend): 排班詳情 Modal 上傳/標記後依現用篩選刷新`
- `fix(frontend): 後台當月總支出橫幅於換頁時重新整理`
- `docs: 更動報告 code review 修正`

## 做了什麼

### 修正1（regression，最高優先）— 詳情 Modal 刷新抓不到舊排班
- 背景：需求5 讓「無篩選」的排班列表只回最近建立的 100 筆。
- 問題：詳情 Modal 在上傳診斷／標記 no_show 後，用 `getAdminSchedules({})`（無篩選＝最近 100 筆）再 `find(detailRecord.id)` 刷新；若該排班是較舊、不在 100 筆內 → `find` 回 undefined → Modal 維持舊資料。
- 修法：
  - `fetchData` 改為回傳取得的列表（`Promise<ScheduleItem[]>`）。
  - 新增 `refreshAndSyncDetail`：呼叫 `fetchData()`（沿用目前 `filters`）後，用同一份列表同步開啟中的 `detailRecord`。因為詳情是從「目前篩選結果」開啟的，用同一篩選刷新可保證該筆一定在結果內。
  - 兩個 `getAdminSchedules({})` 刷新呼叫移除，改用 `refreshAndSyncDetail`（同時少一次重複請求）。

### 修正2 — 當月總支出橫幅整個 session 不刷新
- 問題：橫幅只在 AppLayout mount 抓一次（deps `[isAdmin]`），AppLayout 換頁不 remount → 改了某病人實付後橫幅維持舊值、跨午夜後月份標籤也不更新。
- 修法：effect deps 加入 `location.pathname`，每次後台換頁重新抓當月總額。沿用既有 best-effort（失敗靜默）。

## TDD / 驗證

- 前端（vitest）`ScheduleManagement.test.tsx` 新增 regression 測試：套用 location 篩選 → 開詳情 → 開上傳 → 觸發 `onUploaded`，斷言刷新呼叫 `getAdminSchedules({location:'VGH'})` 且**從未**以 `{}` 呼叫。
- 前端（vitest）`AppLayout.test.tsx` 新增：admin 換頁時 `getMonthlyTotal` 由 1 次→2 次（不再 stale）。既有 banner 顯示/非 admin 不請求測試維持綠。
- `tsc` 乾淨、eslint 乾淨、前端全套 **18 檔 / 98 測試全綠**。

## 影響檔案

- `frontend/src/pages/admin/ScheduleManagement.tsx`（fetchData 回傳列表 + refreshAndSyncDetail）
- `frontend/src/components/AppLayout.tsx`（橫幅換頁刷新）
- 對應測試 `ScheduleManagement.test.tsx`、`AppLayout.test.tsx`

## 備註（review 其餘findings，未在本次處理）

- 效能：`GetHistory` 的 N+1 照片查詢 + 每次改區間重抓整份歷史、且日期過濾在 Go 端（未下推 SQL）。
- 一致性：三條「區間加總」邊界慣例不一（歷史閉區間 vs SQL 半開）；「實付總額」三種時間範圍共用相近標籤。
- 清理：`SumActualByDateRange` / `SumActualByPatientDateRange` 幾乎重複；`NT$` 格式化重複 4 處；`schedules.date` 無索引。
  以上列為後續可處理項，未動本次範圍。
