# 部署指南 — 交付與部署選項

> 這份文件給「拿到這套系統、要把它架起來」的人。
> 我們**不預設**你的環境（自架主機 / 雲端、有沒有 Docker、怎麼對外都由你決定）。
> 技術規格細節見 [DEPLOYMENT_SPEC.md](../DEPLOYMENT_SPEC.md)。

---

## 0. 系統長什麼樣（先有概念）

這套系統由四個部分組成，彼此的關係是固定的，但**跑在哪、怎麼對外，你自由選**：

```
   使用者瀏覽器 / 手機
        │  HTTPS（由你選的對外方式提供）
        ▼
   ┌─────────────────────────────┐
   │ 反向代理（必要）             │  ← 服務前端靜態檔 +
   │  · 服務前端 SPA              │    把 /api、/uploads 轉給後端
   │  · /api  → 後端 :8080        │    （Docker 版用內建 nginx；
   │  · /uploads → 後端 :8080     │      裸機版你自己擺 nginx/Caddy）
   └───────────────┬─────────────┘
                   │
                   ▼
   ┌─────────────────────────────┐      ┌──────────────┐
   │ 後端 API（Go，單一 binary） │─────▶│ PostgreSQL   │  ← 內建容器
   │  監聽 :8080                  │      │  資料庫      │    或你自己的 DB
   └───────────────┬─────────────┘      └──────────────┘
                   │ （選用）OTLP
                   ▼
             ┌──────────────┐
             │ Jaeger 追蹤  │  ← 選用，可完全不裝
             └──────────────┘
```

**三個重點**（決定了下面所有選擇）：
1. **反向代理是必要的**，因為前端固定用相對路徑 `/api` 打後端。Docker 版已內建 nginx；不用 Docker 就得自己擺一個。
2. **資料庫可換**：預設用內建 PostgreSQL 容器，但你也可以接自己的 / 雲端 DB。
3. **對外公開完全是你的事**：系統只在本機開一個 HTTP port，要用什麼擺在前面（自架反代 + 你的網域、雲端負載平衡器、Cloudflare Tunnel…）由你決定。

---

## 1. 你要做的三個決定

