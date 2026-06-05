# 翻譯員打卡系統 — 產品規格書（Product Spec）

> 版本：v2（2026-06）。本文件由資深產品經理視角撰寫，描述**完整產品藍圖**，並對每一項功能標註**實作狀態**。
> 撰寫原則：敘述用繁體中文，技術名詞（API、欄位、status、role…）用英文。
> 與本文件並行的技術規格書見 [ARCHITECTURE_SPEC.md](ARCHITECTURE_SPEC.md)。舊版產品概要保留在 [SPEC.md](SPEC.md)（未刪除）。

## 實作狀態圖例

| 標記 | 意義 |
|------|------|
| ✅ | 已實作且有測試 / E2E 覆蓋 |
| 🟡 | 部分實作（核心可用，但缺少原始規劃的部分子情境） |
| ⬜ | 未實作（僅資料欄位或藍圖，程式尚未接） |

---

## 1. 產品定位

一套供翻譯公司／NGO 使用的**現場翻譯出勤與服務紀錄系統**。翻譯員到醫院為病人（多為泰語、移工、難民）做現場翻譯；系統負責：

1. 管理員排班（可一場排班服務**多位病人**）。
2. 翻譯員到現場**打卡**（自拍照 + GPS）並逐一**上傳每位病人的診斷證明**或標記**未到（no_show）**。
3. 管理員查核出勤、查核診斷結果、匯出報表（Excel / Google Sheet）、定期自動寄送。
4. 稽核（audit log）、通知（LINE / Email 排程提醒）。

**不做什麼（產品邊界）**
- 不做病歷管理 / 醫療診斷本身（只存「診斷證明照片」當佐證）。
- 不做薪資結算（只匯出出勤資料供他系統計算）。
- 不做即時通訊 / 聊天。
- 不做翻譯員自助註冊（帳號一律由管理員建立）。

---

## 2. 角色與權限（Roles）

| 角色 | role 值 | 能做什麼 |
|------|---------|----------|
| **管理員 Admin** | `admin` | 管理 translator/admin/patient 帳號、排班、查核打卡與診斷結果、匯出、定期匯出設定、稽核日誌、代理上傳診斷／代標 no_show |
| **翻譯員 Translator** | `translator` | 查看**自己的**排班、打卡（到達/離開）、補打卡、逐病人上傳診斷照片 / 標記 no_show、查看自己的打卡歷史與統計、查看（受限的）病人清單 |

**權限機制（已實作 ✅）**
- 登入後發 JWT，內含 `user_id / role / must_change_pw`。
- Route group 以 middleware 把關：`JWTAuth` → `RequirePasswordChanged` → `RoleRequired(...)`。
- 跨資源存取一律驗證 ownership（例：翻譯員只能對自己 schedule 底下的 SchedulePatient 上傳診斷）。
- 詳見 [middleware spec](backend/internal/middleware/MIDDLEWARE_SPEC.md)。

---

## 3. 帳號與認證（Auth）

### 3.1 登入 ✅
- 方式：`email + password`，回傳 JWT + user 物件。
- 密碼以 **bcrypt** 雜湊儲存。
- **帳號鎖定（lockout）✅**：連續登入失敗達 `MAX_LOGIN_ATTEMPTS`（預設 5）次，鎖定 `LOCK_DURATION_MINUTES`（預設 15）分鐘；成功登入後計數歸零。鎖定期間回傳剩餘時間。
- 停用帳號（status=`disabled`）無法登入。

### 3.2 首次登入強制改密碼 ✅
- 新帳號 `must_change_pw=true`。
- 前端：`RequireAuth` 偵測 `mustChangePW` → 強制導向 `/change-password`。
- 後端：除了 change-password 本身外，所有受保護 route 套 `RequirePasswordChanged`，token 仍帶 must_change_pw 時回 403 `PASSWORD_CHANGE_REQUIRED`。
- 改密碼成功後**重新簽發 JWT**（清掉 must_change_pw claim），前端更新 user 狀態。

### 3.3 日常改密碼 ✅
- `POST /api/auth/change-password`：驗證舊密碼 → 更新 → 回新 JWT。

### 3.4 管理員重設翻譯員密碼 ✅
- `POST /api/admin/translators/:id/reset-password`：管理員覆寫他人密碼，目標帳號被強制 `must_change_pw=true`。
- 不可用此端點重設自己的密碼（`ErrCannotResetSelf`）。

