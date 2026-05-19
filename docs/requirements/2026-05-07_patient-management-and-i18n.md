# 需求文檔 — 病人管理、多病人排班、多語系

**版本**：v1.0 定稿
**日期**：2026-05-07
**狀態**：已確認，待實作

---

## 1. 目標總覽

| # | 功能 | 影響範圍 |
|---|------|---------|
| 1 | 多語系（en/zh-TW/th，預設英文）| 全前端 + 後端錯誤訊息 |
| 2 | 病人資料庫（姓名、電話、ID、歷史紀錄）| 新 model + 新頁面 |
| 3 | 排班支援多病人 + 每病人個別時段 | Schedule schema 重構 |
| 4 | 翻譯員看到自己排班的病人 | 翻譯員頁面 |
| 5 | 每病人診斷證明上傳（最多 3 張照片）| 新 model |
| 6 | 病人沒來標記 + 原因 | 上述同 |
| 7 | 拿掉環境照（保留自拍）| 打卡流程 |

---

## 2. 多語系（i18n）

### 2.1 範圍
- **包含**：所有前端 UI 文字、後端 API 錯誤訊息
- **不包含**：資料庫資料（病人姓名、地點等）

### 2.2 支援語言
| 代碼 | 語言 | 備註 |
|------|------|------|
| `en` | English | 預設 |
| `zh-TW` | 繁體中文 | |
| `th` | ภาษาไทย | |

### 2.3 行為
- 新使用者**一律預設英文**（不偵測瀏覽器語言）
- Header 右上角下拉切換，即時生效
- 切換選擇存 localStorage（不跟使用者帳號綁定，省 schema 變更）

### 2.4 技術選型建議
- 前端：`react-i18next`（業界標準、輕量、antd 也支援 ConfigProvider locale 切換）
- 後端：訊息 key 化，handler 回傳 `{"code": "EMAIL_TAKEN", "message": "..."}`，前端用 i18next 對應翻譯
  - 或後端依 `Accept-Language` header 回傳對應語言訊息（兩種都可，前者較乾淨）

---

## 3. 病人資料庫

### 3.1 Schema

```go
type Patient struct {
    ID        uint
    Name      string  // 必填
    Phone     string  // 必填
    IDType    string  // 必填，enum: "passport" | "hn" | "unid"
    IDNumber  string  // 必填
    CreatedAt time.Time
    UpdatedAt time.Time
    // (no Status — 只支援刪除)
}
```

唯一性建議：`(IDType, IDNumber)` 加 unique index，避免重複建立同一個人。

### 3.2 ID Type 說明（記錄於文檔，避免日後誤解）
| Code | 意義 |
|------|------|
| `passport` | 護照號碼 |
| `hn` | Hospital Number（醫院病歷號）|
| `unid` | 推測為難民識別編號（具體定義待醫院確認）|

### 3.3 權限
- **建立 / 編輯 / 刪除**：僅 admin
- **查看**：admin 看全部欄位；翻譯員只看姓名、電話、ID Number（隱藏其他）

### 3.4 歷史看診紀錄

**自動由「該病人出現過的排班 + 看診結果」彙總**，不開放手動新增。

顯示內容（按時間倒序）：
- 看診日期、時段
- 排班翻譯員
- 地點
- 結果狀態（**已上傳診斷證明** / **未到**）
- 診斷證明照片（縮圖）
- 沒來時：未到原因

### 3.5 看診結果

- 每**一次排班**一個結果（不是病人層級）
- 由翻譯員上傳的診斷證明照片構成
- 「沒來」也算一筆結果（標記未到 + 原因）

### 3.6 後台頁面：病人管理（`/admin/patients`）

- 列表：姓名、電話、ID Type、ID Number、建立時間、操作
- 搜尋：姓名、電話、ID Number
- 操作：新增、編輯、刪除（modal confirm）、查看歷史紀錄
- 點「歷史紀錄」開新頁或 Modal，顯示所有就診紀錄

---

## 4. 排班多病人改造

### 4.1 Schema 變動

**舊版**：
```go
type Schedule struct {
    // ...
    PatientName string
}
```

