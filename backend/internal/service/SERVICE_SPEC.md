# service — 規格（overview）

> 對應檔案：`backend/internal/service/*.go`
> 上層：[ARCHITECTURE_SPEC.md](../../../ARCHITECTURE_SPEC.md)

## 1. 定位與職責
商業邏輯層。所有跨欄位驗證、權限 ownership、狀態轉換、外部服務協調都在這裡。

**通用慣例**
- 每個從 handler 呼叫的方法第一參數是 `ctx context.Context`，內部 `repo.WithCtx(ctx)`。
- 錯誤一律回**套件級 sentinel error**（`var ErrXxx = errors.New(...)`），由 [error_mapper](../handler/HANDLER_SPEC.md) 轉 HTTP code。
- 選用相依用 **builder 注入**（`WithXxxRepo(...)` 回 `*Service` 鏈式），讓舊測試可用精簡建構子。
- service 不碰 `*gin.Context`、不寫 HTTP status。

## 2. 模組清單

| 模組 | 重點 | 規格 |
|------|------|------|
| **AuthService** | login + lockout、改密碼重簽 JWT、admin reset | [重型 ★](AUTH_SERVICE_SPEC.md) |
| **CheckinService** | 打卡守衛、逾時自動 makeup、leave 阻擋 pending、統計 | [重型 ★](CHECKIN_SERVICE_SPEC.md) |
| **ScheduleService** | 多病人 transaction、週期展開、批次匯入 V1/V2 | [重型 ★](SCHEDULE_SERVICE_SPEC.md) |
| **DiagnosisService** | 逐病人照片(≤3) 上傳/刪除/再補傳、no_show、ownership、結果總覽 | [重型 ★](DIAGNOSIS_SERVICE_SPEC.md) |
| PatientService | CRUD、id_number 正規化、(idType,idNumber) 唯一、translator scope、就診歷史 | 本檔 §3 |
| TranslatorService | translator CRUD / disable（軟停用）| 本檔 §3 |
| AdminService | admin CRUD、不可刪自己、seed | 本檔 §3 |
| ExportService | Excel(excelize) / Google Sheet / 月報 cron | 本檔 §3 |
| NotificationService | 隔日提醒 LINE push + Email | 本檔 §3 |
| GeocodingService | Nominatim reverse geocode（失敗不阻斷）| 本檔 §3 |
| MailService | SMTP 寄信（PLAIN auth、單附件、RFC2047 主旨）| 本檔 §3 |
| CleanupService | cron 清除逾期照片 | 本檔 §3 |
| AuditService | 寫稽核（錯誤吞掉，不阻斷）/ 查詢 | 本檔 §3 |
| StatsService | 後台金額統計（當月病人總實付）| 本檔 §3 |

## 3. 輕量模組對外契約

### PatientService（`patient_service.go`）
- `Create/Update`：先 `normalizeIDNumber`（ToUpper+Trim），再查 `FindByIDTypeAndNumber` 防重 → `ErrPatientDuplicate`。Update 排除自身。
- **xlsx 匯入/匯出**：`ImportPatients(rows)` 逐列驗證（必填、idType∈{passport,hn,unid}）+ 走 `Create`，**重複/非法略過並回報**（`dto.PatientImportResult{created,skipped,errors[{row,reason}]}`，row 為 sheet 列號）；`BuildExcel`（全部病人，匯入相容欄位）、`BuildPatientTemplate`（表頭 + 範例列）用 excelize。
- `List`（全部）vs `ListForTranslator`（有 spRepo 時限縮 scope，未注入則退回全部 — legacy 行為）。
- `GetHistory(ctx,id,from,to)`：注入 history repos 時做真實彙整（schedule_patients ⨝ schedules ⨝ users + diagnosis_photos，date DESC）；未注入回空 history。
  - `from/to`（皆 `YYYY-MM-DD`，選填）：非空且非合法日期 → `ErrInvalidDateFormat`（fail closed，避免靜默丟掉條件）。日期區間**下推 SQL**（閉區間，上界以 +1 天轉半開 `< nextDay(to)`，安全跨 sqlite RFC3339 與 postgres date）。
  - 照片以 `FindBySchedulePatientIDs` **單次批次查**再依 sp_id 分組（避免 N+1）。回傳 `ActualTotal` = 區間內 entries 的 `actual_amount` 總和（無區間=全時段）。
- **金額統計**：`ActualTotals(ctx,ids)` 批次取多位病人全時段實付（複用 `SumActualByPatients`，無 N+1，供病人列表欄）；`PatientYearActualTotal(ctx,id,year)` 取單一病人某年實付（半開區間，供排班建立時的「年度已實付」提示）。未注入 spRepo 時回 0/空。
- 不變式：id_number 正規化是**人工維持**，繞過 service 寫 DB 會破壞唯一比對。