### 3.5 緊急 CLI 重設密碼 ✅
- `server -reset-password <email> <newPassword>`：直接連 DB 重設，供 Docker 容器內救援。

### 3.6 個人設定頁（綁定 LINE/Telegram） ⬜
- 原始規劃的「個人設定」頁與綁定流程**未實作**。`line_user_id` 目前需由 DB/管理員手動寫入；`telegram_chat_id` 欄位存在但無任何程式使用。

---

## 4. 翻譯員帳號管理（Admin）✅

- `GET/POST/PUT/DELETE /api/admin/translators`
- 欄位：name、email、phone、password（初始密碼）、status（active/disabled）。
- 停用（DELETE = 設 status=disabled，**不真刪**）：保留歷史打卡與排班；翻譯員立即無法登入。
- 所有異動寫 audit log。

**情境**
- 新進報到 → 建帳號 → 線下告知初始密碼 → 翻譯員首登改密碼。
- 離職 → 停用帳號；未來排班仍在，管理員可自行刪除或重指派。

---

## 5. 管理員帳號管理（Admin）✅

- `GET/POST/DELETE /api/admin/admins`
- 新建 admin 一樣 `must_change_pw=true`。
- **不可刪除自己**（`ErrCannotDeleteSelf`）；只能刪 role=admin 的帳號。
- 系統開機時自動 seed 一個 `admin@admin.com`（密碼取自 `ADMIN_DEFAULT_PASSWORD`，未設則隨機產生並印在 log 一次）。

---

## 6. 病人管理（Patient）✅

> 原始 SPEC 沒有獨立病人實體；現行系統已將病人正規化成獨立 `patients` 表，排班改為關聯病人。

### 6.1 資料模型
| 欄位 | 說明 |
|------|------|
| name | 姓名 |
| phone | 電話 |
| idType | 證件類型：`passport`（護照）/ `hn`（病歷號）/ `unid`（難民/無證件） |
| idNumber | 證件號碼（**儲存時自動轉大寫 + trim**，比對不分大小寫） |

- **唯一性 ✅**：`(idType, idNumber)` 唯一。重複建立 / 改成已存在組合 → 409 `PATIENT_DUPLICATE`。

### 6.2 操作
- `GET/POST/PUT/DELETE /api/admin/patients`：搜尋（name/phone/idNumber）+ 分頁。
- `GET /api/admin/patients/:id/history` ✅：彙整該病人**所有就診紀錄**（跨排班）：日期、時段、地點、翻譯員、status（pending/completed/no_show）、no_show 原因、診斷證明照片。
- 翻譯員端 `GET /api/patients` ✅：**只看得到自己排班內出現過的病人**（scope 限縮），用於打卡 UI 顯示。

---

## 7. 排班管理（Schedule）

### 7.1 單場排班 + 多病人（multi-patient）✅
一筆 schedule 代表「某翻譯員、某天、某地點、某整體時段」，底下掛 1..N 個 **SchedulePatient**（每位病人有自己的 start/end 與 status）。

| 層級 | 欄位 |
|------|------|
| Schedule | translatorId、date、startTime、endTime（整體時段）、location、note |
| SchedulePatient | patientId、startTime、endTime（**須落在整體時段內**）、order、status、noShowReason |

**驗證規則（已實作 ✅）**
- 至少一位病人（多病人模式）：`SCHEDULE_PATIENTS_REQUIRED`。
- 每位病人 end > start：`PATIENT_END_BEFORE_START`。
- 每位病人時段 ⊆ 整體時段：`PATIENT_TIME_OUT_OF_RANGE`。
- 同一排班不可重複同一病人：`DUPLICATE_PATIENT_IN_SCHEDULE`。
- patientId 必須存在：`PATIENT_NOT_FOUND`。
- 建立 / 更新（替換整份病人清單）皆在單一 transaction 內完成。

**向後相容 🟡**：舊資料的單一 `patient_name`（free text）欄位仍保留可讀；新排班走 SchedulePatient。

### 7.2 週期排班（recurrence）✅
- 規則：`daily`、`weekly:1,3,5`（0=週日）、`monthly:5,20`。
- 由 start date 展開到 `recurrenceUntil`，同組共用 `recurrence_group_id`。
- `monthly` 對短月自動 clamp（例 31 → 2 月為 28/29）。
- 展開後每筆**各自獨立**，可單獨編輯/刪除。
- **限制 🟡**：週期排班目前走 legacy `patientName` 路徑，**尚未支援每場帶多病人**展開。

