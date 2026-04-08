# Phase 1 修正與驗證報告

**日期：** 2026-04-08
**版本：** Phase 1 MVP

---

## 一、發現問題與修正摘要

### 1. antd v6 → v5 降版

| 項目 | 說明 |
|------|------|
| **問題** | `package.json` 安裝了 `antd@6.3.5`，但程式碼為 v5 API 撰寫 |
| **錯誤訊息** | `TypeError: J.some is not a function`（Select 內部 useMemo） |
| **修正** | `npm install antd@5`，重新 build frontend Docker image |
| **影響頁面** | 所有含 `Select`、`TimePicker` 元件的頁面（排班管理等） |

---

### 2. 前端 API 路徑錯誤

| 錯誤路徑 | 正確路徑 | 檔案 |
|----------|----------|------|
| `GET /api/translators` | `GET /api/admin/translators` | `src/api/translators.ts` |
| `POST /api/translators` | `POST /api/admin/translators` | `src/api/translators.ts` |
| `PUT /api/translators/:id` | `PUT /api/admin/translators/:id` | `src/api/translators.ts` |
| `PATCH /api/translators/:id/disable` | `DELETE /api/admin/translators/:id` | `src/api/translators.ts` |
| `GET /api/my/schedules` | `GET /api/schedules` | `src/api/schedules.ts` |

**根本原因：** 前端 API 函數撰寫時路徑與後端 Gin 路由不一致。

---

### 3. 後端回應格式包裝問題

| 項目 | 說明 |
|------|------|
| **問題** | 後端所有列表 API 統一回傳 `{"data": [...]}` 包裝，前端直接用 axios `r.data` 期望拿到陣列 |
| **錯誤訊息** | `TypeError: n.map is not a function` |
| **修正** | 在 `src/api/client.ts` 的 axios response interceptor 加入自動展開邏輯：若回應 body 只有單一 `data` 欄位，自動展開為其值 |

```ts
// client.ts interceptor
if (
  response.data !== null &&
  typeof response.data === 'object' &&
  'data' in response.data &&
  Object.keys(response.data).length === 1
) {
  response.data = response.data.data;
}
```

---

### 4. JSON 欄位命名 snake_case vs camelCase 不一致

| 項目 | 說明 |
|------|------|
| **問題** | 後端 Go DTO JSON tag 使用 snake_case，前端 TypeScript 型別使用 camelCase |
| **影響** | 所有欄位對應失敗：`must_change_pw`、`translator_id`、`start_time` 等均無法正確讀取 |
| **修正** | 修改後端四個 DTO 檔案，統一改為 camelCase JSON tag |

**修改的 DTO 欄位對照：**

| 檔案 | 原本 | 修正後 |
|------|------|--------|
| `dto/auth.go` | `must_change_pw` | `mustChangePW` |
| `dto/auth.go` | `old_password` / `new_password` | `oldPassword` / `newPassword` |
| `dto/schedule.go` | `translator_id`, `translator_name` | `translatorId`, `translatorName` |
| `dto/schedule.go` | `start_time`, `end_time` | `startTime`, `endTime` |
| `dto/schedule.go` | `patient_name`, `checkin_status` | `patientName`, `checkinStatus` |
| `dto/schedule.go` | `date_from`, `date_to` (query) | `dateFrom`, `dateTo` |
| `dto/translator.go` | `created_at` | `createdAt` |
| `dto/checkin.go` | `schedule_id`, `translator_id` | `scheduleId`, `translatorId` |
| `dto/checkin.go` | `checkin_time`, `selfie_url` | `checkinTime`, `selfieUrl` |
| `dto/checkin.go` | `environment_url`, `is_makeup` | `environmentUrl`, `isMakeup` |
| `dto/checkin.go` | `makeup_reason`, `created_at` | `makeupReason`, `createdAt` |

---

## 二、驗證結果

### 管理員功能

| 功能 | 測試方式 | 結果 |
|------|----------|------|
| 登入（admin@admin.com） | API + 瀏覽器 | ✅ 成功，正確跳轉 |
| 首次登入強制改密碼 | `mustChangePW: true` 重導向 | ✅ 正確觸發 |
| 翻譯員列表 | `GET /api/admin/translators` | ✅ 正確顯示 |
| 新增翻譯員 | `POST /api/admin/translators` | ✅ 成功建立 |
| 排班列表 | `GET /api/admin/schedules` | ✅ 正確顯示 |
| 新增排班 | `POST /api/admin/schedules` | ✅ 成功建立，回傳 camelCase |

### 翻譯員功能

| 功能 | 測試方式 | 結果 |
|------|----------|------|
| 登入（wang@test.com） | API | ✅ 成功 |
| 我的排班頁面 | `GET /api/schedules` | ✅ 正確顯示排班卡片（日期、時間、地點、病患） |
| 到達打卡頁面 | 點擊「到達打卡」按鈕 | ✅ 頁面正確顯示排班資訊、拍照區塊、GPS 定位區塊 |

### API 回應範例

**排班建立回應（修正後）：**
```json
{
  "data": {
    "id": 1,
    "translatorId": 3,
    "translatorName": "王小明",
    "date": "2026-04-08",
    "startTime": "09:00",
    "endTime": "11:00",
    "location": "台大醫院 3F 翻譯室",
    "patientName": "陳小華",
    "note": "泰語翻譯",
    "checkinStatus": "none"
  }
}
```

---

## 三、已知限制

| 項目 | 說明 |
|------|------|
| 瀏覽器自動化登入 | Ant Design Form 受控輸入不回應 DOM 事件注入，僅影響自動化測試工具，真實使用者輸入正常 |
| GPS 定位 | 瀏覽器需授予定位權限，本機測試環境會顯示「正在取得定位…」，實際裝置可正常取得 |
| 照片上傳 | 需要裝置相機，桌面瀏覽器可改用上傳圖片檔案 |

---

## 四、修改檔案清單

```
frontend/
  package.json                       # antd ^6 → ^5
  src/api/client.ts                  # 加入 data envelope 自動展開 interceptor
  src/api/translators.ts             # 修正 API 路徑、HTTP method
  src/api/schedules.ts               # 修正 /my/schedules → /schedules

backend/
  internal/dto/auth.go               # JSON tag → camelCase
  internal/dto/schedule.go           # JSON tag → camelCase
  internal/dto/translator.go         # JSON tag → camelCase
  internal/dto/checkin.go            # JSON tag → camelCase
```