**新版**：
```go
type Schedule struct {
    // ...
    // PatientName 欄位移除
    // StartTime / EndTime 依舊存在，代表整個排班的時段範圍
}

type SchedulePatient struct {
    ID         uint
    ScheduleID uint
    PatientID  uint
    StartTime  string  // HH:mm，必須落在 Schedule.StartTime ~ EndTime 內
    EndTime    string  // HH:mm
    Order      int     // 顯示順序（按 StartTime 排即可，此欄位選擇性）
    // 看診狀態：
    Status     string  // enum: "pending" | "completed" | "no_show"
    NoShowReason string // 未到原因（Status=no_show 時）
    CreatedAt  time.Time
    UpdatedAt  time.Time
}

type DiagnosisPhoto struct {
    ID                uint
    SchedulePatientID uint   // 屬於哪個排班的哪個病人
    PhotoURL          string
    UploadedAt        time.Time
}
```

**Schedule.StartTime / EndTime 的意義**：
- 仍保留作為整個排班的時段（admin 在建立排班時填）
- 每個 SchedulePatient 的 (StartTime, EndTime) 必須落在 Schedule 範圍內
- 排班整體的 startTime/endTime 用於：到達/離開打卡的時間判斷、超時偵測

### 4.2 舊資料遷移
- 確認決議：**直接丟棄 `Schedule.PatientName`**
- 不做遷移腳本
- AutoMigrate 會把該欄位留著但變成 nullable（GORM 不會自動 drop column），不影響運作
- 若要徹底乾淨，可手動下 `ALTER TABLE schedules DROP COLUMN patient_name`（一次性操作，不寫進程式）

### 4.3 排班建立 / 編輯 UI

Admin 在排班 Modal 內：
- 翻譯員（下拉）
- 日期
- 整體時段（start ~ end）
- 地點
- **病人區塊**：可動態新增多筆，每筆包含：
  - 病人（從病人資料庫挑選，下拉 + 搜尋）
  - 個別時段（start ~ end，預設帶入整體時段）
- 備註

驗證：
- 至少一個病人
- 每個病人時段必須落在排班整體時段內
- 同一排班不可重複選同一個病人

### 4.4 Excel 批次匯入格式（Q3-4 我的提案）

採用「**扁平化 + 排班代號合併**」格式，一筆病人一列：

| A | B | C | D | E | F | G | H | I |
|---|---|---|---|---|---|---|---|---|
| 排班代號 | 翻譯員ID | 日期 | 整體開始 | 整體結束 | 地點 | 病人ID | 病人開始 | 病人結束 |
| SCH-001 | 3 | 2026-05-10 | 09:00 | 12:00 | 台大醫院 | 12 | 09:00 | 09:30 |
| SCH-001 | 3 | 2026-05-10 | 09:00 | 12:00 | 台大醫院 | 15 | 09:30 | 10:00 |
| SCH-002 | 5 | 2026-05-10 | 14:00 | 17:00 | 榮總 | 22 | 14:00 | 15:00 |

**邏輯**：相同「排班代號」會合併為一筆排班，多列代表多病人。

**好處**：一個 Excel 一次匯入多筆排班 + 多病人。

**缺點**：admin 要自己編排班代號，且必須在同一檔案內保持一致。

✅ **已確認採用扁平化方案。**

下載範本會更新為這個格式，含 2-3 列範例資料示範同代號合併。

---

## 5. 翻譯員流程改造

### 5.1 排班列表（MySchedules）

每個排班的展開內容增加：
- 病人清單：姓名、電話、ID Number、看診時段
- 看診狀態：未開始 / 進行中 / 已完成（依到達打卡狀態判斷）

### 5.2 到達打卡（CheckIn）

- 流程不變：自拍 → 送出
- **拿掉環境照**（拍照、上傳、顯示全部移除）
- 自拍仍必拍

### 5.3 病人看診處理（新增頁面或 Modal）

到達打卡完成後，排班頁面變成「進行中」狀態，每個病人列出：
- 病人資訊
- **動作按鈕**：
  - 「上傳診斷證明」→ 開 Modal，最多 3 張照片，送出後 Status=completed
  - 「標記未到」→ 開 Modal，填未到原因，送出後 Status=no_show

只支援照片格式（jpg/png/heic），不支援 PDF。

### 5.4 離開打卡（CheckOut）

- 前置檢查：**所有病人必須處於 completed 或 no_show 狀態**
  - 若還有 pending 病人 → 按鈕 disabled + tooltip 提示