### TranslatorService / AdminService
- 共用 `ErrEmailTaken`、`ErrPasswordHashFailed`。
- 新帳號一律 `must_change_pw=true`、bcrypt 雜湊。
- Translator `Disable` = status→disabled（**不真刪**，保留歷史）；只能操作 role=translator。
- Admin `DeleteAdmin`：`requesterID==targetID` → `ErrCannotDeleteSelf`；target 非 admin → `ErrNotAnAdmin`（真刪）。

### ExportService（`export_service.go`）
- `BuildCheckinExcel`：用 `checkinService.AdminList` 取資料 → excelize in-memory file（中文表頭、type/makeup 中文化）。
- `CreateCheckinGoogleSheet`：需 `GOOGLE_CREDENTIALS_FILE`（service account JWT）→ 建 Sheet → 回 URL；未設定回錯。
- `RunExportForAdmin`：算**上一個自然月**範圍 → 依 format 產 Excel(寄附件)/Sheet(寄連結) → `UpdateLastRun`。由月報 cron 與 `/export/schedule/run` 呼叫。

### NotificationService（`notification_service.go`）
- `PushLine(ctx,lineUserID,msg)`：需 `LINE_CHANNEL_ACCESS_TOKEN`，POST LINE push API；httpClient 包 otelhttp。
- `SendScheduleReminders`：撈**明日**所有排班，對每位 translator：有 line_user_id → LINE push；有 email 且 SMTP 設定 → 寄信。個別失敗只 log 不中斷。開自己的 trace span。
- ⬜ 只實作「明日提醒」一種情境。

### GeocodingService（`geocoding_service.go`）
- `ReverseGeocode(ctx,lat,lng)`：呼叫 Nominatim `/reverse`（zh-TW、User-Agent 必填），回 display_name。`SetBaseURL` 供測試。**呼叫端可忽略錯誤**（打卡不因第三方失敗而擋）。

### MailService（`mail_service.go`）
- `Send(to,subject,body,*Attachment)`：需 `SMTP_HOST`+`SMTP_FROM`，未設定回錯（不靜默成功）。支援單一 base64 附件、RFC2047 中文主旨。

### CleanupService（`cleanup_service.go`）
- `RunPhotoCleanup`：walk `UPLOAD_DIR`，刪除 mtime 早於 `PHOTO_RETENTION_DAYS` 的影像檔（限 jpg/png/gif/webp）。每日 03:00 cron。
- **`PHOTO_RETENTION_DAYS=0`（預設）= 永久保存**：直接 log 並 return，永不刪除任何檔案。只有設正整數才會清。
- ⚠️ 設正整數時純依檔案 mtime，**不檢查是否仍被 DB 引用**；保留期需大於業務查詢期。

### AuditService（`audit_service.go`）
- `Log(ctx,adminID,action,targetType,targetID,detail)`：補 admin_name 後寫入；**錯誤吞掉**（稽核不得阻斷主流程）。
- `List(filter)`：分頁查詢。

### StatsService（`stats_service.go`）
- `CurrentMonthActualTotal(ctx)`：回 `(yearMonth, total)` — 全病人**當月**實付總額（依 `schedules.date`），供後台全域橫幅。
- `monthRange(now)`（純函式，可單測）：以伺服器本地時間（容器 TZ Asia/Taipei）算當月半開區間 `[YYYY-MM-01, 次月-01)` 與 `YYYY-MM` 標籤，含 12 月跨年。
- 透過 `SumActualByDateRange` 加總；未注入 spRepo 時回 0。

## 4. 測試考量
- DB 用 in-memory SQLite（`testutil_test.go`）；各 service 都有 `_test.go`。
- 難測點：ExportService 的 Google/SMTP、NotificationService 的 LINE/HTTP — 靠 base URL override 或避免實際外呼；GeocodingService 用 `SetBaseURL` 注入 fake server。
- 純函式可單測：`expandRecurrenceDates`、`normalizeIDNumber`、`getCheckinStatus`、`encodeSubject`、`monthRange`、`nextDay`、`validDateOrEmpty`。

## 5. 協作者
依賴 [repository](../repository/REPOSITORY_SPEC.md)、[dto](../dto/DTO_SPEC.md)、[config](../config/CONFIG_SPEC.md)、[middleware](../middleware/MIDDLEWARE_SPEC.md)(GenerateToken)；被 [handler](../handler/HANDLER_SPEC.md) 與 cron（[cmd/server](../../cmd/server/SERVER_SPEC.md)）呼叫。
