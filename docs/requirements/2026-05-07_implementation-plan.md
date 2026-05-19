# 實作計畫 — 病人管理、多病人排班、多語系

**對應需求文檔**：`docs/requirements/2026-05-07_patient-management-and-i18n.md`
**日期**：2026-05-07

---

## 階段間依賴順序

```
階段 1 (i18n)  ──┐
                 ├──→ 階段 3 (排班多病人) ──→ 階段 4 (打卡流程)
階段 2 (病人) ───┘                              ↑
       │                                         │
       └──── 歷史紀錄 placeholder ───────────────┘
              (階段 2 留接口，階段 4 填實作)
```

- 階段 1 與階段 2 **可完全並行**
- 階段 3 強依賴階段 2（排班 Modal 內的 PatientPicker 必須等病人 CRUD 完成）
- 階段 4 強依賴階段 3（診斷證明掛在 SchedulePatient 上）

---

## 階段 1：i18n 基礎建設（工作量：中）

### 後端
- `dto/error.go`（新）：統一 `ErrorResponse{ Code, Message }` + 錯誤碼常量
- 全部 handler 改寫：`c.JSON(4xx, gin.H{"error": "..."})` → `c.JSON(4xx, dto.NewError(code, msg))`
- service 改用 sentinel error，handler 做 map

### 前端
- `package.json` 加 `react-i18next` / `i18next` / `i18next-browser-languagedetector`
- 新增 `frontend/src/i18n/index.ts` + `locales/en.json` / `zh-TW.json` / `th.json`
- `main.tsx`：i18n 初始化 + antd ConfigProvider locale 動態切換
- `AppLayout.tsx`：Header 加語言切換器（寫 localStorage）
- 所有 pages 字串改 `t('...')`
- `api/client.ts`：axios interceptor 攔 `{code, message}` → 翻譯

### Commits（9 個）
1. `feat(backend): add error code dto and sentinel errors`
2. `refactor(backend): migrate auth/admin handlers to error code response`
3. `refactor(backend): migrate schedule/checkin/translator handlers to error code response`
4. `feat(frontend): bootstrap react-i18next with en/zh-TW/th resources`
5. `feat(frontend): add language switcher in AppLayout header`
6. `refactor(frontend): localize admin pages`
7. `refactor(frontend): localize translator pages`
8. `feat(frontend): map backend error codes to i18n keys via axios interceptor`
9. `docs(changelog): record i18n rollout`

### 驗證 6 項
1. 預設語言進站永遠是英文（清 localStorage 後刷新驗證）
2. 切到 zh-TW 後刷新仍為 zh-TW（持久化）
3. 切語言時 antd DatePicker / Modal confirm 文字也跟著切
4. 故意輸錯密碼，三語各看到對應翻譯
5. 後端回傳未知 code 時 fallback 顯示 message
6. 三語 JSON key 完整對齊，無缺漏

### 風險
- antd ConfigProvider 切換不重渲染 → 用 React state 包 locale
- 泰文長度撐爆 button → 翻完逐頁視覺 review
- 後端 sentinel error 漏接 → wrap helper `respondError(c, err)` 集中 map

---

## 階段 2：病人資料庫（工作量：中）

### 後端
- `model/patient.go`（新）：Patient + unique index `(id_type, id_number)`
- `dto/patient.go`（新）：CRUD DTO + History DTO
- `repository/patient_repo.go`（新）：含 `WithCtx`, `Create/Update/Delete/FindByID/List/FindByIDTypeAndNumber`
- `service/patient_service.go`（新）：CRUD + 重複檢查 + history **placeholder**（階段 4 才實作真實聚合）
- `handler/patient_handler.go`（新）：admin CRUD + translator GET
- `main.go`：`AutoMigrate(&Patient{})` + 路由註冊
  - `GET/POST /api/admin/patients`
  - `PUT/DELETE /api/admin/patients/:id`
  - `GET /api/admin/patients/:id/history`（先回空陣列）
  - `GET /api/patients`（翻譯員用，階段 3 才收斂為「自己排班內」）

### 前端
- `types/index.ts` 加 `Patient`, `IDType`, `PatientHistory`
- `api/patients.ts`（新）
- `pages/admin/PatientManagement.tsx`（新）+ `PatientHistory.tsx`（placeholder）
- 三語 JSON 補 `patients.*`
- `App.tsx` 加路由與 menu 項

