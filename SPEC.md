# 翻譯員打卡系統 — 產品規格書（原始版）

> ⚠️ **這是專案最初的原始需求規格書，僅供溯源參考，不再逐項維護。**
> 現況以 [`PRODUCT_SPEC.md`](PRODUCT_SPEC.md)（維護中的 living spec，含 ✅ 完成標記與後續新增功能）為準；
> 各層細節以對應的 `*SPEC.md`（[ARCHITECTURE_SPEC](ARCHITECTURE_SPEC.md) 為入口）為準。
> 本檔下方的 API 清單與 DB schema 屬最初設計，與後續多次 migration（病人金額、`schedules.date` 索引、金額統計等）已脫節，**勿據此實作**。

## 1. 專案概述

一套供翻譯公司使用的打卡管理系統，翻譯員可查看自己的排班、於到達/離開時打卡（含照片與 GPS），管理員可於後台管理排班、查看打卡紀錄並匯出報表。

---

## 2. 角色定義

| 角色 | 說明 |
|------|------|
| **管理員** | 翻譯公司管理員。可建立帳號、管理排班、查看所有打卡紀錄、匯出報表 |
| **翻譯員** | 現場翻譯人員。可查看自己的排班、執行打卡（到達/離開）、補打卡 |

---

## 3. 功能規格

### 3.1 帳號管理（管理員）

- 管理員可建立、編輯、停用翻譯員帳號
- 帳號欄位：
  - 翻譯員 ID（系統自動產生）
  - 姓名
  - Email
  - 電話
  - 密碼（由管理員設定初始密碼，翻譯員首次登入後可自行修改）
  - 帳號狀態（啟用 / 停用）
- 登入方式：帳號（Email）+ 密碼

### 3.2 排班管理（管理員）

- 管理員可新增、編輯、刪除排班
- 單筆排班欄位：

| 欄位 | 說明 | 範例 |
|------|------|------|
| 翻譯員 | 下拉選擇 | 王大明 |
| 日期 | 日期選擇器 | 2026-04-05 |
| 時間 | 開始 ~ 結束 | 09:00 ~ 11:00 |
| 地點 | 文字輸入 | 台大醫院 3F 翻譯室 |
| 病人姓名 | 文字輸入 | 陳小華 |
| 備註 | 文字輸入（選填） | 泰語翻譯 |

- **週期排班**：可設定重複規則（每日 / 每週特定幾天 / 每月特定日期），系統自動展開產生排班紀錄
- 管理員可在排班列表以「日期範圍 / 翻譯員 / 地點」篩選

### 3.3 排班查看（翻譯員）

- 翻譯員登入後看到 **自己的** 排班列表（不可看到其他人的排班）
- 預設顯示「今日 & 未來」的排班，可切換查看歷史
- 每筆排班顯示：日期、時間、地點、病人姓名、打卡狀態
- 打卡狀態標示：
  - ⬜ 未打卡
  - 🟡 已到達（尚未離開）
  - ✅ 已完成（到達 + 離開皆已打卡）
  - 🔵 已補打卡

### 3.4 打卡功能（翻譯員）

每筆排班有 **兩次打卡**：到達打卡 & 離開打卡。

#### 打卡流程

1. 翻譯員在排班列表點擊「到達打卡」或「離開打卡」
2. 系統要求上傳 **2 張照片**：
   - 自拍照（開啟前鏡頭拍攝）
   - 現場環境照（切換後鏡頭拍攝）
3. 系統自動取得 GPS 定位（經緯度 + 反查地址）
4. 翻譯員確認後送出
5. 系統記錄：打卡時間（伺服器時間）、GPS 座標、地址、2 張照片

#### 打卡資料模型

| 欄位 | 說明 |
|------|------|
| 打卡 ID | 系統自動產生 |
| 排班 ID | 關聯的排班 |
| 翻譯員 ID | 打卡人 |
| 打卡類型 | 到達 / 離開 |
| 打卡時間 | 伺服器時間戳 |
| GPS 經度 | float |
| GPS 緯度 | float |
| 地址 | 反查後的地址文字 |
| 自拍照 URL | 檔案儲存路徑 |
| 環境照 URL | 檔案儲存路徑 |
| 是否補打卡 | boolean |
| 補打卡備註 | 選填，補打卡時填寫原因 |

### 3.5 補打卡（翻譯員）

- 翻譯員可對「未打卡」的排班執行補打卡
- 補打卡流程與一般打卡相同（照片 + GPS），額外需填寫 **補打卡原因**
- 補打卡紀錄會標記為「補打卡」，不需管理員審核
- 管理員在後台可篩選查看所有補打卡紀錄

