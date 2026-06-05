# stores/authStore — 規格（輕量）

> 對應檔案：`frontend/src/stores/authStore.tsx`
> 上層：[FRONTEND_SPEC](../FRONTEND_SPEC.md)

## 1. 定位與職責
以 React Context 持有登入狀態（user / token），並與 localStorage 同步。**不做** API 呼叫（登入請求在 Login 頁，成功後呼叫 `login()`）。

## 2. 對外契約（useAuth）
| 名稱 | 說明 |
|------|------|
| `user / token` | 目前登入者；初始值從 localStorage 還原 |
| `login(token, user)` | 寫入 state + localStorage(`token`,`user`) |
| `logout()` | 清 state + localStorage |
| `updateUser(partial)` | 局部更新 user（如改密碼後 must_change_pw=false）並回寫 localStorage |
| `isAdmin / isTranslator` | 由 `user.role` 推導 |

`useAuth` 必須在 `AuthProvider` 內使用，否則 throw。

## 3. 不變式
| 不變式 | 保證 |
|--------|------|
| localStorage key 用 `token` / `user` | 人工維持（[api client](../api/API_CLIENT_SPEC.md) interceptor 直接讀同 key）|
| user JSON 解析失敗 → 視為未登入 | 機制保證（`loadUser` try/catch 回 null）|

## 4. 邊界條件
- 初次載入：從 localStorage 還原，支援重整後保持登入。
- token 與 user 兩者**獨立存**：理論上可能只剩其一；guard（RequireAuth）要求兩者都在才算登入。

## 5. 已知技術債
- token 過期 store 無感知，靠 API 401 才觸發登出（見 api client）。
- 與 client interceptor 各自讀寫 localStorage，登出邏輯分散兩處。

## 6. 測試考量
`stores/__tests__/authStore.test.tsx`：login/logout/updateUser、localStorage 還原、Provider 外使用報錯。
