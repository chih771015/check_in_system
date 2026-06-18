# handler — 規格（輕量）

> 對應檔案：`backend/internal/handler/*.go`
> 上層：[ARCHITECTURE_SPEC.md](../../../ARCHITECTURE_SPEC.md)

## 1. 定位與職責
HTTP 邊界轉接層：bind DTO、取 context（userID/role）、存上傳檔、呼叫 service、**把 sentinel error 統一轉 HTTP code**、寫 audit log。**不放**商業邏輯。

## 2. 檔案分工（對應 route，見 [SERVER_SPEC](../../cmd/server/SERVER_SPEC.md)）
| handler | 端點群 |
|---------|--------|
| auth_handler | login、change-password |
| translator_handler | /admin/translators CRUD + reset-password |
| admin_handler | /admin/admins CRUD |
| patient_handler | /admin/patients CRUD + history、/admin/patients/import（xlsx 匯入）、/admin/export/patients（匯出）、/admin/export/patients-template（範本）、/patients(translator) |
| schedule_handler | /admin/schedules CRUD + import + group delete、/schedules(translator) |
| checkin_handler | /checkins、/checkins/makeup、/checkins、/checkins/stats、/admin/checkins、/admin/export/excel、google-sheet |
| diagnosis_handler | /checkins/diagnosis、/checkins/diagnosis/photos（GET 列表 ?schedulePatientId / DELETE :photoId）、/checkins/no-show、/admin/diagnosis、/admin/diagnosis/photos（GET 列表 / DELETE :photoId）、/admin/no-show、/checkins/diagnosis/amount、/admin/diagnosis/amount、/admin/diagnosis-results、/admin/export/diagnosis、/admin/schedule-patients/:id/photos |
| export_schedule_handler | /admin/export/schedule (get/upsert/run) |
| audit_handler | /admin/audit-logs |
| error_mapper | 共用錯誤轉換（非 handler，是工具）|
| test_reset_handler / _stub | E2E reset（build tag 切換）|

## 3. error_mapper.go（最關鍵）
| 函式 | 用途 |
|------|------|
| `respondError(c, err)` | `mapError(err)` → `(status, code)` → `dto.NewError`；未知錯 → 500 `INTERNAL_ERROR` |
| `respondBadRequest(c, err)` | binding/驗證錯 → 400 `BAD_REQUEST` |
| `respondCode(c, status, code, msg)` | 無 sentinel 的特例（檔案、ID 解析…）|

`mapError` 是 **service sentinel ↔ dto code ↔ HTTP status** 的單一對照表。新增錯誤必須同步更新此檔（見 [dto spec](../dto/DTO_SPEC.md) §3 checklist）。

## 4. 共同慣例
- 一律傳 `c.Request.Context()` 給 service（tracing）。
- 取 caller 身分：`c.GetUint("userID")` / `c.Get("userID")`（由 [middleware](../middleware/MIDDLEWARE_SPEC.md) 設）。
- 上傳檔：`saveUploadedFile(c, field)` / `saveMultipartFile(c, fh, prefix)` 存進 `UPLOAD_DIR`，回 `/uploads/...` URL；multipart 表單同時用 `c.PostForm(...)` 取純值（lat/lng/address）。
- 列表回應慣例包 `{ "data": ... }` 信封（前端 client 會 unwrap，見 [api client](../../../frontend/src/api/API_CLIENT_SPEC.md)）。
- 管理員寫操作成功後 `auditService.Log(...)`。

## 5. 邊界條件
| 情境 | 行為 |
|------|------|
| 缺 selfie 檔 | 400 `SELFIE_REQUIRED`（environment 選填）|
| :id 非數字 | 400 `INVALID_*_ID` |
| bind 失敗 | 400 `BAD_REQUEST` |
| userID 不在 context | 401 `USER_CONTEXT_MISSING`（理論上 middleware 已擋）|

## 6. 已知技術債
- 各 handler 重複「parseID → bind → call → mapError → audit」樣板，可抽 helper。
- audit detail 多為空字串，資訊量有限。

## 7. 測試考量
`error_mapper_test.go` 驗 sentinel→code 對照齊全。handler 主要靠 service 單測 + E2E 覆蓋。
