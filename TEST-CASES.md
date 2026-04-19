# 翻譯員打卡系統 — 完整測試案例

> **版本**：v1.0  
> **撰寫日期**：2026-04-19  
> **適用範圍**：後端 API + 前端 UI + 基礎設施  
> **測試環境**：Docker Compose（PostgreSQL + Jaeger + Backend + Frontend）

---

## 目錄

1. [測試環境與前置準備](#1-測試環境與前置準備)
2. [TC-AUTH：認證模組](#2-tc-auth認證模組)
3. [TC-TM：翻譯員管理](#3-tc-tm翻譯員管理)
4. [TC-SCH：排班管理](#4-tc-sch排班管理)
5. [TC-CK：打卡功能](#5-tc-ck打卡功能)
6. [TC-EXP：匯出功能](#6-tc-exp匯出功能)
7. [TC-AUD：稽核紀錄](#7-tc-aud稽核紀錄)
8. [TC-MW：中介層與權限控制](#8-tc-mw中介層與權限控制)
9. [TC-TRACE：OpenTelemetry / Jaeger 追蹤](#9-tc-traceopentelemetry--jaeger-追蹤)
10. [TC-CRON：排程任務](#10-tc-cron排程任務)
11. [TC-SEC：安全性測試](#11-tc-sec安全性測試)
12. [TC-E2E：端對端流程](#12-tc-e2e端對端流程)
13. [TC-UI：前端 UI 測試](#13-tc-ui前端-ui-測試)

---

## 1. 測試環境與前置準備

### 1.1 環境啟動

```bash
docker compose -f docker/docker-compose.yml up -d --build
```

### 1.2 測試資料

| 代號 | 角色 | email | 密碼 | 說明 |
|------|------|-------|------|------|
| ADMIN_1 | admin | admin@admin.com | （初始密碼或 env 設定） | 系統預設管理員 |
| TRANS_1 | translator | trans1@test.com | Test1234 | 測試翻譯員 A |
| TRANS_2 | translator | trans2@test.com | Test5678 | 測試翻譯員 B |

### 1.3 共用變數

```
BASE_URL = http://localhost:8080
ADMIN_TOKEN = （登入後取得）
TRANS_TOKEN = （登入後取得）
```

---

## 2. TC-AUTH：認證模組

### TC-AUTH-001：正常登入

| 項目 | 內容 |
|------|------|
| **ID** | TC-AUTH-001 |
| **名稱** | 管理員以正確帳密登入 |
| **前置條件** | ADMIN_1 帳號存在，status=active |
| **測試步驟** | 1. `POST /api/auth/login`<br>2. Body: `{"email":"admin@admin.com","password":"正確密碼"}` |
| **預期結果** | 1. HTTP 200<br>2. 回傳 `token`（非空 JWT 字串）<br>3. 回傳 `user.id`、`user.email`、`user.role="admin"`、`user.status="active"` |
| **驗證重點** | token 可被成功解碼，claims 中 userID/role 正確 |

### TC-AUTH-002：密碼錯誤

| 項目 | 內容 |
|------|------|
| **ID** | TC-AUTH-002 |
| **名稱** | 以錯誤密碼登入 |
| **前置條件** | ADMIN_1 帳號存在 |
| **測試步驟** | 1. `POST /api/auth/login`<br>2. Body: `{"email":"admin@admin.com","password":"wrongpassword"}` |
| **預期結果** | 1. HTTP 401<br>2. `{"error":"invalid email or password"}` |

### TC-AUTH-003：不存在的帳號

| 項目 | 內容 |
|------|------|
| **ID** | TC-AUTH-003 |
| **名稱** | 以不存在的 email 登入 |
| **前置條件** | 無 |
| **測試步驟** | 1. `POST /api/auth/login`<br>2. Body: `{"email":"nobody@test.com","password":"whatever"}` |
| **預期結果** | 1. HTTP 401<br>2. `{"error":"invalid email or password"}` |

### TC-AUTH-004：email 格式不合法

| 項目 | 內容 |
|------|------|
| **ID** | TC-AUTH-004 |
| **名稱** | email 欄位不是合法 email 格式 |
| **前置條件** | 無 |
| **測試步驟** | 1. `POST /api/auth/login`<br>2. Body: `{"email":"not-an-email","password":"123456"}` |
| **預期結果** | 1. HTTP 400<br>2. 回傳 binding validation 錯誤訊息 |

### TC-AUTH-005：空白必填欄位

| 項目 | 內容 |
|------|------|
| **ID** | TC-AUTH-005 |
| **名稱** | email 或 password 為空 |
| **前置條件** | 無 |
| **測試步驟** | 1. Body: `{"email":"","password":"123456"}` → 預期 400<br>2. Body: `{"email":"a@b.com","password":""}` → 預期 400<br>3. Body: `{}` → 預期 400 |
| **預期結果** | 全部回傳 HTTP 400 + binding 驗證錯誤 |

### TC-AUTH-006：請求體非 JSON

| 項目 | 內容 |
|------|------|
| **ID** | TC-AUTH-006 |
| **名稱** | Content-Type 非 JSON 或 body 非法 |
| **前置條件** | 無 |
| **測試步驟** | 1. `POST /api/auth/login`，body 為純文字 `"hello"`<br>2. 不帶 Content-Type |
| **預期結果** | HTTP 400 |

### TC-AUTH-007：已停用帳號登入

| 項目 | 內容 |
|------|------|
| **ID** | TC-AUTH-007 |
| **名稱** | status=disabled 的使用者嘗試登入 |
| **前置條件** | TRANS_1 的 status 已被管理員設為 disabled |
| **測試步驟** | 1. `POST /api/auth/login`<br>2. Body: `{"email":"trans1@test.com","password":"Test1234"}` |
| **預期結果** | 1. HTTP 401<br>2. `{"error":"account is disabled"}` |

### TC-AUTH-008：帳號鎖定 — 連續失敗觸發

| 項目 | 內容 |
|------|------|
| **ID** | TC-AUTH-008 |
| **名稱** | 連續 5 次錯誤密碼後帳號被鎖定 |
| **前置條件** | TRANS_1 帳號 active，LoginAttempts=0 |
| **測試步驟** | 1. 連續送 5 次 `POST /api/auth/login`，密碼皆錯誤<br>2. 第 6 次送正確密碼 |
| **預期結果** | 1. 前 5 次皆回傳 401 `"invalid email or password"`<br>2. 第 6 次回傳 401 `"account locked, try again in XXs"`（XX 為剩餘秒數） |
| **驗證重點** | 剩餘秒數 ≤ 900（15 分鐘 × 60） |

### TC-AUTH-009：帳號鎖定 — 鎖定期間正確密碼仍被擋

| 項目 | 內容 |
|------|------|
| **ID** | TC-AUTH-009 |
| **名稱** | 鎖定期間即使密碼正確也無法登入 |
| **前置條件** | TC-AUTH-008 執行完畢，帳號處於鎖定狀態 |
| **測試步驟** | 1. 立即送 `POST /api/auth/login`，使用正確密碼 |
| **預期結果** | HTTP 401，`"account locked, try again in XXs"` |

### TC-AUTH-010：帳號鎖定 — 到期後恢復

| 項目 | 內容 |
|------|------|
| **ID** | TC-AUTH-010 |
| **名稱** | 鎖定到期後可正常登入 |
| **前置條件** | 帳號鎖定中（可將 LOCK_DURATION_MINUTES 設為 1 加速測試）|
| **測試步驟** | 1. 等待鎖定到期<br>2. 送 `POST /api/auth/login`，使用正確密碼 |
| **預期結果** | HTTP 200，登入成功，LoginAttempts 被重設為 0 |

### TC-AUTH-011：失敗 4 次後成功登入重設計數器

| 項目 | 內容 |
|------|------|
| **ID** | TC-AUTH-011 |
| **名稱** | 未達鎖定門檻時成功登入重設計數器 |
| **前置條件** | TRANS_1 帳號 active |
| **測試步驟** | 1. 連續 4 次錯誤密碼（未達 5 次門檻）<br>2. 第 5 次用正確密碼登入 → 成功<br>3. 再連續 4 次錯誤密碼<br>4. 第 5 次用正確密碼登入 |
| **預期結果** | 步驟 2 和步驟 4 都回傳 200（計數器在步驟 2 被重設） |

### TC-AUTH-012：mustChangePW 使用者登入

| 項目 | 內容 |
|------|------|
| **ID** | TC-AUTH-012 |
| **名稱** | 新建翻譯員首次登入帶有 mustChangePW 標記 |
| **前置條件** | 剛由管理員建立的翻譯員，mustChangePW=true |
| **測試步驟** | 1. `POST /api/auth/login` 用正確帳密 |
| **預期結果** | 1. HTTP 200<br>2. `user.mustChangePW = true`<br>3. token 中 claims.mustChangePW = true |

### TC-AUTH-013：改密碼 — 正常流程

| 項目 | 內容 |
|------|------|
| **ID** | TC-AUTH-013 |
| **名稱** | 使用者以正確舊密碼修改為新密碼 |
| **前置條件** | 使用者已登入，持有合法 token |
| **測試步驟** | 1. `POST /api/auth/change-password`<br>2. Header: `Authorization: Bearer {TOKEN}`<br>3. Body: `{"oldPassword":"原密碼","newPassword":"NewPass123"}` |
| **預期結果** | 1. HTTP 200<br>2. `{"message":"Password changed successfully","token":"新JWT"}`<br>3. 新 token 中 mustChangePW=false |
| **後續驗證** | 用舊密碼登入 → 401；用新密碼登入 → 200 |

### TC-AUTH-014：改密碼 — 舊密碼錯誤

| 項目 | 內容 |
|------|------|
| **ID** | TC-AUTH-014 |
| **名稱** | 輸入錯誤的舊密碼 |
| **前置條件** | 使用者已登入 |
| **測試步驟** | 1. Body: `{"oldPassword":"wrong","newPassword":"NewPass123"}` |
| **預期結果** | HTTP 400，`{"error":"old password is incorrect"}` |

### TC-AUTH-015：改密碼 — 新密碼太短

| 項目 | 內容 |
|------|------|
| **ID** | TC-AUTH-015 |
| **名稱** | 新密碼少於 6 字元 |
| **前置條件** | 使用者已登入 |
| **測試步驟** | 1. Body: `{"oldPassword":"正確密碼","newPassword":"12345"}`（5 字元）|
| **預期結果** | HTTP 400，binding 驗證錯誤 |

### TC-AUTH-016：改密碼 — 新密碼恰好 6 字元

| 項目 | 內容 |
|------|------|
| **ID** | TC-AUTH-016 |
| **名稱** | 新密碼恰好 6 字元（邊界值） |
| **前置條件** | 使用者已登入 |
| **測試步驟** | 1. Body: `{"oldPassword":"正確密碼","newPassword":"123456"}` |
| **預期結果** | HTTP 200，改密碼成功 |

### TC-AUTH-017：改密碼 — 未帶 token

| 項目 | 內容 |
|------|------|
| **ID** | TC-AUTH-017 |
| **名稱** | 不帶 Authorization header 呼叫改密碼 |
| **前置條件** | 無 |
| **測試步驟** | 1. `POST /api/auth/change-password`，不帶 header |
| **預期結果** | HTTP 401，`{"error":"Authorization header is required"}` |

### TC-AUTH-018：改密碼後 mustChangePW 標記清除

| 項目 | 內容 |
|------|------|
| **ID** | TC-AUTH-018 |
| **名稱** | mustChangePW=true 的使用者改密碼後標記清除 |
| **前置條件** | 使用者 mustChangePW=true，已登入持有 token |
| **測試步驟** | 1. 改密碼成功<br>2. 用新 token 呼叫 `/api/admin/translators` 或 `/api/schedules` |
| **預期結果** | 1. 回傳新 token 中 mustChangePW=false<br>2. 步驟 2 不再被 RequirePasswordChanged 攔截 |

---

## 3. TC-TM：翻譯員管理

### TC-TM-001：列表 — 取得所有翻譯員

| 項目 | 內容 |
|------|------|
| **ID** | TC-TM-001 |
| **名稱** | 管理員取得翻譯員列表 |
| **前置條件** | 已存在 2 名翻譯員，ADMIN_TOKEN 有效 |
| **測試步驟** | 1. `GET /api/admin/translators`<br>2. Header: `Authorization: Bearer {ADMIN_TOKEN}` |
| **預期結果** | 1. HTTP 200<br>2. `data` 為陣列，每筆含 `id, email, name, phone, status, createdAt`<br>3. 不含 admin 帳號 |

### TC-TM-002：列表 — status 篩選

| 項目 | 內容 |
|------|------|
| **ID** | TC-TM-002 |
| **名稱** | 以 status 篩選翻譯員 |
| **前置條件** | 有 active 和 disabled 翻譯員各至少 1 名 |
| **測試步驟** | 1. `GET /api/admin/translators?status=active`<br>2. `GET /api/admin/translators?status=disabled` |
| **預期結果** | 1. 只回傳 status=active 的翻譯員<br>2. 只回傳 status=disabled 的翻譯員 |

### TC-TM-003：建立 — 正常流程

| 項目 | 內容 |
|------|------|
| **ID** | TC-TM-003 |
| **名稱** | 建立新翻譯員（所有必填欄位） |
| **前置條件** | ADMIN_TOKEN 有效 |
| **測試步驟** | 1. `POST /api/admin/translators`<br>2. Body: `{"email":"new@test.com","password":"Pass1234","name":"新翻譯員","phone":"0912345678"}` |
| **預期結果** | 1. HTTP 201<br>2. `{"message":"Translator created successfully"}`<br>3. 用 `GET /api/admin/translators` 可看到新帳號<br>4. DB 中 role="translator", status="active", mustChangePW=true |
| **稽核驗證** | audit_logs 有一筆 action="create_translator"，detail 含 email 和 name |

### TC-TM-004：建立 — 重複 email

| 項目 | 內容 |
|------|------|
| **ID** | TC-TM-004 |
| **名稱** | 用已存在的 email 建立翻譯員 |
| **前置條件** | trans1@test.com 已存在 |
| **測試步驟** | 1. Body: `{"email":"trans1@test.com","password":"Pass1234","name":"重複"}` |
| **預期結果** | HTTP 400，`{"error":"email already exists"}` |

### TC-TM-005：建立 — 密碼太短

| 項目 | 內容 |
|------|------|
| **ID** | TC-TM-005 |
| **名稱** | 密碼少於 6 字元 |
| **前置條件** | 無 |
| **測試步驟** | 1. Body: `{"email":"short@test.com","password":"12345","name":"短密碼"}` |
| **預期結果** | HTTP 400，binding 驗證錯誤 |

### TC-TM-006：建立 — 缺少必填欄位

| 項目 | 內容 |
|------|------|
| **ID** | TC-TM-006 |
| **名稱** | 分別缺少 email / password / name |
| **前置條件** | 無 |
| **測試步驟** | 1. 缺 email → 400<br>2. 缺 password → 400<br>3. 缺 name → 400 |
| **預期結果** | 全部 HTTP 400 |

### TC-TM-007：建立 — phone 選填不帶

| 項目 | 內容 |
|------|------|
| **ID** | TC-TM-007 |
| **名稱** | 不帶 phone 欄位建立翻譯員 |
| **前置條件** | 無 |
| **測試步驟** | 1. Body: `{"email":"nophone@test.com","password":"Pass1234","name":"沒手機"}` |
| **預期結果** | HTTP 201，phone 為空字串 |

### TC-TM-008：更新 — 部分欄位更新

| 項目 | 內容 |
|------|------|
| **ID** | TC-TM-008 |
| **名稱** | 只更新 name，其他欄位不變 |
| **前置條件** | TRANS_1 存在 |
| **測試步驟** | 1. `PUT /api/admin/translators/{TRANS_1_ID}`<br>2. Body: `{"name":"新名稱"}` |
| **預期結果** | 1. HTTP 200<br>2. 查詢後 name="新名稱"，phone/status 不變 |

### TC-TM-009：更新 — 無效 status 值

| 項目 | 內容 |
|------|------|
| **ID** | TC-TM-009 |
| **名稱** | status 設為非法值 |
| **前置條件** | TRANS_1 存在 |
| **測試步驟** | 1. Body: `{"status":"suspended"}` |
| **預期結果** | HTTP 400，`"status must be 'active' or 'disabled'"` |

### TC-TM-010：更新 — 對 admin 操作

| 項目 | 內容 |
|------|------|
| **ID** | TC-TM-010 |
| **名稱** | 試圖更新 admin 帳號 |
| **前置條件** | ADMIN_1 的 ID 已知 |
| **測試步驟** | 1. `PUT /api/admin/translators/{ADMIN_ID}`<br>2. Body: `{"name":"hack"}` |
| **預期結果** | HTTP 400，`{"error":"user is not a translator"}` |

### TC-TM-011：更新 — 不存在的 ID

| 項目 | 內容 |
|------|------|
| **ID** | TC-TM-011 |
| **名稱** | 更新不存在的翻譯員 |
| **前置條件** | 無 |
| **測試步驟** | 1. `PUT /api/admin/translators/99999` |
| **預期結果** | HTTP 400，`{"error":"translator not found"}` |

### TC-TM-012：停用翻譯員

| 項目 | 內容 |
|------|------|
| **ID** | TC-TM-012 |
| **名稱** | 停用一個 active 翻譯員 |
| **前置條件** | TRANS_1 status=active |
| **測試步驟** | 1. `DELETE /api/admin/translators/{TRANS_1_ID}` |
| **預期結果** | 1. HTTP 200，`"Translator disabled successfully"`<br>2. 查詢後 status=disabled<br>3. 該翻譯員嘗試登入 → 401 `"account is disabled"` |
| **稽核驗證** | audit_logs 有 action="disable_translator" |

### TC-TM-013：重設密碼 — 正常流程

| 項目 | 內容 |
|------|------|
| **ID** | TC-TM-013 |
| **名稱** | 管理員為翻譯員重設密碼 |
| **前置條件** | TRANS_1 存在，ADMIN_TOKEN 有效 |
| **測試步驟** | 1. `POST /api/admin/translators/{TRANS_1_ID}/reset-password`<br>2. Body: `{"newPassword":"NewPass88"}` |
| **預期結果** | 1. HTTP 200，`"Password reset successfully"`<br>2. TRANS_1 用舊密碼登入 → 401<br>3. TRANS_1 用 "NewPass88" 登入 → 200，mustChangePW=true |
| **稽核驗證** | audit_logs 有 action="reset_password" |

### TC-TM-014：重設密碼 — 管理員對自己

| 項目 | 內容 |
|------|------|
| **ID** | TC-TM-014 |
| **名稱** | 管理員試圖重設自己的密碼 |
| **前置條件** | ADMIN_TOKEN 有效 |
| **測試步驟** | 1. `POST /api/admin/translators/{ADMIN_ID}/reset-password`<br>2. Body: `{"newPassword":"SelfReset"}` |
| **預期結果** | HTTP 400，包含 "cannot reset your own password" 錯誤訊息 |

### TC-TM-015：重設密碼 — 密碼低於 8 字元

| 項目 | 內容 |
|------|------|
| **ID** | TC-TM-015 |
| **名稱** | 重設密碼時新密碼少於 8 字元 |
| **前置條件** | TRANS_1 存在 |
| **測試步驟** | 1. Body: `{"newPassword":"1234567"}`（7 字元）|
| **預期結果** | HTTP 400，binding 驗證錯誤 |
| **邊界測試** | `"newPassword":"12345678"`（8 字元）→ HTTP 200 成功 |

---

## 4. TC-SCH：排班管理

### TC-SCH-001：列表 — 無篩選

| 項目 | 內容 |
|------|------|
| **ID** | TC-SCH-001 |
| **名稱** | 管理員取得所有排班 |
| **前置條件** | 已建立數筆排班 |
| **測試步驟** | 1. `GET /api/admin/schedules` |
| **預期結果** | 1. HTTP 200<br>2. `data` 陣列每筆含：id, translatorId, translatorName, date, startTime, endTime, location, patientName, note, checkinStatus, recurrenceGroupId |

### TC-SCH-002：列表 — 日期區間篩選

| 項目 | 內容 |
|------|------|
| **ID** | TC-SCH-002 |
| **名稱** | 以 dateFrom + dateTo 篩選排班 |
| **前置條件** | 1/15、2/15、3/15 各有排班 |
| **測試步驟** | 1. `GET /api/admin/schedules?dateFrom=2026-02-01&dateTo=2026-02-28` |
| **預期結果** | 只回傳 2/15 的排班 |

### TC-SCH-003：列表 — 翻譯員 + 地點組合篩選

| 項目 | 內容 |
|------|------|
| **ID** | TC-SCH-003 |
| **名稱** | 同時用 translatorId 和 location 篩選 |
| **前置條件** | TRANS_1 和 TRANS_2 各有多筆排班，地點不同 |
| **測試步驟** | 1. `GET /api/admin/schedules?translatorId={TRANS_1_ID}&location=醫院` |
| **預期結果** | 只回傳 TRANS_1 且地點包含「醫院」的排班（ILIKE 模式匹配） |

### TC-SCH-004：建立 — 單次排班

| 項目 | 內容 |
|------|------|
| **ID** | TC-SCH-004 |
| **名稱** | 建立一筆單次排班 |
| **前置條件** | TRANS_1 存在 |
| **測試步驟** | 1. `POST /api/admin/schedules`<br>2. Body:<br>```json<br>{"translatorId":TRANS_1_ID,"date":"2026-05-01","startTime":"09:00","endTime":"12:00","location":"台北醫院","patientName":"王先生","note":"注意事項"}<br>``` |
| **預期結果** | 1. HTTP 201<br>2. `data` 含完整排班資訊<br>3. checkinStatus="none"<br>4. recurrenceGroupId 為 null |
| **稽核驗證** | audit_logs 有 action="create_schedule" |

### TC-SCH-005：建立 — 缺少必填欄位

| 項目 | 內容 |
|------|------|
| **ID** | TC-SCH-005 |
| **名稱** | 分別缺少各必填欄位 |
| **前置條件** | 無 |
| **測試步驟** | 逐一測試缺少：translatorId / date / startTime / endTime / location / patientName |
| **預期結果** | 全部回傳 HTTP 400 |

### TC-SCH-006：建立 — 日期格式錯誤

| 項目 | 內容 |
|------|------|
| **ID** | TC-SCH-006 |
| **名稱** | 日期非 YYYY-MM-DD 格式 |
| **前置條件** | 無 |
| **測試步驟** | 1. `"date":"01/15/2026"` → 400<br>2. `"date":"2026-13-01"` → 400<br>3. `"date":"abc"` → 400 |
| **預期結果** | 全部回傳 400，`"invalid date format, use YYYY-MM-DD"` |

### TC-SCH-007：建立 — translatorId 不存在

| 項目 | 內容 |
|------|------|
| **ID** | TC-SCH-007 |
| **名稱** | translatorId 指向不存在的使用者 |
| **前置條件** | 無 |
| **測試步驟** | 1. `"translatorId":99999` |
| **預期結果** | HTTP 400，`"translator not found"` |

### TC-SCH-008：建立 — translatorId 是 admin

| 項目 | 內容 |
|------|------|
| **ID** | TC-SCH-008 |
| **名稱** | translatorId 指向 admin 帳號 |
| **前置條件** | ADMIN_1 的 ID 已知 |
| **測試步驟** | 1. `"translatorId":ADMIN_ID` |
| **預期結果** | HTTP 400，`"user is not a translator"` |

### TC-SCH-009：建立 — 重複排班（daily）

| 項目 | 內容 |
|------|------|
| **ID** | TC-SCH-009 |
| **名稱** | 建立 daily 重複排班 |
| **前置條件** | TRANS_1 存在 |
| **測試步驟** | 1. Body:<br>```json<br>{"translatorId":TRANS_1_ID,"date":"2026-05-01","startTime":"09:00","endTime":"12:00","location":"A","patientName":"B","recurrenceRule":"daily","recurrenceUntil":"2026-05-05"}<br>``` |
| **預期結果** | 1. HTTP 201<br>2. DB 中產生 5 筆排班（5/1~5/5）<br>3. 所有排班共用同一個 recurrenceGroupId（UUID）|

### TC-SCH-010：建立 — 重複排班（weekly）

| 項目 | 內容 |
|------|------|
| **ID** | TC-SCH-010 |
| **名稱** | 建立 weekly:1,3,5（週一三五）重複排班 |
| **前置條件** | TRANS_1 存在 |
| **測試步驟** | 1. `"recurrenceRule":"weekly:1,3,5","date":"2026-05-01","recurrenceUntil":"2026-05-31"` |
| **預期結果** | 1. HTTP 201<br>2. 只在 5 月的週一三五產生排班<br>3. 驗證 DB 中每筆的 Weekday 為 Monday/Wednesday/Friday |

### TC-SCH-011：建立 — 重複排班（monthly 日期 clamping）

| 項目 | 內容 |
|------|------|
| **ID** | TC-SCH-011 |
| **名稱** | monthly:31 在短月份的日期自動 clamp |
| **前置條件** | TRANS_1 存在 |
| **測試步驟** | 1. `"recurrenceRule":"monthly:31","date":"2026-01-01","recurrenceUntil":"2026-04-30"` |
| **預期結果** | 1. 1 月 → 1/31<br>2. 2 月 → 2/28（2026 非閏年）<br>3. 3 月 → 3/31<br>4. 4 月 → 4/30<br>5. 共 4 筆，無重複 |

### TC-SCH-012：建立 — monthly:29,30,31 在 2 月不重複

| 項目 | 內容 |
|------|------|
| **ID** | TC-SCH-012 |
| **名稱** | 多個日期 clamp 到同一天時不重複 |
| **前置條件** | 無 |
| **測試步驟** | 1. `"recurrenceRule":"monthly:29,30,31","date":"2026-02-01","recurrenceUntil":"2026-02-28"` |
| **預期結果** | 只產生 1 筆（2/28），不重複 |

### TC-SCH-013：建立 — 有 recurrenceRule 但沒 recurrenceUntil

| 項目 | 內容 |
|------|------|
| **ID** | TC-SCH-013 |
| **名稱** | 設定重複規則但缺少結束日期 |
| **前置條件** | 無 |
| **測試步驟** | 1. `"recurrenceRule":"daily"` 但不帶 recurrenceUntil |
| **預期結果** | HTTP 400，`"recurrenceUntil is required when recurrenceRule is set"` |

### TC-SCH-014：建立 — recurrenceUntil 早於 date

| 項目 | 內容 |
|------|------|
| **ID** | TC-SCH-014 |
| **名稱** | 結束日期早於開始日期 |
| **前置條件** | 無 |
| **測試步驟** | 1. `"date":"2026-05-10","recurrenceRule":"daily","recurrenceUntil":"2026-05-01"` |
| **預期結果** | HTTP 400，`"recurrenceUntil must be after or equal to date"` |

### TC-SCH-015：建立 — 未知 recurrenceRule

| 項目 | 內容 |
|------|------|
| **ID** | TC-SCH-015 |
| **名稱** | 使用不支援的重複規則 |
| **前置條件** | 無 |
| **測試步驟** | 1. `"recurrenceRule":"yearly:1"` |
| **預期結果** | HTTP 400，包含 `"unknown rule"` |

### TC-SCH-016：建立 — weekday 值超出範圍

| 項目 | 內容 |
|------|------|
| **ID** | TC-SCH-016 |
| **名稱** | weekly 規則中 weekday 值不在 0-6 |
| **前置條件** | 無 |
| **測試步驟** | 1. `"recurrenceRule":"weekly:7"` → 400<br>2. `"recurrenceRule":"weekly:-1"` → 400 |
| **預期結果** | HTTP 400，`"weekday values must be 0-6"` |

### TC-SCH-017：更新排班

| 項目 | 內容 |
|------|------|
| **ID** | TC-SCH-017 |
| **名稱** | 更新排班的各個欄位 |
| **前置條件** | 已建立排班 SCH_ID |
| **測試步驟** | 1. `PUT /api/admin/schedules/{SCH_ID}`<br>2. Body: `{"date":"2026-06-01","location":"新地點"}` |
| **預期結果** | 1. HTTP 200<br>2. data 中 date 和 location 已更新<br>3. 其他欄位不變 |

### TC-SCH-018：刪除單筆排班

| 項目 | 內容 |
|------|------|
| **ID** | TC-SCH-018 |
| **名稱** | 刪除一筆排班 |
| **前置條件** | SCH_ID 存在 |
| **測試步驟** | 1. `DELETE /api/admin/schedules/{SCH_ID}` |
| **預期結果** | 1. HTTP 200，`"Schedule deleted successfully"`<br>2. 再查詢該 ID → 找不到 |
| **稽核驗證** | audit_logs 有 action="delete_schedule" |

### TC-SCH-019：刪除整組重複排班

| 項目 | 內容 |
|------|------|
| **ID** | TC-SCH-019 |
| **名稱** | 對重複排班中的一筆執行整組刪除 |
| **前置條件** | TC-SCH-009 建立的 5 筆 daily 排班 |
| **測試步驟** | 1. 取得其中任一筆的 ID<br>2. `DELETE /api/admin/schedules/{任一ID}/group` |
| **預期結果** | 1. HTTP 200，`"deleted":5`<br>2. 同 recurrenceGroupId 的所有排班全被刪除 |

### TC-SCH-020：刪除整組 — 非重複排班 fallback

| 項目 | 內容 |
|------|------|
| **ID** | TC-SCH-020 |
| **名稱** | 對無 recurrenceGroupId 的排班執行 /group 刪除 |
| **前置條件** | 單次排班 SCH_SINGLE 存在 |
| **測試步驟** | 1. `DELETE /api/admin/schedules/{SCH_SINGLE}/group` |
| **預期結果** | HTTP 200，`"deleted":1`（只刪單筆）|

### TC-SCH-021：Excel 匯入 — 全部成功

| 項目 | 內容 |
|------|------|
| **ID** | TC-SCH-021 |
| **名稱** | 上傳合法 Excel 全部成功匯入 |
| **前置條件** | Excel 檔案含 3 行資料，translatorId 皆合法 |
| **測試步驟** | 1. `POST /api/admin/schedules/import`<br>2. multipart/form-data，field name=`file` |
| **預期結果** | HTTP 200，`{"success":3,"failed":0,"total":3}` |
| **稽核驗證** | audit_logs 有 action="import_schedules" |

### TC-SCH-022：Excel 匯入 — 部分失敗

| 項目 | 內容 |
|------|------|
| **ID** | TC-SCH-022 |
| **名稱** | Excel 含合法與不合法行 |
| **前置條件** | 5 行資料，其中 2 行 translatorId 不存在 |
| **測試步驟** | 1. 上傳 Excel |
| **預期結果** | HTTP 200，`{"success":3,"failed":2,"total":5}` |

### TC-SCH-023：Excel 匯入 — 未上傳檔案

| 項目 | 內容 |
|------|------|
| **ID** | TC-SCH-023 |
| **名稱** | 不帶檔案呼叫匯入 endpoint |
| **前置條件** | 無 |
| **測試步驟** | 1. `POST /api/admin/schedules/import` 空 body |
| **預期結果** | HTTP 400，`"file is required"` |

### TC-SCH-024：翻譯員查看自己的排班

| 項目 | 內容 |
|------|------|
| **ID** | TC-SCH-024 |
| **名稱** | 翻譯員只能看到自己的排班 |
| **前置條件** | TRANS_1 和 TRANS_2 各有排班 |
| **測試步驟** | 1. 用 TRANS_1 的 token 呼叫 `GET /api/schedules` |
| **預期結果** | 1. HTTP 200<br>2. 全部排班的 translatorId 皆為 TRANS_1_ID<br>3. 看不到 TRANS_2 的排班 |

---

## 5. TC-CK：打卡功能

### TC-CK-001：到達打卡 — 正常流程

| 項目 | 內容 |
|------|------|
| **ID** | TC-CK-001 |
| **名稱** | 翻譯員正常到達打卡 |
| **前置條件** | TRANS_1 有今日排班 SCH_ID |
| **測試步驟** | 1. `POST /api/checkins`<br>2. multipart/form-data:<br>- scheduleId: SCH_ID<br>- type: arrive<br>- latitude: 25.0330<br>- longitude: 121.5654<br>- address: 台北市<br>- selfie: (image file)<br>- environment: (image file) |
| **預期結果** | 1. HTTP 201<br>2. data.type="arrive"<br>3. data.isMakeup=false<br>4. data.selfieUrl 為 `/uploads/selfie_...` 格式<br>5. data.environmentUrl 為 `/uploads/environment_...` 格式<br>6. data.translatorName 為 TRANS_1 的名字 |

### TC-CK-002：離開打卡 — 正常流程

| 項目 | 內容 |
|------|------|
| **ID** | TC-CK-002 |
| **名稱** | 已到達後正常離開打卡 |
| **前置條件** | TC-CK-001 已執行（已有 arrive 紀錄）|
| **測試步驟** | 1. `POST /api/checkins`，type=leave |
| **預期結果** | HTTP 201，data.type="leave" |

### TC-CK-003：未到達就離開

| 項目 | 內容 |
|------|------|
| **ID** | TC-CK-003 |
| **名稱** | 未先到達就嘗試離開打卡 |
| **前置條件** | SCH_ID 無任何打卡紀錄 |
| **測試步驟** | 1. `POST /api/checkins`，type=leave |
| **預期結果** | HTTP 400，`"must check in (arrive) before checking out (leave)"` |

### TC-CK-004：重複到達打卡

| 項目 | 內容 |
|------|------|
| **ID** | TC-CK-004 |
| **名稱** | 同一排班重複到達打卡 |
| **前置條件** | 已有 arrive 紀錄 |
| **測試步驟** | 1. 再次送 type=arrive |
| **預期結果** | HTTP 400，`"already checked in with type: arrive"` |

### TC-CK-005：重複離開打卡

| 項目 | 內容 |
|------|------|
| **ID** | TC-CK-005 |
| **名稱** | 同一排班重複離開打卡 |
| **前置條件** | 已有 arrive + leave 紀錄 |
| **測試步驟** | 1. 再次送 type=leave |
| **預期結果** | HTTP 400，`"already checked in with type: leave"` |

### TC-CK-006：排班不屬於該翻譯員

| 項目 | 內容 |
|------|------|
| **ID** | TC-CK-006 |
| **名稱** | 嘗試打別人的排班 |
| **前置條件** | SCH_ID 屬於 TRANS_2 |
| **測試步驟** | 1. 用 TRANS_1 的 token 對 SCH_ID 打卡 |
| **預期結果** | HTTP 400，`"schedule does not belong to this translator"` |

### TC-CK-007：排班不存在

| 項目 | 內容 |
|------|------|
| **ID** | TC-CK-007 |
| **名稱** | scheduleId 不存在 |
| **前置條件** | 無 |
| **測試步驟** | 1. scheduleId: 99999 |
| **預期結果** | HTTP 400，`"schedule not found"` |

### TC-CK-008：缺少 selfie 照片

| 項目 | 內容 |
|------|------|
| **ID** | TC-CK-008 |
| **名稱** | 不上傳 selfie 照片 |
| **前置條件** | 無 |
| **測試步驟** | 1. multipart 中只有 environment，無 selfie |
| **預期結果** | HTTP 400，錯誤訊息包含 "Selfie photo is required" |

### TC-CK-009：缺少 environment 照片

| 項目 | 內容 |
|------|------|
| **ID** | TC-CK-009 |
| **名稱** | 不上傳 environment 照片 |
| **前置條件** | 無 |
| **測試步驟** | 1. multipart 中只有 selfie，無 environment |
| **預期結果** | HTTP 400，錯誤訊息包含 "Environment photo is required" |

### TC-CK-010：反向地理編碼

| 項目 | 內容 |
|------|------|
| **ID** | TC-CK-010 |
| **名稱** | 不帶 address 但帶 GPS 座標，自動反向地理編碼 |
| **前置條件** | GeocodingService 可用 |
| **測試步驟** | 1. 不帶 address，帶 latitude/longitude |
| **預期結果** | 1. HTTP 201<br>2. data.address 被自動填入（非空） |

### TC-CK-011：反向地理編碼失敗不阻擋

| 項目 | 內容 |
|------|------|
| **ID** | TC-CK-011 |
| **名稱** | 地理編碼服務不可用時打卡仍成功 |
| **前置條件** | 模擬 Nominatim 不可達 |
| **測試步驟** | 1. 不帶 address，帶座標 |
| **預期結果** | 1. HTTP 201（打卡成功）<br>2. data.address 為空字串 |

### TC-CK-012：補打卡

| 項目 | 內容 |
|------|------|
| **ID** | TC-CK-012 |
| **名稱** | 翻譯員進行補打卡 |
| **前置條件** | SCH_ID 無打卡紀錄 |
| **測試步驟** | 1. `POST /api/checkins/makeup`<br>2. type=arrive, makeupReason="忘記打卡" |
| **預期結果** | 1. HTTP 201<br>2. data.isMakeup=true<br>3. data.makeupReason="忘記打卡" |

### TC-CK-013：個人統計 — 準時 vs 遲到

| 項目 | 內容 |
|------|------|
| **ID** | TC-CK-013 |
| **名稱** | 驗證準時/遲到的 5 分鐘門檻 |
| **前置條件** | 排班 startTime="09:00"<br>- 打卡 A：checkinTime=09:04 → 準時<br>- 打卡 B：checkinTime=09:06 → 遲到 |
| **測試步驟** | 1. `GET /api/checkins/stats` |
| **預期結果** | 1. onTimeCount 含打卡 A<br>2. lateCount 含打卡 B<br>3. 5 分鐘整（09:05:00）算準時，09:05:01 算遲到 |

### TC-CK-014：個人統計 — 各計數正確

| 項目 | 內容 |
|------|------|
| **ID** | TC-CK-014 |
| **名稱** | 統計各項計數正確性 |
| **前置條件** | TRANS_1 有多筆打卡（含 arrive, leave, makeup）|
| **測試步驟** | 1. `GET /api/checkins/stats` |
| **預期結果** | 1. total = 所有打卡總數<br>2. arriveCount = type="arrive" 的數量<br>3. leaveCount = type="leave" 的數量<br>4. makeupCount = isMakeup=true 的數量 |

### TC-CK-015：管理員列表 — 組合篩選

| 項目 | 內容 |
|------|------|
| **ID** | TC-CK-015 |
| **名稱** | 管理員以多條件篩選打卡紀錄 |
| **前置條件** | 多筆打卡紀錄 |
| **測試步驟** | 1. `GET /api/admin/checkins?translatorId=X&type=arrive&isMakeup=false&dateFrom=2026-01-01&dateTo=2026-12-31` |
| **預期結果** | 只回傳符合全部條件的紀錄 |

### TC-CK-016：管理員編輯打卡

| 項目 | 內容 |
|------|------|
| **ID** | TC-CK-016 |
| **名稱** | 管理員修改打卡時間和地址 |
| **前置條件** | 打卡紀錄 CK_ID 存在 |
| **測試步驟** | 1. `PUT /api/admin/checkins/{CK_ID}`<br>2. Body: `{"checkinTime":"2026-05-01T08:30:00Z","address":"修正地址"}` |
| **預期結果** | 1. HTTP 200，`"Checkin updated successfully"`<br>2. 查詢該筆紀錄，時間和地址已更新<br>3. 其他欄位不變 |
| **稽核驗證** | audit_logs 有 action="update_checkin" |

### TC-CK-017：管理員編輯 — 不傳任何欄位

| 項目 | 內容 |
|------|------|
| **ID** | TC-CK-017 |
| **名稱** | 不傳任何可更新欄位 |
| **前置條件** | CK_ID 存在 |
| **測試步驟** | 1. Body: `{}` |
| **預期結果** | HTTP 400，`"no fields to update"` |

### TC-CK-018：管理員刪除打卡

| 項目 | 內容 |
|------|------|
| **ID** | TC-CK-018 |
| **名稱** | 管理員刪除一筆打卡紀錄 |
| **前置條件** | CK_ID 存在 |
| **測試步驟** | 1. `DELETE /api/admin/checkins/{CK_ID}` |
| **預期結果** | 1. HTTP 200，`"Checkin deleted successfully"`<br>2. 查詢該 ID → 找不到 |
| **稽核驗證** | audit_logs 有 action="delete_checkin" |

### TC-CK-019：打卡狀態邏輯 — 完整驗證

| 項目 | 內容 |
|------|------|
| **ID** | TC-CK-019 |
| **名稱** | 排班的 checkinStatus 隨打卡紀錄正確變化 |
| **前置條件** | 排班 SCH_X 無打卡紀錄 |
| **測試步驟** | 1. 查排班列表 → checkinStatus="none"<br>2. 到達打卡 → 查排班列表 → "arrived"<br>3. 離開打卡 → 查排班列表 → "completed"<br>4. 刪除離開打卡 → 查排班列表 → "arrived"<br>5. 刪除到達打卡 → 查排班列表 → "none" |
| **預期結果** | 每步 checkinStatus 如上所述 |

### TC-CK-020：打卡狀態 — makeup 優先級

| 項目 | 內容 |
|------|------|
| **ID** | TC-CK-020 |
| **名稱** | 有 makeup 打卡時 status 為 "makeup" |
| **前置條件** | 排班 SCH_Y |
| **測試步驟** | 1. 補打卡 arrive（isMakeup=true）<br>2. 正常打卡 leave（isMakeup=false）<br>3. 查排班列表 |
| **預期結果** | checkinStatus="makeup"（makeup 優先級最高） |

---

## 6. TC-EXP：匯出功能

### TC-EXP-001：Excel 匯出

| 項目 | 內容 |
|------|------|
| **ID** | TC-EXP-001 |
| **名稱** | 管理員匯出打卡紀錄為 Excel |
| **前置條件** | 有打卡紀錄，ADMIN_TOKEN 有效 |
| **測試步驟** | 1. `GET /api/admin/export/excel` |
| **預期結果** | 1. HTTP 200<br>2. Content-Type: `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`<br>3. Content-Disposition 含 `checkins.xlsx`<br>4. 回傳二進位 Excel 檔案 |

### TC-EXP-002：Excel 匯出 — 帶篩選

| 項目 | 內容 |
|------|------|
| **ID** | TC-EXP-002 |
| **名稱** | 帶日期區間篩選後匯出 |
| **前置條件** | 有多日打卡紀錄 |
| **測試步驟** | 1. `GET /api/admin/export/excel?dateFrom=2026-05-01&dateTo=2026-05-31` |
| **預期結果** | Excel 中只含 5 月份的打卡資料 |

### TC-EXP-003：Google Sheet 匯出 — 正常

| 項目 | 內容 |
|------|------|
| **ID** | TC-EXP-003 |
| **名稱** | 匯出為 Google Sheet |
| **前置條件** | GOOGLE_CREDENTIALS_FILE 已設定 |
| **測試步驟** | 1. `POST /api/admin/export/google-sheet`<br>2. Body: `{"title":"測試匯出"}` |
| **預期結果** | 1. HTTP 200<br>2. 回傳 `url`（Google Sheets URL）和 `title` |

### TC-EXP-004：Google Sheet 匯出 — 未設定憑證

| 項目 | 內容 |
|------|------|
| **ID** | TC-EXP-004 |
| **名稱** | 未設定 Google Credentials 時匯出 |
| **前置條件** | GOOGLE_CREDENTIALS_FILE 未設定 |
| **測試步驟** | 1. `POST /api/admin/export/google-sheet` |
| **預期結果** | HTTP 503，`"Google credentials not configured..."` |

### TC-EXP-005：匯出排程 — 查詢無設定

| 項目 | 內容 |
|------|------|
| **ID** | TC-EXP-005 |
| **名稱** | 管理員尚未設定匯出排程 |
| **前置條件** | 該管理員無 ExportSchedule 記錄 |
| **測試步驟** | 1. `GET /api/admin/export/schedule` |
| **預期結果** | HTTP 200，`data: null` |

### TC-EXP-006：匯出排程 — 建立設定

| 項目 | 內容 |
|------|------|
| **ID** | TC-EXP-006 |
| **名稱** | 建立匯出排程 |
| **前置條件** | 無 |
| **測試步驟** | 1. `POST /api/admin/export/schedule`<br>2. Body: `{"frequency":"monthly","dayOfMonth":15,"format":"excel","emailTo":"admin@test.com","enabled":true}` |
| **預期結果** | 1. HTTP 200，`"Export schedule saved"`<br>2. `GET /api/admin/export/schedule` 回傳剛設定的值 |

### TC-EXP-007：匯出排程 — upsert 覆蓋

| 項目 | 內容 |
|------|------|
| **ID** | TC-EXP-007 |
| **名稱** | 重複設定會覆蓋而非新增 |
| **前置條件** | TC-EXP-006 已建立排程 |
| **測試步驟** | 1. 再次 POST，dayOfMonth 改為 20，format 改為 google_sheet |
| **預期結果** | 查詢後 dayOfMonth=20, format="google_sheet"（覆蓋，非新增第二筆） |

### TC-EXP-008：匯出排程 — dayOfMonth 邊界值

| 項目 | 內容 |
|------|------|
| **ID** | TC-EXP-008 |
| **名稱** | dayOfMonth 驗證邊界 |
| **前置條件** | 無 |
| **測試步驟** | 1. dayOfMonth=0 → 400<br>2. dayOfMonth=1 → 200<br>3. dayOfMonth=28 → 200<br>4. dayOfMonth=29 → 400 |
| **預期結果** | 合法範圍 1-28 |

### TC-EXP-009：匯出排程 — format 驗證

| 項目 | 內容 |
|------|------|
| **ID** | TC-EXP-009 |
| **名稱** | format 只接受 excel 或 google_sheet |
| **前置條件** | 無 |
| **測試步驟** | 1. format="csv" → 400<br>2. format="excel" → 200<br>3. format="google_sheet" → 200 |
| **預期結果** | 非法值回 400 |

### TC-EXP-010：手動觸發匯出

| 項目 | 內容 |
|------|------|
| **ID** | TC-EXP-010 |
| **名稱** | 手動觸發一次匯出 |
| **前置條件** | SMTP 設定完成，匯出排程已設定 |
| **測試步驟** | 1. `POST /api/admin/export/schedule/run` |
| **預期結果** | 1. HTTP 200，回傳 `"Export executed successfully"` + result 物件<br>2. result 含 rangeFrom / rangeTo<br>3. emailTo 收到信件 |

---

## 7. TC-AUD：稽核紀錄

### TC-AUD-001：查詢全部

| 項目 | 內容 |
|------|------|
| **ID** | TC-AUD-001 |
| **名稱** | 無篩選查詢稽核紀錄 |
| **前置條件** | 已執行多種管理操作 |
| **測試步驟** | 1. `GET /api/admin/audit-logs` |
| **預期結果** | 1. HTTP 200<br>2. `data` 陣列，每筆含 id, admin_id, admin_name, action, target_type, target_id, detail, created_at<br>3. `total` 為全部紀錄數<br>4. 預設分頁 pageSize=20 |

### TC-AUD-002：action 篩選

| 項目 | 內容 |
|------|------|
| **ID** | TC-AUD-002 |
| **名稱** | 以 action 篩選稽核紀錄 |
| **前置條件** | 有 create_translator 和 delete_schedule 紀錄 |
| **測試步驟** | 1. `GET /api/admin/audit-logs?action=create_translator` |
| **預期結果** | 全部結果的 action 皆為 "create_translator" |

### TC-AUD-003：日期篩選

| 項目 | 內容 |
|------|------|
| **ID** | TC-AUD-003 |
| **名稱** | 以 startDate / endDate 篩選 |
| **前置條件** | 有跨月的稽核紀錄 |
| **測試步驟** | 1. `GET /api/admin/audit-logs?startDate=2026-05-01&endDate=2026-05-31` |
| **預期結果** | 全部結果的 created_at 在 5 月份範圍內 |

### TC-AUD-004：分頁功能

| 項目 | 內容 |
|------|------|
| **ID** | TC-AUD-004 |
| **名稱** | 分頁瀏覽稽核紀錄 |
| **前置條件** | 超過 20 筆紀錄 |
| **測試步驟** | 1. `GET /api/admin/audit-logs?page=0&pageSize=5` → 前 5 筆<br>2. `GET /api/admin/audit-logs?page=1&pageSize=5` → 第 6~10 筆 |
| **預期結果** | 1. 兩次回傳不同紀錄<br>2. total 不受分頁影響<br>3. 依 created_at DESC 排序 |

### TC-AUD-005：完整性 — 所有操作皆記錄

| 項目 | 內容 |
|------|------|
| **ID** | TC-AUD-005 |
| **名稱** | 驗證所有管理操作都產生稽核紀錄 |
| **前置條件** | 無 |
| **測試步驟** | 依序執行以下操作並查詢 audit_logs：<br>1. 建立翻譯員 → action="create_translator"<br>2. 更新翻譯員 → action="update_translator"<br>3. 停用翻譯員 → action="disable_translator"<br>4. 重設密碼 → action="reset_password"<br>5. 建立排班 → action="create_schedule"<br>6. 更新排班 → action="update_schedule"<br>7. 刪除排班 → action="delete_schedule"<br>8. 刪除排班組 → action="delete_schedule_group"<br>9. 匯入排班 → action="import_schedules"<br>10. 編輯打卡 → action="update_checkin"<br>11. 刪除打卡 → action="delete_checkin" |
| **預期結果** | 每步操作後 audit_logs 新增一筆，action / target_type / target_id 正確 |

---

## 8. TC-MW：中介層與權限控制

### TC-MW-001：無 token 存取受保護路由

| 項目 | 內容 |
|------|------|
| **ID** | TC-MW-001 |
| **名稱** | 不帶 Authorization header |
| **前置條件** | 無 |
| **測試步驟** | 1. `GET /api/admin/translators`（不帶 header） |
| **預期結果** | HTTP 401，`"Authorization header is required"` |

### TC-MW-002：格式錯誤的 token

| 項目 | 內容 |
|------|------|
| **ID** | TC-MW-002 |
| **名稱** | Authorization header 格式不正確 |
| **前置條件** | 無 |
| **測試步驟** | 1. Header: `Authorization: Token xxx`（無 Bearer 前綴）<br>2. Header: `Authorization: Bearer`（無 token）|
| **預期結果** | HTTP 401，`"Authorization header must be Bearer {token}"` |

### TC-MW-003：無效 / 過期 token

| 項目 | 內容 |
|------|------|
| **ID** | TC-MW-003 |
| **名稱** | 使用無效或已過期的 JWT |
| **前置條件** | 無 |
| **測試步驟** | 1. Header: `Authorization: Bearer invalid.jwt.token`<br>2. 使用已過期的 token |
| **預期結果** | HTTP 401，`"Invalid or expired token"` |

### TC-MW-004：RequirePasswordChanged 攔截

| 項目 | 內容 |
|------|------|
| **ID** | TC-MW-004 |
| **名稱** | mustChangePW=true 被強制改密碼中介層攔截 |
| **前置條件** | 使用者 mustChangePW=true，持有含此 flag 的 token |
| **測試步驟** | 1. `GET /api/admin/translators` → 被攔截<br>2. `GET /api/schedules` → 被攔截<br>3. `POST /api/checkins` → 被攔截<br>4. `POST /api/auth/change-password` → 不被攔截 |
| **預期結果** | 步驟 1-3：HTTP 403，`{"code":"PASSWORD_CHANGE_REQUIRED","error":"password change required"}`<br>步驟 4：正常處理（不被攔截） |

### TC-MW-005：角色權限 — admin only

| 項目 | 內容 |
|------|------|
| **ID** | TC-MW-005 |
| **名稱** | translator 角色存取 admin-only endpoint |
| **前置條件** | TRANS_TOKEN 有效 |
| **測試步驟** | 1. `GET /api/admin/translators` 用 TRANS_TOKEN<br>2. `POST /api/admin/schedules` 用 TRANS_TOKEN<br>3. `GET /api/admin/checkins` 用 TRANS_TOKEN<br>4. `GET /api/admin/audit-logs` 用 TRANS_TOKEN |
| **預期結果** | 全部 HTTP 403，`"Insufficient permissions"` |

### TC-MW-006：角色權限 — translator only

| 項目 | 內容 |
|------|------|
| **ID** | TC-MW-006 |
| **名稱** | admin 角色存取 translator-only endpoint |
| **前置條件** | ADMIN_TOKEN 有效 |
| **測試步驟** | 1. `POST /api/checkins` 用 ADMIN_TOKEN<br>2. `GET /api/schedules` 用 ADMIN_TOKEN<br>3. `GET /api/checkins/stats` 用 ADMIN_TOKEN |
| **預期結果** | 全部 HTTP 403，`"Insufficient permissions"` |

---

## 9. TC-TRACE：OpenTelemetry / Jaeger 追蹤

### TC-TRACE-001：服務註冊

| 項目 | 內容 |
|------|------|
| **ID** | TC-TRACE-001 |
| **名稱** | Jaeger UI 可見服務名稱 |
| **前置條件** | 系統啟動完成 |
| **測試步驟** | 1. 開啟 http://localhost:16686<br>2. Service 下拉選單 |
| **預期結果** | 可見 `translator-checkin`（或 OTEL_SERVICE_NAME 設定值）|

### TC-TRACE-002：HTTP span 自動建立

| 項目 | 內容 |
|------|------|
| **ID** | TC-TRACE-002 |
| **名稱** | 每個 API 請求自動建立 HTTP server span |
| **前置條件** | 無 |
| **測試步驟** | 1. 呼叫 `POST /api/auth/login`<br>2. Jaeger 查詢 service=translator-checkin |
| **預期結果** | 可見 operation=`POST /api/auth/login` 的 trace |

### TC-TRACE-003：SQL span 巢狀在 HTTP span 下

| 項目 | 內容 |
|------|------|
| **ID** | TC-TRACE-003 |
| **名稱** | GORM SQL 查詢 span 為 HTTP span 的子 span |
| **前置條件** | WithCtx 正確傳播 |
| **測試步驟** | 1. 呼叫 `POST /api/auth/login`（正確帳密）<br>2. Jaeger 展開該 trace |
| **預期結果** | 1. 根 span：`POST /api/auth/login`<br>2. 子 span：SQL 查詢（如 `SELECT * FROM users...`）<br>3. 巢狀關係正確（非 orphaned 獨立 trace）|

### TC-TRACE-004：敏感資訊過濾

| 項目 | 內容 |
|------|------|
| **ID** | TC-TRACE-004 |
| **名稱** | span attributes 不包含 PII |
| **前置條件** | 無 |
| **測試步驟** | 1. 呼叫 login API<br>2. Jaeger 查看 span attributes |
| **預期結果** | 不含 email、password、query string 等敏感資訊 |

### TC-TRACE-005：打卡 trace 含外部 HTTP 呼叫

| 項目 | 內容 |
|------|------|
| **ID** | TC-TRACE-005 |
| **名稱** | 打卡觸發反向地理編碼時 trace 含 HTTP client span |
| **前置條件** | GeocodingService 可用，打卡不帶 address |
| **測試步驟** | 1. 打卡時不帶 address<br>2. Jaeger 查看 POST /api/checkins trace |
| **預期結果** | trace 中含 Nominatim HTTP GET 子 span |

---

## 10. TC-CRON：排程任務

### TC-CRON-001：定期匯出 — dayOfMonth 匹配觸發

| 項目 | 內容 |
|------|------|
| **ID** | TC-CRON-001 |
| **名稱** | 當今天 == dayOfMonth 且 enabled=true 時觸發匯出 |
| **前置條件** | ExportSchedule：dayOfMonth=今天日期, enabled=true, SMTP 設定完成 |
| **測試步驟** | 1. 等待 08:00 cron tick（或暫時改 cron 為 `*/1 * * * *`）|
| **預期結果** | 1. emailTo 收到匯出信件<br>2. last_run_at 更新 |

### TC-CRON-002：定期匯出 — dayOfMonth 不匹配

| 項目 | 內容 |
|------|------|
| **ID** | TC-CRON-002 |
| **名稱** | 今天 != dayOfMonth 時不觸發 |
| **前置條件** | dayOfMonth 設為明天日期 |
| **測試步驟** | 1. 等待 cron tick |
| **預期結果** | 不觸發匯出，last_run_at 不變 |

### TC-CRON-003：定期匯出 — enabled=false

| 項目 | 內容 |
|------|------|
| **ID** | TC-CRON-003 |
| **名稱** | 排程停用時不觸發 |
| **前置條件** | enabled=false |
| **測試步驟** | 1. 等待 cron tick |
| **預期結果** | 不觸發匯出 |

### TC-CRON-004：照片清理

| 項目 | 內容 |
|------|------|
| **ID** | TC-CRON-004 |
| **名稱** | 超過 retention 期限的照片被清理 |
| **前置條件** | uploads 目錄有超過 PHOTO_RETENTION_DAYS 的照片 |
| **測試步驟** | 1. 等待 03:00 cron tick |
| **預期結果** | 1. 過期照片被刪除<br>2. 未過期照片保留 |

### TC-CRON-005：排班提醒

| 項目 | 內容 |
|------|------|
| **ID** | TC-CRON-005 |
| **名稱** | 隔日有排班時發送提醒 |
| **前置條件** | 明天有排班，LINE/email 設定完成 |
| **測試步驟** | 1. 等待 07:00 cron tick |
| **預期結果** | 翻譯員收到隔日排班提醒（LINE 或 email） |

---

## 11. TC-SEC：安全性測試

### TC-SEC-001：SQL Injection

| 項目 | 內容 |
|------|------|
| **ID** | TC-SEC-001 |
| **名稱** | SQL injection 防護 |
| **前置條件** | 無 |
| **測試步驟** | 1. 登入 email: `"admin@admin.com' OR '1'='1"`<br>2. 篩選 location: `"'; DROP TABLE schedules; --"` |
| **預期結果** | 1. 回傳正常錯誤訊息（401 或空結果），非 SQL 錯誤<br>2. 資料庫表未被影響 |

### TC-SEC-002：密碼儲存安全

| 項目 | 內容 |
|------|------|
| **ID** | TC-SEC-002 |
| **名稱** | 密碼以 bcrypt 雜湊儲存 |
| **前置條件** | 建立一個翻譯員 |
| **測試步驟** | 1. 直接查詢 DB：`SELECT password_hash FROM users WHERE email='trans1@test.com'` |
| **預期結果** | password_hash 以 `$2a$` 開頭（bcrypt 格式），非明文 |

### TC-SEC-003：API response 不洩漏密碼

| 項目 | 內容 |
|------|------|
| **ID** | TC-SEC-003 |
| **名稱** | 所有 API 回傳不含 password_hash |
| **前置條件** | 無 |
| **測試步驟** | 1. `GET /api/admin/translators` 檢查回傳<br>2. `POST /api/auth/login` 檢查回傳 |
| **預期結果** | 回傳 JSON 中無 password_hash / passwordHash 欄位 |

### TC-SEC-004：跨使用者資料隔離

| 項目 | 內容 |
|------|------|
| **ID** | TC-SEC-004 |
| **名稱** | 翻譯員無法存取其他翻譯員的資料 |
| **前置條件** | TRANS_1 和 TRANS_2 各有排班和打卡 |
| **測試步驟** | 1. TRANS_1 token 呼叫 `GET /api/schedules` → 只有自己的<br>2. TRANS_1 token 呼叫 `GET /api/checkins` → 只有自己的<br>3. TRANS_1 token 對 TRANS_2 的排班打卡 → 400 |
| **預期結果** | 完全無法存取他人資料 |

### TC-SEC-005：帳號鎖定防暴力破解

| 項目 | 內容 |
|------|------|
| **ID** | TC-SEC-005 |
| **名稱** | 暴力破解被帳號鎖定機制防禦 |
| **前置條件** | MAX_LOGIN_ATTEMPTS=5 |
| **測試步驟** | 1. 自動化腳本嘗試 100 次錯誤密碼 |
| **預期結果** | 第 6 次起全部回傳 "account locked"，不會嘗試密碼比對 |

### TC-SEC-006：mustChangePW 繞過防護

| 項目 | 內容 |
|------|------|
| **ID** | TC-SEC-006 |
| **名稱** | 直接帶 mustChangePW=true 的 token 呼叫 API |
| **前置條件** | 取得含 mustChangePW=true 的 token |
| **測試步驟** | 1. 帶該 token 呼叫 `GET /api/admin/translators`<br>2. 帶該 token 呼叫 `POST /api/checkins`<br>3. 帶該 token 呼叫 `GET /api/schedules` |
| **預期結果** | 全部回傳 403 `PASSWORD_CHANGE_REQUIRED` |

---

## 12. TC-E2E：端對端流程

### TC-E2E-001：新人入職完整流程

| 項目 | 內容 |
|------|------|
| **ID** | TC-E2E-001 |
| **名稱** | 新翻譯員從建立到首次打卡的完整流程 |
| **前置條件** | ADMIN_TOKEN 有效 |
| **測試步驟** | 1. 管理員建立翻譯員帳號 → 201<br>2. 翻譯員用初始密碼登入 → 200, mustChangePW=true<br>3. 翻譯員嘗試查看排班 → 403 PASSWORD_CHANGE_REQUIRED<br>4. 翻譯員改密碼 → 200, 取得新 token<br>5. 翻譯員用新 token 查看排班 → 200<br>6. 管理員為翻譯員建立排班 → 201<br>7. 翻譯員查看排班 → 看到新排班<br>8. 翻譯員到達打卡 → 201<br>9. 翻譯員離開打卡 → 201<br>10. 翻譯員查看統計 → total=2 |
| **預期結果** | 全流程順暢，各步驟狀態正確 |

### TC-E2E-002：密碼重設完整流程

| 項目 | 內容 |
|------|------|
| **ID** | TC-E2E-002 |
| **名稱** | 翻譯員忘記密碼，管理員重設，翻譯員改密碼 |
| **前置條件** | TRANS_1 存在，已忘記密碼 |
| **測試步驟** | 1. 管理員重設 TRANS_1 密碼為 "TempPass8" → 200<br>2. TRANS_1 用 TempPass8 登入 → 200, mustChangePW=true<br>3. TRANS_1 嘗試呼叫 API → 403<br>4. TRANS_1 改密碼為 "FinalPass123" → 200<br>5. TRANS_1 用新密碼正常使用系統 → 200 |
| **預期結果** | 完整重設流程可行 |

### TC-E2E-003：重複排班建立與整組刪除

| 項目 | 內容 |
|------|------|
| **ID** | TC-E2E-003 |
| **名稱** | 建立每週重複排班後整組刪除 |
| **前置條件** | TRANS_1 存在 |
| **測試步驟** | 1. 建立 weekly:1,3,5 排班，date=5/1, until=5/31 → 201<br>2. 查詢排班列表 → 看到多筆排班（共用 groupId）<br>3. 對其中一筆執行 /group 刪除 → 200<br>4. 查詢排班列表 → 該組全部消失 |
| **預期結果** | 整組刪除完整且乾淨 |

### TC-E2E-004：打卡紀錄修正流程

| 項目 | 內容 |
|------|------|
| **ID** | TC-E2E-004 |
| **名稱** | 管理員修正錯誤打卡紀錄 |
| **前置條件** | TRANS_1 有一筆到達打卡（時間錯誤）|
| **測試步驟** | 1. 管理員查看打卡列表 → 看到打卡紀錄<br>2. 管理員編輯打卡時間 → 200<br>3. 確認紀錄已更新<br>4. 管理員刪除一筆多餘打卡 → 200<br>5. 確認排班 checkinStatus 正確更新 |
| **預期結果** | 修正流程完整，狀態一致 |

### TC-E2E-005：匯出設定與執行

| 項目 | 內容 |
|------|------|
| **ID** | TC-E2E-005 |
| **名稱** | 設定匯出排程後手動觸發驗證 |
| **前置條件** | SMTP 設定完成 |
| **測試步驟** | 1. `POST /api/admin/export/schedule` 設定 format=excel, emailTo, enabled=true → 200<br>2. `GET /api/admin/export/schedule` → 確認設定正確<br>3. `POST /api/admin/export/schedule/run` → 200<br>4. 檢查 emailTo 信箱 |
| **預期結果** | 信箱收到附 Excel 的信件 |

### TC-E2E-006：Excel 匯入排班流程

| 項目 | 內容 |
|------|------|
| **ID** | TC-E2E-006 |
| **名稱** | 準備 Excel → 匯入 → 翻譯員看到排班 |
| **前置條件** | TRANS_1 存在 |
| **測試步驟** | 1. 準備 Excel 檔案含 3 行合法資料<br>2. `POST /api/admin/schedules/import` 上傳 → 200, success=3<br>3. TRANS_1 查看排班 → 看到 3 筆新排班 |
| **預期結果** | 匯入結果正確，翻譯員可見 |

### TC-E2E-007：稽核追蹤完整流程

| 項目 | 內容 |
|------|------|
| **ID** | TC-E2E-007 |
| **名稱** | 一系列操作後稽核紀錄完整可查 |
| **前置條件** | 無 |
| **測試步驟** | 1. 建立翻譯員<br>2. 建立排班<br>3. 更新排班<br>4. 翻譯員打卡<br>5. 管理員編輯打卡<br>6. 管理員刪除排班<br>7. 查詢 audit-logs → 驗證每步都有紀錄 |
| **預期結果** | audit_logs 中可看到步驟 1,2,3,5,6 的紀錄（步驟 4 不產生稽核）|

---

## 13. TC-UI：前端 UI 測試

### TC-UI-001：登入頁面

| # | 場景 | 操作 | 預期結果 |
|---|------|------|---------|
| 001-1 | 正常登入 | 輸入正確帳密，按登入 | 導向 dashboard |
| 001-2 | 錯誤帳密 | 輸入錯誤密碼 | 顯示錯誤 toast/alert |
| 001-3 | 帳號鎖定 | 連續 5 次錯誤 | 顯示鎖定訊息含剩餘時間 |
| 001-4 | mustChangePW 導轉 | 新使用者登入 | 自動導向 /change-password |

### TC-UI-002：改密碼頁面

| # | 場景 | 操作 | 預期結果 |
|---|------|------|---------|
| 002-1 | 正常改密碼 | 填舊密碼 + 新密碼 | 成功 toast，導向 dashboard |
| 002-2 | 新密碼太短 | 少於 6 字元 | 前端驗證提示 |
| 002-3 | 舊密碼錯誤 | 輸入錯誤舊密碼 | 錯誤 toast |
| 002-4 | token 更新 | 改密碼成功後 | authStore 中 token 已更新 |

### TC-UI-003：翻譯員管理頁面

| # | 場景 | 操作 | 預期結果 |
|---|------|------|---------|
| 003-1 | 列表渲染 | 進入頁面 | 表格顯示所有翻譯員 |
| 003-2 | 建立 Modal | 按「新增」按鈕 | 開啟表單 Modal，含 email/password/name/phone |
| 003-3 | 建立驗證 | 不填 email 按確定 | 表單驗證提示 |
| 003-4 | 編輯 Modal | 按編輯按鈕 | 開啟 Modal 預填現有資料 |
| 003-5 | 停用確認 | 按停用按鈕 | 彈出二次確認 Modal |
| 003-6 | 重設密碼 Modal | 按「重設密碼」 | Modal 含新密碼 + 確認密碼欄位 |
| 003-7 | 重設密碼驗證 | 新密碼 <8 字元 | 前端驗證提示 |
| 003-8 | 重設密碼驗證 | 兩次密碼不一致 | 前端驗證提示 |
| 003-9 | status 篩選 | 切換篩選下拉 | 列表按 active/disabled 篩選 |

### TC-UI-004：排班管理頁面

| # | 場景 | 操作 | 預期結果 |
|---|------|------|---------|
| 004-1 | 列表渲染 | 進入頁面 | 顯示排班列表 |
| 004-2 | 篩選功能 | 選日期區間 + 翻譯員 | 列表更新 |
| 004-3 | 建立排班 | 填寫表單，按確定 | 成功 toast，列表刷新 |
| 004-4 | 建立重複排班 | 選擇重複規則 | 成功 toast |
| 004-5 | 編輯排班 | 按編輯，修改欄位 | 成功 toast |
| 004-6 | 刪除單筆 | 按刪除，確認 | 成功，該筆消失 |
| 004-7 | 刪除整組 | 有 groupId 時顯示「刪除整組」 | 整組消失 |
| 004-8 | 無 groupId | 非重複排班 | 不顯示「刪除整組」按鈕 |
| 004-9 | Excel 匯入 | 上傳 Excel | 顯示 success/failed 計數 |
| 004-10 | checkinStatus | 不同狀態 | 顯示正確的顏色/標籤 |

### TC-UI-005：打卡紀錄頁面（管理員）

| # | 場景 | 操作 | 預期結果 |
|---|------|------|---------|
| 005-1 | 列表渲染 | 進入頁面 | 顯示所有打卡紀錄 |
| 005-2 | 多條件篩選 | 日期 + 翻譯員 + 類型 | 列表更新 |
| 005-3 | 編輯打卡 | 按編輯，修改時間 | 成功 toast |
| 005-4 | 刪除打卡 | 按刪除，二次確認 | 成功，該筆消失 |
| 005-5 | Excel 匯出 | 按匯出按鈕 | 下載 checkins.xlsx |

### TC-UI-006：個人打卡頁面（翻譯員）

| # | 場景 | 操作 | 預期結果 |
|---|------|------|---------|
| 006-1 | 歷史列表 | 進入頁面 | 顯示個人打卡歷史 |
| 006-2 | 統計數據 | 檢查統計卡片 | total/arrive/leave/makeup/onTime/late |
| 006-3 | 日期篩選 | 選日期區間 | 列表 + 統計同步更新 |

### TC-UI-007：匯出設定頁面

| # | 場景 | 操作 | 預期結果 |
|---|------|------|---------|
| 007-1 | 無設定時 | 進入頁面 | 顯示空白表單 / 預設值 |
| 007-2 | 有設定時 | 進入頁面 | 表單填入已存設定 |
| 007-3 | 儲存設定 | 填寫後按儲存 | 成功 toast |
| 007-4 | 立即執行 | 按「立即執行一次」 | loading → 成功 toast |
| 007-5 | dayOfMonth 驗證 | 輸入 0 或 29 | 前端驗證提示 |

### TC-UI-008：稽核紀錄頁面

| # | 場景 | 操作 | 預期結果 |
|---|------|------|---------|
| 008-1 | 列表渲染 | 進入頁面 | 表格顯示稽核紀錄 |
| 008-2 | action 篩選 | 下拉選單選 action | 列表篩選 |
| 008-3 | 日期篩選 | 選日期區間 | 列表篩選 |
| 008-4 | 分頁 | 點下一頁 | 切頁正常 |

### TC-UI-009：403 / 401 攔截器

| # | 場景 | 操作 | 預期結果 |
|---|------|------|---------|
| 009-1 | PASSWORD_CHANGE_REQUIRED | API 回 403 + code | 自動導向 /change-password |
| 009-2 | Insufficient permissions | API 回 403 | 顯示權限不足訊息 |
| 009-3 | 401 Unauthorized | token 過期 | 自動登出，導向登入頁 |

### TC-UI-010：導航與路由保護

| # | 場景 | 操作 | 預期結果 |
|---|------|------|---------|
| 010-1 | admin 側邊選單 | admin 登入 | 顯示：翻譯員管理、排班、打卡紀錄、稽核、匯出設定 |
| 010-2 | translator 側邊選單 | translator 登入 | 顯示：我的排班、打卡、個人紀錄 |
| 010-3 | 路由保護 | translator 直接輸入 /admin/... URL | 跳轉或 403 |

---

## 附錄 A：測試案例統計

| 模組 | 案例數 |
|------|--------|
| TC-AUTH：認證模組 | 18 |
| TC-TM：翻譯員管理 | 15 |
| TC-SCH：排班管理 | 24 |
| TC-CK：打卡功能 | 20 |
| TC-EXP：匯出功能 | 10 |
| TC-AUD：稽核紀錄 | 5 |
| TC-MW：中介層與權限 | 6 |
| TC-TRACE：追蹤 | 5 |
| TC-CRON：排程任務 | 5 |
| TC-SEC：安全性 | 6 |
| TC-E2E：端對端流程 | 7 |
| TC-UI：前端 UI | 10 大項 (40+ 子項) |
| **合計** | **~170 案例** |

## 附錄 B：優先級分級

### 🔴 P0 — 阻塞上線（Must Pass）

- TC-AUTH-001~018（全部認證測試）
- TC-MW-001~006（全部中介層權限）
- TC-SEC-001~006（全部安全性測試）
- TC-CK-001~009（打卡核心流程 + 照片上傳）
- TC-CK-019（打卡狀態邏輯）
- TC-E2E-001（新人入職完整流程）

### 🟡 P1 — 重要功能

- TC-TM-003~015（翻譯員 CRUD + 重設密碼）
- TC-SCH-004~020（排班建立/刪除/匯入）
- TC-CK-012~018（補打卡/統計/管理員編輯刪除）
- TC-EXP-001~010（匯出全部）
- TC-AUD-001~005（稽核紀錄）
- TC-E2E-002~007（其他端對端流程）

### 🟢 P2 — 一般

- TC-TM-001~002（列表篩選）
- TC-SCH-001~003（列表篩選）
- TC-TRACE-001~005（Jaeger 追蹤）
- TC-CRON-001~005（排程任務）
- TC-UI-001~010（前端 UI）
