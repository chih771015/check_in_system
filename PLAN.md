# 翻譯員打卡系統 — 開發計畫

---

## Phase 1：MVP — 帳號 + 排班 + 基本打卡

> 目標：最小可用版本，翻譯員能看排班、打卡，管理員能建帳號和排班。

### 1.1 專案初始化

- [ ] 建立 monorepo 結構
  ```
  /frontend   — React + TypeScript + Vite
  /backend    — Go module
  /docker     — Dockerfile + docker-compose.yml
  ```
- [ ] 前端：React + Vite + TypeScript 初始化，安裝 React Router、Axios、Ant Design (或 Tailwind)
- [ ] 後端：Go module 初始化，安裝 Gin + GORM + jwt-go + bcrypt
- [ ] Docker Compose：PostgreSQL + Go backend + React dev server
- [ ] 設定 .env 管理環境變數

### 1.2 資料庫

- [ ] 建立 migration 機制（golang-migrate 或 GORM AutoMigrate）
- [ ] 建立 users table
- [ ] 建立 schedules table
- [ ] 建立 checkins table
- [ ] Seed 一筆管理員帳號

### 1.3 認證模組

- [ ] `POST /api/auth/login` — 帳密登入，回傳 JWT
- [ ] `POST /api/auth/change-password` — 修改密碼
- [ ] JWT middleware（驗證 token、注入 user context）
- [ ] Role middleware（區分 admin / translator 權限）
- [ ] 前端：登入頁
- [ ] 前端：首次登入強制改密碼頁
- [ ] 前端：JWT 存儲 + Axios interceptor（自動帶 token、401 跳登入）

### 1.4 帳號管理（管理員）

- [ ] `GET /api/admin/translators` — 翻譯員列表
- [ ] `POST /api/admin/translators` — 新增翻譯員
- [ ] `PUT /api/admin/translators/:id` — 編輯翻譯員
- [ ] `DELETE /api/admin/translators/:id` — 停用翻譯員
- [ ] 前端：翻譯員管理頁（列表 + 新增 / 編輯 / 停用）

### 1.5 排班管理（管理員）

- [ ] `GET /api/admin/schedules` — 排班列表（支援日期、翻譯員、地點篩選）
- [ ] `POST /api/admin/schedules` — 新增單筆排班
- [ ] `PUT /api/admin/schedules/:id` — 編輯排班
- [ ] `DELETE /api/admin/schedules/:id` — 刪除排班
- [ ] 前端：排班管理頁（列表 + 篩選 + CRUD 表單）

### 1.6 我的排班（翻譯員）

- [ ] `GET /api/schedules` — 取得自己的排班（從 JWT 取 user ID）
- [ ] 前端：我的排班頁（今日/未來列表 + 打卡狀態標示）
- [ ] 前端：歷史排班切換

### 1.7 打卡功能

- [ ] 照片上傳 API — 接收檔案，存至 S3/GCS（或初期先存本地磁碟），回傳 URL
- [ ] `POST /api/checkins` — 打卡（到達/離開）
  - 接收：schedule_id、type（arrive/leave）、selfie、environment photo、GPS 座標
  - 驗證：該排班屬於此翻譯員、未重複打卡、到達卡必須在離開卡之前
  - 反查地址：呼叫 Google Geocoding API（或 OpenStreetMap Nominatim）
- [ ] 前端：打卡頁
  - 前鏡頭拍照（自拍）
  - 後鏡頭拍照（環境）
  - 瀏覽器 Geolocation API 取 GPS
  - 顯示地址預覽
  - 確認送出

### 1.8 Phase 1 收尾

- [ ] RWD 調整：確保手機瀏覽器可正常操作所有翻譯員功能
- [ ] 錯誤處理：API 統一錯誤格式、前端 toast 提示
- [ ] 基本 loading / empty state
- [ ] 手動測試完整流程

---

## Phase 2：核心完整 — 補打卡 + 後台紀錄 + Excel 匯出

> 目標：管理員能看到完整打卡紀錄並匯出，翻譯員能補打卡。

### 2.1 補打卡

- [ ] `POST /api/checkins/makeup` — 補打卡 API（同打卡，額外接收 makeup_reason）
- [ ] 前端：補打卡頁（同打卡頁 + 補打卡原因欄位）
- [ ] 排班列表中區分「補打卡」標記

### 2.2 後台打卡紀錄

- [ ] `GET /api/admin/checkins` — 打卡紀錄列表（支援篩選：日期、翻譯員、地點、類型、是否補打卡）
- [ ] `GET /api/admin/checkins/:id` — 打卡詳情
- [ ] 前端：打卡紀錄頁
  - 列表 + 多條件篩選
  - 展開詳情：照片預覽（可放大）、GPS 地圖嵌入（Google Maps embed）
  - 補打卡紀錄特別標示

### 2.3 Dashboard

- [ ] `GET /api/admin/dashboard` — 今日統計（應打卡數、已到達、已完成、未打卡）
- [ ] 前端：Dashboard 頁（數字卡片 + 快速連結）

### 2.4 Excel 匯出