### 3.6 後台管理（管理員）

#### 3.6.1 打卡紀錄查看

- 列表顯示所有打卡紀錄
- 篩選條件：日期範圍、翻譯員、地點、打卡類型（到達/離開）、是否補打卡
- 每筆紀錄可展開查看詳情：照片預覽、GPS 地圖標記、打卡時間

#### 3.6.2 報表匯出

匯出欄位：

| 欄位 | 說明 |
|------|------|
| 翻譯員 ID | |
| 翻譯員姓名 | |
| 打卡時間 | 到達時間 & 離開時間 |
| 地點 | 排班地點 |
| GPS 地址 | 實際打卡地址 |
| 自拍照 URL | 可點擊開啟 |
| 環境照 URL | 可點擊開啟 |
| 是否補打卡 | 是/否 |
| 補打卡原因 | |

匯出格式：
- **Excel (.xlsx)**：即時下載
- **Google Sheet**：即時產生，回傳 Google Sheet 連結
- 支援 **定期匯出**：管理員可設定每月自動產生報表（寄送 Email 或存至指定 Google Drive）

### 3.7 通知功能

以 **LINE Bot** 或 **Telegram Bot** 擇一實作（建議先做 LINE Bot，台灣使用率較高）。

通知情境：

| 情境 | 對象 | 時機 |
|------|------|------|
| 打卡提醒 | 翻譯員 | 排班開始前 30 分鐘 |
| 未打卡提醒 | 翻譯員 | 排班開始後 15 分鐘仍未打到達卡 |
| 新排班通知 | 翻譯員 | 管理員新增排班時 |
| 補打卡通知 | 管理員 | 翻譯員執行補打卡時（僅通知，不需審核） |

翻譯員需在系統中綁定 LINE / Telegram 帳號。

---

## 4. 頁面清單

### 翻譯員端

| 頁面 | 說明 |
|------|------|
| 登入頁 | Email + 密碼 |
| 首次登入改密碼 | 強制修改初始密碼 |
| 我的排班 | 排班列表 + 打卡入口 |
| 打卡頁 | 拍照 + GPS + 確認送出 |
| 補打卡頁 | 同打卡頁 + 原因欄位 |
| 個人設定 | 修改密碼、綁定 LINE/Telegram |

### 管理員端

| 頁面 | 說明 |
|------|------|
| 登入頁 | 同上 |
| Dashboard | 今日打卡概覽（已打卡/未打卡人數） |
| 翻譯員管理 | CRUD 翻譯員帳號 |
| 排班管理 | CRUD 排班 + 週期排班設定 |
| 打卡紀錄 | 列表 + 篩選 + 詳情展開 |
| 報表匯出 | 篩選條件 + 匯出按鈕 + 定期匯出設定 |

---

## 5. 技術架構

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   Frontend   │────▶│   Backend    │────▶│   Database   │
│  React SPA   │     │   Go (Gin/   │     │  PostgreSQL  │
│  (RWD)       │◀────│   Echo)      │◀────│              │
└──────────────┘     └──────┬───────┘     └──────────────┘
                            │
                    ┌───────┼───────┐
                    ▼       ▼       ▼
              ┌────────┐ ┌──────┐ ┌──────────┐
              │ Object │ │ LINE │ │ Google   │
              │Storage │ │ Bot  │ │ Sheets   │
              │(S3/GCS)│ │ API  │ │ API      │
              └────────┘ └──────┘ └──────────┘