### Commits（9 個）
1. `feat(backend): add patient model and migration`
2. `feat(backend): add patient repository`
3. `feat(backend): add patient service with duplicate check`
4. `feat(backend): add patient admin/translator handlers and routes`
5. `feat(frontend): add patient types and api client`
6. `feat(frontend): add admin patient management page`
7. `feat(frontend): add patient history placeholder view`
8. `feat(frontend): add patient menu entry and i18n keys`
9. `docs(changelog): patient database stage 2`

### 驗證 6 項
1. 重複 `(idType, idNumber)` 建立 → `PATIENT_DUPLICATE` 錯誤翻譯
2. 翻譯員打 `/api/admin/patients` → 403
3. 翻譯員打 `/api/patients` → 看得到病人但無建立時間
4. 刪除病人 → 列表更新
5. 搜尋（姓名/電話/ID）皆命中
6. 三種 idType 在新增/編輯/列表 i18n 化

### 風險
- idNumber 大小寫敏感 → 入庫前 `strings.ToUpper`
- 翻譯員 `/api/patients` 暫無權限收斂 → 階段 3 補上，issue 記錄
- 列表分頁 → 後端一開始就支援 page/pageSize

---

## 階段 3：排班多病人改造（工作量：大）

### 後端
- `model/schedule_patient.go`（新）：含 Status enum
- `model/diagnosis_photo.go`（新）：schema only，階段 4 才用
- `model/schedule.go`：`PatientName` 改 `*string`、加 `Patients []SchedulePatient` 關聯
- `dto/schedule.go`：請求/回應加 `Patients []SchedulePatientPayload`
- `dto/schedule_patient.go`（新）
- `repository/schedule_patient_repo.go`（新）
- `repository/diagnosis_photo_repo.go`（新，schema only）
- `repository/schedule_repo.go`：load 時 Preload Patients
- `service/schedule_service.go`：
  - Create/Update 改 transactional（schedule + patients 同一 transaction）
  - 驗證：至少 1 病人、無重複、子時段落在整體內
  - **Excel 匯入**改寫：按 column A 合併同代號為一筆 schedule
- 翻譯員 `GET /api/patients` 收斂為「join SchedulePatient → 自己排班內」

### 前端
- `types/index.ts` 加 `SchedulePatient`, `SchedulePatientStatus`
- `api/schedules.ts` payload 更新
- `components/PatientPicker.tsx`（新）：下拉 + 搜尋
- `components/SchedulePatientListEditor.tsx`（新）：動態加減列
- `pages/admin/ScheduleManagement.tsx`：Modal 加病人區塊
- `pages/translator/MySchedules.tsx`：展開列顯示病人清單
- Excel 範本下載按鈕指向新格式

### Commits（10 個）
1. `feat(backend): add schedule_patient and diagnosis_photo models with migration`
2. `feat(backend): add schedule_patient and diagnosis_photo repositories`
3. `refactor(backend): schedule service supports multi-patient create/update with validation`
4. `refactor(backend): excel import merges rows by schedule code`
5. `refactor(backend): schedule handlers and translator handler return patient list`
6. `feat(backend): restrict GET /api/patients to translator's own schedules`
7. `feat(frontend): add PatientPicker and SchedulePatientListEditor components`
8. `refactor(frontend): admin ScheduleManagement supports multi-patient`
9. `refactor(frontend): translator MySchedules shows patient list per schedule`
10. `docs(changelog): multi-patient schedule rollout`

### 驗證 7 項
1. 建立排班含 3 病人 → DB 三筆 schedule_patients
2. 子時段超出整體 → 拒絕，回 `SCHEDULE_PATIENT_TIME_OUT_OF_RANGE`
3. 同排班重複病人 → 拒絕
4. 編輯排班移除某病人 → DB 該列被刪
5. Excel 兩列同代號 → 1 排班 2 病人
6. 翻譯員看不到別人排班的病人
7. 舊資料（PatientName 還有值）詳情頁不爆錯

### 風險
- 舊 `patient_name` 殘留 → 列表 fallback 顯示
- Transaction 邊界 → 同一 `db.Transaction()` 寫
- Excel 排班代號為空或不一致 → 明確錯誤行號回前端
- PatientPicker 大量資料卡頓 → 搜尋 API 限 20 筆 + debounce
- `UNIQUE(schedule_id, patient_id)` 必須加

---

## 階段 4：打卡流程改造（工作量：極大）