- [ ] `GET /api/admin/export/excel` — 依篩選條件產生 .xlsx（使用 excelize）
- [ ] 匯出欄位：翻譯員 ID、姓名、打卡時間、地點、GPS 地址、自拍照 URL、環境照 URL、是否補打卡、原因
- [ ] 前端：報表匯出頁（篩選條件 + 下載按鈕）

---

## Phase 3：進階功能 — Google Sheet + 定期匯出 + 週期排班

> 目標：自動化匯出流程，減少排班重複操作。

### 3.1 Google Sheet 匯出

- [ ] 整合 Google Sheets API（Service Account 或 OAuth）
- [ ] `POST /api/admin/export/google-sheet` — 建立新 Sheet 並寫入資料，回傳連結
- [ ] 前端：匯出頁新增「匯出至 Google Sheet」按鈕 + 顯示產生的連結

### 3.2 定期匯出

- [ ] 建立 export_schedules table
- [ ] `POST /api/admin/export/schedule` — CRUD 定期匯出設定
- [ ] 後端排程（Go cron 或外部 cron job）：每日檢查是否有需要執行的定期匯出
- [ ] 匯出後寄送 Email（SMTP / SendGrid）或存至 Google Drive
- [ ] 前端：定期匯出設定表單

### 3.3 週期排班

- [ ] 排班新增 API 支援 recurrence_rule 參數
- [ ] 後端解析規則，自動展開產生多筆排班（共用 recurrence_group_id）
- [ ] 前端：排班表單加入「重複」選項（每日 / 每週特定幾天 / 每月特定日期）+ 結束日期
- [ ] 支援單筆刪除（不影響同組其他排班）

---

## Phase 4：通知整合 — LINE Bot / Telegram Bot

> 目標：翻譯員透過即時通訊收到排班和打卡提醒。

### 4.1 Bot 建立與綁定

- [ ] 建立 LINE Official Account + Messaging API channel（或 Telegram Bot）
- [ ] `POST /api/profile/bind-line` — 綁定 LINE（透過 webhook 取得 user ID）
- [ ] `POST /api/profile/bind-telegram` — 綁定 Telegram
- [ ] 前端：個人設定頁加入綁定入口（QR Code / 連結）

### 4.2 通知排程

- [ ] 後端排程：每分鐘掃描即將到來的排班
- [ ] 打卡提醒：排班開始前 30 分鐘，發送 LINE/Telegram 訊息
- [ ] 未打卡提醒：排班開始後 15 分鐘仍無到達卡，發送提醒
- [ ] 新排班通知：管理員新增排班時，即時推送給對應翻譯員
- [ ] 補打卡通知：翻譯員補打卡時，推送給管理員

### 4.3 訊息模板

- [ ] 設計各情境的訊息模板（含排班資訊摘要）
- [ ] LINE：使用 Flex Message 或純文字
- [ ] Telegram：使用 Markdown 格式

---

## 專案結構（預覽）

```
Thai/
├── frontend/
│   ├── src/
│   │   ├── pages/
│   │   │   ├── Login.tsx
│   │   │   ├── ChangePassword.tsx
│   │   │   ├── translator/
│   │   │   │   ├── MySchedules.tsx
│   │   │   │   ├── CheckIn.tsx
│   │   │   │   ├── MakeupCheckIn.tsx
│   │   │   │   └── Settings.tsx
│   │   │   └── admin/
│   │   │       ├── Dashboard.tsx
│   │   │       ├── TranslatorManagement.tsx
│   │   │       ├── ScheduleManagement.tsx
│   │   │       ├── CheckInRecords.tsx
│   │   │       └── ExportReport.tsx
│   │   ├── components/
│   │   ├── hooks/
│   │   ├── api/
│   │   ├── stores/
│   │   └── utils/
│   ├── package.json
│   └── vite.config.ts
├── backend/
│   ├── cmd/
│   │   └── server/
│   │       └── main.go
│   ├── internal/
│   │   ├── handler/      — HTTP handlers (controllers)
│   │   ├── service/      — Business logic
│   │   ├── repository/   — Database queries
│   │   ├── model/        — GORM models
│   │   ├── middleware/    — JWT, role check
│   │   ├── dto/          — Request/response structs
│   │   └── config/       — App config
│   ├── migrations/
│   ├── go.mod
│   └── go.sum
├── docker/
│   ├── Dockerfile.frontend
│   ├── Dockerfile.backend
│   └── docker-compose.yml
├── SPEC.md
├── USER_STORIES.md
└── PLAN.md
```

---

## 開發順序總覽

```
Phase 1 (MVP)
  ├── 專案初始化 & DB
  ├── 認證模組
  ├── 帳號管理
  ├── 排班 CRUD
  ├── 我的排班（翻譯員）
  └── 打卡（到達/離開 + 照片 + GPS）

Phase 2 (核心完整)
  ├── 補打卡
  ├── 後台打卡紀錄 + Dashboard
  └── Excel 匯出

Phase 3 (進階)
  ├── Google Sheet 匯出
  ├── 定期匯出
  └── 週期排班

Phase 4 (通知)
  ├── LINE / Telegram Bot 建立 + 綁定
  ├── 通知排程
  └── 訊息模板
```
