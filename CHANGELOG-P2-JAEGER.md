# P2 功能 + Jaeger 分散式追蹤 — 完整變更報告

## 目錄

1. [變更總覽](#1-變更總覽)
2. [Commit 歷程](#2-commit-歷程)
3. [P2 功能明細](#3-p2-功能明細)
4. [Jaeger/OpenTelemetry 追蹤](#4-jaegeropentelemetry-追蹤)
5. [Context 傳播架構](#5-context-傳播架構)
6. [檔案清單](#6-檔案清單)
7. [驗證方式](#7-驗證方式)

---

## 1. 變更總覽

| 類別 | 新增檔案 | 修改檔案 | 程式碼行數 |
|------|---------|---------|-----------|
| 後端 models/config/dto/middleware | 1 | 6 | +135 |
| 後端 新 services | 6 | 0 | +613 |
| 後端 repos/services/handlers | 2 | 13 | +912 |
| Jaeger/OTel 追蹤 | 1 | 4 | +466 |
| 前端 UI | 3 | 14 | +669 |
| **合計** | **13** | **37** | **~2,800** |

---

## 2. Commit 歷程

```
commit 1: feat(backend): P2 基礎建設 — 模型、設定、DTO、中介層
commit 2: feat(backend): P2 新服務層 — 稽核、通知、地理編碼、清理、郵件、匯出
commit 3: feat(backend): P2 完整接線 — repos、services、handlers（含 ctx 傳播）
commit 4: feat: Jaeger/OpenTelemetry 分散式追蹤 — 基礎設施 + main.go 接線
commit 5: feat(frontend): P2 介面更新 — 稽核紀錄、個人打卡、排班匯入、匯出設定
commit 6: docs: P2 + Jaeger 完整變更報告（本文件）
```

每個 commit 可獨立閱讀，按以下順序講述功能建構過程：
1. 先建立資料模型與基礎設施
2. 再建立獨立的服務模組
3. 把服務接上 repository 與 handler，同時加入 context 傳播
4. 加入 Jaeger 追蹤基礎設施
5. 最後補齊前端 UI

---

## 3. P2 功能明細

### 3.1 稽核紀錄系統 (Audit Logging)

**問題**：管理員操作無紀錄，無法追溯誰做了什麼。

**解決方案**：
- `AuditLog` 模型：記錄 adminID、action、targetType、targetID、detail
- `AuditService.Log(ctx, ...)` 在以下操作自動記錄：
  - 建立/修改/停用翻譯員
  - 重設密碼
  - 建立/修改/刪除排班（含批次匯入、整組刪除）
  - 編輯/刪除打卡紀錄
- 前端 `/admin/audit-logs` 頁面：分頁 + action/日期篩選

### 3.2 帳號鎖定 (Login Lockout)

**問題**：暴力破解密碼無任何防護。

**解決方案**：
- `User` 模型新增 `LoginAttempts`、`LockedUntil` 欄位
- 連續失敗 N 次（預設 5）後鎖定 M 分鐘（預設 15）
- 成功登入自動重設計數器
- 環境變數 `MAX_LOGIN_ATTEMPTS`、`LOCK_DURATION_MINUTES` 可調整

### 3.3 後端強制改密碼 (mustChangePW Enforcement)

**問題**：`mustChangePW` 只在前端攔截，直接呼叫 API 可繞過。

**解決方案**：
- `RequirePasswordChanged()` middleware 讀取 JWT claims 中的 `must_change_pw`
- 回傳 `403 {code: "PASSWORD_CHANGE_REQUIRED"}`
- 套用在所有 admin/translator route group
- 例外：`/api/auth/change-password` 不套用
- 改密碼成功後回傳新 token（前端更新 authStore）

### 3.4 管理員重設密碼

**問題**：使用者忘記密碼後完全無法恢復。

**解決方案**：
- `POST /api/admin/translators/:id/reset-password`
- 管理員輸入新密碼，目標使用者被強制改密碼
- 禁止對自己使用此 endpoint

### 3.5 管理員編輯/刪除打卡紀錄

**問題**：打卡錯誤只能進 DB 修改。

**解決方案**：
- `PUT /api/admin/checkins/:id` — 可修改 checkinTime、address、makeupReason
- `DELETE /api/admin/checkins/:id` — 硬刪除

### 3.6 批次刪除整組排班

**問題**：重複排班只能逐筆刪除。

**解決方案**：
- `DELETE /api/admin/schedules/:id/group`
- 根據 `recurrence_group_id` 整組刪除

### 3.7 Excel 批次匯入排班

**解決方案**：
- `POST /api/admin/schedules/import` (multipart)
- 逐行驗證，回傳 success/failed 計數 + 錯誤明細

### 3.8 個人打卡紀錄與統計

**解決方案**：
- `GET /api/checkins` — 翻譯員查看自己的打卡歷史
- `GET /api/checkins/stats` — 統計：總數、到達、離開、補打卡、準時、遲到

### 3.9 通知服務

**解決方案**：
- LINE Messaging API 推播 + SMTP email 備援
- 每日 07:00 cron 發送隔日排班提醒

### 3.10 反向地理編碼

**解決方案**：
- 打卡時若缺少地址，自動呼叫 OSM Nominatim 反查
- 失敗靜默處理，不阻擋打卡

### 3.11 照片清理

**解決方案**：
- 每日 03:00 cron 刪除超過 retention 期限的打卡照片
- 環境變數 `PHOTO_RETENTION_DAYS`（預設 90）

### 3.12 定期匯出 + 手動觸發

**解決方案**：
- `RunExportForAdmin(ctx, adminID)` 共用邏輯
- cron 每日 08:00 檢查 day_of_month 是否匹配
- `POST /api/admin/export/schedule/run` 手動觸發
- 支援 Excel 附件 / Google Sheet 連結 email

### 3.13 Monthly 日期 Clamping

**解決方案**：
- `monthly:31` 在 2 月產生 28/29、4 月產生 30
- 使用 `time.Date` rollover + `seen` map 防重複

---

## 4. Jaeger/OpenTelemetry 追蹤

### 架構圖

```
┌─────────────┐     OTLP/gRPC      ┌──────────────┐
│   Backend   │ ──────────────────> │    Jaeger     │
│  (Go + Gin) │     port 4317      │  all-in-one   │
│             │                     │  port 16686   │
└─────────────┘                     └──────────────┘

span 結構：
┌──────────────────────────────────────────────────┐
│ POST /api/auth/login  (otelgin)                  │
│  ├── select users     (gorm otel plugin)         │
│  ├── update users     (gorm otel plugin)         │
│  └── ...                                         │
└──────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────┐
│ POST /api/checkins  (otelgin)                    │
│  ├── select schedules (gorm)                     │
│  ├── select checkins  (gorm)                     │
│  ├── HTTP GET nominatim (otelhttp)               │
│  ├── select users     (gorm)                     │
│  └── insert checkins  (gorm)                     │
└──────────────────────────────────────────────────┘
```

### 元件

| 元件 | 角色 |
|------|------|
| `tracing.Init()` | 建立 TracerProvider + OTLP exporter |
| `otelgin.Middleware` | 自動為每個 HTTP 請求建立 server span |
| `gormtracing.NewPlugin` | 自動為每個 SQL 查詢建立 span |
| `otelhttp.NewTransport` | 包裝外部 HTTP 呼叫（LINE、Nominatim）|
| `repo.WithCtx(ctx)` | 讓 SQL span 嵌套在 HTTP span 下 |
| `scrubSensitiveSpanAttributes` | 過濾 query string 防止 PII 洩漏 |
| `tracer.Start(ctx, "cron.xxx")` | cron 任務各自建立 root span |

### 環境變數

| 變數 | 預設值 | 說明 |
|------|--------|------|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `jaeger:4317` | OTLP gRPC 端點 |
| `OTEL_SERVICE_NAME` | `translator-checkin` | Jaeger 服務名稱 |
| `DEPLOY_ENV` | `dev` | 部署環境標籤 |

---

## 5. Context 傳播架構

### 設計模式

```go
// Repository 層：WithCtx 回傳綁定 context 的副本
func (r *UserRepository) WithCtx(ctx context.Context) *UserRepository {
    return &UserRepository{db: r.db.WithContext(ctx)}
}

// Service 層：接收 ctx，建立 scoped repo
func (s *AuthService) Login(ctx context.Context, email, password string) (...) {
    repo := s.userRepo.WithCtx(ctx)
    user, err := repo.FindByEmail(email)
    // ...
}

// Handler 層：傳入 c.Request.Context()
func (h *AuthHandler) Login(c *gin.Context) {
    resp, err := h.authService.Login(c.Request.Context(), req.Email, req.Password)
}
```

### 結果

| 狀態 | 說明 |
|------|------|
| 修改前 | SQL span 為獨立 trace，無法追溯來源 API |
| 修改後 | SQL span 嵌套在對應的 HTTP server span 下 |

已套用 WithCtx 的 repository：
- UserRepository
- CheckinRepository
- ScheduleRepository
- ExportScheduleRepository
- AuditLogRepository

---

## 6. 檔案清單

### 新增檔案（13）

```
backend/internal/model/audit_log.go
backend/internal/repository/audit_log_repo.go
backend/internal/service/audit_service.go
backend/internal/service/notification_service.go
backend/internal/service/geocoding_service.go
backend/internal/service/cleanup_service.go
backend/internal/service/mail_service.go
backend/internal/service/export_service.go
backend/internal/handler/audit_handler.go
backend/internal/tracing/tracing.go
frontend/src/api/audit.ts
frontend/src/pages/admin/AuditLogs.tsx
frontend/src/pages/translator/MyCheckins.tsx
```

### 修改檔案（37）

```
backend/cmd/server/main.go
backend/go.mod / go.sum
backend/internal/config/config.go
backend/internal/dto/auth.go / checkin.go / schedule.go
backend/internal/middleware/auth.go
backend/internal/model/user.go
backend/internal/repository/user_repo.go / checkin_repo.go /
    schedule_repo.go / export_schedule_repo.go
backend/internal/service/auth_service.go / translator_service.go /
    schedule_service.go / checkin_service.go
backend/internal/handler/auth_handler.go / translator_handler.go /
    schedule_handler.go / checkin_handler.go / export_schedule_handler.go
docker/docker-compose.yml
frontend/src/App.tsx
frontend/src/api/auth.ts / checkins.ts / client.ts / export.ts /
    schedules.ts / translators.ts
frontend/src/components/AppLayout.tsx
frontend/src/pages/ChangePassword.tsx
frontend/src/pages/admin/CheckinRecords.tsx / ExportSettings.tsx /
    ScheduleManagement.tsx / TranslatorManagement.tsx
frontend/src/types/index.ts
```

---

## 7. 驗證方式

### 快速驗證

```bash
# 1. 啟動服務
docker compose -f docker/docker-compose.yml up -d --build

# 2. 確認 Jaeger UI
open http://localhost:16686

# 3. 觸發幾個 API 呼叫
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@admin.com","password":"xxx"}'

# 4. 在 Jaeger 查看 service=translator-checkin
#    選 operation=POST /api/auth/login
#    應看到 2 層巢狀：HTTP span → SQL span
```

### 驗證 context 傳播

```bash
# 查詢最近 login traces
curl -s "http://localhost:16686/api/traces?\
service=translator-checkin&\
operation=POST%20%2Fapi%2Fauth%2Flogin&\
limit=3" | python3 -m json.tool

# 預期結果：每個 trace 有 2+ spans，SQL span 為 HTTP span 的子 span
```

### 功能驗證清單

- [ ] 管理員重設翻譯員密碼 → 翻譯員用新密碼登入 → 被導向改密碼
- [ ] curl 帶 mustChangePW=true 的 token 打 API → 403
- [ ] 建立 weekly:1,3,5 重複排班 → 按「刪除整組」→ 全部消失
- [ ] 編輯打卡紀錄的時間 → DB 反映變更
- [ ] ExportSettings 設好 emailTo → 按「立即執行」→ 收到信件
- [ ] Jaeger UI 可見所有 API trace，SQL span 正確嵌套
