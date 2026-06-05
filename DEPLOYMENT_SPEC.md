# 部署規格與使用說明（Deployment Spec）

> 上層：[ARCHITECTURE_SPEC.md](ARCHITECTURE_SPEC.md)｜容器細節見 [DOCKER_SPEC](docker/DOCKER_SPEC.md)｜對外穿透原文見 [docs/local-expose.md](docs/local-expose.md)

## 1. 部署型態
本系統設計為 **Docker Compose 單機部署**（使用者 <100 人）。三種情境：

| 情境 | 用途 | 指令來源 |
|------|------|----------|
| **本機開發** | 日常開發 | dev compose |
| **對外穿透（expose）** | 讓外部人員臨時試用本機環境，免雲端部署 | expose overlay + cloudflared |
| **正式單機** | 部署到一台 server 長期運行 | dev compose + 強化 env |

## 2. 前置需求
- Docker（Desktop 或 daemon）。
- `backend/.env`（複製 `backend/.env.example`）。
- **`JWT_SECRET` 必填、≥32 字**（`openssl rand -hex 32`），否則 backend 拒絕啟動。

## 3. 環境變數（部署必看）

| 變數 | 必填 | 說明 |
|------|------|------|
| `JWT_SECRET` | ✅ | HS256 金鑰，≥32 字且非預設 |
| `DB_HOST/PORT/USER/PASSWORD/NAME` | ✅ | Docker 內 `DB_HOST=postgres` |
| `ADMIN_DEFAULT_PASSWORD` | 建議 | 首次啟動 seed admin 密碼；未設則隨機產生印在 log |
| `UPLOAD_DIR` | — | 預設 `/app/uploads`（已綁 volume）|
| `JWT_EXPIRY_HOURS` | — | 預設 24 |
| `MAX_LOGIN_ATTEMPTS` / `LOCK_DURATION_MINUTES` | — | 帳號鎖定（5 / 15）|
| `PHOTO_RETENTION_DAYS` | — | 照片保留天數（90）；cron 每日 03:00 清 |
| `GOOGLE_CREDENTIALS_FILE` | 選填 | Google Sheet 匯出 service account |
| `SMTP_HOST/PORT/USER/PASSWORD/FROM` | 選填 | 寄信（定期匯出報表）|
| `LINE_CHANNEL_ACCESS_TOKEN` | 選填 | LINE 排班提醒 |
| `OTEL_EXPORTER_OTLP_ENDPOINT` / `OTEL_SERVICE_NAME` / `DEPLOY_ENV` | 選填 | tracing；`OTEL_TRACES_EXPORTER=none` 可關 |

完整範本見 [backend/.env.example](backend/.env.example)。

## 4. 部署步驟

### 4.1 正式單機
```bash
# 1. 準備 env
cp backend/.env.example backend/.env
#   填入 JWT_SECRET（openssl rand -hex 32）、ADMIN_DEFAULT_PASSWORD、DB_HOST=postgres ...

# 2. 啟動
docker compose -f docker/docker-compose.yml up -d --build

# 3. 確認
docker compose -f docker/docker-compose.yml ps
docker compose -f docker/docker-compose.yml logs -f backend     # 看 seed admin 密碼（若未設）

# 4. 升級（拉新 code 後）
docker compose -f docker/docker-compose.yml up -d --build        # 重建 image，volume 資料保留
```
- 對外服務埠：frontend `3000`。前面通常再擺一層 TLS 反代（nginx/caddy/雲端 LB）。
- DB schema 由 backend 啟動時 `AutoMigrate` 自動建立／升級（見 [SERVER_SPEC](backend/cmd/server/SERVER_SPEC.md)）。

### 4.2 對外穿透（cloudflared）
```bash
bash scripts/expose.sh           # 預設 3000；可帶參數指定 port
bash scripts/stop.sh             # 停止
```
`expose.sh` 會：自動裝 cloudflared（macOS/Homebrew）→ 用 `docker-compose.yml + docker-compose.expose.yml` 起 stack（換 `nginx.expose.conf`：信任 `CF-Connecting-IP`、`X-Forwarded-Proto: https`）→ 等就緒 → `cloudflared tunnel --url` 產生臨時 HTTPS 網址。Ctrl+C 關隧道但容器續跑。架構圖與細節見 [docs/local-expose.md](docs/local-expose.md)。

## 5. 資料持久化與備份
| 資料 | 位置 | 備份方式 |
|------|------|----------|
| PostgreSQL | volume `postgres_data` | `pg_dump`（或備份 volume）|
| 上傳照片 | bind mount `backend/uploads/` | 直接備份目錄 |
- `docker compose down` 保留 volume；`down -v` 會**刪除資料**，正式環境慎用。
- 照片留存 `PHOTO_RETENTION_DAYS` 天後由 cron 自動清除 → 業務查詢期需短於保留期，必要的證據請另存。

## 6. 背景排程（cron，backend 內建）
| 時間 | 工作 | 需求 |
|------|------|------|
| 03:00 | 清除逾期照片 | — |
| 07:00 | 明日排班提醒（LINE+Email）| LINE token / SMTP |
| 08:00 | 定期匯出當天到期者 → email | SMTP（+Google 憑證）|
時間依容器時區（`TimeZone=Asia/Taipei`）。

## 7. 安全檢查清單（上線前）
- [ ] `JWT_SECRET` 為隨機 ≥32 字（非範例值）。
- [ ] `ADMIN_DEFAULT_PASSWORD` 已設，且首位 admin 已登入改密碼（must_change_pw）。
- [ ] **CORS 收斂**：dev compose 預設 `AllowAllOrigins`，正式應限白名單（見 [SERVER_SPEC](backend/cmd/server/SERVER_SPEC.md) §10）。
- [ ] 前端前面有 TLS（HTTPS）。
- [ ] DB 密碼非預設 `postgres/postgres`。
- [ ] `ENABLE_TEST_RESET` **未設**（正式 build 也不含 e2e tag，雙重保險）。
- [ ] 照片改雲端儲存 + 簽名 URL（目前為本機靜態，⬜ 待強化，見 [PRODUCT_SPEC §14](PRODUCT_SPEC.md)）。

## 8. 可觀測性
- Jaeger UI：http://localhost:16686（dev stack）。
- backend 對 OTLP `jaeger:4317`；collector 不可達不影響服務（背景重試）。
- 細節見 [TRACING_SPEC](backend/internal/tracing/TRACING_SPEC.md)。

## 9. 疑難排解
| 症狀 | 排查 |
|------|------|
| backend 起不來 | `.env` 缺 `JWT_SECRET` / 太短；看 `logs backend` |
| 忘記 admin 密碼 | `docker exec <backend> ./server -reset-password admin@admin.com "NewPass123"` |
| 照片上傳 413 | 反代 `client_max_body_size`（compose 內 nginx 已設 20m）|
| 外部連不到 | 確認用 expose overlay + cloudflared，且前端走相對 `/api` |
| 報表沒寄出 | SMTP env 未設或錯；定期匯出僅在 day_of_month 命中當天 08:00 跑 |

## 10. 協作者
容器與 nginx 細節見 [DOCKER_SPEC](docker/DOCKER_SPEC.md)；啟動序列/守衛見 [SERVER_SPEC](backend/cmd/server/SERVER_SPEC.md) 與 [CONFIG_SPEC](backend/internal/config/CONFIG_SPEC.md)；產品面狀態見 [PRODUCT_SPEC](PRODUCT_SPEC.md)。
