# DiagnosisService — 規格（重型 ★）

> 對應檔案：`backend/internal/service/diagnosis_service.go`
> 上層：[service overview](SERVICE_SPEC.md) ← [ARCHITECTURE_SPEC.md](../../../ARCHITECTURE_SPEC.md)

## 1. 定位與職責
逐 **SchedulePatient** 的服務結果：上傳診斷證明照片（≤3）標記 `completed`、標記 `no_show`（需原因）、查詢照片、**診斷結果總覽（分頁 + 篩選）**。translator 操作需 ownership；admin 有 surrogate 變體（無 ownership 檢查）。
- **不做**：照片存檔（handler 存好 URL 才呼叫）、打卡（CheckinService，但兩者透過 SchedulePatient.status 連動）。

## 2. 對外契約
| 方法 | ownership | 說明 |
|------|-----------|------|
| `UploadDiagnosis(ctx,translatorID,spID,urls)` | ✅ | 累計照片≤3，標 completed |
| `MarkNoShow(ctx,translatorID,spID,reason)` | ✅ | reason 必填，標 no_show |
| `AdminUploadDiagnosis(ctx,spID,urls)` | ✗ | 代理上傳 |
| `AdminMarkNoShow(ctx,spID,reason)` | ✗ | 代理標記 |
| `GetPhotos(ctx,spID)` | ✗ | 回照片 URL（upload time 排序）|
| `ListResults(ctx,query)` | ✗ | terminal(completed/no_show) 總覽，分頁 |

常數 `MaxDiagnosisPhotos = 3`。Sentinel：`ErrSchedulePatientNotFound / ErrDiagnosisPhotoLimit / ErrDiagnosisNotOwned / ErrNoShowReasonRequired`。

## 3. 狀態模型（SchedulePatient.status）

### 3b. 狀態機
```mermaid
stateDiagram-v2
  [*] --> pending: 排班建立時
  pending --> completed: UploadDiagnosis (≥1 photo)
  pending --> no_show: MarkNoShow (reason 必填)
  completed --> completed: 再上傳 (累計≤3) / admin 重傳
  no_show --> completed: admin/translator 補上傳 (status 被覆寫)
  completed --> no_show: MarkNoShow 覆寫
```
> status 轉換**無單向限制**：upload 一律覆寫成 completed、no_show 一律覆寫成 no_show（允許更正）。

### 3c. 不變式
| 不變式 | 保證 |
|--------|------|
| 每 SchedulePatient 照片 ≤ 3 | 人工維持（upload 前 `CountBySchedulePatientID + len > 3` 檢查；DB 不擋）|
| translator 只能動自己排班的病人 | 機制保證（assertOwnedSchedulePatient 比對 schedule.translator_id）|
| no_show 必附原因 | 機制保證（reason=="" → 直接回錯）|
| 與離開打卡連動：pending 擋 leave | 在 CheckinService（跨 service 不變式）|

## 4. 主要流程（UploadDiagnosis）
```mermaid
sequenceDiagram
  participant H as diagnosis_handler
  participant S as DiagnosisService
  participant SP as SchedulePatientRepo
  participant P as DiagnosisPhotoRepo
  H->>S: UploadDiagnosis(translatorID, spID, urls)
  S->>SP: FindByID(spID) 無→SchedulePatientNotFound
  S->>SP: schedule.translator_id==translatorID? 否→DiagnosisNotOwned
  S->>P: CountBySchedulePatientID
  S->>S: existing+len(urls)>3 → DiagnosisPhotoLimit
  loop each url
    S->>P: Create(DiagnosisPhoto)
  end
  S->>SP: UpdateStatus(completed, "")
```

## 5. 邊界條件表
| 情境 | 事件 | 行為 |
|------|------|------|
| spID 不存在 | 任一 | `SCHEDULE_PATIENT_NOT_FOUND` |
| 非自己排班 | translator upload/no_show | `DIAGNOSIS_NOT_OWNED` (403) |
| 既有+新 > 3 | upload | `DIAGNOSIS_PHOTO_LIMIT` (400) |
| no_show 無 reason | no_show | `NO_SHOW_REASON_REQUIRED` (400) |
| admin 代理 | Admin* | 跳過 ownership |
| ListResults 含 pending | ListResults | **排除**（只回 completed/no_show）|

## 6. ListResults（診斷結果總覽）
- raw join `schedule_patients ⨝ schedules ⨝ users ⨝ patients`，where status in (completed,no_show)。
- 篩選：status / translatorID / dateFrom / dateTo / patientName(LIKE)。
- 排序 `s.date DESC, sp.start_time DESC, sp.id DESC`，分頁（預設 page1/size20）。
- **N+1 防護**：先收集本頁 spIDs，一次撈所有照片再 map 回各列。
- date 跨 DB 相容：sqlite 回 RFC3339、postgres 回 YYYY-MM-DD，統一 trim `T` 之後。

## 7. 並發假設
- 照片上限「先 count 再寫」非原子；同一 SchedulePatient 並發上傳極端情況可能超過 3（低頻內部可接受）。

## 8. 測試考量
- `diagnosis_service_test.go`、`diagnosis_photos_get_test.go`、`diagnosis_results_test.go`。
- 縫：三個 repo 皆可 SQLite fake；ListResults 用 raw SQL，需在 SQLite 與 PG 都驗 date 處理。

## 9. 已知技術債
- 照片上限非 DB 約束。
- ListResults / 病人歷史的 raw join 寫在 service，repo 職責外溢（與 [repository spec](../repository/REPOSITORY_SPEC.md) §4 同源問題）。

## 10. 重構方向
- 把 terminal 查詢與照片 batch-load 抽進 repository。
- 照片上限改 DB 觸發或計數欄位。
