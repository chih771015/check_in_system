# 如何跑 E2E 測試 — 完整流程

## TL;DR — 三條指令

```bash
cd e2e
./run-e2e.sh          # 自動處理：依賴、stack 啟動、等就緒、跑測試
./run-e2e.sh --down   # 跑完順便把 stack 收乾淨
```

如果你連 `./run-e2e.sh` 都不想記，npm script 也有：

```bash
cd e2e
npm run e2e           # 同 ./run-e2e.sh
npm run e2e:ui        # Playwright UI mode（互動式 debug）
npm run e2e:once      # 跑完自動 down -v
```

> 第一次跑會比較久（docker pull + go build + npm install + playwright install），大約 3-5 分鐘。之後每次大概 30 秒就完成 stack 啟動。

---

## 前置需求

開始前確認以下都裝好：

| 工具 | 最低版本 | 檢查指令 |
|---|---|---|
| Docker Desktop（或 Docker Engine） | 啟動中 | `docker info` |
| docker compose（v2）或 docker-compose（v1） | 任一即可 | `docker compose version` |
| Node.js | ≥ 18 | `node -v` |
| npm | ≥ 9 | `npm -v` |
| curl | 任一 | `curl --version` |

如果你 Mac 用 Homebrew：
```bash
brew install --cask docker
brew install node
```

裝完 Docker 記得**啟動 Docker Desktop**（mac 右上選單列要看到鯨魚圖示），不然 `docker info` 會掛。

---

## 第一次跑：詳細流程拆解

```bash
cd /Users/chiangchihhsuan/Desktop/Thai/e2e
./run-e2e.sh
```

腳本會依序做這些事，每一步失敗都會明確報錯：

### Step 1：檢查環境
看到 `✓ docker / node 20 / npm / curl available` 就 OK。常見問題：
- ❌ `Docker daemon is not running` → 開 Docker Desktop
- ❌ `Node 16 found; need >= 18` → `brew upgrade node` 或裝 nvm

### Step 2：裝 npm 依賴
第一次會看到 `Installing npm dependencies (first run)`，跑 `npm install`，約 30 秒。
之後 `node_modules` 存在就跳過。

### Step 3：裝 Playwright Chromium
第一次會下載 Chromium 二進位（約 150MB），約 1 分鐘。
之後跳過。

> 如果公司網路擋 Playwright 的 CDN，可以改 `PLAYWRIGHT_DOWNLOAD_HOST` 環境變數指到內部 mirror。

### Step 4：啟動 docker-compose stack
```
docker compose -f ../docker/docker-compose.e2e.yml -p thai-e2e up -d --build
```
- 第一次會 build backend image（`go build -tags e2e`，約 1 分鐘）+ build frontend image（npm install + vite build，約 2 分鐘）
- 之後只重啟容器，幾秒就好

啟動的 services：
| Service | Container 名 | Host port |
|---|---|---|
| postgres | thai-e2e-postgres-1 | 55432 |
| backend  | thai-e2e-backend-1  | 8081 |
| frontend | thai-e2e-frontend-1 | 3001 |

不會與 dev stack（5432 / 8080 / 3000）衝突，**可同時跑**。

### Step 5：等待後端就緒
```
Waiting for backend to be ready (timeout 90s)
....
✓ Backend ready, DB reset to clean seed state
```
腳本每 2 秒 POST 一次 `http://localhost:3001/api/test/reset`，回 200 才繼續。最多等 90 秒。

如果這步逾時：
1. 看 backend log：`npm run stack:logs` 或 `docker compose -f ../docker/docker-compose.e2e.yml -p thai-e2e logs backend`
2. 常見原因：
   - 第一次跑 GORM migration 較慢 → 把 `WAIT_TIMEOUT` 加到 180 再跑：`WAIT_TIMEOUT=180 ./run-e2e.sh`
   - postgres healthcheck 還沒過 → 等幾秒重跑
   - port 8081 被佔走 → `lsof -i :8081` 找出來殺掉

### Step 6：跑 Playwright
```
Running 25 tests using 1 worker
  ✓  1 [chromium-desktop] › auth.spec.ts (1.2s)
  ✓  2 [chromium-desktop] › auth.spec.ts (0.8s)
  ...
```

跑完顯示報告路徑：
```
Report: file:///Users/.../e2e/playwright-report/index.html
  (open with: npx playwright show-report)
```

