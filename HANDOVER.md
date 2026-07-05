# 交接說明 / Handover Guide

> 翻譯員打卡系統。接手第一份先看這個。
> Translator Check-in System. **Start here.**

---

## 中文

### 這是什麼
一套「翻譯員到院打卡」系統：管理員排班、翻譯員到現場用 **GPS + 拍照**打卡，
管理員可看報表、匯出 Excel / Google Sheet、發 LINE / Email 提醒。
技術：Go + PostgreSQL 後端、React 前端、全部用 Docker Compose 跑。

### 你會拿到什麼、怎麼跑起來
你拿到的是**原始碼**（不是預先做好的 image）。跑起來只要三步：

```bash
# 1. 裝好 Docker（含 docker compose v2），終端機打 docker info 不報錯即可
# 2. 進到專案資料夾
cd 翻譯員打卡系統

# 3. 一鍵部署（第一次會自動建設定檔、產生密碼、建置容器）
bash scripts/deploy-prod.sh
```

跑完終端機會印出 **管理員帳號與初始密碼**，記下來。系統會在
`http://127.0.0.1:3000` 運行（只有這台機器看得到）。

- 要讓外出的翻譯員用手機連 → 看 [`docs/PRODUCTION_DEPLOY.md`](docs/PRODUCTION_DEPLOY.md) 第三節（Cloudflare Tunnel）。
- 升級（拿到新版原始碼後）→ 再跑一次 `bash scripts/deploy-prod.sh`，資料不會不見。

### 你只需要碰一個設定檔
`backend/.env.production`（第一次跑腳本時自動產生）。裡面每個欄位都有中文註解。
- 密碼類（JWT、資料庫、admin 初始密碼）→ **腳本自動產生，不用管**。
- 選填整合（SMTP 寄信、Google Sheets、LINE、對外 Tunnel）→ **要用才填**，填法看 [`docs/PRODUCTION_DEPLOY.md`](docs/PRODUCTION_DEPLOY.md)。

### 日常維運與備份
指令表、備份方式、上線前安全檢查清單，全部在 [`docs/PRODUCTION_DEPLOY.md`](docs/PRODUCTION_DEPLOY.md)。
- ⚠️ **千萬不要**對正式環境下 `docker compose ... down -v`，`-v` 會刪光資料庫。
- 備份：資料庫在 Docker volume `postgres_data`；上傳照片在 `backend/uploads/`。

### 文件地圖
| 檔案 | 給誰看 |
|------|--------|
| **HANDOVER.md**（本檔） | 接手的人，第一份 |
| [`docs/PRODUCTION_DEPLOY.md`](docs/PRODUCTION_DEPLOY.md) | 部署與維運（白話中文） |
| [`README.md`](README.md) | 想改程式的工程師（架構 / 開發） |
| [`DEPLOYMENT_SPEC.md`](DEPLOYMENT_SPEC.md) · [`ARCHITECTURE_SPEC.md`](ARCHITECTURE_SPEC.md) | 技術規格細節 |

---

## English

### What this is
A "translator on-site check-in" system: admins schedule assignments; interpreters
check in on location with **GPS + photo** proof; admins get dashboards, Excel /
Google Sheet export, and LINE / Email reminders.
Stack: Go + PostgreSQL backend, React frontend, all run via Docker Compose.

### What you get & how to run it
You receive the **source code** (not a prebuilt image). Three steps to run:

```bash
# 1. Install Docker (with docker compose v2); `docker info` must succeed
# 2. Enter the project folder
cd translator-checkin-system

# 3. One-command deploy (first run auto-creates config, generates secrets, builds)
bash scripts/deploy-prod.sh
```

The script prints the **admin account and initial password** at the end — save it.
The app runs at `http://127.0.0.1:3000` (visible on this machine only).

- To expose it to interpreters' phones → see [`docs/PRODUCTION_DEPLOY.md`](docs/PRODUCTION_DEPLOY.md) section 3 (Cloudflare Tunnel; note: that file is in Chinese).
- To upgrade (after pulling new source) → run `bash scripts/deploy-prod.sh` again; data is preserved.

### You only touch one config file
`backend/.env.production` (auto-created on first run). Every field has a comment.
- Secrets (JWT, DB, admin password) → **auto-generated, leave them alone**.
- Optional integrations (SMTP, Google Sheets, LINE, public Tunnel) → **fill only if used**.

### Operations & backup
Command cheatsheet, backup steps, and a pre-launch security checklist are all in
[`docs/PRODUCTION_DEPLOY.md`](docs/PRODUCTION_DEPLOY.md).
- ⚠️ **Never** run `docker compose ... down -v` on production — `-v` wipes the database.
- Backup: database lives in the `postgres_data` Docker volume; uploaded photos in `backend/uploads/`.
