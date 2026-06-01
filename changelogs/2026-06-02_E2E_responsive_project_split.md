# 更動報告 — 2026-06-02：E2E responsive spec 限定 mobile-chrome 跑

## 背景

前一輪 `9adb025` 修完後剩 1 個 fail：
- `responsive / mobile sidebar collapses after menu tap` 在 chromium-desktop 上失敗（timeout）

playwright.config.ts 的 mobile-chrome project 設了 `testMatch: /responsive/`，但 chromium-desktop 沒設 `testIgnore`，所以 responsive spec **兩邊都跑**了。Desktop viewport 上沒有抽屜可收，menu item 永遠被 main layout 攔住，必然 timeout。

## 變更

`e2e/playwright.config.ts`：chromium-desktop project 加 `testIgnore: /responsive\.spec\.ts/`。

之後：
- chromium-desktop 跑所有 spec **除了** responsive
- mobile-chrome **只跑** responsive

## Commit

| Hash | 說明 |
|---|---|
| (本 commit) | fix(e2e): chromium-desktop 加 testIgnore 排除 responsive spec |