| 決定 | 選項 | 看哪一節 |
|------|------|---------|
| **A. 怎麼跑** | Docker Compose（推薦，最省事） / 不用 Docker 裸機跑 | [第 3 節](#3-決定-a怎麼跑) |
| **B. 資料庫** | 內建 PostgreSQL 容器（預設） / 你自己的或雲端 Postgres | [第 4 節](#4-決定-b資料庫放哪) |
| **C. 對外公開** | 完全你決定：自架反代 / 雲端 LB / Cloudflare Tunnel / 只在內網 | [第 5 節](#5-決定-c怎麼對外公開完全你決定) |

其餘（SMTP、Google Sheets、LINE、Jaeger 追蹤）**全部選用**，要用才開，見 [第 6 節](#6-選用整合全部-opt-in)。

---

## 2. 最低環境需求

不管你選哪條路，這台機器最少要有：

- **一台你能 SSH / 操作的主機**（Linux 最常見；Windows Server、macOS、雲端 VM 都行）。
- **對外連得到網際網路**（build 時要抓相依套件與 base image）。
- 依你選的路線，二選一：
  - **走 Docker**：裝好 Docker Engine + Docker Compose v2（`docker info`、`docker compose version` 不報錯即可）。
  - **走裸機**：Go 1.26+、Node 20+、一個 PostgreSQL 16、一個反向代理（nginx / Caddy）。
- **硬碟空間**：照片預設**永久保存**，請預留足夠空間並安排備份（見 [第 7 節](#7-日常維運與備份)）。

---

## 3. 決定 A：怎麼跑

### 路線 A（推薦）：Docker Compose 一鍵部署

最省事。一個指令搞定 build、產生密鑰、啟動、健康檢查。

```bash
# 1. 進到專案資料夾
cd /path/to/專案

# 2. 一鍵部署（第一次會自動建立 backend/.env.production、產生密碼、build 容器）
bash scripts/deploy-prod.sh

# 3. 記下終端機最後印出的「管理員帳號與初始密碼」
```

跑完系統在 `http://127.0.0.1:3000`（**只有本機看得到**，對外看第 5 節）。
之後升級（拿到新版原始碼）只要再跑一次 `bash scripts/deploy-prod.sh`，資料不會遺失。

> 內建服務：postgres、後端、前端(nginx)、jaeger。前端只綁 `127.0.0.1:3000`，不直接曝到公網。

### 路線 B：不用 Docker（裸機 / 自備環境）

當客戶環境不給裝 Docker、或已有既定的部署方式時走這條。你要自己備齊
**PostgreSQL + 跑後端 binary + build 前端靜態檔 + 一個反向代理**。

**B-1. 準備資料庫**
建一個 PostgreSQL 16 資料庫，記下 host / port / user / password / dbname（下一步要填）。
（資料表由後端啟動時自動建立，不用手動跑 migration。）

**B-2. 建置並啟動後端**（Go 單一 binary）

```bash
cd backend
cp .env.example .env        # 然後編輯 .env，見下方必填項
go build -o server ./cmd/server
./server                    # 監聽 PORT（預設 8080）
```

`.env` 至少要填：
```ini
DB_HOST=你的DB主機     # 裸機同一台就填 localhost
DB_PORT=5432
DB_USER=...
DB_PASSWORD=...        # 不要用預設 postgres
DB_NAME=translator_checkin
JWT_SECRET=            # 必填，≥32 字：用 openssl rand -hex 32 產生
UPLOAD_DIR=./uploads   # 照片存這裡，記得納入備份
PORT=8080
# 選用整合見第 6 節；不填不影響核心功能
```
> 正式環境建議用 systemd / supervisor 等把 `./server` 顧成常駐服務（開機自動拉起）。
> 忘記 admin 密碼時：`./server -reset-password admin@admin.com '新密碼'`。

**B-3. build 前端靜態檔**

```bash
cd frontend
npm install
npm run build          # 產出靜態檔到 frontend/dist/
```

**B-4. 擺一個反向代理**（**這步不能省**）

前端固定用相對路徑 `/api` 打後端，所以你的反代必須做三件事：
1. 服務 `frontend/dist/` 的靜態檔，且 **SPA fallback**（找不到檔就回 `index.html`）。
2. 把 `/api/` 轉給後端 `http://127.0.0.1:8080`。
3. 把 `/uploads/` 也轉給後端（否則照片會壞掉）。
4. **放大上傳上限**到 100MB（手機照片一張 3–8MB，一次上傳多張會撞到預設 1MB 上限）。

現成範本就是 [`docker/nginx.conf`](../docker/nginx.conf)，把裡面的 `proxy_pass http://backend:8080`
改成 `http://127.0.0.1:8080` 即可直接用。Caddy 的等價設定也很短，原則相同。

> **同源最省事**：讓前端和 `/api` 走同一個網域（就是上面這個反代），就**不需要處理 CORS**。
> 只有當你把前端和後端拆到不同網域時，才需要另外設定 CORS。

---

## 4. 決定 B：資料庫放哪

### 選項 1（預設）：用內建的 PostgreSQL 容器
Docker 路線預設就是這個，`deploy-prod.sh` 會自動產生強密碼，你什麼都不用做。
資料存在 Docker volume `postgres_data`。

### 選項 2：接你自己的 / 雲端 Postgres（RDS、Cloud SQL、既有 DB…）
1. 在 `backend/.env.production`（Docker）或 `backend/.env`（裸機）把 `DB_HOST/PORT/USER/PASSWORD/DB_NAME`
   改成你那顆 DB 的連線資訊。
2. **若走 Docker**：編輯 [`docker/docker-compose.prod.yml`](../docker/docker-compose.prod.yml)，
   把 `postgres` 服務區塊、以及 backend 的 `depends_on: postgres` 註解掉（因為你不再用內建 DB）。
3. 確認後端所在網路連得到那顆 DB（防火牆 / VPC 安全群組放行 5432）。

> 資料表會由後端在啟動時自動建立（AutoMigrate），你只要給一個空的 database 即可。

---

## 5. 決定 C：怎麼對外公開（完全你決定）

**系統本身只做一件事：在本機開一個 HTTP port（Docker 版是 `127.0.0.1:3000`）。**
要不要對外、用什麼網域、HTTPS 憑證怎麼來——**通通是你的決定，系統不強制、也不綁定任何一種**。

下面是幾個常見做法，挑一個就好（不是非得照哪個）：

| 你的情況 | 建議做法 |
|---------|---------|
| 主機有公開 IP / 在雲端 | 在前面擺你自己的 **nginx / Caddy** 反代，綁你的網域、上你的 TLS 憑證（Let's Encrypt 等），反代到本機的 3000（或裸機的前端埠）。 |
| 用雲端平台 | 直接用雲端的 **負載平衡器 / Ingress**（ALB、Cloud Load Balancing、K8s Ingress…）指到這台的埠，TLS 在 LB 上終結。 |
| 主機在內網、沒有公開 IP | 用 **Cloudflare Tunnel**（或 Tailscale、frp…）把它安全地接出去，免開防火牆 port、自動 HTTPS。**這只是其中一個選項**，見下方。 |
| 只在內網 / 區網用 | 什麼都不用加，區網內用 `http://這台的內網IP:3000` 連即可。 |

### （選項之一）Cloudflare Tunnel
如果你剛好符合「內網無公開 IP」且想用 Cloudflare，系統有內建支援：

1. Cloudflare Zero Trust → Networks → Tunnels → Create a tunnel → 選 Cloudflared，取得 token。
2. 把 token 填進 `backend/.env.production` 的 `TUNNEL_TOKEN=`。
3. 在 Tunnel 的 Public Hostname 設定：你的網域 → Service **HTTP** → `frontend:80`。
4. 部署時加參數：`bash scripts/deploy-prod.sh --tunnel`。

不用 Tunnel 就別填 `TUNNEL_TOKEN`、也別加 `--tunnel`，這個服務根本不會啟動（它掛在
compose 的 `tunnel` profile 底下，預設關）。

---

## 6. 選用整合（全部 opt-in）

以下**全部選用**，留空 / 不設定該功能就是關閉，**不影響核心打卡功能**。
每個欄位在 `backend/.env.production`（或 `.env`）裡都有中文註解說明怎麼取得。

| 整合 | 用途 | 怎麼開 |
|------|------|--------|
| **SMTP** | 定期報表 email、提醒 email | 填 `SMTP_HOST/PORT/USER/PASSWORD/FROM` |
| **Google Sheets** | 把打卡資料匯出到 Google 試算表 | 放憑證 JSON、填 `GOOGLE_CREDENTIALS_FILE`，Docker 版另需在 compose 打開對應 volume |
| **LINE** | 每天推播明日排班提醒 | 填 `LINE_CHANNEL_ACCESS_TOKEN` |
| **Jaeger 追蹤** | 分散式追蹤 / 觀測 | Docker 版預設開；不想要可在 `.env` 設 `OTEL_TRACES_EXPORTER=none`。裸機不裝 Jaeger 也能跑 |

---

## 7. 日常維運與備份

| 你想做的事 | Docker 指令 |
|-----------|------|
| 看服務狀態 | `docker compose -f docker/docker-compose.prod.yml ps` |
| 看後端 log | `docker compose -f docker/docker-compose.prod.yml logs -f backend` |
| 停止（保留資料） | `docker compose -f docker/docker-compose.prod.yml down` |
| 升級（拉新 code 後） | `bash scripts/deploy-prod.sh` |
| 重設 admin 密碼 | `docker compose -f docker/docker-compose.prod.yml exec backend ./server -reset-password admin@admin.com '新密碼'` |

（裸機版對應：`systemctl status/restart` 你的服務、直接看 binary 的 log、重跑 `go build` 升級。）

### 備份（重要）
- **資料庫**：內建容器 → 備份 Docker volume `postgres_data`，或定期 `pg_dump`；雲端 DB → 用雲端的備份機制。
- **上傳照片**：存在後端的 `UPLOAD_DIR`（Docker 版綁到 `backend/uploads/`）。直接複製整個資料夾即可。
- ⚠️ **千萬不要**對正式環境下 `docker compose ... down -v`——那個 `-v` 會**刪掉資料庫資料**。

---

## 8. 上線前安全檢查清單

- [ ] `JWT_SECRET`、`DB_PASSWORD` 都是強值（非預設 / 非空）。走腳本會自動產生；裸機請自己用 `openssl rand` 產。
- [ ] 第一次用 admin 登入後，已照提示改掉初始密碼。
- [ ] 對外是 **HTTPS**（不論用哪種對外方式，憑證都由你這端確保）。
- [ ] `backend/.env` / `backend/.env.production`（含密碼金鑰）**沒有**被 commit 進 git。
- [ ] 已安排資料庫與照片的定期備份。
- [ ] 照片預設**永久保存**（`PHOTO_RETENTION_DAYS=0`）；確認硬碟空間足夠。
