# 翻譯員打卡系統 — 完整測試項目清單

> 根據現有所有功能列出的測試項目，涵蓋 API 測試、業務邏輯、邊界條件、安全性、整合測試。

---

## 目錄

1. [認證模組 (Auth)](#1-認證模組-auth)
2. [翻譯員管理 (Translator Management)](#2-翻譯員管理-translator-management)
3. [排班管理 (Schedule Management)](#3-排班管理-schedule-management)
4. [打卡功能 (Check-in)](#4-打卡功能-check-in)
5. [匯出功能 (Export)](#5-匯出功能-export)
6. [稽核紀錄 (Audit Log)](#6-稽核紀錄-audit-log)
7. [中介層與安全性 (Middleware & Security)](#7-中介層與安全性-middleware--security)
8. [OpenTelemetry / Jaeger 追蹤](#8-opentelemetry--jaeger-追蹤)
9. [Cron 排程任務](#9-cron-排程任務)
10. [前端 UI 測試](#10-前端-ui-測試)
11. [管理員帳號管理 (Admin Accounts)](#11-管理員帳號管理-admin-accounts)
12. [病人管理 (Patient Management)](#12-病人管理-patient-management)
13. [多病人排班 (Multi-Patient Schedule)](#13-多病人排班-multi-patient-schedule)
14. [診斷證明 / 未到 / 結果總覽 (Diagnosis / No-show / Results)](#14-診斷證明--未到--結果總覽-diagnosis--no-show--results)

> 註：本文件原版（2026-04-19）涵蓋 1~10 章。第 11~14 章與第 4 章的 ✏️ 標記列為
> stage 3/4（管理員帳號、病人正規化、多病人排班、診斷證明）後補。

---

## 1. 認證模組 (Auth)

### 1.1 登入 `POST /api/auth/login`

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 1.1.1 | 正確 email + 密碼登入 | 200，回傳 token + user 資訊 |
| 1.1.2 | 錯誤密碼登入 | 401，回傳 "invalid email or password" |
| 1.1.3 | 不存在的 email 登入 | 401，回傳錯誤訊息 |
| 1.1.4 | 空白 email 欄位 | 400，binding 驗證錯誤 |
| 1.1.5 | 空白 password 欄位 | 400，binding 驗證錯誤 |
| 1.1.6 | 無效 email 格式（如 "abc"） | 400，email 格式驗證錯誤 |
| 1.1.7 | 請求體不是 JSON | 400，binding 錯誤 |
| 1.1.8 | 空請求體 | 400，binding 錯誤 |
| 1.1.9 | 已停用帳號登入 (status=disabled) | 401，回傳 "account is disabled" |
| 1.1.10 | 連續失敗 5 次後再次嘗試 | 401，回傳 "account locked, try again in Xs" |
| 1.1.11 | 帳號鎖定期間以正確密碼登入 | 401，仍然鎖定 |
| 1.1.12 | 帳號鎖定到期後用正確密碼登入 | 200，登入成功 |
| 1.1.13 | 連續失敗 4 次後成功登入 | 200，且失敗計數器被重設 |
| 1.1.14 | 登入成功後 token 包含正確 role | 解碼 JWT，role 為 "admin" 或 "translator" |
| 1.1.15 | mustChangePW=true 的使用者登入 | 200，token 含 mustChangePW=true，user 物件含 mustChangePW=true |
| 1.1.16 | admin 角色登入 | 200，role="admin" |
| 1.1.17 | translator 角色登入 | 200，role="translator" |
| 1.1.18 | MAX_LOGIN_ATTEMPTS 環境變數自訂為 3 | 連續 3 次失敗即鎖定 |
| 1.1.19 | LOCK_DURATION_MINUTES 環境變數自訂為 5 | 鎖定 5 分鐘 |

### 1.2 改密碼 `POST /api/auth/change-password`

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 1.2.1 | 正確舊密碼 + 合法新密碼 | 200，回傳 "Password changed successfully" + 新 token |
| 1.2.2 | 錯誤舊密碼 | 400，"old password is incorrect" |
| 1.2.3 | 新密碼少於 6 字元 | 400，binding 驗證錯誤 |
| 1.2.4 | 新密碼恰好 6 字元 | 200，成功 |
| 1.2.5 | 未帶 token 呼叫 | 401，未授權 |
| 1.2.6 | 帶過期 token 呼叫 | 401，token 過期 |
| 1.2.7 | mustChangePW=true 的使用者改密碼後 | 回傳新 token 中 mustChangePW=false |
| 1.2.8 | 改密碼後用舊密碼登入 | 401，登入失敗 |
| 1.2.9 | 改密碼後用新密碼登入 | 200，登入成功 |
| 1.2.10 | 空白 oldPassword | 400，binding 驗證錯誤 |
| 1.2.11 | 空白 newPassword | 400，binding 驗證錯誤 |
| 1.2.12 | mustChangePW=true 時此 endpoint 不被 RequirePasswordChanged 擋 | 200，可正常改密碼 |

---

## 2. 翻譯員管理 (Translator Management)

### 2.1 列表 `GET /api/admin/translators`

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 2.1.1 | 管理員取得所有翻譯員 | 200，回傳翻譯員陣列 |
| 2.1.2 | 用 status=active 篩選 | 200，只回傳 active 翻譯員 |
| 2.1.3 | 用 status=disabled 篩選 | 200，只回傳 disabled 翻譯員 |
| 2.1.4 | 無翻譯員時 | 200，回傳空陣列 |
| 2.1.5 | translator 角色呼叫 | 403，權限不足 |
| 2.1.6 | 未帶 token 呼叫 | 401，未授權 |
| 2.1.7 | mustChangePW=true 的 admin 呼叫 | 403，PASSWORD_CHANGE_REQUIRED |

### 2.2 建立翻譯員 `POST /api/admin/translators`

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 2.2.1 | 填寫所有必填欄位建立 | 201，建立成功 |
| 2.2.2 | 缺少 email | 400，驗證錯誤 |
| 2.2.3 | 缺少 password | 400，驗證錯誤 |
| 2.2.4 | 缺少 name | 400，驗證錯誤 |
| 2.2.5 | email 格式不合法 | 400，email 格式錯誤 |
| 2.2.6 | 重複的 email | 400，"email already exists" |
| 2.2.7 | 密碼少於 6 字元 | 400，驗證錯誤 |
| 2.2.8 | 不帶 phone（選填欄位） | 201，建立成功，phone 為空 |
| 2.2.9 | 新建翻譯員的 role 為 "translator" | DB 中 role = "translator" |
| 2.2.10 | 新建翻譯員的 status 為 "active" | DB 中 status = "active" |
| 2.2.11 | 新建翻譯員的 mustChangePW 為 true | DB 中 must_change_pw = true |
| 2.2.12 | 建立後產生稽核紀錄 | audit_logs 有 action="create_translator" |
| 2.2.13 | translator 角色呼叫 | 403，權限不足 |

### 2.3 更新翻譯員 `PUT /api/admin/translators/:id`

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 2.3.1 | 更新 name | 200，name 更新 |
| 2.3.2 | 更新 phone | 200，phone 更新 |
| 2.3.3 | 更新 status 為 disabled | 200，status 變 disabled |
| 2.3.4 | 更新 status 為 active | 200，status 變 active |
| 2.3.5 | 無效的 status 值（如 "suspended"）| 400，status 必須為 active 或 disabled |
| 2.3.6 | 不存在的 ID | 400，"translator not found" |
| 2.3.7 | 對 admin 角色的使用者操作 | 400，"user is not a translator" |
| 2.3.8 | ID 非數字（如 "abc"）| 400，invalid ID |
| 2.3.9 | 空請求體（不更新任何欄位）| 200，無變動 |
| 2.3.10 | 只更新部分欄位 | 200，其他欄位不變 |
| 2.3.11 | 更新後產生稽核紀錄 | audit_logs 有 action="update_translator" |

### 2.4 停用翻譯員 `DELETE /api/admin/translators/:id`

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 2.4.1 | 停用一個 active 翻譯員 | 200，status 變 disabled |
| 2.4.2 | 不存在的 ID | 400，"translator not found" |
| 2.4.3 | 對 admin 使用者操作 | 400，"user is not a translator" |
| 2.4.4 | 已經是 disabled 的翻譯員 | 200，仍是 disabled（冪等）|
| 2.4.5 | 停用後該翻譯員無法登入 | 401，"account is disabled" |
| 2.4.6 | 停用後產生稽核紀錄 | audit_logs 有 action="disable_translator" |

### 2.5 管理員重設密碼 `POST /api/admin/translators/:id/reset-password`

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 2.5.1 | 正確重設翻譯員密碼 | 200，"Password reset successfully" |
| 2.5.2 | 管理員對自己重設 | 400，"cannot reset own password" |
| 2.5.3 | 不存在的 ID | 400，target user not found |
| 2.5.4 | 新密碼少於 8 字元 | 400，binding 驗證錯誤 |
| 2.5.5 | 新密碼恰好 8 字元 | 200，成功 |
| 2.5.6 | 重設後翻譯員用舊密碼登入 | 401，失敗 |
| 2.5.7 | 重設後翻譯員用新密碼登入 | 200，成功且 mustChangePW=true |
| 2.5.8 | 重設後翻譯員的 mustChangePW 被設為 true | 確認 DB 欄位 |
| 2.5.9 | 重設後產生稽核紀錄 | audit_logs 有 action="reset_password" |
| 2.5.10 | translator 角色呼叫 | 403，權限不足 |

---

## 3. 排班管理 (Schedule Management)

### 3.1 管理員列表 `GET /api/admin/schedules`

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 3.1.1 | 無篩選條件，取得所有排班 | 200，回傳所有排班 |
| 3.1.2 | dateFrom + dateTo 日期區間篩選 | 200，只回傳範圍內排班 |
| 3.1.3 | translatorId 篩選 | 200，只回傳該翻譯員排班 |
| 3.1.4 | location 篩選（部分匹配） | 200，ILIKE 模式匹配 |
| 3.1.5 | 多條件組合篩選 | 200，交集結果 |
| 3.1.6 | 無結果時 | 200，空陣列 |
| 3.1.7 | 每筆回傳正確的 checkinStatus | 驗證 none/arrived/completed/makeup |
| 3.1.8 | 每筆回傳 translatorName | 非空字串 |
| 3.1.9 | 回傳 recurrenceGroupId（重複排班）| UUID 或 null |
| 3.1.10 | translatorId 非數字時 | 忽略篩選，回傳全部 |

### 3.2 建立排班 `POST /api/admin/schedules`

#### 3.2.1 單次排班

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 3.2.1.1 | 填寫所有必填欄位建立 | 201，回傳排班資料 |
| 3.2.1.2 | 缺少 translatorId | 400 |
| 3.2.1.3 | 缺少 date | 400 |
| 3.2.1.4 | 缺少 startTime | 400 |
| 3.2.1.5 | 缺少 endTime | 400 |
| 3.2.1.6 | 缺少 location | 400 |
| 3.2.1.7 | 缺少 patientName | 400 |
| 3.2.1.8 | 日期格式錯誤（如 "01/15/2026"）| 400，"invalid date format" |
| 3.2.1.9 | 不存在的 translatorId | 400，"translator not found" |
| 3.2.1.10 | translatorId 是 admin 的 ID | 400，"user is not a translator" |
| 3.2.1.11 | note 為選填，不帶也可 | 201，note 為空 |
| 3.2.1.12 | 建立後產生稽核紀錄 | audit_logs 有 action="create_schedule" |

#### 3.2.2 重複排班（每日）

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 3.2.2.1 | recurrenceRule="daily", date 到 recurrenceUntil 產生正確數量 | 驗證 DB 中記錄數 |
| 3.2.2.2 | daily 且 date == recurrenceUntil | 只產生 1 筆 |
| 3.2.2.3 | 有 recurrenceRule 但沒 recurrenceUntil | 400，"recurrenceUntil is required" |
| 3.2.2.4 | recurrenceUntil 早於 date | 400，"recurrenceUntil must be after or equal to date" |
| 3.2.2.5 | 所有記錄共用同一個 recurrenceGroupId | UUID 一致 |

#### 3.2.3 重複排班（每週）

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 3.2.3.1 | weekly:1,3,5（週一三五）| 只在對應星期幾產生排班 |
| 3.2.3.2 | weekly:0（週日） | 只在週日產生 |
| 3.2.3.3 | weekly:0,6（週末） | 只在週六日產生 |
| 3.2.3.4 | weekday 值超過 6 | 400，"weekday values must be 0-6" |
| 3.2.3.5 | weekday 值為負數 | 400 |
| 3.2.3.6 | weekly 格式錯誤（如 "weekly:abc"）| 400 |

#### 3.2.4 重複排班（每月）

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 3.2.4.1 | monthly:5,20（每月 5 號和 20 號）| 正確產生 |
| 3.2.4.2 | monthly:31 在 2 月（非閏年）| 產生 2/28 |
| 3.2.4.3 | monthly:31 在 2 月（閏年）| 產生 2/29 |
| 3.2.4.4 | monthly:31 在 4 月（30 天月份）| 產生 4/30 |
| 3.2.4.5 | monthly:31 在 1 月（31 天月份）| 產生 1/31 |
| 3.2.4.6 | monthly:29,30,31 在 2 月 | 不重複，只產生 1 筆（28 或 29）|
| 3.2.4.7 | day 值超過 31 | 400 |
| 3.2.4.8 | day 值為 0 | 400 |
| 3.2.4.9 | 未知 recurrenceRule（如 "yearly:1"）| 400，"unknown rule" |

### 3.3 更新排班 `PUT /api/admin/schedules/:id`

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 3.3.1 | 更新 date | 200，日期更新 |
| 3.3.2 | 更新 startTime / endTime | 200 |
| 3.3.3 | 更新 location | 200 |
| 3.3.4 | 更新 patientName | 200 |
| 3.3.5 | 更新 note | 200 |
| 3.3.6 | 不存在的 schedule ID | 400，"schedule not found" |
| 3.3.7 | 更新後 checkinStatus 仍正確 | 驗證狀態不變 |
| 3.3.8 | 部分更新（只傳 note） | 200，其他欄位不變 |
| 3.3.9 | 無效日期格式 | 400 |
| 3.3.10 | 更新後產生稽核紀錄 | audit_logs 有 action="update_schedule" |

### 3.4 刪除排班 `DELETE /api/admin/schedules/:id`

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 3.4.1 | 刪除存在的排班 | 200，"Schedule deleted successfully" |
| 3.4.2 | 不存在的 ID | 400，"schedule not found" |
| 3.4.3 | 刪除後再查詢該排班 | 找不到 |
| 3.4.4 | 刪除後產生稽核紀錄 | audit_logs 有 action="delete_schedule" |

### 3.5 刪除整組排班 `DELETE /api/admin/schedules/:id/group`

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 3.5.1 | 對有 recurrenceGroupId 的排班刪除整組 | 200，回傳 deleted 數量 |
| 3.5.2 | 同組所有排班都被刪除 | 查詢同 groupId 回傳 0 筆 |
| 3.5.3 | 對沒有 recurrenceGroupId 的排班 | 200，deleted=1，只刪單筆 |
| 3.5.4 | 不存在的 ID | 400，"schedule not found" |
| 3.5.5 | 刪除後產生稽核紀錄 | audit_logs 有 action="delete_schedule_group" |

### 3.6 Excel 匯入排班 `POST /api/admin/schedules/import`

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 3.6.1 | 上傳合法 Excel，全部成功 | 200，success=N, failed=0 |
| 3.6.2 | 未上傳檔案 | 400，"File is required" |
| 3.6.3 | 上傳非 Excel 檔案 | 400，解析錯誤 |
| 3.6.4 | 部分行有錯（如 translatorId 不存在）| 200，success=M, failed=K |
| 3.6.5 | 全部行都有錯 | 200，success=0, failed=N |
| 3.6.6 | 空的 Excel（只有表頭）| 200，success=0, failed=0 |
| 3.6.7 | translatorId 為空 | 該行 failed |
| 3.6.8 | date 格式錯誤 | 該行 failed |
| 3.6.9 | translatorId 指向 admin 使用者 | 該行 failed，"translator not found" |
| 3.6.10 | 跳過完全空白行 | 不計入 total |
| 3.6.11 | 匯入後產生稽核紀錄 | audit_logs 有 action="import_schedules" |

### 3.7 翻譯員查看排班 `GET /api/schedules`

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 3.7.1 | 翻譯員查看自己的排班 | 200，只有自己的排班 |
| 3.7.2 | 用 date_from / date_to 篩選 | 200，範圍內排班 |
| 3.7.3 | 無排班時 | 200，空陣列 |
| 3.7.4 | admin 角色呼叫 | 403，權限不足 |
| 3.7.5 | 回傳正確的 checkinStatus | 驗證各狀態 |

---

## 4. 打卡功能 (Check-in)

### 4.1 一般打卡 `POST /api/checkins`

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 4.1.1 | 正常到達打卡 (type=arrive) | 201，回傳打卡紀錄 |
| 4.1.2 | 到達後離開打卡 (type=leave) | 201 |
| 4.1.3 | 未到達就離開 | 400，"must check in (arrive) before checking out (leave)" |
| 4.1.4 | 重複到達打卡 | 400，"already checked in with type: arrive" |
| 4.1.5 | 重複離開打卡 | 400，"already checked in with type: leave" |
| 4.1.6 | 排班不存在 | 400，"schedule not found" |
| 4.1.7 | 排班不屬於該翻譯員 | 400，"schedule does not belong to this translator" |
| 4.1.8 | 缺少 selfie 照片 | 400，`SELFIE_REQUIRED` |
| ~~4.1.9~~ | ~~缺少 environment 照片~~ ✏️ **已廢除** — stage 4 移除環境照需求，environment 照不再必填 | （不再驗證；保留欄位向後相容）|
| 4.1.10 | 帶 GPS 座標（lat/lng）| 201，座標被儲存 |
| 4.1.11 | 不帶 GPS 座標 | 201，lat/lng 為 0 |
| 4.1.12 | 不帶 address 但帶 GPS | 201，自動反向地理編碼填入 address |
| 4.1.13 | 反向地理編碼失敗 | 201，address 為空（不阻擋打卡）|
| 4.1.14 | 回傳的 selfieUrl 路徑格式正確 | /uploads/selfie_yyyymmdd_... |
| ~~4.1.15~~ | ~~回傳的 environmentUrl 路徑格式正確~~ ✏️ **已廢除**（環境照移除） | — |
| 4.1.16 | checkinTime 為伺服器時間 | 接近 time.Now() |
| 4.1.17 | isMakeup 預設為 false | false |
| 4.1.18 | admin 角色呼叫 | 403，權限不足 |
| 4.1.19 | ✏️ 打卡時間已超過排班 endTime 且未標 makeup | 201，後端自動標 isMakeup=true 並填系統補登原因 |
| 4.1.20 | ✏️ 離開打卡但仍有 pending 病人（尚未上傳診斷/未到） | 400，`CHECKOUT_BLOCKED_BY_PENDING` |
| 4.1.21 | ✏️ 離開打卡且所有病人皆 completed/no_show | 201，放行 |
| 4.1.22 | ✏️ makeup 離開打卡可略過 pending gate | 201，補登模式不擋 pending |

### 4.2 補打卡 `POST /api/checkins/makeup`

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 4.2.1 | 正常補打卡 | 201，isMakeup=true |
| 4.2.2 | 有 makeupReason | 201，makeupReason 被儲存 |
| 4.2.3 | 到達 / 離開序列規則同一般打卡 | 同 4.1.3~4.1.5 |
| 4.2.4 | 同一排班先正常打卡再補打卡 | 400，重複類型 |
| 4.2.5 | 回傳 isMakeup=true 和 makeupReason | 確認欄位 |

### 4.3 翻譯員查看打卡紀錄 `GET /api/checkins`

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 4.3.1 | 查看自己的打卡歷史 | 200，只回傳自己的 |
| 4.3.2 | dateFrom / dateTo 篩選 | 200，範圍內紀錄 |
| 4.3.3 | 無紀錄時 | 200，空陣列 |
| 4.3.4 | 回傳 translatorName | 非空 |

### 4.4 翻譯員統計 `GET /api/checkins/stats`

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 4.4.1 | 回傳正確 total 數量 | 打卡總數 |
| 4.4.2 | arriveCount 正確 | 到達打卡數量 |
| 4.4.3 | leaveCount 正確 | 離開打卡數量 |
| 4.4.4 | makeupCount 正確 | 補打卡數量 |
| 4.4.5 | 準時到達（排班開始 5 分鐘內） | onTimeCount +1 |
| 4.4.6 | 遲到到達（排班開始 5 分鐘後） | lateCount +1 |
| 4.4.7 | 恰好 5 分鐘邊界 | onTimeCount +1（不算遲到）|
| 4.4.8 | 無紀錄時所有計數為 0 | 全部 0 |
| 4.4.9 | dateFrom / dateTo 篩選影響統計 | 只計算範圍內 |

### 4.5 管理員列表 `GET /api/admin/checkins`

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 4.5.1 | 無篩選，取得所有打卡 | 200，全部紀錄 |
| 4.5.2 | dateFrom / dateTo 篩選 | 200 |
| 4.5.3 | translatorId 篩選 | 200，只回傳該翻譯員 |
| 4.5.4 | type=arrive 篩選 | 200，只回傳到達打卡 |
| 4.5.5 | type=leave 篩選 | 200，只回傳離開打卡 |
| 4.5.6 | isMakeup=true 篩選 | 200，只回傳補打卡 |
| 4.5.7 | isMakeup=false 篩選 | 200，只回傳正常打卡 |
| 4.5.8 | 多條件組合 | 200，交集 |
| 4.5.9 | 每筆回傳 translatorName | 非空 |

### 4.6 管理員編輯打卡 `PUT /api/admin/checkins/:id`

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 4.6.1 | 更新 checkinTime | 200，時間更新 |
| 4.6.2 | 更新 address | 200，地址更新 |
| 4.6.3 | 更新 makeupReason | 200 |
| 4.6.4 | 不存在的 checkin ID | 400，"checkin not found" |
| 4.6.5 | 不傳任何欄位 | 400，"no fields to update" |
| 4.6.6 | 部分更新（只傳 address） | 200，其他欄位不變 |
| 4.6.7 | 更新後產生稽核紀錄 | audit_logs 有 action="update_checkin" |
| 4.6.8 | translator 角色呼叫 | 403 |

### 4.7 管理員刪除打卡 `DELETE /api/admin/checkins/:id`

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 4.7.1 | 刪除存在的打卡 | 200，"Checkin deleted successfully" |
| 4.7.2 | 不存在的 ID | 400，"checkin not found" |
| 4.7.3 | 刪除後排班 checkinStatus 回到 none | 狀態更新 |
| 4.7.4 | 只刪 leave 後 checkinStatus 變 arrived | 確認邏輯 |
| 4.7.5 | 刪除後產生稽核紀錄 | audit_logs 有 action="delete_checkin" |
| 4.7.6 | translator 角色呼叫 | 403 |

### 4.8 打卡狀態邏輯 (checkinStatus)

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 4.8.1 | 無打卡 → status=none | 排班列表中顯示 none |
| 4.8.2 | 只有 arrive → status=arrived | 排班列表中顯示 arrived |
| 4.8.3 | arrive + leave → status=completed | 顯示 completed |
| 4.8.4 | 有 isMakeup=true → status=makeup | makeup 優先級最高 |
| 4.8.5 | arrive(makeup) + leave(normal) → status=makeup | 只要有 makeup 就是 makeup |

---

## 5. 匯出功能 (Export)

### 5.1 Excel 匯出 `GET /api/admin/export/excel`

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 5.1.1 | 無篩選匯出所有打卡 | 200，下載 .xlsx 檔案 |
| 5.1.2 | Content-Type 正確 | application/vnd.openxml... |
| 5.1.3 | Content-Disposition 包含檔名 | attachment; filename="checkins.xlsx" |
| 5.1.4 | dateFrom / dateTo 篩選 | 檔案只含範圍內資料 |
| 5.1.5 | 無資料時 | 200，空白 Excel（只有表頭）|
| 5.1.6 | translator 角色呼叫 | 403 |

### 5.2 Google Sheet 匯出 `POST /api/admin/export/google-sheet`

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 5.2.1 | 設定好 Google Credentials 後匯出 | 200，回傳 url + title |
| 5.2.2 | 未設定 Google Credentials | 503 |
| 5.2.3 | 自訂 title | 200，回傳自訂 title |
| 5.2.4 | 不帶 title | 200，使用預設時間戳 title |

### 5.3 匯出排程設定 `GET /api/admin/export/schedule`

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 5.3.1 | 有設定的管理員查詢 | 200，回傳設定 |
| 5.3.2 | 無設定的管理員查詢 | 200，data=null |

### 5.4 匯出排程儲存 `POST /api/admin/export/schedule`

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 5.4.1 | 建立新的匯出排程 | 200，"Export schedule saved" |
| 5.4.2 | 更新既有匯出排程（upsert）| 200，覆蓋舊設定 |
| 5.4.3 | dayOfMonth=0 | 400，驗證錯誤 |
| 5.4.4 | dayOfMonth=29 | 400，超過上限 28 |
| 5.4.5 | dayOfMonth=28 | 200 |
| 5.4.6 | dayOfMonth=1 | 200 |
| 5.4.7 | format 不是 excel/google_sheet | 400 |
| 5.4.8 | format=excel | 200 |
| 5.4.9 | format=google_sheet | 200 |
| 5.4.10 | enabled=true | 200，排程啟用 |
| 5.4.11 | enabled=false | 200，排程停用 |

### 5.5 手動觸發匯出 `POST /api/admin/export/schedule/run`

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 5.5.1 | 有設定好 SMTP 後觸發 | 200，回傳匯出結果 |
| 5.5.2 | 未設定 SMTP | 500，SMTP 錯誤 |
| 5.5.3 | format=excel 時信件附 Excel | 確認附件 |
| 5.5.4 | format=google_sheet 時信件含連結 | 確認連結 |
| 5.5.5 | 執行後更新 lastRunAt | DB 中 last_run_at 更新 |
| 5.5.6 | translator 角色呼叫 | 403 |

---

## 6. 稽核紀錄 (Audit Log)

### 6.1 查詢稽核紀錄 `GET /api/admin/audit-logs`

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 6.1.1 | 無篩選取得所有紀錄 | 200，分頁回傳 |
| 6.1.2 | action 篩選（如 "create_translator"）| 200，只回傳該 action |
| 6.1.3 | targetType 篩選 | 200 |
| 6.1.4 | adminId 篩選 | 200 |
| 6.1.5 | startDate / endDate 日期篩選 | 200 |
| 6.1.6 | 分頁：page=1, pageSize=10 | 200，最多 10 筆 |
| 6.1.7 | 分頁：page=2 | 200，偏移正確 |
| 6.1.8 | 預設 pageSize=20 | 不帶 pageSize 時回傳最多 20 筆 |
| 6.1.9 | 回傳 total 為全部符合條件的數量 | 不受分頁影響 |
| 6.1.10 | 每筆包含 adminName | 非空 |
| 6.1.11 | 依 createdAt DESC 排序 | 最新的在最前面 |
| 6.1.12 | translator 角色呼叫 | 403 |

### 6.2 稽核紀錄完整性

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 6.2.1 | 建立翻譯員 → 有紀錄 | action="create_translator" |
| 6.2.2 | 更新翻譯員 → 有紀錄 | action="update_translator" |
| 6.2.3 | 停用翻譯員 → 有紀錄 | action="disable_translator" |
| 6.2.4 | 重設密碼 → 有紀錄 | action="reset_password" |
| 6.2.5 | 建立排班 → 有紀錄 | action="create_schedule" |
| 6.2.6 | 更新排班 → 有紀錄 | action="update_schedule" |
| 6.2.7 | 刪除排班 → 有紀錄 | action="delete_schedule" |
| 6.2.8 | 刪除排班組 → 有紀錄 | action="delete_schedule_group" |
| 6.2.9 | 匯入排班 → 有紀錄 | action="import_schedules" |
| 6.2.10 | 編輯打卡 → 有紀錄 | action="update_checkin" |
| 6.2.11 | 刪除打卡 → 有紀錄 | action="delete_checkin" |
| 6.2.12 | 所有紀錄包含正確 adminId | 操作者 ID |
| 6.2.13 | 所有紀錄包含正確 targetType | "translator" / "schedule" / "checkin" |
| 6.2.14 | 所有紀錄包含正確 targetId | 目標 ID |

---

## 7. 中介層與安全性 (Middleware & Security)

### 7.1 JWT 認證 (JWTAuth)

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 7.1.1 | 不帶 Authorization header | 401 |
| 7.1.2 | Authorization 格式不正確（無 "Bearer " 前綴）| 401 |
| 7.1.3 | 無效的 JWT token | 401 |
| 7.1.4 | 過期的 JWT token | 401 |
| 7.1.5 | 有效 token 正確解析 userID | context 中有 userID |
| 7.1.6 | 有效 token 正確解析 role | context 中有 role |
| 7.1.7 | 有效 token 正確解析 mustChangePW | context 中有值 |

### 7.2 強制改密碼 (RequirePasswordChanged)

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 7.2.1 | mustChangePW=true 呼叫受保護 endpoint | 403，code="PASSWORD_CHANGE_REQUIRED" |
| 7.2.2 | mustChangePW=false 呼叫受保護 endpoint | 通過，繼續 |
| 7.2.3 | /api/auth/change-password 不套用此中介層 | 不被擋 |
| 7.2.4 | /api/auth/login 不套用此中介層 | 不被擋 |
| 7.2.5 | 所有 /api/admin/* 路由套用 | 被擋 |
| 7.2.6 | 所有 /api/schedules 路由套用 | 被擋 |
| 7.2.7 | 所有 /api/checkins/* 路由套用 | 被擋 |

### 7.3 角色權限 (RoleRequired)

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 7.3.1 | admin 呼叫 admin-only endpoint | 200，通過 |
| 7.3.2 | translator 呼叫 admin-only endpoint | 403 |
| 7.3.3 | translator 呼叫 translator-only endpoint | 200，通過 |
| 7.3.4 | admin 呼叫 translator-only endpoint | 403 |

### 7.4 安全性測試

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 7.4.1 | SQL injection 在 email 欄位 | 無效，被參數化查詢保護 |
| 7.4.2 | SQL injection 在篩選參數 | 無效 |
| 7.4.3 | 密碼以 bcrypt 雜湊儲存 | DB 中不含明文密碼 |
| 7.4.4 | API 回傳不含 password_hash | 確認 response 無密碼欄位 |
| 7.4.5 | JWT secret 足夠複雜 | 非預設值 |
| 7.4.6 | 帳號鎖定防暴力破解 | 連續失敗後鎖定 |
| 7.4.7 | 翻譯員無法存取其他翻譯員資料 | 只看到自己的排班/打卡 |
| 7.4.8 | 翻譯員無法打其他人的排班 | 400，schedule 不屬於該翻譯員 |
| 7.4.9 | Path traversal in file upload | 檔名使用時間戳生成，不受影響 |
| 7.4.10 | CORS 設定正確 | 只允許預期的來源 |

---

## 8. OpenTelemetry / Jaeger 追蹤

### 8.1 追蹤基礎設施

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 8.1.1 | 服務在 Jaeger 中註冊 | service=translator-checkin 可見 |
| 8.1.2 | 每個 HTTP 請求產生 server span | otelgin middleware 運作 |
| 8.1.3 | SQL 查詢產生子 span | gorm tracing plugin 運作 |
| 8.1.4 | SQL span 嵌套在 HTTP span 下 | WithCtx 正確傳播 context |
| 8.1.5 | 外部 HTTP 呼叫（Nominatim）產生 span | otelhttp transport 運作 |
| 8.1.6 | span attributes 不含 PII | scrubSensitiveSpanAttributes 過濾 |

### 8.2 Context 傳播驗證

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 8.2.1 | POST /api/auth/login 的 trace 有巢狀 SQL spans | select users → update users |
| 8.2.2 | POST /api/checkins 的 trace 有完整 span 鏈 | select schedules → select checkins → HTTP Nominatim → insert |
| 8.2.3 | GET /api/admin/schedules 的 trace | select + 多個 checkin 查詢 |
| 8.2.4 | 所有 repository 的 WithCtx 都被呼叫 | 無 orphaned SQL span |
| 8.2.5 | Cron 任務建立獨立 root span | tracer.Start(ctx, "cron.xxx") |
| 8.2.6 | OTEL_EXPORTER_OTLP_ENDPOINT 可設定 | 自訂 endpoint 運作 |
| 8.2.7 | OTEL_SERVICE_NAME 可設定 | Jaeger 顯示自訂名稱 |

---

## 9. Cron 排程任務

### 9.1 排班提醒 (每日 07:00)

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 9.1.1 | 隔日有排班 → 發送 LINE 推播 | 翻譯員收到通知 |
| 9.1.2 | 隔日無排班 → 不發送 | 無通知 |
| 9.1.3 | LINE API 失敗 → 嘗試 email 備援 | email 寄出 |
| 9.1.4 | 多位翻譯員同時有排班 | 各自收到通知 |

### 9.2 照片清理 (每日 03:00)

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 9.2.1 | 超過 retention 的照片被刪除 | 檔案消失 |
| 9.2.2 | 未超過 retention 的照片保留 | 檔案存在 |
| 9.2.3 | PHOTO_RETENTION_DAYS 預設 90 天 | 90 天前照片被刪 |
| 9.2.4 | 自訂 PHOTO_RETENTION_DAYS | 依設定執行 |

### 9.3 定期匯出 (每日 08:00)

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 9.3.1 | 今天 == dayOfMonth 且 enabled=true | 執行匯出 |
| 9.3.2 | 今天 != dayOfMonth | 不執行 |
| 9.3.3 | enabled=false | 不執行 |
| 9.3.4 | 執行後 last_run_at 更新 | DB 欄位更新 |
| 9.3.5 | 匯出區間為上個月 1 號至最後一天 | 資料範圍正確 |
| 9.3.6 | 多位管理員各有設定 | 各自獨立執行 |

---

## 10. 前端 UI 測試

### 10.1 登入頁面

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 10.1.1 | 正確帳密登入 | 導向 dashboard |
| 10.1.2 | 錯誤帳密顯示錯誤訊息 | toast/alert 提示 |
| 10.1.3 | 帳號鎖定顯示剩餘時間 | 提示鎖定訊息 |
| 10.1.4 | mustChangePW=true 導向改密碼頁 | 自動跳轉 /change-password |

### 10.2 改密碼頁面

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 10.2.1 | 輸入正確舊密碼 + 新密碼 | 成功，導向 dashboard |
| 10.2.2 | 新密碼少於 6 字元 | 前端驗證提示 |
| 10.2.3 | 改密碼後 authStore 更新 token | 新 token 被儲存 |
| 10.2.4 | 改密碼後不再被攔截 | 可正常使用系統 |

### 10.3 翻譯員管理頁面

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 10.3.1 | 列表顯示所有翻譯員 | 表格渲染正確 |
| 10.3.2 | 建立新翻譯員 Modal | 表單驗證 + 提交 |
| 10.3.3 | 編輯翻譯員 Modal | 顯示現有資料，可修改 |
| 10.3.4 | 停用翻譯員確認彈窗 | 二次確認 |
| 10.3.5 | 重設密碼 Modal | 輸入新密碼 + 確認密碼 |
| 10.3.6 | 重設密碼前端驗證（≥8 字元、兩次一致）| 驗證通過才送出 |
| 10.3.7 | status 篩選功能 | 切換顯示 active/disabled |

### 10.4 排班管理頁面

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 10.4.1 | 排班列表顯示 | 表格渲染 |
| 10.4.2 | 篩選條件（日期、翻譯員、地點）| 列表即時更新 |
| 10.4.3 | 建立單次排班 | 成功 toast |
| 10.4.4 | 建立重複排班 | 成功 toast |
| 10.4.5 | 編輯排班 | 成功 toast |
| 10.4.6 | 刪除單筆排班確認彈窗 | 二次確認 + 刪除 |
| 10.4.7 | 刪除整組排班按鈕（有 recurrenceGroupId 時顯示）| 顯示「刪除整組」|
| 10.4.8 | 非重複排班不顯示「刪除整組」按鈕 | 只有「刪除」|
| 10.4.9 | Excel 匯入功能 | 上傳 + 回傳結果 |
| 10.4.10 | checkinStatus 正確顯示顏色/標籤 | none/arrived/completed/makeup |

### 10.5 打卡紀錄頁面（管理員）

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 10.5.1 | 列表顯示所有打卡紀錄 | 表格渲染 |
| 10.5.2 | 篩選功能 | 日期/翻譯員/類型/補打卡 |
| 10.5.3 | 編輯打卡 Modal | 只顯示可編輯欄位 |
| 10.5.4 | 刪除打卡確認彈窗 | 二次確認 |
| 10.5.5 | Excel 匯出按鈕 | 下載檔案 |
| 10.5.6 | Google Sheet 匯出按鈕 | 開啟新視窗 |

### 10.6 個人打卡頁面（翻譯員）

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 10.6.1 | 顯示個人打卡歷史 | 列表渲染 |
| 10.6.2 | 統計數據顯示 | total/arrive/leave/makeup/onTime/late |
| 10.6.3 | 日期篩選功能 | 列表 + 統計更新 |

### 10.7 稽核紀錄頁面

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 10.7.1 | 列表顯示 | 表格渲染 |
| 10.7.2 | action 篩選 | 下拉選單篩選 |
| 10.7.3 | 日期篩選 | 範圍篩選 |
| 10.7.4 | 分頁功能 | 切頁正常 |

### 10.8 匯出設定頁面

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 10.8.1 | 顯示現有設定 | 表單填入現有值 |
| 10.8.2 | 無設定時顯示空表單 | 預設值 |
| 10.8.3 | 儲存設定 | 成功 toast |
| 10.8.4 | 「立即執行一次」按鈕 | loading + 成功 toast |
| 10.8.5 | dayOfMonth 驗證（1-28）| 前端驗證 |
| 10.8.6 | format 下拉選單 | excel / google_sheet |

### 10.9 403 處理（前端攔截器）

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 10.9.1 | API 回 403 PASSWORD_CHANGE_REQUIRED | 自動導向 /change-password |
| 10.9.2 | API 回 403 Insufficient permissions | 顯示權限不足訊息 |
| 10.9.3 | API 回 401 | 自動登出，導向登入頁 |

### 10.10 側邊選單 / 導航

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 10.10.1 | admin 角色顯示管理選單 | 翻譯員管理、排班、打卡、稽核、匯出 |
| 10.10.2 | translator 角色顯示翻譯員選單 | 我的排班、打卡、個人紀錄 |
| 10.10.3 | 路由保護（translator 存取 /admin/*）| 跳轉或顯示 403 |

### 10.11 病人選擇器 PatientPicker ✏️（新元件）

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 10.11.1 | 掛載時抓病人清單 | 呼叫 API 並列出選項 |
| 10.11.2 | 選取病人觸發 onChange | 回傳 patientId |
| 10.11.3 | 搜尋輸入 debounce 後重新查詢 | 只送一次查詢 |
| 10.11.4 | value 在掛載後設定時顯示病人名 | 正確顯示 |

### 10.12 多病人清單編輯器 SchedulePatientListEditor ✏️（新元件）

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 10.12.1 | 初始空值顯示一列空白 | 1 列 |
| 10.12.2 | 依 value 渲染既有列 | 列數正確 |
| 10.12.3 | 點 Add 新增一列 | 列數 +1 |
| 10.12.4 | 點 Delete 移除一列 | 列數 -1 |
| 10.12.5 | clampPatientTimes 將病人時段夾到整體時段內 | util 單元測試 |
| 10.12.6 | validatePatientTimes 偵測 end≤start / 超出範圍 / 重複病人 / 缺 patientId | 回對應錯誤碼 |

### 10.13 診斷上傳 / 未到 Modal ✏️（新元件）

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 10.13.1 | DiagnosisUploadModal 未選檔時送出禁用 | 按鈕 disabled |
| 10.13.2 | 選檔超過 3 張自動截斷並提示（不擋送出） | 最多 3 張 |
| 10.13.3 | 選 1~3 張可送出並呼叫 upload | 觸發上傳 |
| 10.13.4 | NoShowModal 原因空白時送出禁用 | 按鈕 disabled |
| 10.13.5 | NoShowModal 填原因後可送出並呼叫 markNoShow | 觸發 |

### 10.14 管理員 / 診斷結果頁 ✏️

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 10.14.1 | 管理員帳號管理頁列表 / 新增 / 刪除 | UI 操作正確 |
| 10.14.2 | 診斷結果總覽頁篩選 + 分頁 | 呼叫對應 API |
| 10.14.3 | 手機側邊選單點選後自動收起 | sidebar 收合 |

---

## 11. 管理員帳號管理 (Admin Accounts)

> stage：`AdminService` + `GET/POST/DELETE /api/admin/admins`。新增 admin 強制 `mustChangePW=true`。
> 對應測試：`backend/internal/service/admin_service_test.go`。

### 11.1 列表 `GET /api/admin/admins`

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 11.1.1 | 取得所有 admin 帳號 | 200，只回 role=admin，含 id/email/name/status/createdAt |
| 11.1.2 | 回傳不含 passwordHash | 任何欄位皆無密碼雜湊 |
| 11.1.3 | translator 角色呼叫 | 403 |

### 11.2 建立 admin `POST /api/admin/admins`

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 11.2.1 | 正常建立 | 成功，role="admin"、status="active" |
| 11.2.2 | 新建帳號 mustChangePW=true | 建立後該帳號首次登入須改密碼 |
| 11.2.3 | email 已存在 | 409，`EMAIL_TAKEN` |
| 11.2.4 | 密碼以 bcrypt 雜湊儲存 | DB 內非明文 |
| 11.2.5 | 缺少必填欄位（email/password/name）| 400，binding 驗證錯誤 |

### 11.3 刪除 admin `DELETE /api/admin/admins/:id`

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 11.3.1 | 刪除其他 admin | 成功 |
| 11.3.2 | 刪除自己的帳號 | 400，`CANNOT_DELETE_SELF` |
| 11.3.3 | 目標 ID 不存在 | 404，`ADMIN_NOT_FOUND` |
| 11.3.4 | 目標是 translator（非 admin） | 400，`NOT_AN_ADMIN` |
| 11.3.5 | 無效 ID 格式 | 400，`INVALID_ADMIN_ID` |

---

## 12. 病人管理 (Patient Management)

> stage 2/4：`patients` 表正規化，`(idType, idNumber)` 唯一。idNumber 儲存自動轉大寫+trim。
> 對應測試：`patient_service_test.go`、`patient_history_test.go`、`patient_translator_scope_test.go`、
> `repository/schedule_patient_repo_test.go`。

### 12.1 建立病人 `POST /api/admin/patients`

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 12.1.1 | 正常建立 | 成功，回傳病人含 id |
| 12.1.2 | idNumber 自動正規化（小寫+空白→大寫trim） | 儲存值為大寫且去前後空白 |
| 12.1.3 | (idType, idNumber) 重複 | 409，`PATIENT_DUPLICATE` |
| 12.1.4 | 同 idNumber 但不同 idType | 成功（唯一鍵是組合） |
| 12.1.5 | name/phone 前後空白 | 自動 trim |
| 12.1.6 | 缺少必填欄位 | 400，binding 驗證錯誤 |
| 12.1.7 | idType 非法值（非 passport/hn/unid） | 400 |

### 12.2 更新病人 `PUT /api/admin/patients/:id`

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 12.2.1 | 正常更新 | 成功 |
| 12.2.2 | 不改 idNumber 的 no-op 更新（自我排除） | 成功，不誤判重複 |
| 12.2.3 | 改成已被別人占用的 (idType, idNumber) | 409，`PATIENT_DUPLICATE` |
| 12.2.4 | 不存在的 ID | 404，`PATIENT_NOT_FOUND` |

### 12.3 刪除病人 `DELETE /api/admin/patients/:id`

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 12.3.1 | 刪除存在的病人 | 成功 |
| 12.3.2 | 不存在的 ID | 404，`PATIENT_NOT_FOUND` |

### 12.4 列表 / 搜尋 `GET /api/admin/patients`

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 12.4.1 | 無篩選列表 + 分頁 | 200，回 data + total |
| 12.4.2 | search 命中 name / phone / idNumber | 只回符合者 |
| 12.4.3 | page/pageSize 分頁 | 正確切片與 total |

### 12.5 病人歷史 `GET /api/admin/patients/:id/history`

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 12.5.1 | 跨多筆排班彙整就診紀錄，依日期 DESC | 回傳 patient + history[] |
| 12.5.2 | 每筆含 date/時段/location/翻譯員/status/noShowReason/診斷照片 | 欄位齊全 |
| 12.5.3 | 病人無任何排班 | history 為空陣列（仍 200） |
| 12.5.4 | 不存在的病人 ID | 404，`PATIENT_NOT_FOUND` |

### 12.6 翻譯員端病人清單 `GET /api/patients`（scope 限縮）

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 12.6.1 | 只回自己排班內出現過的病人 | 不含其他翻譯員的病人 |
| 12.6.2 | 自己沒有任何排班 | 回空清單 |
| 12.6.3 | search + 分頁在 scope 內生效 | 正確 |

---

## 13. 多病人排班 (Multi-Patient Schedule)

> stage 3：一筆 schedule 掛 1..N 個 SchedulePatient，每位病人有自己 start/end/status。
> 對應測試：`schedule_service_multipatient_test.go`、`schedule_excel_test.go`、`schedule_patient_repo_test.go`。

### 13.1 建立多病人排班 `POST /api/admin/schedules`

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 13.1.1 | 帶 1 位病人成功 | 成功，建立對應 SchedulePatient |
| 13.1.2 | 帶多位病人成功 | 全數寫入，order 依序 |
| 13.1.3 | patients 空清單 | 400，`SCHEDULE_PATIENTS_REQUIRED` |
| 13.1.4 | 同一排班重複同一病人 | 400，`DUPLICATE_PATIENT_IN_SCHEDULE` |
| 13.1.5 | 病人時段超出整體時段 | 400，`PATIENT_TIME_OUT_OF_RANGE` |
| 13.1.6 | 病人 end <= start | 400，`PATIENT_END_BEFORE_START` |
| 13.1.7 | patientId 不存在 | 400，`PATIENT_NOT_FOUND` |
| 13.1.8 | 建立在單一 transaction 內（中途失敗全回滾） | 失敗時不留半筆資料 |
| 13.1.9 | 向後相容：仍可用 legacy patientName | 成功（不走 SchedulePatient 路徑） |

### 13.2 更新多病人排班 `PUT /api/admin/schedules/:id`

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 13.2.1 | 更新時整份替換病人清單 | 舊 SchedulePatient 清掉、寫入新清單 |
| 13.2.2 | 替換清單沿用相同驗證規則 | 同 13.1.3~13.1.7 |

### 13.3 刪除連帶級聯 `DELETE /api/admin/schedules/:id` / `/group`

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 13.3.1 | 刪單筆排班連帶刪 schedule_patients | 無 FK 殘留 |
| 13.3.2 | 刪整組重複排班連帶刪各場 schedule_patients | 全組清乾淨 |
| 13.3.3 | 刪排班連帶刪關聯 checkins | 滿足 FK |

### 13.4 Excel V2 扁平匯入 `POST /api/admin/schedules/import`

> 欄位 `Code|TranslatorID|Date|OverallStart|OverallEnd|Location|PatientID|PatientStart|PatientEnd|Note`，相同 Code 合併成一筆多病人排班。

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 13.4.1 | 相同 Code 多列合併為一筆多病人排班 | 1 筆 schedule + N 個 SchedulePatient |
| 13.4.2 | 同 Code 內 meta（Date/時段/Location）衝突 | 該 Code 群組失敗 |
| 13.4.3 | 某群組內病人非法只跳過該群組 | 其他群組仍成功 |
| 13.4.4 | Code 空白 | 該列/群組拒絕 |
| 13.4.5 | 成功匯入確實寫入 DB | schedule + schedule_patients 落地 |
| 13.4.6 | 回傳成功/失敗明細 | 逐群組結果可辨識 |
| 13.4.7 | 未上傳檔案 | 400，`FILE_REQUIRED` |

---

## 14. 診斷證明 / 未到 / 結果總覽 (Diagnosis / No-show / Results)

> stage 4：打卡只是到場，逐病人的診斷證明才是服務證據。最多 3 張照片。
> 對應測試：`diagnosis_service_test.go`、`diagnosis_results_test.go`、`diagnosis_photos_get_test.go`。

### 14.1 翻譯員上傳診斷證明 `POST /api/checkins/diagnosis`

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 14.1.1 | 上傳 1~3 張照片成功 | 該 SchedulePatient status→completed |
| 14.1.2 | 既有 + 新增超過 3 張 | 400，`DIAGNOSIS_PHOTO_LIMIT` |
| 14.1.3 | 操作不屬於自己排班的病人 | 403，`DIAGNOSIS_NOT_OWNED` |
| 14.1.4 | SchedulePatient 不存在 | 404，`SCHEDULE_PATIENT_NOT_FOUND` |
| 14.1.5 | 上傳後可由離開 gate 放行 | 配合 4.1.20~4.1.21 |

### 14.2 翻譯員標記未到 `POST /api/checkins/no-show`

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 14.2.1 | 帶 reason 標記未到成功 | status→no_show，存 reason |
| 14.2.2 | 未帶 reason | 400，`NO_SHOW_REASON_REQUIRED` |
| 14.2.3 | 操作不屬於自己排班的病人 | 403，`DIAGNOSIS_NOT_OWNED` |

### 14.3 管理員代理操作 `POST /api/admin/diagnosis` / `/api/admin/no-show`

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 14.3.1 | 管理員代上傳診斷（無 ownership 限制） | 成功，status→completed |
| 14.3.2 | 管理員代標未到 | 成功，status→no_show |
| 14.3.3 | 管理員代標未到未帶 reason | 400，`NO_SHOW_REASON_REQUIRED` |
| 14.3.4 | 對不存在的 SchedulePatient | 404，`SCHEDULE_PATIENT_NOT_FOUND` |

### 14.4 診斷結果總覽 `GET /api/admin/diagnosis-results`

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 14.4.1 | 只列 terminal（completed/no_show），排除 pending | pending 不出現 |
| 14.4.2 | 依日期 DESC、再 startTime DESC、再 id DESC 排序 | 順序正確 |
| 14.4.3 | status 篩選 | 只回該狀態 |
| 14.4.4 | translatorId 篩選 | 只回該翻譯員 |
| 14.4.5 | 日期區間 dateFrom/dateTo 篩選 | 範圍內 |
| 14.4.6 | patientName 模糊搜尋 | LIKE 命中 |
| 14.4.7 | 分頁 page/pageSize（預設 20） | 正確切片 + total |
| 14.4.8 | 每筆含病人欄位 + 診斷照片（batch load 無 N+1） | 欄位齊全 |
| 14.4.9 | 每筆含 updatedAt | 有值 |

### 14.5 單一病人照片 `GET /api/admin/schedule-patients/:id/photos`

| # | 測試項目 | 預期結果 |
|---|---------|---------|
| 14.5.1 | 回傳該 SchedulePatient 的照片 URL（依上傳時間） | 陣列正確 |
| 14.5.2 | 尚無上傳 | 回空陣列 |
| 14.5.3 | SchedulePatient 不存在 | 404，`SCHEDULE_PATIENT_NOT_FOUND` |

---

## 附錄 A：測試統計

| 模組 | 測試項目數 |
|------|-----------|
| 認證模組 | 31 |
| 翻譯員管理 | 33 |
| 排班管理 | 49 |
| 打卡功能 | 53 |
| 匯出功能 | 22 |
| 稽核紀錄 | 26 |
| 中介層與安全性 | 21 |
| 追蹤 (Jaeger) | 13 |
| Cron 排程 | 12 |
| 前端 UI | 55 |
| 管理員帳號管理 | 13 |
| 病人管理 | 23 |
| 多病人排班 | 21 |
| 診斷證明 / 未到 / 結果 | 24 |
| **合計** | **396** |

---

## 附錄 B：測試優先級建議

### P0 — 必須（阻塞上線）
- 1.1.1~1.1.11（登入 + 鎖定）
- 1.2.1~1.2.9（改密碼）
- 7.1.1~7.1.7（JWT 認證）
- 7.2.1~7.2.7（強制改密碼）
- 7.3.1~7.3.4（角色權限）
- 4.1.1~4.1.7（打卡核心流程）
- 7.4.3~7.4.8（安全性核心）

### P1 — 重要（影響功能完整性）
- 2.2.1~2.2.8（建立翻譯員）
- 2.5.1~2.5.9（重設密碼）
- 3.2.1.1~3.2.4.9（排班建立全部）
- 4.1.8~4.1.18（打卡邊界條件）
- 4.6.1~4.7.6（管理員編輯/刪除打卡）
- 3.5.1~3.5.5（整組刪除）
- 5.5.1~5.5.6（手動匯出）

### P2 — 一般
- 其餘所有測試項目