- 通過後流程同到達打卡（自拍 → 送出）

### 5.5 補打卡（Makeup）— 採用方案 B

✅ **已確認採用方案 B：補打卡時允許處理病人，但不強制。**

具體行為：
- 補打卡頁面顯示該排班的病人清單
- 翻譯員可選擇上傳診斷證明 / 標記未到，皆為**選填**
- 沒處理的病人保持 `pending` 狀態
- Admin 後台「打卡紀錄」會顯示「有 N 個病人未處理」提示，便於後續收尾
- Admin 可在後台「打卡紀錄」內代為上傳或代為標記未到（新增 admin 補登功能）
- 補打卡原因（`makeupReason`）仍為必填

### 5.6 環境照處理

- **新打卡不再拍環境照**（前端拿掉 UI、後端不再要求 EnvironmentURL）
- **舊資料保留**（envrionment_url 欄位仍在 DB，舊紀錄詳情頁仍能看，但新紀錄為空）
- Admin Excel 匯出移除環境照欄位
- Admin 詳情 Modal 移除環境照區塊

---

## 6. API 變動清單

### 新增
| Method | Path | 用途 |
|--------|------|------|
| GET | `/api/admin/patients` | 列表（含搜尋）|
| POST | `/api/admin/patients` | 新增 |
| PUT | `/api/admin/patients/:id` | 編輯 |
| DELETE | `/api/admin/patients/:id` | 刪除 |
| GET | `/api/admin/patients/:id/history` | 看診歷史 |
| GET | `/api/patients` | 翻譯員可查（限自己排班內的病人）|
| POST | `/api/checkins/diagnosis` | 翻譯員上傳病人診斷證明（含 SchedulePatientID + 多張照片）|
| POST | `/api/checkins/no-show` | 標記病人未到（SchedulePatientID + reason）|

### 修改
| Path | 變動 |
|------|------|
| `POST /api/admin/schedules` | body 改為含 `patients: [{patientId, startTime, endTime}]` |
| `PUT /api/admin/schedules/:id` | 同上 |
| `POST /api/admin/schedules/import` | Excel 格式更動（見 4.4）|
| `POST /api/checkins` | 移除 environment_photo 欄位 |
| `POST /api/checkins/makeup` | 同上 |
| `GET /api/schedules` (翻譯員) | response 含 patients 陣列 |
| `GET /api/admin/checkins` | response 含 patients 陣列 + 診斷證明 |

### 移除
- 環境照相關所有欄位（保留 DB，僅前後端不再讀寫）

### 多語系錯誤訊息
所有錯誤 response 改為：
```json
{ "code": "PATIENT_NOT_FOUND", "message": "Patient not found" }
```
前端用 `code` 對應 i18n key，`message` 作為 fallback。

---

## 7. 資料庫 Schema 變動總結

### 新增 Tables
- `patients`
- `schedule_patients`
- `diagnosis_photos`

### 修改 Tables
- `schedules`：`patient_name` 變 nullable（不強制 drop）
- `checkins`：`environment_url` 變 nullable（保留歷史資料）

---

## 8. 階段拆分建議（實作順序）

**為了控制風險，建議分四個階段交付：**

### 階段 1：i18n 基礎建設（最快交付）
- react-i18next + ConfigProvider 接好
- 翻譯所有現有頁面（en/zh-TW/th）
- Header 加切換器
- 後端錯誤訊息 code 化

### 階段 2：病人資料庫
- 新 model、API、admin 頁面
- 不影響排班，純新功能

### 階段 3：排班多病人改造
- Schema 變動（新 SchedulePatient）
- 排班建立 / 編輯 UI 改造
- Excel 匯入格式更新
- 翻譯員排班頁面顯示病人

### 階段 4：打卡流程改造
- 拿掉環境照
- 加入病人診斷證明 / 未到流程
- 離開打卡前置檢查
- 補打卡流程調整（依 Q8-2 決議）
- 病人歷史看診紀錄頁面

---

## 9. 決議紀錄

| # | 議題 | 決議 |
|---|------|------|
| 1 | Excel 匯入格式 | 扁平化（同代號合併為一筆排班）|
| 2 | 補打卡與病人流程 | 方案 B（允許處理但不強制，admin 後台可代補登）|

需求已定稿，待開工。