### 7.3 刪除
- `DELETE /api/admin/schedules/:id` ✅：刪單筆，先級聯刪 checkins + schedule_patients（滿足 FK）。
- `DELETE /api/admin/schedules/:id/group` ✅：刪同一 `recurrence_group_id` 全部；非群組則退化成刪單筆。

### 7.4 批次匯入（import）✅
- `POST /api/admin/schedules/import`
- V1：每列一筆單病人排班。
- V2：扁平 Excel，欄位 `Code|TranslatorID|Date|OverallStart|OverallEnd|Location|PatientID|PatientStart|PatientEnd|Note`，**相同 Code 合併成一筆多病人排班**；逐群組驗證，壞群組不影響其他群組，回傳成功/失敗明細。

### 7.5 篩選 ✅
- 管理員：date 範圍 / translator / location。
- 翻譯員 `GET /api/schedules`：只回自己的；預設今日與未來，可查歷史。

### 7.6 排班狀態（checkinStatus，前端顯示用）✅
由打卡紀錄推導：
- `none` 未打卡 ⬜
- `arrived` 已到達（只有 arrive）🟡
- `makeup` 有補打卡但尚未完成 🔵
- `completed` arrive + leave 皆有 ✅

---

## 8. 打卡（Check-in）

### 8.1 到達 / 離開打卡 ✅
`POST /api/checkins`（multipart form）

- 必填：`scheduleId`、`type`（arrive/leave）、`selfie`（**自拍照，必填**）。
- 選填：`environment`（環境照，**stage 4 後改為選填**）、`latitude/longitude`、`address`。
- GPS：前端用瀏覽器 Geolocation 取得座標，並以 **Nominatim** 反查地址；若前端沒帶 address，後端會再反查一次（失敗則略過，不擋打卡）。

**業務守衛（已實作 ✅）**
- 排班必須存在且屬於該翻譯員（`SCHEDULE_NOT_OWNED`）。
- 同一排班同一 type 不可重複打卡（`DUPLICATE_CHECKIN`）。
- 打 leave 前必須先有 arrive（`ARRIVE_BEFORE_LEAVE`）。
- **離開前置條件**：該排班所有 SchedulePatient 都不可是 `pending`（必須 completed 或 no_show），否則 `CHECKOUT_BLOCKED_BY_PENDING`（補打卡豁免此規則）。
- **逾時自動標記補打卡 ✅**：若打卡當下已超過排班 end time 且非主動補打卡，系統自動標 `is_makeup=true` 並補上原因「打卡時間超過排班結束時間（系統自動標記）」。

**紀錄欄位**：type、checkin_time（伺服器時間）、lat/lng、address、selfie_url、environment_url、is_makeup、makeup_reason。

### 8.2 補打卡（makeup）✅
`POST /api/checkins/makeup`：流程同上 + 必填**補打卡原因**；`is_makeup=true`，不需審核。GPS 記錄「補打卡當下」實際位置（可能非醫院）。

### 8.3 翻譯員打卡歷史 / 統計 ✅
- `GET /api/checkins`：自己的打卡列表（可帶日期範圍）。
- `GET /api/checkins/stats`：彙總 total / arrive / leave / makeup / onTime / late（late = arrive 晚於 start+5 分鐘）。

### 8.4 異常情境（已實作守衛）
| 情境 | 行為 |
|------|------|
| 重複打同 type | 擋下，回 `DUPLICATE_CHECKIN` ✅ |
| 未到達就打離開 | 擋下，`ARRIVE_BEFORE_LEAVE` ✅ |
| 還有 pending 病人就打離開 | 擋下，`CHECKOUT_BLOCKED_BY_PENDING` ✅ |
| 缺自拍照 | 擋下，`SELFIE_REQUIRED` ✅ |
| GPS 失敗 / 拒絕授權 | 前端進入 denied/timeout 狀態提示；後端不強制 GPS（可空座標打卡）🟡 |
| 反查地址第三方失敗 | 靜默略過，仍可打卡 ✅ |

---

## 9. 診斷證明與未到（Diagnosis / No-show）✅

