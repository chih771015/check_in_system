# 更動報告 — 修復前端 build：測試 fixture 缺必填 prepaidAmount

- 日期：2026-06-26
- 分支：`feature/local-expose`
- 發現於：實跑 e2e（`./run-e2e.sh`）時，frontend docker image 的 `npm run build` 失敗。

## Commit

- `fix(frontend): 測試 fixture 補上必填 prepaidAmount，修復 tsc -b 編譯`

## 問題

`SchedulePatientPayload` 已將 `prepaidAmount: number` 列為必填，但兩個測試檔的 fixture 仍用舊形狀（只有 patientId/startTime/endTime），導致 `tsc -b` 報 TS2741（28 處）：
- `src/utils/__tests__/schedulePatient.test.ts`（21 處，**非本 session 既有破口**）
- `src/components/__tests__/SchedulePatientListEditor.test.tsx`（7 處）

### 為何之前沒被擋下來（重要）
- 根 `tsconfig.json` 是 `{"files": [], "references": [...]}`。直接跑 `npx tsc --noEmit`（不帶 `-b`）**不會**走 project references → 等於檢查 0 個檔案，永遠 pass。
- vitest 用 esbuild，不做型別檢查。
- 真正會檢查的是 `npm run build` 的 `tsc -b`（走 references，`tsconfig.app.json` 的 `include: ["src"]` 含測試檔）。只有 frontend docker build 會跑到，於是這個破口一直沒在本機/單元測試被發現，直到實跑 e2e。

→ 結論：本分支的 frontend 正式 build（含 CI）其實是壞的；這次一併修好。

## 修法

- 兩檔所有 `SchedulePatientPayload` fixture 補 `prepaidAmount: 0`（這些是純函式 / 純 UI 測試，prepaidAmount 不影響被測邏輯，0 為自然預設）。

## 驗證

- `npx tsc -b` → exit 0（權威型別檢查通過）。
- `npx vitest run`（兩檔）→ 19 測試全綠。
- **`./run-e2e.sh` 全套實跑：38 passed / 1 skipped**（skipped 為既有 `schedule-validation` 的 `.skip`）。本 session 新增的 6 個 e2e（money-stats ×4、schedule-crud 最新創建按鈕 + 年度已實付）全綠。跑完已 `stack:down` 清除 volume。

## 後續建議（未在本次處理）

- 本機缺一個會走 `tsc -b` 的 typecheck 指令，建議在 `frontend/package.json` 加 `"typecheck": "tsc -b"` 並納入 commit/CI 前置檢查，避免再次只靠 docker build 才發現型別破口。
