# 更動報告 — 2026-05-06

## Commits

| Hash | 說明 |
|------|------|
| `7ff58f8` | chore: 新增根目錄 .gitignore |
| `bea09eb` | fix(docker): 移除 Dockerfile.backend 中不存在的 migrations COPY |

## 變更摘要

### 做了什麼

1. **新增根目錄 `.gitignore`**
   - 專案根目錄原本沒有 `.gitignore`，導致 `backend/uploads/`（runtime 上傳目錄）出現在 `git status` 中觸發 Stop hook
   - 加入常見忽略規則：`backend/uploads/`、`.DS_Store`、`frontend/dist`、`node_modules`、`.env` 系列、`.claude/settings.local.json`

2. **修復 `Dockerfile.backend` 的 migrations COPY 錯誤**
   - 第 21 行有 `COPY migrations ./migrations`，但此專案使用 GORM AutoMigrate，不存在 `migrations/` 目錄
   - 任何 `docker compose up --build` 都會報錯：`"/migrations": not found`
   - 移除該行，build 恢復正常
   - 此修復同時存在於 `master` 與 `feature/local-expose` 兩個分支

### 影響範圍

| 範圍 | 檔案 |
|------|------|
| Git 設定 | `.gitignore`（新增） |
| Docker 建置 | `docker/Dockerfile.backend` |

### 注意事項

- `Dockerfile.backend` 修復是透過 cherry-pick 從 `feature/local-expose` 帶回 `master`
- `.gitignore` 新增後，`backend/uploads/` 的已存在檔案若曾被追蹤，需手動 `git rm --cached` 移除（本專案該目錄未曾被追蹤，無此問題）
