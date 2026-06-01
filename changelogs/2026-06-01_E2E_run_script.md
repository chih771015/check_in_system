# 更動報告 — 2026-06-01：E2E 一鍵腳本 + 操作文件

延續本日早些時候的 E2E framework setup，補上「怎麼跑」的部分。

## Commits

| Hash | 說明 |
|---|---|
| `d01573d` | feat(e2e): 加 run-e2e.sh 一鍵腳本 + 詳細操作文件 |

## 變更摘要

### `e2e/run-e2e.sh`（新，可執行）

一鍵把 E2E 從零跑起來的 bash 腳本。內容：

1. 檢查環境：docker / node ≥ 18 / npm / curl + Docker daemon 在跑
2. `npm install` 如果沒裝過
3. `npx playwright install chromium` 如果沒裝過
4. `docker compose up -d --build` 啟動 e2e stack
5. 每 2 秒 POST `/api/test/reset` 探測，最多 90s
6. 跑 Playwright
7. 印報告路徑 + 視需要 tear down

支援 flag：
- `--no-stack`：跳過啟 stack（適用 stack 已在跑時只重跑測試）
- `--down`：跑完拆 stack + 刪 volume
- `--ui`：開 Playwright UI mode debug
- `--keep-going`：測試失敗也 exit 0（搭配 --down 用）
- `-h / --help`：印用法

支援 docker compose v2 與 v1 自動偵測，自動定位 e2e/ 目錄（從任何地方執行都行）。

### `e2e/package.json` 新增 npm script

- `npm run e2e` → `./run-e2e.sh`
- `npm run e2e:ui` → `./run-e2e.sh --ui`
- `npm run e2e:once` → `./run-e2e.sh --down`
- `npm run stack:logs` → tail compose log

### `e2e/HOW_TO_RUN.md`（新）

完整繁中操作文件：
- TL;DR 三行指令
- 前置需求 + 各工具最低版本
- 第一次跑的 6 步驟逐項解釋（每一步預期看到的訊息 + 失敗時的修法）
- 常用情境：只跑單 spec / debug UI / 手動 reset / 看 log
- 故障排除：port 衝突 / backend 起不來 / selector 找不到 / ENABLE_TEST_RESET 失效
- CI 整合的 reference YAML
- dev stack vs e2e stack 完整對照表

## 影響檔案

- `e2e/run-e2e.sh`（新，755）
- `e2e/HOW_TO_RUN.md`（新）
- `e2e/package.json`：加 e2e / e2e:ui / e2e:once / stack:logs script

## 後續

可以開始實際跑了：
```bash
cd e2e
./run-e2e.sh
```
首次預估 3-5 分鐘（docker pull + build + playwright download）。之後每次重跑大約 30 秒到一兩分鐘。

Selectors 可能需要微調（含中文/英文/泰文 regex 是首版，跑過真實 stack 後會發現哪幾個按鈕對不上）。
