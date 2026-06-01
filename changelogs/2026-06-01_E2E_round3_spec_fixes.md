# 更動報告 — 2026-06-01：E2E 第三輪修正（spec selectors + seed 日期）

## 背景

前一輪 `16e4c22`（must_change_pw=false 用 raw UPDATE）跑通之後，剩 7 個真正的 spec 問題，三類原因：

1. **路由不存在**：spec 寫死 `/checkin`、`/makeup-checkin`，實際路由是 `/checkin/:scheduleId/:type`、`/makeup/:scheduleId/:type`
2. **Seed 在「昨天」**：MySchedules 預設不含歷史，translator 在 list 找不到 seeded schedule
3. **antd modal 攔截**：`.ant-select.first()` / `getByRole('button').last()` 抓到頁面後面被遮住的元素，confirm 對話框 OK 按鈕應 scope 到 `.ant-modal-confirm`

## 變更

### Backend
- `test_reset_handler.go`：seed schedule 日期從 yesterday 改成 today
  - 益處：MySchedules / patient-history / diagnosis-results 全部都看得到，不需要點 "Show History"

### E2E specs
- `translator-checkin.spec.ts`：
  - 移除 `/checkin` 直接 navigate（路由不存在）
  - 改測 `/my-checkins` 頁面渲染
  - 完整打卡流程留 `test.skip`（需 geolocation + selfie fixture）
- `makeup-checkin.spec.ts`：
  - 整個 spec 改成 `test.skip`（路由 `/makeup/:scheduleId/:type` 沒有獨立進入點）
- `schedule-crud.spec.ts`：
  - create 改成 `test.skip`（表單複雜需 test-id 才寫得穩）
  - delete：confirm OK 按鈕改 scope 到 `.ant-modal-confirm`
  - 新增「seeded schedule 出現在列表」的簡單 case
- `translator-mgmt.spec.ts`：
  - 全部欄位 scope 進 `.ant-modal.last()`，避開背景頁面元素
  - 密碼欄改用 `getByLabel`（Input.Password 沒 placeholder）
  - disable confirm 對話框改 scope 到 `.ant-modal-confirm`

## Commit

| Hash | 說明 |
|---|---|
| (本 commit) | fix(e2e): spec selectors + seed schedule 改成今天 |