失敗時：
- 報告自動截圖 + 錄影（在 `test-results/` 底下）
- `npx playwright show-report` 開瀏覽器看 trace

---

## 常用情境

### 我只想跑一個 spec
```bash
cd e2e
./run-e2e.sh --no-stack    # 跳過啟動 stack（假設 stack 已在跑）
# 或
npx playwright test auth.spec.ts
```

### 我想 debug 看畫面
```bash
cd e2e
npm run e2e:ui   # UI mode，可以單步、看 DOM、回放
# 或
./run-e2e.sh --ui
```

### Stack 已經在跑了，只是想重新測
```bash
cd e2e
npx playwright test           # 直接跑，不重啟 stack
# 但記得：tests 之間共享 DB，每個 spec 的 beforeAll 會 reset
```

### 想完全清乾淨
```bash
cd e2e
./run-e2e.sh --down            # 跑完 + 拆 stack + 砍 volume
# 或單獨 down
npm run stack:down
```

### 想看後端在做什麼
```bash
cd e2e
npm run stack:logs             # tail -f 全部 service log
# 或單看 backend
docker compose -f ../docker/docker-compose.e2e.yml -p thai-e2e logs -f backend
```

### 手動操作 reset endpoint
```bash
curl -X POST http://localhost:3001/api/test/reset
# 回傳：
# {"status":"ok","adminEmail":"admin@admin.com","password":"Test1234!",...}
```

### 手動開瀏覽器看 E2E stack
直接開 `http://localhost:3001`，用 `admin@admin.com` / `Test1234!` 登入。
這個 stack 是給測試用的乾淨環境，亂玩不會影響 dev。

---

## 故障排除

### `docker info` 掛了 / Cannot connect to the Docker daemon
Docker Desktop 沒開。打開它，等 30 秒。

### `port is already allocated`
有其他東西佔用 55432 / 8081 / 3001。找出來殺：
```bash
lsof -i :8081
kill <PID>
```
或改 `docker/docker-compose.e2e.yml` 把 host port 換掉。

### `Backend never came up`
1. `npm run stack:logs` 看 backend log
2. 如果看到 `Failed to connect to database` → postgres 還沒起，等久一點重跑
3. 如果看到 panic / error → 貼上來查

### Playwright 報 `Target page, context or browser has been closed`
通常是 frontend 還沒準備好。在 selector 前加 `await page.waitForLoadState('networkidle')`。

### 測試 selector 找不到元素
這份 spec 是首版，selector 用 regex 對中文/英文/泰文。如果某個 button 文字改了，會跑不過。
- 暫時 workaround：用 `npx playwright test --debug auth.spec.ts` 開 inspector，重抓 selector
- 長期解：在前端關鍵元素加 `data-testid`，spec 改用 `getByTestId('xxx')`

### `ENABLE_TEST_RESET` 不起作用 / endpoint 404
代表 backend image 沒用 `Dockerfile.backend.e2e` build。腳本會強制 `--build`，但如果你手動操作 compose 忘了 `--build`，會用舊 image。重來：
```bash
npm run stack:down
./run-e2e.sh
```

---

## CI 整合（未做，留 reference）

如果之後要把 E2E 加進 GitHub Actions：

```yaml
# .github/workflows/e2e.yml（範例）
name: E2E
on: [push, pull_request]
jobs:
  e2e:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: 20
      - name: Run E2E
        run: cd e2e && ./run-e2e.sh --down
      - name: Upload report
        if: failure()
        uses: actions/upload-artifact@v4
        with:
          name: playwright-report
          path: e2e/playwright-report/
```

預估 CI 時間 +5 分鐘。

---

## 與 dev 環境的關係

| | dev stack (`docker-compose.yml`) | e2e stack (`docker-compose.e2e.yml`) |
|---|---|---|
| 觸發方式 | 手動 `docker compose up` | `./run-e2e.sh` |
| Backend build | 標準 (`go build`) | `go build -tags e2e` |
| Reset endpoint | 不存在 | 存在 |
| DB volume | `postgres_data` | `postgres_e2e_data`（獨立） |
| Uploads | `../backend/uploads`（host dir） | `uploads_e2e`（docker volume） |
| Host ports | 5432 / 8080 / 3000 | 55432 / 8081 / 3001 |
| 適合用途 | 日常開發 | 自動化測試 |

**兩個 stack 可同時跑**，互不干擾。
