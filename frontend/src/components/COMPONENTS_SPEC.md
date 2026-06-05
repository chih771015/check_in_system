# components — 規格（輕量）

> 對應檔案：`frontend/src/components/*.tsx`
> 上層：[FRONTEND_SPEC](../FRONTEND_SPEC.md)

## 1. 定位與職責
跨頁共用的 UI 元件。多數為 controlled component（`value` + `onChange`），並**用 props 注入 API 函式以利測試**（預設指向真實 api）。

## 2. 元件清單

### AppLayout.tsx
受保護頁的外框（側欄/導覽 + 語言切換 + 登出）。依角色顯示不同選單。

### PatientPicker.tsx
typeahead `Select`：mount 載入病人、輸入時 **debounce 250ms** 重查（PAGE_SIZE 20）。`value`=patientId、`onChange(patientId)`。用於排班編輯掛病人。

### SchedulePatientListEditor.tsx
排班 modal 內的多病人列編輯：每列 = PatientPicker + start/end TimePicker。
- `value: SchedulePatientPayload[]` + `onChange`，`overallStart/End` 供新列預設值。
- 空清單首次渲染顯示一條空白列（使用者永遠有起點）。
- 用 `clampPatientTimes`（[utils](../FRONTEND_SPEC.md)）把時段夾進整體範圍，前端先擋 `PATIENT_TIME_OUT_OF_RANGE`。

### DiagnosisUploadModal.tsx
上傳 ≤3 張診斷照片到一個 SchedulePatient（multipart）。`MAX_PHOTOS=3` 前端先擋（後端 `DIAGNOSIS_PHOTO_LIMIT` 為最終守衛）。`upload` 可注入（測試）。成功 → `onUploaded()`。

### NoShowModal.tsx
輸入 no-show 原因並標記。原因必填：**前端 disable 送出**給即時回饋，後端 `NO_SHOW_REASON_REQUIRED` 為最終守衛。`markNoShow` 可注入。

### MapLink.tsx
顯示地址 + 可點地圖圖示：Apple 裝置 → Apple Maps，其餘 → Google Maps；`lat/lng` 皆 0（未定位）時不顯示連結。

## 3. 不變式
| 不變式 | 保證 |
|--------|------|
| 前端上限/必填只是 UX 先擋，真正守衛在後端 | 人工維持（兩邊規則須一致：3 張、reason 必填、時段範圍）|
| API 函式以 props 注入、預設真實實作 | 人工維持（測試靠此縫注 mock）|

## 4. 邊界條件
| 元件 | 邊界 |
|------|------|
| PatientPicker | 快速輸入 → debounce 取消前一查；無結果顯示 spin/空 |
| SchedulePatientListEditor | 刪到 0 列仍保一空白列；時段超範圍自動 clamp |
| DiagnosisUploadModal | 選超過 3 張 → 擋下 |
| MapLink | 0,0 座標 → 純文字無連結 |

## 5. 測試考量
`components/__tests__/`：DiagnosisUploadModal、NoShowModal、PatientPicker、SchedulePatientListEditor、MapLink 皆有測試，主要驗證互動 + 注入的 API mock 被正確呼叫。

## 6. 協作者
被 [pages](../pages/PAGES_SPEC.md) 使用；呼叫 [api](../api/API_CLIENT_SPEC.md)；文案走 [i18n](../i18n/I18N_SPEC.md)。
