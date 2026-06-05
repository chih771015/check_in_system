# dto — 規格（輕量）

> 對應檔案：`backend/internal/dto/*.go`
> 上層：[ARCHITECTURE_SPEC.md](../../../ARCHITECTURE_SPEC.md)

## 1. 定位與職責
定義 HTTP 邊界的 **request / response payload**，以及**統一錯誤信封 + error code registry**。把 API 形狀與 model 解耦：model 改欄位不一定動 API；DTO 控制 json 對外命名與選填語意（指標型別 = optional / partial update）。

**不做**：商業邏輯、DB 操作、驗證以外的決策（gin binding tag 做基本必填驗證，跨欄位規則在 service）。

## 2. 檔案分工
| 檔案 | 內容 |
|------|------|
| `error.go` | `ErrorResponse{code,message}` + `NewError` + **所有 error code 常數** |
| `auth.go` | LoginRequest/Response、UserResponse、ChangePassword、ResetPassword |
| `admin.go` | Admin list/create |
| `translator.go` | Translator list/create/update（update 用指標 = partial）|
| `patient.go` | Patient CRUD、list query（search/page）、history、response（camelCase）|
| `schedule.go` | CreateScheduleRequest（含 `Patients []SchedulePatientPayload`、recurrence）、Update（指標 + `*[]Payload`）、ScheduleResponse、SchedulePatientResponse |
| `checkin.go` | CheckinRequest、CheckinMakeupRequest、AdminUpdateCheckinRequest（指標）、CheckinResponse、AdminListParams 對應 |
| `diagnosis_result.go` | DiagnosisResultsQuery / Response / Entry（分頁 + 篩選）|

## 3. 錯誤碼契約（最關鍵）
`error.go` 是**前後端 i18n 的單一真實來源**：code 為 `SCREAMING_SNAKE_CASE`，前端對應 `errors.<CODE>`。

分組：Generic / Auth / Admin·Translator / Schedule / Checkin / Patient / Stage4(SchedulePatient·Diagnosis) / Audit。

**新增 code 的不變式**（人工維持）：
- [ ] 此檔加 `Code...` 常數
- [ ] service 加對應 sentinel error
- [ ] `handler/error_mapper.go` 加 `mapError` case
- [ ] 三語 locale（en/zh-TW/th）加 `errors.<CODE>`

漏任一步 → 前端只會顯示後端英文原文（fallback），或回 `INTERNAL_ERROR`。

完整 code 清單見 `error.go`（約 60 條），與 [error_mapper](../handler/HANDLER_SPEC.md) 一一對應。

## 4. 命名風格注意
- 多數舊 DTO 用 snake_case json（auth/checkin/schedule response）。
- patient / diagnosis / schedule_patient 系列用 **camelCase**（idType、patientId、startTime…）。
- 前端 `types/index.ts` 必須與此處 json tag 對齊；不一致會造成欄位讀不到（靜默 undefined）。

## 5. partial update 慣例
update 類 DTO 欄位用**指標**（`*string`、`*[]Payload`）：`nil` = 不動該欄位，非 nil = 覆寫。service 依此判斷要不要更新（見 ScheduleService.Update / TranslatorService.Update）。

## 6. 協作者
被 [handler](../handler/HANDLER_SPEC.md) bind、[service](../service/SERVICE_SPEC.md) 產出；error code 被 [error_mapper](../handler/HANDLER_SPEC.md) 與前端 [api client](../../../frontend/src/api/API_CLIENT_SPEC.md) 使用。
