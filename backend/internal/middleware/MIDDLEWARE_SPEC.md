# middleware — 規格（輕量）

> 對應檔案：`backend/internal/middleware/auth.go`
> 上層：[ARCHITECTURE_SPEC.md](../../../ARCHITECTURE_SPEC.md)

## 1. 定位與職責
HTTP 進入點的認證/授權守衛 + JWT 簽發/解析工具。**不做**商業邏輯（不查使用者狀態，只看 token claim）。

## 2. 對外契約
| 函式 / middleware | 作用 |
|------|------|
| `GenerateToken(userID, role, mustChangePW)` | 簽 HS256 JWT；過期時數 = `config.JWTExpiryHrs` |
| `ParseToken(tokenString)` | 驗章 + 取 (userID, role, mustChangePW)；拒絕非 HMAC 簽法 |
| `JWTAuth()` | 解 `Authorization: Bearer <token>`，set `userID/userRole/mustChangePW` 進 context；缺/壞 → 401 abort |
| `RequirePasswordChanged()` | token 仍帶 must_change_pw → 403 `PASSWORD_CHANGE_REQUIRED` abort |
| `RoleRequired(roles...)` | role 不在白名單 → 403 abort |

Claims 結構：`{ user_id, role, must_change_pw } + RegisteredClaims(exp, iat)`。

## 3. 套用順序（main.go route group）
```
admin 群組:        JWTAuth → RequirePasswordChanged → RoleRequired("admin")
translator 群組:   JWTAuth → RequirePasswordChanged → RoleRequired("translator")
change-password:   JWTAuth 　(故意不套 RequirePasswordChanged，否則首登無法改密碼)
login:             無 (public)
```

## 4. 不變式
| 不變式 | 保證 |
|--------|------|
| change-password 端點不套 RequirePasswordChanged | 人工維持（套了會死鎖：必須改密碼卻不能呼叫改密碼）|
| 受保護端點皆先 JWTAuth 再取 userID | 人工維持（handler `c.GetUint("userID")` 仰賴此）|
| 簽法限 HMAC | 機制保證（ParseToken 檢查 method，擋 alg=none/RS 混淆）|

## 5. 邊界條件
| 情境 | 行為 |
|------|------|
| 無 Authorization header | 401 "Authorization header is required" |
| 非 `Bearer x` 格式 | 401 |
| token 過期/簽章錯 | 401 "Invalid or expired token" |
| token 無 must_change_pw（理論上不會）| RequirePasswordChanged 放行 |
| role 不符 | 403 "Insufficient permissions" |

> 注意：JWTAuth/RoleRequired 的錯誤是 `{"error": "..."}` 原始格式，**非** dto.ErrorResponse 信封；只有 RequirePasswordChanged 回 `{code, error}`。前端 client 對 401/403+PASSWORD_CHANGE_REQUIRED 有特別處理（見 [api client](../../../frontend/src/api/API_CLIENT_SPEC.md)）。

## 6. 已知技術債
- middleware 錯誤格式與 dto.ErrorResponse 不一致（混用 `error` 與 `code`）。
- token 無法撤銷（無黑名單）；停用帳號者在 token 過期前仍可呼叫 API（除非端點另查 status）。

## 7. 測試考量
`auth_test.go`：token 簽/解、過期、role gate、must_change_pw gate。
