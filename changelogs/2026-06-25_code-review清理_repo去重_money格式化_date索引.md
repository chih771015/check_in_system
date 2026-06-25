# 更動報告 — Code Review 清理：repo 去重 + NT$ 格式化抽共用 + schedules.date 索引

- 日期：2026-06-25
- 分支：`feature/local-expose`
- 來源：5 項需求 code review 的清理類 findings（剩餘三項）

## Commits

- `refactor(backend): 合併實付區間加總 SQL + schedules.date 加索引`
- `refactor(frontend): NT$ 金額格式化抽出 utils/money`
- `docs: 更動報告 code review 清理`

## 做了什麼

### 後端 — repo 去重
- `SumActualByDateRange`（全病人）與 `SumActualByPatientDateRange`（單病人）原本是同一段 JOIN+COALESCE(SUM) 查詢、只差一個 `patient_id` 條件。
- 抽出私有 `sumActualInRange(patientID, from, to)`（patientID==0＝全部），兩個公開方法改為薄包裝委派。
- 公開 API 與呼叫端、測試皆不變；SQL 只剩一份，未來改 join／日期／COALESCE 不會兩邊分歧。

### 後端 — schedules.date 索引
- `model.Schedule.Date` 的 gorm tag 加 `;index`。當月 banner、年度 hint、病人歷史都會以 `schedules.date` 過濾 join，量大時避免全表掃描。AutoMigrate 下次啟動建立索引。

### 前端 — NT$ 格式化共用
- 新增 `utils/money.ts` 的 `formatNT(amount?)`：`12345 → "NT$ 12,345"`，null/undefined 一律當 0。
- 取代 3 處手寫字串：`AppLayout`（橫幅）、`SchedulePatientListEditor`（年度已實付）、`PatientManagement`（實付總額欄）。
- `PatientHistory` 用的是 antd `Statistic prefix="NT$"`（元件層格式化、字體較大的不同呈現），維持原樣不改。

## TDD / 驗證

- 前端新增 `utils/money.test.ts`：整數千分位 + 前綴、0、null/undefined→0。
- 既有測試作迴歸：`AppLayout`（`NT$ 12,345`）、`SchedulePatientListEditor`（`NT$ 8,000`）斷言維持綠（formatNT 輸出一致）。
- 後端 `go test ./...` 全綠、`go vet`/`gofmt` 乾淨；前端 `tsc`/eslint 乾淨、全套 19 檔 / 100 測試綠。

## 影響檔案

- 後端：`repository/schedule_patient_repo.go`、`model/schedule.go`
- 前端：`utils/money.ts`(新)、`utils/money.test.ts`(新)、`components/AppLayout.tsx`、`components/SchedulePatientListEditor.tsx`、`pages/admin/PatientManagement.tsx`

## 備註

- 至此 5 項需求 code review 的 10 條 findings 全數處理完畢（修正 #1/#3、效能 #2/#5/#6、清理 repo 去重/格式化/索引）。
- 前端 PatientHistory 仍每次改區間重抓（後端已下推 SQL + 批次照片變便宜），刻意未改。
