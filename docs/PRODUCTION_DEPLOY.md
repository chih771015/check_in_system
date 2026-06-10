# 正式環境部署指南（白話版）

> 這份文件帶你把系統部署到一台正式 server 上。技術規格細節見 [DEPLOYMENT_SPEC.md](../DEPLOYMENT_SPEC.md)。

## 這次部署的決策（依你的情境）

| 項目 | 選擇 |
|------|------|
| Server | 診所內部機器 |
| 資料庫 | 用 compose 內建的 PostgreSQL 容器 |
| 對外方式 | Cloudflare Tunnel（自動 HTTPS、免開公開 IP） |
| 選填整合 | SMTP、Google Sheets、LINE、Jaeger 全部啟用 |
| CORS | 維持現狀（前後端同源，風險低） |

---

## 一、Server 需要先準備什麼

1. **一台能裝 Docker 的機器**（Linux 最常見；Windows/Mac 也可）。
2. **安裝 Docker**（含 Docker Compose v2）。安裝完在終端機輸入 `docker info` 不報錯即可。
3. **把這個專案複製到 server 上**（`git clone` 或直接拷貝整個資料夾）。
4. 一個**網域名稱**（你已經有了）。

> 你不需要自己設定資料庫密碼、JWT 金鑰——部署腳本會自動產生。

---

## 二、最快的部署流程（3 步）

```bash
# 1. 進入專案資料夾
cd /path/to/Thai

# 2. 執行部署腳本（第一次會自動建立設定檔、產生密鑰、建置容器）
bash scripts/deploy-prod.sh

# 3. 看終端機最後印出的「登入資訊」——記下 admin 帳號與初始密碼
```

跑完後，系統會在 `http://127.0.0.1:3000` 運行（**只有 server 本機看得到**）。
要讓外出的翻譯員用手機連，繼續看第三步「對外公開」。

> 之後升級（拉了新 code）只要再跑一次 `bash scripts/deploy-prod.sh` 即可，資料不會遺失。

---

## 三、對外公開（Cloudflare Tunnel）

你的 server 在診所內部、沒有公開 IP，所以用 Cloudflare Tunnel 把它安全地接到網際網路，
**不用在防火牆或路由器開任何 port**，而且自動有 HTTPS（🔒）。

### 設定步驟

1. **確認網域在哪管理**（你說「待確認」）：
   - 到當初買網域的網站（GoDaddy / 中華電信 / Gandi…）登入查看，或問幫你買的人。
   - 若不在 Cloudflare，可免費把它的「DNS 代管」轉到 Cloudflare（Cloudflare 會給你兩組 nameserver，到原註冊商換上去即可）。

2. **建立 Tunnel**：
   - 註冊／登入 [Cloudflare](https://dash.cloudflare.com) → 進入 **Zero Trust** → **Networks → Tunnels** → **Create a tunnel**。
   - 選 **Cloudflared**，命名（例如 `clinic-checkin`），建立後它會顯示一段 **token**。

3. **把 token 填進設定檔**：打開 `backend/.env.production`，找到 `TUNNEL_TOKEN=`，把 token 貼在等號後面。

4. **設定對外網址**：在同一個 Tunnel 頁面的 **Public Hostname** 加一筆：
   - Subdomain/Domain：你的網域（例如 `checkin.你的診所.com`）
   - Service：**HTTP** → `frontend:80`

5. **重新部署並啟動 Tunnel**：
   ```bash
   bash scripts/deploy-prod.sh --tunnel
   ```

完成後，翻譯員就能用 `https://checkin.你的診所.com` 從任何地方連進來。

---

## 四、要填的設定檔總整理

只有**一個**檔案要填：`backend/.env.production`
（第一次跑腳本時會自動從 `backend/.env.production.example` 複製產生）

| 欄位 | 要不要填 | 說明 |
|------|---------|------|
| `JWT_SECRET` | 自動產生 | 登入金鑰，腳本會自動填 |
| `DB_PASSWORD` | 自動產生 | 資料庫密碼，腳本會自動填 |
| `ADMIN_DEFAULT_PASSWORD` | 自動產生 | 管理員初始密碼，腳本會自動填並印出來 |
| `SMTP_HOST/USER/PASSWORD/FROM` | **要填** | 寄信功能（報表 email、提醒 email） |
| `GOOGLE_CREDENTIALS_FILE` + 憑證檔 | **要填** | Google Sheets 匯出（見下方） |
| `LINE_CHANNEL_ACCESS_TOKEN` | **要填** | LINE 排班提醒 |
| `TUNNEL_TOKEN` | **要填** | 對外公開（第三步） |
| 其他 | 已有預設 | 一般不用動 |

> 每個欄位在 `backend/.env.production` 檔案裡都有中文註解說明怎麼取得。

### Google Sheets 憑證額外步驟
1. Google Cloud Console → IAM → 服務帳號 → 建立金鑰 → 選 **JSON**。
2. 把下載的檔案改名放到 `backend/google-credentials.json`。
3. 打開 `docker/docker-compose.prod.yml`，把 `google-credentials.json` 那行 volume 的註解 `#` 拿掉。
4. 重跑 `bash scripts/deploy-prod.sh`。

---

## 五、日常維運

| 你想做的事 | 指令 |
|-----------|------|
| 看服務狀態 | `docker compose -f docker/docker-compose.prod.yml ps` |
| 看後端 log | `docker compose -f docker/docker-compose.prod.yml logs -f backend` |
| 停止系統（保留資料） | `docker compose -f docker/docker-compose.prod.yml down` |
| 升級（拉新 code 後） | `bash scripts/deploy-prod.sh` |
| 忘記 admin 密碼 | `docker compose -f docker/docker-compose.prod.yml exec backend ./server -reset-password admin@admin.com '新密碼'` |

### 備份（重要）
- **資料庫**：存在 Docker volume `postgres_data`。定期用 `pg_dump` 匯出，或備份整個 volume。
- **上傳照片**：存在 `backend/uploads/` 資料夾，直接複製整個資料夾即可。
- ⚠️ 千萬不要對正式環境下 `docker compose ... down -v`，那個 `-v` 會**刪掉資料庫資料**。

---

## 六、上線前安全檢查清單

- [ ] 跑過 `deploy-prod.sh`，`JWT_SECRET` / `DB_PASSWORD` 已是自動產生的強值（非預設）。
- [ ] 第一次用 admin 登入後，已照提示改掉初始密碼。
- [ ] 已設定 Cloudflare Tunnel，對外是 `https://`（有 🔒）。
- [ ] `backend/.env.production` **沒有**被 commit 進 git（已由 .gitignore 擋下）。
- [ ] 已安排資料庫與照片的定期備份。
- [ ] 確認照片保留天數 `PHOTO_RETENTION_DAYS`（預設 90 天）符合業務需求。
