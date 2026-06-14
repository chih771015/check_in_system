# docker — 規格與使用說明

> 對應檔案：`docker/*`
> 上層：[ARCHITECTURE_SPEC.md](../ARCHITECTURE_SPEC.md)

## 1. 定位與職責
用 Docker Compose 把整個系統（postgres / jaeger / backend / frontend）一鍵拉起。提供三種 stack：**dev（開發）**、**e2e（測試隔離）**、**expose（對外穿透）**。

## 2. 檔案總覽
| 檔案 | 用途 |
|------|------|
| `Dockerfile.backend` | 後端 production image（多階段 build，Go 1.26 → alpine）|
| `Dockerfile.backend.e2e` | 同上但 `-tags e2e`，編入 `/api/test/reset` 端點 |
| `Dockerfile.frontend` | 前端 build（node 20 → nginx:alpine 靜態服務）|
| `docker-compose.yml` | **dev stack**：postgres + jaeger + backend + frontend |
| `docker-compose.e2e.yml` | **e2e stack**：獨立 port/volume、無 jaeger、reset 端點開啟 |
| `docker-compose.expose.yml` | dev stack 的 overlay：換成對外穿透用 nginx 設定 |
| `nginx.conf` | 前端 SPA fallback + `/api/`、`/uploads/` 反向代理（`client_max_body_size 100m`）|
| `nginx.expose.conf` | 同上 + 信任 cloudflared 的 `CF-Connecting-IP`、強制 `X-Forwarded-Proto: https` |

## 3. dev stack（docker-compose.yml）

| service | image / build | host port | 說明 |
|---------|---------------|-----------|------|
| postgres | postgres:16-alpine | 5432 | healthcheck `pg_isready`；volume `postgres_data` 持久化 |
| jaeger | jaegertracing/all-in-one:1.57 | 16686(UI)/4317(gRPC)/4318(HTTP) | tracing 後端 |
| backend | Dockerfile.backend | 8080 | `env_file: backend/.env` + 覆寫 `DB_HOST=postgres`、OTLP 指向 jaeger；`uploads` 綁本機 volume |
| frontend | Dockerfile.frontend | 3000→80 | nginx 反代 backend |

**啟動 / 停止**
```bash
docker compose -f docker/docker-compose.yml up -d --build
docker compose -f docker/docker-compose.yml down          # 保留資料
docker compose -f docker/docker-compose.yml down -v       # 連 volume 一起刪
```
- 前端 → http://localhost:3000
- Jaeger UI → http://localhost:16686
- 預設 admin：`admin@admin.com` / `ADMIN_DEFAULT_PASSWORD`（未設則看 backend log 的隨機密碼）。

> ⚠️ backend 需要 `backend/.env`（複製 `.env.example`），其中 **`JWT_SECRET` 必填且 ≥32 字**，否則容器啟動即 `os.Exit(1)`（見 [config spec](../backend/internal/config/CONFIG_SPEC.md)）。

## 4. e2e stack（docker-compose.e2e.yml）
與 dev **完全隔離**，可並存：

| 差異 | dev | e2e |
|------|-----|-----|
| backend 映像 | Dockerfile.backend | **Dockerfile.backend.e2e**（-tags e2e）|
| `ENABLE_TEST_RESET` | 無 | `true`（開 `/api/test/reset`）|
| postgres port | 5432 | **55432** |
| backend port | 8080 | **8081** |
| frontend port | 3000 | **3001** |
| jaeger | 有 | 無（`OTEL_TRACES_EXPORTER=none`）|
| volume | postgres_data / uploads | postgres_e2e_data / uploads_e2e |

```bash
docker compose -f docker/docker-compose.e2e.yml -p thai-e2e up -d --build
docker compose -f docker/docker-compose.e2e.yml -p thai-e2e down -v
```
詳見 [E2E_SPEC](../e2e/E2E_SPEC.md)。

## 5. expose stack（對外穿透）
`docker-compose.expose.yml` 是 **overlay**，只覆寫 frontend 的 nginx 設定為 `nginx.expose.conf`：
```bash
docker compose -f docker/docker-compose.yml -f docker/docker-compose.expose.yml up -d --build
```
搭配 cloudflared（`scripts/expose.sh`）即可讓外部瀏覽器透過 HTTPS 隧道存取本機。詳見 [DEPLOYMENT_SPEC](../DEPLOYMENT_SPEC.md) §4 與 [docs/local-expose.md](../docs/local-expose.md)。

## 6. nginx 反代要點（不變式）
| 規則 | 原因 |
|------|------|
| `/api/` → backend:8080 | 前端走相對路徑 `/api`，避免寫死 localhost（外部瀏覽器才連得到）|
| `/uploads/` → backend:8080 | 否則照片路徑會落入 SPA fallback 回傳 index.html（破圖）|
| `client_max_body_size 100m` | 手機照片 3–8MB；診斷照上限 30 張，單批放寬到 100m（前端 ~90MB 預檢，過大提示分批；預設 1MB 會 413）|
| `try_files ... /index.html` | SPA 前端路由 fallback |

## 7. 邊界條件 / 常見問題
| 情境 | 處理 |
|------|------|
| backend 起不來 | 多半是 `.env` 缺 `JWT_SECRET`；看 `docker compose logs backend` |
| 照片上傳 413 | nginx `client_max_body_size`（已設 100m）；自架反代別漏 |
| dev 與 e2e 互撞 | 不會：port/volume/project name 皆不同 |
| Google Sheet 匯出失敗 | 需掛載 `google-credentials.json`（compose 內有註解範例）|

## 8. 協作者
backend image 跑 [cmd/server](../backend/cmd/server/SERVER_SPEC.md)；測試見 [E2E_SPEC](../e2e/E2E_SPEC.md) / [TESTING_SPEC](../TESTING_SPEC.md)；部署見 [DEPLOYMENT_SPEC](../DEPLOYMENT_SPEC.md)。
