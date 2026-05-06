# 更動報告 — 2026-05-06

## Commits（feature/local-expose 分支）

| Hash | 說明 |
|------|------|
| `0b78fe7` | chore(claude): 強化 Stop hook — 加入 changelog 缺失偵測 |
| `90c072f` | fix(claude): stop-check.sh 修正 changelog 計數邏輯 |
| `c4f2f05` | chore: 停止追蹤 .claude/settings.local.json（已加入 .gitignore） |

## 變更摘要

### 做了什麼

1. **SessionStart hook（`.claude/hooks/session-start.sh`）**
   - 新增 SessionStart hook，每次對話開始時記錄當下的 `git HEAD`
   - 儲存至 `/tmp/claude-session-{session_id}.head`
   - 讓 Stop hook 能比對「這次 session 前後 HEAD 是否移動」

2. **Stop hook 重構（`.claude/hooks/stop-check.sh`）**
   - 原 inline bash 移出為獨立 script，邏輯更易維護
   - 保留原有「未提交變更 → exit 2」功能
   - 新增「有新 commit 但 changelogs/ 沒被碰到 → exit 2」偵測
   - 修正 `grep -c` 在無 match 時 exit 1 導致 `|| echo 0` 重複輸出的 bug，改用 `[ -z ]` 判空字串

3. **停止追蹤 `.claude/settings.local.json`**
   - 該檔案已在 `.gitignore` 中列為忽略，但過去已被追蹤
   - 用 `git rm --cached` 移除追蹤，未來本機設定不再進入版控

4. **`settings.json` 更新**
   - 新增 `SessionStart` hook 區塊
   - `Stop` hook command 由 inline bash 改為 `bash .claude/hooks/stop-check.sh`

### 影響範圍

| 範圍 | 檔案 |
|------|------|
| Claude Code hook 腳本 | `.claude/hooks/session-start.sh`（新增）、`.claude/hooks/stop-check.sh`（新增）|
| Claude Code 設定 | `.claude/settings.json` |
| Git 追蹤 | `.claude/settings.local.json`（移出版控）|

### 注意事項

- `bash .claude/hooks/stop-check.sh` 是相對路徑，hooks 從 git repo 根目錄執行，路徑正確
- `SessionStart` hook 需要**重啟 Claude Code 或開啟 `/hooks` 選單**才會在新 session 生效
- Stop hook 現在有兩層防護：① 未提交變更、② 有 commit 但無 changelog；兩者均會 rewake
- `/tmp/claude-session-{id}.head` 為暫存檔，重開機後會消失（無影響，下次 session 重新記錄）
