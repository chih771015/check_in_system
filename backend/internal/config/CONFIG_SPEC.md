# config — 規格（輕量）

> 對應檔案：`backend/internal/config/config.go`
> 上層：[ARCHITECTURE_SPEC.md](../../../ARCHITECTURE_SPEC.md)

## 1. 定位與職責
從環境變數載入設定到全域 `AppConfig`，並在開機時**強制安全前置條件**。`Load()` 由 `cmd/server/main.go` 最先呼叫。

## 2. 設定項

| env | 預設 | 用途 |
|-----|------|------|
| DB_HOST/PORT/USER/PASSWORD/NAME | localhost/5432/postgres/postgres/translator_checkin | Postgres 連線（`DSN()`，TimeZone=Asia/Taipei, sslmode=disable）|
| JWT_SECRET | （無安全預設）| HS256 簽章金鑰 — **見 §3 守衛** |
| JWT_EXPIRY_HOURS | 24 | token 壽命 |
| UPLOAD_DIR | ./uploads | 照片儲存 + 靜態服務根 |
| PORT | 8080 | HTTP 埠 |
| GOOGLE_CREDENTIALS_FILE | "" | Google Sheet 匯出 service account |
| SMTP_HOST/PORT/USER/PASSWORD/FROM | ""/587/""/""/"" | 寄信（MailService）|
| MAX_LOGIN_ATTEMPTS | 5 | lockout 門檻 |
| LOCK_DURATION_MINUTES | 15 | lockout 時長 |
| PHOTO_RETENTION_DAYS | 90 | 照片清除保留期 |
| ADMIN_DEFAULT_PASSWORD | "" | seed admin 密碼（空則隨機產生並 log）|
| LINE_CHANNEL_ACCESS_TOKEN | "" | LINE push |

OTel 相關（`OTEL_*`、`DEPLOY_ENV`）不在此處，由 [tracing](../tracing/TRACING_SPEC.md) 直接讀 env。

## 3. 不變式（安全守衛）
| 不變式 | 保證 |
|--------|------|
| JWT_SECRET 不可為內建預設且須 ≥32 字 | **機制保證**：不符直接 `os.Exit(1)`（拒絕開機，印 openssl 提示）|
| AppConfig 在任何 service 使用前已載入 | 人工維持（main 第一步呼叫 Load）|

## 4. 邊界條件
- 數值 env 解析失敗 → 退回預設（不報錯）。
- 多數外部整合（Google/SMTP/LINE）未設定 → **功能在呼叫時才回錯**，不影響開機。

## 5. 協作者
被全 backend 透過 `config.AppConfig` 讀取；`DSN()` 給 [cmd/server](../../cmd/server/SERVER_SPEC.md) 連 DB。