> stage 4 新增：打卡只是「到場」，真正的服務證據是**逐病人的診斷證明**。

- 逐 SchedulePatient 操作：
  - `POST /api/checkins/diagnosis`（multipart，最多 **3 張** photo）→ status 變 `completed`。
  - `POST /api/checkins/no-show`（需 reason）→ status 變 `no_show`。
- 翻譯員只能操作自己排班下的病人（ownership 驗證）。
- 管理員代理：`POST /api/admin/diagnosis`、`POST /api/admin/no-show`（無 ownership 限制）。
- 照片上限 3 張：超過回 `DIAGNOSIS_PHOTO_LIMIT`。
- 與離開打卡連動：所有病人 completed/no_show 後才能離開（見 8.1）。

**診斷結果總覽（Admin）✅**
- `GET /api/admin/diagnosis-results`：列出所有 terminal（completed/no_show）的 SchedulePatient，可依 status / translator / 日期 / 病人姓名篩選 + 分頁，附診斷照片（batch load 避免 N+1）。
- `GET /api/admin/schedule-patients/:id/photos`：排班詳情 modal 看單一病人照片。

---

## 10. 後台查核與報表（Admin）

### 10.1 打卡紀錄查核 ✅
- `GET /api/admin/checkins`：篩 date / translator / type / is_makeup。
- `PUT /api/admin/checkins/:id` ✅：可修改 checkin_time / address / makeup_reason（**照片與關聯不可改**）。
- `DELETE /api/admin/checkins/:id` ✅。
- 照片可點開、GPS 可連到地圖（前端 MapLink）。

### 10.2 Excel 匯出 ✅
- `GET /api/admin/export/excel`：即時下載 `.xlsx`（excelize）。欄位含打卡 ID、翻譯員、type、時間、地址、GPS、照片 URL、是否補打卡、原因。

### 10.3 Google Sheet 匯出 ✅
- `POST /api/admin/export/google-sheet`：用 service account（`GOOGLE_CREDENTIALS_FILE`）建立新 Sheet 並回傳連結。未設定憑證則回錯。

### 10.4 定期匯出（periodic export）✅
- `GET/POST /api/admin/export/schedule`：設定每月第幾天、格式（excel/google_sheet）、收件 email、enabled。
- `POST /api/admin/export/schedule/run`：立即執行一次。
- Cron（每日 08:00）✅：當天符合 day_of_month 的設定 → 產生**上個月**報表 → email 寄出 → 更新 last_run_at。

### 10.5 Dashboard（今日概覽）⬜
- 原始規劃的管理員 Dashboard（今日應打卡/已到達/已完成/未打卡計數）**未實作**；管理員登入後預設導向 `/admin/translators`。

---

## 11. 稽核日誌（Audit Log）✅

- 管理員的關鍵操作（建/改/刪 translator、admin、patient、schedule、checkin、診斷代理…）寫入 `audit_logs`：admin_id、admin_name、action、target_type、target_id、detail、時間。
- `GET /api/admin/audit-logs`：分頁查詢。
- 設計原則：audit 寫入**永不阻斷主流程**（錯誤被吞）。

---

## 12. 通知（Notification）

| 情境 | 對象 | 時機 | 狀態 |
|------|------|------|------|
| 明日排程提醒 | 翻譯員 | 每日 07:00 cron，推送**隔天**排班 | ✅ LINE + Email |
| 打卡提醒（排班前 30 分） | 翻譯員 | — | ⬜ |
| 未打卡提醒（開始後 15 分） | 翻譯員 | — | ⬜ |
| 新排班通知 | 翻譯員 | 管理員新增排班時 | ⬜ |
| 補打卡通知 | 管理員 | 翻譯員補打卡時 | ⬜ |

- 管道：**LINE Messaging API push**（需 `LINE_CHANNEL_ACCESS_TOKEN` 且 user 有 `line_user_id`）✅；**Email**（需 SMTP 設定）✅。
- **Telegram** ⬜（僅 `telegram_chat_id` 欄位，無程式）。
- **LINE 綁定流程（QR/連結）** ⬜（line_user_id 需後台手動寫入）。

---

## 13. 國際化（i18n）✅

- 前端 i18next，支援 `en`（**預設 / fallback**）、`zh-TW`、`th`。
- 後端錯誤回傳**穩定 error code**（例 `DUPLICATE_CHECKIN`），前端對照 `errors.<code>` 翻譯；找不到才退回後端原文。
- Ant Design locale 同步切換。

