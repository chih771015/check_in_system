# repository — 規格（輕量）

> 對應檔案：`backend/internal/repository/*.go`
> 上層：[ARCHITECTURE_SPEC.md](../../../ARCHITECTURE_SPEC.md)

## 1. 定位與職責
資料存取層（薄 facade over GORM）。每個 repo 包一個 `*gorm.DB`，只負責 **CRUD / 查詢組裝**；不放商業規則（在 service）。

**核心慣例 — `WithCtx`（必備）**
```go
func (r *XxxRepository) WithCtx(ctx context.Context) *XxxRepository {
    return &XxxRepository{db: r.db.WithContext(ctx)}
}
```
service 一律 `repo.WithCtx(ctx).<Method>()`，讓 GORM OTel plugin 把 SQL span nest 在 request span 下。**不傳 ctx → SQL span 變孤兒**（不變式，人工維持）。

部分 repo（ScheduleRepository）另外暴露 `DB() *gorm.DB`，供 service 做跨表 transaction / raw join 查詢（如多病人建立、診斷結果總覽、病人歷史）。

## 2. 各 repository 對外契約

### UserRepository
FindByEmail / FindByID / FindAll(status) / Create / Update / UpdatePasswordAndForceChange / IncrementLoginAttempts / ResetLoginAttempts / LockUser(until) / FindAllAdmins / DeleteByID。
> 註：`Create`/`Update` 用 `Select("*")` 確保 GORM 不跳過零值欄位（must_change_pw=false 等）— 見 E2E changelog。

### ScheduleRepository
FindByID（preload Translator + Patients.Patient）/ FindByTranslator / FindAll（filter translator/date/location）/ Create / CreateBatch / Update / Delete / IDsByRecurrenceGroup / DeleteByRecurrenceGroup。`DB()` 給 service 開 transaction。

### SchedulePatientRepository
CreateBatch / FindByScheduleID / FindByID（preload Patient）/ DeleteByScheduleID(s) / UpdateStatus(id,status,reason)。

### CheckinRepository
FindByScheduleID / **FindByScheduleAndType**（重複打卡 + arrive-before-leave 的關鍵查詢）/ Create / FindByID / UpdateFields(map) / Delete / DeleteByScheduleID(s) / ListAll(ListAllParams: date/translator/type/isMakeup)。

### DiagnosisPhotoRepository
Create / FindBySchedulePatientID（order by uploaded_at）/ CountBySchedulePatientID（上限檢查用）/ **FindByID** / **Delete(id)**（單張刪除，service 再決定是否退回 pending）。

### PatientRepository
Create / Update / Delete / FindByID / **FindByIDTypeAndNumber**（唯一性檢查）/ List(search,page) / **ListForTranslator**（scope 限縮：只回該翻譯員排班內病人）。

### ExportScheduleRepository
Upsert / FindByAdmin / FindAllEnabled / UpdateLastRun。

### AuditLogRepository
Create / List(filter, 分頁)。

## 3. 邊界條件
| 情境 | 行為 |
|------|------|
| FindBy* 查無 | 回 `gorm.ErrRecordNotFound`，由 service 轉成 domain sentinel（如 ErrPatientNotFound）|
| 刪排班但有 FK 子資料 | service 須先刪 checkins + schedule_patients 才能刪 schedule（repo 不自動級聯）|
| 分頁 page/pageSize ≤ 0 | repo 不防呆，由 service 設預設（page=1,size=20）|

## 4. 已知技術債
- 級聯刪除靠 service 手動呼叫多個 repo，而非 DB FK `ON DELETE CASCADE` 或 GORM association；漏呼叫 → FK 錯誤或孤兒列。
- 部分跨表查詢（診斷總覽、病人歷史）寫在 **service** 用 `DB()` raw join，未抽進 repo，repo 層職責不完全一致。

## 5. 協作者
被所有 [service](../service/SERVICE_SPEC.md) 使用；操作 [model](../model/MODEL_SPEC.md)；由 [cmd/server](../../cmd/server/SERVER_SPEC.md) 注入 `*gorm.DB` 建構。
