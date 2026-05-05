# 本機測試環境對外開放指南

> 適用分支：`feature/local-expose`  
> 用途：讓外部人員（不在同一網路）直接透過瀏覽器存取本機測試環境，無需部署到伺服器。

---

## 架構說明

```
外部瀏覽器
    ↓ HTTPS（自動簽憑證）
cloudflared tunnel（在本機執行）
    ↓ HTTP
nginx :3000  ← 靜態前端 + /api/* 反向代理
    ↓
backend :8080
    ↓
PostgreSQL :5432
```

前端使用相對路徑 `/api/...`，所有流量都經過同一個 port，不需要修改任何 API URL。

---

## 前置需求

| 項目 | 說明 |
|------|------|
| Docker Desktop | 需已安裝並執行中 |
| macOS + Homebrew | `cloudflared` 會由腳本自動安裝 |
| `feature/local-expose` 分支 | `git checkout feature/local-expose` |

---

## 啟動步驟

### 1. 切換到 expose 分支

```bash
git checkout feature/local-expose
```

### 2. 執行啟動腳本

```bash
bash scripts/expose.sh
```

腳本會依序：

1. **安裝 cloudflared**（若尚未安裝，透過 Homebrew 自動安裝，僅第一次需要）
2. **啟動 Docker Compose**（使用 expose 專用 nginx 設定）
3. **等待服務就緒**（自動輪詢 localhost:3000）
4. **建立穿透隧道**，並印出可對外存取的 HTTPS URL，例如：

```
https://random-words-xxxx.trycloudflare.com
```

### 3. 分享 URL

把步驟 2 印出的 URL 傳給測試人員，對方在瀏覽器開啟即可使用。

> **注意**：每次重新執行腳本，URL 都會不同（cloudflared 免費快速隧道的限制）。

---

## 停止服務

### 只關閉隧道（Docker 繼續執行）

在執行 `expose.sh` 的終端機按 **Ctrl+C**。

### 關閉隧道 + 停止所有 Docker 服務

```bash
bash scripts/stop.sh
```

---

## 新增的檔案說明

| 檔案 | 說明 |
|------|------|
| `scripts/expose.sh` | 一鍵啟動腳本（安裝工具、啟動 Docker、建立隧道） |
| `scripts/stop.sh` | 停止所有 Docker 服務 |
| `docker/nginx.expose.conf` | Expose 專用 nginx 設定（真實 IP 透傳、HTTPS header） |
| `docker/docker-compose.expose.yml` | Compose override，僅替換 nginx conf，其餘繼承主設定 |

---

## 常見問題

**Q：URL 每次都不一樣，可以固定嗎？**  
A：免費的 cloudflared 快速隧道每次都是隨機 URL。若需要固定 URL，可以：
- 註冊免費 Cloudflare 帳號，建立具名隧道（Named Tunnel）
- 或改用 ngrok 付費方案

**Q：cloudflared 安裝失敗怎麼辦？**  
A：手動安裝：
```bash
brew install cloudflared
```

**Q：要同時對外開放 Jaeger（Tracing UI）嗎？**  
A：目前腳本只開放 port 3000（前端 + API）。若需要也開放 Jaeger，可另外執行：
```bash
cloudflared tunnel --url http://localhost:16686
```

**Q：這份設定安全嗎？**  
A：任何拿到 URL 的人都能存取，包括管理員功能。建議：
- 測試完畢後立即 Ctrl+C 關閉隧道
- 不要在開放期間進行敏感操作（例如刪除正式資料）
- 如需加上基本認證保護，可在 `nginx.expose.conf` 加入 `auth_basic` 設定

---

## 注意事項

- 此分支（`feature/local-expose`）**不應合併到 `master`**，僅供本機對外測試使用
- Docker 資料（PostgreSQL volume）與主分支共用，對外測試的資料變更會保留
- cloudflared 快速隧道有流量與連線數限制（免費版），正式壓力測試請勿使用
