# 更動報告 — 2026-06-01：E2E 預設啟用 HTML reporter

## Commit

| Hash | 說明 |
|---|---|
| (本 commit) | chore(e2e): 預設一律輸出 HTML reporter 方便 debug |

## 變更摘要

`e2e/playwright.config.ts` 的 reporter 設定原本是：
- CI：`[['github'], ['html']]`
- local：`'list'`（只有 console 輸出，沒生 HTML 報告）

結果今天 debug 16 failed tests 時 `npx playwright show-report` 找不到報告。改成：
- CI：`[['github'], ['html', { open: 'never' }]]`
- local：`[['list'], ['html', { open: 'never' }]]`

兩種環境都生 HTML 報告但不自動開瀏覽器（CI 不需要、local 想看時自己 `npm run report`）。

## 影響檔案

- `e2e/playwright.config.ts`：reporter 欄位