### 後端
- `model/checkin.go`：`EnvironmentURL` 改 `*string`
- `dto/checkin.go`：移除 environment_photo、新增 UploadDiagnosis / NoShow / AdminMakeupPatient
- `repository/checkin_repo.go`：preload SchedulePatient + DiagnosisPhotos
- `repository/diagnosis_photo_repo.go`：補實 Create/Find/Count
- `service/checkin_service.go`：
  - CheckIn/CheckOut 不再要求 environment_photo
  - CheckOut 前置：所有 SchedulePatient 必須 completed/no_show，否則 `CHECKOUT_BLOCKED_PENDING_PATIENTS`
  - Makeup：reason 仍必填，patient 處理選填
- 新增診斷上傳 + 標記未到 service
- `service/patient_service.go`：完成 history 真實聚合（join schedule + schedule_patient + diagnosis_photo）
- `handler/checkin_handler.go`：移除 env 處理 + 新增 diagnosis/no-show endpoints
- `handler/admin_handler.go`：加 admin 代補登 endpoint
- Excel 匯出移除環境照欄位、加診斷證明列數欄位

### 前端
- `types/index.ts` 加 `DiagnosisPhoto`
- `api/checkins.ts`：移除 env、加 uploadDiagnosis / markNoShow / adminUploadDiagnosis / adminMarkNoShow
- `components/DiagnosisUploadModal.tsx`（新，最多 3 張、image only）
- `components/NoShowModal.tsx`（新）
- `pages/translator/CheckIn.tsx`：移除環境照
- `pages/translator/MakeupCheckIn.tsx`：移除環境照 + 加病人處理區塊（選填）
- `pages/translator/MySchedules.tsx`：每病人「上傳診斷證明」/「標記未到」按鈕；CheckOut 視 pending 數量 disabled
- `pages/admin/CheckinRecords.tsx`：移除環境照、加「未處理病人 N」提示、admin 代補登
- `pages/admin/PatientHistory.tsx`：完成真實渲染

### Commits（13 個）
1. `refactor(backend): make environment_url nullable and stop requiring on new checkins`
2. `feat(backend): diagnosis photo and no-show service logic`
3. `feat(backend): checkout pre-check blocks pending patients`
4. `feat(backend): translator endpoints for upload diagnosis / mark no-show`
5. `feat(backend): admin endpoints for surrogate diagnosis / no-show`
6. `feat(backend): patient history aggregation completed`
7. `refactor(backend): export removes environment column`
8. `refactor(frontend): remove environment photo from checkin and makeup`
9. `feat(frontend): diagnosis upload and no-show modals`
10. `feat(frontend): MySchedules per-patient action and checkout gating`
11. `feat(frontend): admin makeup actions in CheckinRecords`
12. `feat(frontend): real patient history view`
13. `docs(changelog): checkin flow phase 4`

### 驗證 8 項
1. 新打卡完全沒有環境照欄位（Network 確認）
2. CheckOut 有 pending 病人 → 按鈕 disabled + 強打 API 也擋
3. 上傳第 4 張診斷證明 → 後端拒絕
4. 非 image MIME → 拒絕
5. Makeup 不處理病人也能提交，admin 看到「未處理 N」
6. Admin 代上傳 → 病人 history 顯示且 audit log 標記 surrogate
7. 病人 history 按時間倒序，no_show + completed 都列入
8. 三語錯誤訊息皆翻譯

### 風險
- 舊環境照資料 → dto 保留，有值才顯示
- CheckOut race → server 再驗一次
- Audit log 必須記 admin 代補登（surrogate flag）
- 歷史紀錄 N+1 → join + preload
- HEIC 上傳 → MIME 白名單 `image/jpeg|png|heic|heif`

---

## 總工程量

| 階段 | 工作量 | 後端 commits | 前端 commits | 並行 |
|------|--------|------------|------------|------|
| 階段 1 i18n | 中 | 3 | 5 | 可與階段 2 並行 |
| 階段 2 病人 | 中 | 4 | 4 | 可與階段 1 並行 |
| 階段 3 排班多病人 | 大 | 6 | 3 | 需階段 2 |
| 階段 4 打卡流程 | 極大 | 7 | 6 | 需階段 3 |

**總計：約 38 個 commits，工程相對單位 1+1+2+3 = 7。**

---

## 建議節奏

| 週 | 內容 |
|----|------|
| W1 | 階段 1 + 階段 2 並行 |
| W2 | 階段 3 後端 → 前端 → Excel 匯入（風險高放最後）|
| W3 | 階段 4 後端（env 下線 → diagnosis → checkout gating → admin 代補登 → history）+ 前端 |
| W4 | 緩衝：跨階段回歸、三語視覺、舊資料相容、changelog 收尾 |

**每階段結束打 git tag（`v-phase1-i18n` 等），方便回滾。階段 3、4 上線前 staging 跑完整 happy path + 失敗 path。**