---

## 14. 非功能需求（NFR）

| 項目 | 規格 | 狀態 |
|------|------|------|
| 使用者規模 | < 100 人 | — |
| RWD / 手機優先 | 翻譯員主要用手機 | ✅（responsive E2E）|
| 安全 | HTTPS、JWT（HS256，secret 須 ≥32 字且非預設，否則拒絕開機）、bcrypt、lockout | ✅ |
| 照片 | 多檔上傳存本機 `uploads/`，靜態服務 `/uploads`；保留 `PHOTO_RETENTION_DAYS`（預設 90）天後 cron 清除 | ✅ |
| 物件儲存（S3/GCS）| 原規劃雲端儲存 | ⬜（目前本機檔案系統）|
| 照片 URL 簽名 | 防未授權存取 | ⬜（目前靜態路徑可直接存取）|
| Tracing | OpenTelemetry → OTLP/gRPC → Jaeger；SQL/HTTP span，PII 清洗 | ✅ |
| 時區 | Asia/Taipei | ✅ |

---

## 15. 頁面清單（Frontend Routes）

### 翻譯員端
| 路徑 | 頁面 | 狀態 |
|------|------|------|
| `/login` | 登入 | ✅ |
| `/change-password` | 改密碼（首登強制）| ✅ |
| `/my-schedules` | 我的排班 + 打卡入口 + 逐病人診斷/no_show | ✅ |
| `/my-checkins` | 我的打卡歷史 / 統計 | ✅ |
| `/checkin/:scheduleId/:type` | 打卡（拍照 + GPS + 送出）| ✅ |
| `/makeup/:scheduleId/:type` | 補打卡（+ 原因）| ✅ |
| 個人設定（綁定通知）| — | ⬜ |

### 管理員端
| 路徑 | 頁面 | 狀態 |
|------|------|------|
| `/admin/translators` | 翻譯員管理（登入預設頁）| ✅ |
| `/admin/admins` | 管理員管理 | ✅ |
| `/admin/patients` | 病人管理 | ✅ |
| `/admin/patients/:id/history` | 病人就診歷史 | ✅ |
| `/admin/schedules` | 排班管理（含多病人、匯入、週期）| ✅ |
| `/admin/checkins` | 打卡紀錄查核 | ✅ |
| `/admin/diagnosis-results` | 診斷結果總覽 | ✅ |
| `/admin/export-settings` | 定期匯出設定 | ✅ |
| `/admin/audit-logs` | 稽核日誌 | ✅ |
| Dashboard | 今日概覽 | ⬜ |

---

## 16. 與原始 SPEC.md 的差異摘要

| 項目 | 原始 SPEC | 現況 |
|------|-----------|------|
| 病人 | 排班內 free-text patient_name | 獨立 `patients` 表 + 多病人 SchedulePatient ✅ |
| 服務證據 | 無 | 逐病人診斷證明照片（≤3）+ no_show ✅ |
| 離開打卡 | 直接打 | 需所有病人處理完才可離開 ✅ |
| 環境照 | 必填 | 改選填（自拍照仍必填）✅ |
| 通知 | LINE/Telegram 四種情境 | 僅「明日提醒」LINE+Email；其餘 ⬜，Telegram ⬜ |
| 個人設定/綁定 | 有 | ⬜ |
| Dashboard | 有 | ⬜ |
| i18n | 預留 | 已實作 en/zh-TW/th ✅ |
| 稽核 | 無 | audit log ✅ |
| 帳號鎖定 | 無 | lockout ✅ |
| Tracing | 無 | OTel + Jaeger ✅ |
| 照片儲存 | S3/GCS + 簽名 URL | 本機檔案 + 靜態服務（無簽名）🟡 |

---

## 17. 後續產品 backlog（建議優先序）

1. **LINE 綁定流程 + 個人設定頁**（解鎖所有 LINE 通知情境的前提）。
2. 通知情境補齊：排班前提醒、遲到未打卡提醒、新排班/補打卡通知。
3. 管理員 **Dashboard**（今日出勤概覽）。
4. 週期排班支援**多病人展開**。
5. 照片改雲端儲存 + 簽名 URL（資安強化）。
6. Telegram 管道（次要，台灣 LINE 為主）。