```

| 層級 | 技術選擇 |
|------|----------|
| Frontend | React + TypeScript + Vite, RWD（手機優先） |
| Backend | Go + Gin (or Echo) + GORM |
| Database | PostgreSQL |
| 檔案儲存 | AWS S3 或 GCP Cloud Storage（照片） |
| 認證 | JWT Token |
| 通知 | LINE Messaging API / Telegram Bot API |
| 匯出 | excelize (Go Excel library) + Google Sheets API |
| 部署 | Docker + Docker Compose |

---

## 6. API 概要

### Auth
- `POST /api/auth/login` — 登入
- `POST /api/auth/change-password` — 修改密碼

### 翻譯員帳號（管理員）
- `GET /api/admin/translators` — 列表
- `POST /api/admin/translators` — 新增
- `PUT /api/admin/translators/:id` — 編輯
- `DELETE /api/admin/translators/:id` — 停用

### 排班（管理員）
- `GET /api/admin/schedules` — 列表（支援篩選）
- `POST /api/admin/schedules` — 新增（含週期規則）
- `PUT /api/admin/schedules/:id` — 編輯
- `DELETE /api/admin/schedules/:id` — 刪除

### 排班（翻譯員）
- `GET /api/schedules` — 我的排班列表

### 打卡
- `POST /api/checkins` — 打卡（到達/離開）
- `POST /api/checkins/makeup` — 補打卡
- `GET /api/admin/checkins` — 打卡紀錄列表（管理員，支援篩選）
- `GET /api/admin/checkins/:id` — 打卡詳情

### 匯出（管理員）
- `GET /api/admin/export/excel` — 下載 Excel
- `POST /api/admin/export/google-sheet` — 產生 Google Sheet
- `POST /api/admin/export/schedule` — 設定定期匯出

### 通知
- `POST /api/profile/bindline` — 綁定 LINE
- `POST /api/profile/bind-telegram` — 綁定 Telegram

---

## 7. 資料庫 Schema（概要）

```sql
-- 使用者
CREATE TABLE users (
    id            SERIAL PRIMARY KEY,
    email         VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name          VARCHAR(100) NOT NULL,
    phone         VARCHAR(20),
    role          VARCHAR(20) NOT NULL,  -- 'admin' | 'translator'
    status        VARCHAR(20) DEFAULT 'active',  -- 'active' | 'disabled'
    must_change_pw BOOLEAN DEFAULT TRUE,
    line_user_id   VARCHAR(255),
    telegram_chat_id VARCHAR(255),
    created_at    TIMESTAMP DEFAULT NOW(),
    updated_at    TIMESTAMP DEFAULT NOW()
);

-- 排班
CREATE TABLE schedules (
    id              SERIAL PRIMARY KEY,
    translator_id   INT REFERENCES users(id),
    date            DATE NOT NULL,
    start_time      TIME NOT NULL,
    end_time        TIME NOT NULL,
    location        VARCHAR(500) NOT NULL,
    patient_name    VARCHAR(100) NOT NULL,
    note            TEXT,
    recurrence_rule VARCHAR(255),  -- 'daily' | 'weekly:1,3,5' | 'monthly:5,20' | NULL
    recurrence_group_id UUID,     -- 同一組週期排班的 group ID
    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW()
);

-- 打卡
CREATE TABLE checkins (
    id              SERIAL PRIMARY KEY,
    schedule_id     INT REFERENCES schedules(id),
    translator_id   INT REFERENCES users(id),
    type            VARCHAR(10) NOT NULL,  -- 'arrive' | 'leave'
    checkin_time    TIMESTAMP NOT NULL,
    latitude        DECIMAL(10, 7),
    longitude       DECIMAL(10, 7),
    address         VARCHAR(500),
    selfie_url      VARCHAR(1000) NOT NULL,
    environment_url VARCHAR(1000) NOT NULL,
    is_makeup       BOOLEAN DEFAULT FALSE,
    makeup_reason   TEXT,
    created_at      TIMESTAMP DEFAULT NOW()
);

-- 定期匯出設定
CREATE TABLE export_schedules (
    id              SERIAL PRIMARY KEY,
    admin_id        INT REFERENCES users(id),
    frequency       VARCHAR(20) NOT NULL,  -- 'monthly'
    day_of_month    INT,
    export_format   VARCHAR(20) NOT NULL,  -- 'excel' | 'google_sheet'
    email_to        VARCHAR(255),
    google_drive_folder_id VARCHAR(255),
    created_at      TIMESTAMP DEFAULT NOW()
);
```

---

## 8. 非功能需求

| 項目 | 規格 |
|------|------|
| 使用者規模 | < 100 人 |
| RWD | 手機優先設計（翻譯員主要用手機操作） |
| 瀏覽器支援 | Chrome、Safari（最新兩版） |
| 照片大小限制 | 單張 < 5MB，上傳前前端壓縮 |
| GPS 精度 | 使用瀏覽器 Geolocation API |
| 安全性 | HTTPS、JWT、密碼 bcrypt 雜湊、照片 URL 簽名（防止未授權存取） |
| 語言 | 繁體中文（UI），可預留 i18n 架構 |

---

## 9. 開發階段建議

| 階段 | 內容 | 預估範圍 |
|------|------|----------|
| **Phase 1** | 帳號管理 + 排班 CRUD + 基本打卡（到達/離開 + 照片 + GPS） | MVP |
| **Phase 2** | 補打卡 + 後台打卡紀錄查看 + Excel 匯出 | 核心完整 |
| **Phase 3** | Google Sheet 匯出 + 定期匯出 + 週期排班 | 進階功能 |
| **Phase 4** | LINE Bot / Telegram Bot 通知 | 通知整合 |
