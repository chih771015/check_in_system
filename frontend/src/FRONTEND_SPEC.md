# frontend — 規格（overview）

> 對應檔案：`frontend/src/**`
> 上層：[ARCHITECTURE_SPEC.md](../../ARCHITECTURE_SPEC.md)

## 1. 定位與職責
React + TypeScript + Vite + Ant Design v5 SPA。手機優先（翻譯員主要用手機）。透過 `/api`（相對路徑，由 nginx/Vite proxy 轉發）打後端。

## 2. 結構地圖
| 區塊 | 路徑 | 規格 |
|------|------|------|
| 進入點 / 路由 / guards | `main.tsx`、`App.tsx` | 本檔 §3 |
| 認證狀態 | `stores/authStore.tsx` | [AUTH_STORE_SPEC](stores/AUTH_STORE_SPEC.md) |
| API 層 | `api/client.ts` + `api/*.ts` | [API_CLIENT_SPEC](api/API_CLIENT_SPEC.md) |
| 頁面 | `pages/(admin|translator)/*` | [PAGES_SPEC](pages/PAGES_SPEC.md) |
| 共用元件 | `components/*` | [COMPONENTS_SPEC](components/COMPONENTS_SPEC.md) |
| 定位 hook | `hooks/useGeolocation.ts` | [HOOKS_SPEC ★](hooks/HOOKS_SPEC.md) |
| 國際化 | `i18n/*` | [I18N_SPEC](i18n/I18N_SPEC.md) |
| 型別 | `types/index.ts` | 本檔 §4 |
| 工具 | `utils/apiError.ts`、`utils/schedulePatient.ts` | 本檔 §5 |

## 3. 路由與守衛（App.tsx）
- 三層 guard：`RequireAuth`（無 token/user → /login；must_change_pw → /change-password）、`RequireAdmin`、`RequireTranslator`（角色不符互導）。
- `AppLayout` 包所有受保護頁；`DefaultRedirect` 處理 `*`（admin→/admin/translators，translator→/my-schedules）。
- Ant Design locale 隨 i18n 語言切換（en/zh-TW/th）。
- 完整路由表見 [PRODUCT_SPEC §15](../../PRODUCT_SPEC.md)。

## 4. 型別契約（types/index.ts）
前端 interface 必須對齊後端 [dto](../../backend/internal/dto/DTO_SPEC.md) 的 json tag：
- camelCase 系列：Patient、SchedulePatient、DiagnosisResult、ScheduleItem.patients…
- snake/camel 混用是後端遺留，前端以實際 json 為準。
- `IDType = 'passport' | 'hn' | 'unid'`、`SchedulePatientStatus = 'pending'|'completed'|'no_show'`、`checkinStatus = 'none'|'arrived'|'completed'|'makeup'`。

## 5. 工具
- `utils/apiError.ts`：`extractApiError(err)` 取出 `translatedMessage`/後端 message，給 toast 顯示。
- `utils/schedulePatient.ts`：`clampPatientTimes(...)` 把病人時段夾進整體時段（與後端 `PATIENT_TIME_OUT_OF_RANGE` 對應，前端先擋）。

## 6. 不變式
| 不變式 | 保證 |
|--------|------|
| token 存 localStorage(`token`)、user 存 `user` | 人工維持（authStore 與 client interceptor 都讀寫這兩個 key）|
| 所有 API 經 `api/client`（帶 token、unwrap、error→i18n）| 人工維持 |
| error code 對得到 `errors.<CODE>` 翻譯 | 人工維持（後端新增 code 要同步三語）|

## 7. 測試考量
vitest + @testing-library/react；既有測試涵蓋 Login、AdminManagement、ChangePassword、各 component、authStore、apiError、schedulePatient、i18n。E2E 見 `e2e/`。
