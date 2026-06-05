# pages — 規格（輕量）

> 對應檔案：`frontend/src/pages/**`
> 上層：[FRONTEND_SPEC](../FRONTEND_SPEC.md)

## 1. 定位與職責
路由對應的畫面，組合 Ant Design 元件 + [api](../api/API_CLIENT_SPEC.md) + [components](../components/COMPONENTS_SPEC.md)。商業邏輯盡量在後端；頁面負責互動、表單驗證（前端先擋）、狀態顯示。

## 2. 公用頁
| 檔案 | 路由 | 重點 |
|------|------|------|
| `Login.tsx` | /login | email+密碼登入；401 由本頁顯示 toast（client 不導向）|
| `ChangePassword.tsx` | /change-password | 首登強制 / 日常改密碼；成功後 `updateUser({mustChangePW:false})` |

## 3. 翻譯員頁（`pages/translator/`）
| 檔案 | 路由 | 重點 |
|------|------|------|
| `MySchedules.tsx` | /my-schedules | 自己的排班；每病人可開 [DiagnosisUploadModal](../components/COMPONENTS_SPEC.md) / [NoShowModal]；打卡入口 |
| `MyCheckins.tsx` | /my-checkins | 打卡歷史 + 統計 |
| `CheckIn.tsx` | /checkin/:scheduleId/:type | **打卡頁**：自拍照 + GPS（用 [useGeolocation ★](../hooks/HOOKS_SPEC.md)）；送出 disabled until GPS success |
| `MakeupCheckIn.tsx` | /makeup/:scheduleId/:type | 補打卡（+ 原因）|

> ⚠️ 與原始 SPEC 不同：CheckIn 頁**只收自拍照（selfie）+ GPS**，環境照（environment）UI 未提供（後端為選填）。

## 4. 管理員頁（`pages/admin/`）
| 檔案 | 路由 | 重點 |
|------|------|------|
| `TranslatorManagement.tsx` | /admin/translators | 翻譯員 CRUD + 停用 + 重設密碼（登入預設頁）|
| `AdminManagement.tsx` | /admin/admins | 管理員 CRUD（不可刪自己）|
| `PatientManagement.tsx` | /admin/patients | 病人 CRUD + 搜尋分頁 |
| `PatientHistory.tsx` | /admin/patients/:id/history | 病人就診歷史 |
| `ScheduleManagement.tsx` | /admin/schedules | 排班 CRUD + 多病人（[SchedulePatientListEditor]）+ 匯入 + 週期 + 群組刪 |
| `CheckinRecords.tsx` | /admin/checkins | 打卡查核 + 篩選 + 編修/刪 + [MapLink] |
| `DiagnosisResults.tsx` | /admin/diagnosis-results | 診斷結果總覽（分頁/篩選/看照片）|
| `ExportSettings.tsx` | /admin/export-settings | 定期匯出設定 + 立即執行 + 即時 Excel/Sheet |
| `AuditLogs.tsx` | /admin/audit-logs | 稽核日誌分頁 |

## 5. 共同慣例
- 錯誤：`message.error(extractApiError(err))`（[utils/apiError](../FRONTEND_SPEC.md)）。
- 所有文案走 i18n `t(...)`；無寫死中英字串（除少數 emoji/符號）。
- 列表端點回 `{data}` 由 client 自動 unwrap。
- 重複點擊防呆（見 changelog 2026-05-31）。

## 6. 已知技術債
- 頁面普遍自行 `useState + useEffect + try/catch` 撈資料，無共用 data-fetching 抽象（可導入 react-query）。

## 7. 測試考量
`pages/__tests__/`：Login、AdminManagement、ChangePassword 有 RTL 測試；其餘頁面主要由 [E2E](../../../e2e/README.md) 覆蓋。
