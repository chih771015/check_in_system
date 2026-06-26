# e2e — 規格與使用說明（Playwright）

> 對應檔案：`e2e/*`
> 上層：[ARCHITECTURE_SPEC.md](../ARCHITECTURE_SPEC.md)｜實作步驟另見 [README.md](README.md) / [HOW_TO_RUN.md](HOW_TO_RUN.md)

## 1. 定位與職責
用 Playwright 在真實瀏覽器跑端對端流程，打的是**獨立的 e2e docker stack**（不碰 dev 資料）。重點不是覆蓋每個分支，而是驗證「使用者實際走得通」的關鍵流程。

## 2. 架構與隔離機制
```
Playwright (e2e/) ──HTTP──▶ frontend :3001 (nginx) ──/api──▶ backend :8081 (-tags e2e)
                                                                     │
                                       POST /api/test/reset ◀────────┘  (清庫 + 重新 seed)
                                                                     ▼
                                                            postgres :55432 (獨立 volume)
```
- backend 用 `-tags e2e` 編譯，才會註冊 **`POST /api/test/reset`**。
- 該端點受**三層保護**（任一不符就不註冊，見 `backend/internal/handler/test_reset_handler.go`）：
  1. build tag `e2e`（production build 是 no-op stub）
  2. env `ENABLE_TEST_RESET=true`
  3. `GIN_MODE != release`
- 每個 spec 在 `beforeAll`/`beforeEach` 呼叫 `resetDB()`（`support/seed.ts`）→ truncate 全表 + 清 upload 目錄 + 重建固定 seed。**spec 永遠不可直接連 postgres**。

## 3. Seed 資料（單一真實來源）
`support/seed.ts` 的常數**必須與** `test_reset_handler.go` 同步：
| 角色 | email | 備註 |
|------|-------|------|
| admin | admin@admin.com | E2E Admin |
| translator(active) | alice@translator.com | 可登入 |
| translator(disabled) | bob@translator.com | 停用，測登入被擋 |
| 密碼（全部）| `Test1234!` | |
| patients | passport `A123456` / hn `HN001` / unid `UN-XYZ` | |
| 今日排班 | alice，地點 `E2E Clinic, Bangkok`，兩位病人 | patients[0] completed + 1 張照片、**實付 1500**；patients[1] pending |

> seed 帳號 `must_change_pw` 由 reset 端點以 explicit UPDATE 強制設定（避免 GORM 零值跳過，見 changelog 2026-06-01/02）。
> 金額：patients[0] 今日已完成看診的 `actual_amount=1500`（`seed.ts` 鏡像常數 `seededActualPaidTotal`），是唯一的 seed 實付，故等於當月橫幅總額與 patients[0] 的列表/歷史實付總額；money-stats spec 以此斷言確切金額。

## 4. Playwright 設定（playwright.config.ts）
| 設定 | 值 | 原因 |
|------|----|------|
| baseURL | `E2E_BASE_URL` 或 http://localhost:3001 | 指向 e2e frontend |
| globalSetup | `global-setup.ts` | 開跑前 reset 一次，並 fail-fast 提示「stack 沒起」|
| fullyParallel / workers | false / 1 | **共用 DB state，必須序列化** |
| retries | CI=2、本機=0 | |
| locale / timezone | zh-TW / Asia/Bangkok | 模擬真實使用者環境 |
| project: chromium-desktop | Desktop Chrome，`testIgnore: responsive.spec` | 桌機過不了手機抽屜行為 |
| project: mobile-chrome | Pixel 7，`testMatch: responsive.spec` | 只跑 RWD 測試 |
| reporter | CI: github+html／本機: list+html | |

## 5. 測試案例（tests/）
| spec | 流程 |
|------|------|
| auth | 登入 / 改密碼 / 停用帳號被擋 / 鎖定 |
| translator-mgmt | 翻譯員 CRUD + 重設密碼 + **重複 email 顯示 EMAIL_TAKEN 訊息** |
| schedule-crud / schedule-validation | 排班建立、多病人、時段驗證 + **「最新創建排班」預設按鈕 + 建立 modal 顯示病人年度已實付** |
| translator-checkin | 到達/離開打卡流程 |
| makeup-checkin | 補打卡 + 原因 |
| diagnosis-flow | 逐病人診斷上傳 / no_show（admin 結果總覽）|
| diagnosis-manage | **診斷照片上傳 → 補傳 → 刪除 → 刪光退回 pending（API 層，避開打卡 UI）+ 刪不存在回 404** |
| patient-import-export | **病人 xlsx 匯出 → 再匯入（重複略過）round-trip + 範本下載 + 非 xlsx 回 INVALID_EXCEL（API 層）** |
| patient-history | 病人就診歷史 |
| export | Excel 匯出 |
| money-stats | **後台當月總支出橫幅（admin 可見 / translator 不可見）+ 病人列表實付總額欄 + 病人歷史實付總額與日期區間篩選** |
| errors | 錯誤碼 → i18n 訊息 |
| i18n | 語言切換 |
| responsive | 手機版 RWD（僅 mobile-chrome）|

## 6. 怎麼跑（三種方式，由簡到細）

### A. 一鍵腳本（最省事，推薦）
```bash
cd e2e
./run-e2e.sh            # 自動：檢查 docker/node → 裝依賴 → 裝 Chromium → 起 stack → 等就緒 → 跑測試
./run-e2e.sh --down     # 跑完順手拆 stack + 刪 volume
./run-e2e.sh --ui       # 開 Playwright UI 模式
./run-e2e.sh --no-stack # stack 已在跑，只跑測試
```
腳本退出碼：`0` 全綠、`1` 缺前置工具、`2` stack 起不來（印 backend log）、`3` 測試失敗。

### B. npm scripts（手動分步）
```bash
cd e2e
npm run stack:up     # docker compose -p thai-e2e up -d --build
npm run reset        # POST /api/test/reset（確認後端就緒）
npm test             # playwright test
npm run report       # 開 HTML 報告
npm run stack:down   # 拆 stack + 刪 volume
```

### C. 純 docker + playwright
見 [DOCKER_SPEC §4](../docker/DOCKER_SPEC.md) 手動拉 stack，再 `npx playwright test`。

## 7. 失敗排查
| 症狀 | 原因 / 處理 |
|------|------------|
| globalSetup 連不到 /api/test/reset | stack 沒起 → `npm run stack:up`；或 backend 還在啟動，等久一點（run-e2e 會等 90s）|
| 測試 flaky | 已 workers=1 序列化；檢查是否漏 `resetDB()` |
| reset 回 404 | backend 不是 e2e build 或 `ENABLE_TEST_RESET` 沒設 |
| 報告在哪 | `e2e/playwright-report/index.html`（`npm run report` 開）|

## 8. 不變式
| 不變式 | 保證 |
|--------|------|
| reset 端點不可能進 production | 機制保證（build tag + env + GIN_MODE 三層）|
| seed 常數前後端一致 | 人工維持（改一邊要改另一邊）|
| 測試只透過 reset 端點操控狀態 | 人工維持（禁止直連 DB）|
| dev 資料不被 e2e 影響 | 機制保證（獨立 port/volume/project）|

## 9. 協作者
依賴 [docker e2e stack](../docker/DOCKER_SPEC.md)、backend 的 reset 端點（[SERVER_SPEC](../backend/cmd/server/SERVER_SPEC.md)）；整體測試策略見 [TESTING_SPEC](../TESTING_SPEC.md)。
