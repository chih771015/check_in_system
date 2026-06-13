# Sync Docs — 功能異動的文件同步清單

每次「新增或修改功能」後，用這份清單把文件補齊。分兩類：

- **固定文件（Fixed）**：每次都要走一遍，不論改了什麼。
- **浮動文件（Floating）**：依你「實際改了哪些 code」相對對應 — 規則是機械式的：**每個你動到的 code 檔，找它同目錄（或最近上層）的 `*_SPEC.md` 一起改。**

> 本專案慣例：幾乎每個 code 目錄都放一份**同層 `*_SPEC.md`**。所以「浮動文件」不用猜，照下面對照表把你改到的目錄對到它的 spec 即可。

---

## 步驟

### 1. 先盤點你改了什麼
```bash
git status -s
git diff --stat HEAD
```
把改動分成：後端 model/dto/config/repo/service/handler/路由、前端 api/pages/components/i18n、基礎設施（docker/env/cron/tracing）。

### 2. 自我稽核：哪些既有文件提到了你動的領域？
用關鍵字反查，避免漏掉「描述舊行為而現在變不正確」的文件：
```bash
md() { find . -name "*.md" -not -path "*/node_modules/*" -not -path "*/.git/*" -not -path "*/changelogs/*"; }
grep -rl -i "<你改的功能關鍵字，如 diagnosis / 照片 / EMAIL_TAKEN / retention>" $(md)
```
逐一打開檢查：**內容是否因這次改動而過期？**（最常見：寫死的 HTTP code、預設值、行為描述、統計數字。）

### 3. 更新「固定文件」
| 文件 | 何時改 | 改什麼 |
|------|--------|--------|
| `changelogs/YYYY-MM-DD_描述.md` | **每次必加** | 由 `/staged-commit` 產生；commit/摘要/影響/測試/注意事項 |
| `PRODUCT_SPEC.md` | 對外可見行為 / API 契約有變 | 對應章節（如 §9 診斷、附錄資料保存表）|
| `USER_STORIES.md` | 新增/改變使用者可見行為 | 加 US 或在既有 US 補「驗收」；異常情境補「場景」|
| `TEST-CASES.md` | 行為可被測試 | 加 `TC-XXX-NNN`；**更新附錄統計數字**；修正過期的預期結果 |
| `TEST-PLAN.md` | 同上 | 加測試項目列；**更新附錄統計數字**；修正過期預期 |
| `e2e/E2E_SPEC.md` | 有動 `e2e/tests/*.spec.ts` | 更新 §5 spec 清單與流程描述 |

### 4. 更新「浮動文件」— 依改動相對對應
你改到左欄的 code，就改右欄的 spec：

| 你改的 code 區域 | 對應 spec（同層）|
|------------------|------------------|
| `backend/internal/model/` | `MODEL_SPEC.md` |
| `backend/internal/config/` | `CONFIG_SPEC.md` |
| `backend/internal/dto/` | `DTO_SPEC.md` |
| `backend/internal/middleware/` | `MIDDLEWARE_SPEC.md` |
| `backend/internal/repository/` | `REPOSITORY_SPEC.md` |
| `backend/internal/service/` | `SERVICE_SPEC.md` + 對應重型（`AUTH_/CHECKIN_/SCHEDULE_/DIAGNOSIS_SERVICE_SPEC.md`）|
| `backend/internal/handler/` | `HANDLER_SPEC.md` |
| `backend/cmd/server/` | `SERVER_SPEC.md`（路由 / cron / wiring）|
| `backend/internal/tracing/` | `TRACING_SPEC.md` |
| `frontend/src/api/` | `API_CLIENT_SPEC.md` |
| `frontend/src/components/` | `COMPONENTS_SPEC.md` |
| `frontend/src/pages/` | `PAGES_SPEC.md` |
| `frontend/src/hooks/` | `HOOKS_SPEC.md` |
| `frontend/src/stores/` | `AUTH_STORE_SPEC.md` |
| `frontend/src/i18n/` | `I18N_SPEC.md` |
| `docker/` | `DOCKER_SPEC.md` |

### 5. 跨切面觸發規則（某種改動 → 一定要連帶改一串）
| 觸發 | 連帶必改（缺一就會出 bug 或文件失真）|
|------|------|
| **新增/改 error code** | `dto/error.go` 常數 → service sentinel → `handler/error_mapper.go` case → **三語 i18n（en/zh-TW/th）** `errors.<CODE>`。（`DTO_SPEC.md §3` 有同款 checklist；i18n 測試會強制三語 key 一致）|
| **新增 API 端點** | `HANDLER_SPEC` + `SERVER_SPEC`(路由) + `PRODUCT_SPEC` + 前端 `API_CLIENT_SPEC` + 前端對應 `api/<domain>.ts` + `TEST-PLAN`/`TEST-CASES` + `e2e`。注意 gin wildcard 衝突（`:id` 與靜態段同層）|
| **新增環境變數** | `CONFIG_SPEC` + `backend/.env.example` + `backend/.env.production.example` + `DEPLOYMENT_SPEC` + `docs/PRODUCTION_DEPLOY.md` |
| **新增/改 cron 或啟動行為** | `SERVER_SPEC §5` |
| **狀態機 / 狀態轉換改變** | 對應 service 的 `*_SERVICE_SPEC.md` 狀態圖（mermaid）|
| **新 repository** | 必含 `WithCtx`（見 `CLAUDE.md`）；`REPOSITORY_SPEC` 加契約 |
| **新增前端 i18n key** | 必同步加到 `en/zh-TW/th` 三檔（i18n 測試強制一致）|
| **改部署 / 容器** | `DEPLOYMENT_SPEC` + `docs/PRODUCTION_DEPLOY.md` + `docker/DOCKER_SPEC.md` |
| **新增「標記未到 / no_show」入口** | `MarkNoShow` 會**清空該 slot 既有照片**。若新入口的 no-show 按鈕**可達 completed slot**（completed 才有照片），必須包一層 `diagnosis.noShowClearsPhotosConfirm` 的 Popconfirm 提示，避免誤點清掉照片。只對非 completed（pending/no_show，無照片）顯示則免。現有入口：DiagnosisResults（有提示）/ ScheduleManagement、MySchedules（僅非 completed，免）|
| **新增「破壞性 / 不可復原」操作入口** | 一般原則：刪除、清空、覆寫等不可逆動作，UI 入口應加 Popconfirm（並在文案點出「會刪掉什麼」），後端維持最終守衛 |

### 6. 收尾檢查
- [ ] 固定文件 5 份都看過（沒變動的可不動，但要確認沒過期）
- [ ] 改到的每個 code 目錄，對應 spec 都同步了
- [ ] 跨切面規則命中的，整串都補齊
- [ ] 測試統計數字有跟著加/改
- [ ] 過期的「預期結果 / 預設值 / HTTP code」已修正（不是只加新內容）
- [ ] 走 `/staged-commit`，文件改動放在「文件層」commit，並產出 changelog

---

## 備註：怎麼判斷「固定 vs 浮動」
- **固定**：描述「系統整體做什麼 / 使用者怎麼用 / 要測什麼」——與單一 code 目錄無關，所以每次都看。
- **浮動**：描述「某層 code 怎麼實作」——和你動到的目錄一一對應，沒動到就不用看。
- 拿不準時，用步驟 2 的 `grep -rl` 反查，讓關鍵字告訴你哪些文件牽涉到。
