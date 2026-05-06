# 更動報告 — 2026-05-06

## Commits

| Hash | 說明 |
|------|------|
| `21ca3e9` | fix(frontend): API baseURL 改為相對路徑，修復外部瀏覽器打到 localhost 的問題 |

（已 cherry-pick 至 master：`4eddc39`）

## 變更摘要

### 問題描述

透過 cloudflared tunnel 對外開放後，在外部瀏覽器操作時所有 API 請求都打到 `localhost:8080`，導致請求失敗。

### 根因

`frontend/src/api/client.ts` 的 axios `baseURL` 寫死為 `http://localhost:8080/api`。瀏覽器會將這個 URL 原封不動送出，外部機器上的 `localhost` 當然不是後端。

### 修復

- `client.ts`：`baseURL` 改為相對路徑 `/api`，瀏覽器會根據當前域名（tunnel URL 或 localhost）自動補完 origin
- `vite.config.ts`：新增 `server.proxy`，本地 `npm run dev` 時將 `/api/*` 轉發到 `http://localhost:8080`，維持開發體驗不變

### 影響範圍

| 範圍 | 檔案 |
|------|------|
| 前端 API 客戶端 | `frontend/src/api/client.ts` |
| 前端開發設定 | `frontend/vite.config.ts` |

### 注意事項

- 改為相對路徑後，nginx 的 `/api/` proxy 設定（`nginx.conf`、`nginx.expose.conf`）是唯一的後端入口，請確保 nginx 設定正確
- `npm run dev` 開發模式需要後端在 `localhost:8080` 才能正常使用
