# i18n — 規格（輕量）

> 對應檔案：`frontend/src/i18n/index.ts` + `i18n/locales/{en,zh-TW,th}.json`
> 上層：[FRONTEND_SPEC](../FRONTEND_SPEC.md)

## 1. 定位與職責
i18next 設定與語系資源。支援 `en`（**預設 + fallback**）、`zh-TW`、`th`。語言選擇存 localStorage(`language`)。

## 2. 對外契約
| 名稱 | 說明 |
|------|------|
| `SUPPORTED_LANGUAGES` | `['en','zh-TW','th']` |
| `setLanguage(lang)` | 寫 localStorage + `i18n.changeLanguage` |
| `default export i18n` | 供非元件處（如 [api client](../api/API_CLIENT_SPEC.md)）呼叫 `i18n.t` |

App.tsx 依當前語言切 Ant Design locale（enUS/zhTW/thTH）。

## 3. 錯誤碼翻譯契約（最關鍵）
每個 locale 必含 `errors.<CODE>` 區塊，對應後端 [dto error code](../../../backend/internal/dto/DTO_SPEC.md)。
- client 收到錯誤 → `i18n.t("errors."+code, {defaultValue: 後端message})`。
- **不變式**（人工維持）：後端新增 error code → 三個 locale 都要補；漏補則顯示後端英文 fallback。

## 4. 邊界條件
| 情境 | 行為 |
|------|------|
| localStorage 無 `language` | 用 'en' |
| 存了不支援的語言 | 退回 'en' |
| key 找不到 | fallbackLng='en'；errors.* 再退回後端 message |

## 5. 已知技術債
- 三份 locale 手動維護，無自動化偵測缺漏 key（可加 lint/測試比對三檔 key 集合）。

## 6. 測試考量
`i18n/__tests__/i18n.test.ts`：語言切換、key 存在性。E2E `i18n.spec.ts` 驗證 UI 實際切換。
