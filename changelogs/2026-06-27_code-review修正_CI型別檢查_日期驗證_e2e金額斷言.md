# 更動報告 — Code Review 修正：CI 型別檢查、歷史日期驗證、e2e 金額斷言

- 日期：2026-06-27
- 分支：`feature/local-expose`
- 來源：對「上次審查後新程式碼」的 code review，處理前三條（最有價值）

## Commits

- `fix(ci): Type-check 步驟改用 npm run typecheck（tsc -b）`
- `fix(backend): 病人歷史日期區間驗證，非法格式回 400`
- `test(e2e): 金額統計 seed 非零實付並斷言確切總額`
- `docs: 更動報告 code review 修正`

## 做了什麼

### #1 CI 的「Type-check」步驟其實沒在檢查
- `.github/workflows/test.yml` 的 Type-check 跑 `npx tsc --noEmit`，但根 `tsconfig.json` 是 `files:[]` + references → **檢查 0 個檔案**，永遠綠燈。型別破口只有後面 `npm run build`（`tsc -b`）才順帶抓得到。
- 改成 `npm run typecheck`（= `tsc -b`，這次先前已加的腳本），讓這個 gate 真的有作用。

### #2 病人歷史日期區間遇非法格式靜默算錯
- 之前重構把日期過濾下推 SQL 後，`to` 無法解析時 `nextDay` 回 false → 上界條件**整個被丟掉** → 回傳超出範圍的資料、總額超報；`from`/`to` 驗證還不對稱。
- `GetHistory` 開頭新增 `validDateOrEmpty` 驗證：`from`/`to` 非空且非 `YYYY-MM-DD` → 回既有的 `ErrInvalidDateFormat`（error_mapper 已對應 400 `INVALID_DATE`）。fail closed，不再靜默。
- 驗證放在 service 層（handler 無測試框架，service 有 fixture），且 `nextDay` 之後永遠收到合法 `to`。

### #3 金額統計 e2e 是空殼斷言
- 原本 seed 的實付金額全是 0 → 畫面全是「NT$ 0」，測試只檢查「有沒有 NT$ 文字」→ 就算 SUM/區間/範圍邏輯壞掉也照樣綠燈。
- seed 的已完成看診（patients[0]，今天）補上 `ActualAmount: 1500`（`test_reset_handler.go`），並在 `support/seed.ts` 加鏡像常數 `seededActualPaidTotal: 1500`。
- `money-stats.spec.ts` 改為斷言**確切金額**：
  - 橫幅（scope 到橫幅元素，非表格欄）顯示 NT$ 1,500
  - 病人列表 patients[0] 欄位 = NT$ 1,500、patients[1] = NT$ 0
  - 病人歷史 Statistic = NT$ 1,500
  - 現在 SUM/區間/scope 任何一個壞掉，數字就會變、測試就會紅。

## TDD / 驗證

- 後端新增 `GetHistory_RejectsMalformedDates`：`2026-06-1`/`2026-13-99`/`garbage`/`nope` 皆回 `ErrInvalidDateFormat`；合法與空字串仍成功。
- 後端 `go test ./...` 全綠、`go build -tags e2e` 編譯過。
- **e2e 全套實跑：38 passed / 1 skipped**（skip 為既有 `schedule-validation` 的 `.skip`）；money-stats 4 案改斷言確切金額後仍綠，seed 改動未影響其他案。跑完已 `stack:down`。

## 影響檔案

- CI：`.github/workflows/test.yml`
- 後端：`internal/service/patient_service.go`（validDateOrEmpty + GetHistory 驗證）、`internal/service/patient_history_test.go`、`internal/handler/test_reset_handler.go`（seed 1500）
- e2e：`support/seed.ts`（鏡像常數）、`tests/money-stats.spec.ts`（確切金額斷言）

## 其餘 review findings（未在本次處理）

- #4 橫幅每次換頁重抓（且原地編輯不刷新）— 需改為「金額變動時 invalidate」，範圍較大，待決定。
- 清理類：三套半開區間建構法、`PatientHistory` 漏用 `formatNT`、`prepaidAmount` fixture 無 factory、`toPatientResponse` 重複、`IN ?` 無分批。
