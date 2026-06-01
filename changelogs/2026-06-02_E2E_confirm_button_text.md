# 更動報告 — 2026-06-02：E2E confirm 對話框按鈕文字補進 regex

## 背景

前一輪 `81e4a71` 修完後只剩 2 個 fail：
- `schedule-crud / delete`
- `translator-mgmt / disable`

兩個都卡在 confirm 對話框的 OK 按鈕點不到。看 page snapshot：

```yaml
- dialog "Confirm":
  - button "Confirm" [active]
```

按鈕文字是 "Confirm" 而不是 "OK" 或 "Delete"。前端用 `okText: t('common.confirm')` 渲染，i18n 對映：
- en: "Confirm"
- zh-TW: "確認"
- th: "ยืนยัน"

我原本 regex `/^(OK|Delete|確認|刪除)$/i` 漏掉 "Confirm"。

## 變更

- `tests/schedule-crud.spec.ts`：regex 補 `Confirm`
- `tests/translator-mgmt.spec.ts`：regex 補 `Confirm`

兩處都註解標明前端是 `okText: t('common.confirm')`，後續維護不會再忘。

## Commit

| Hash | 說明 |
|---|---|
| (本 commit) | fix(e2e): confirm 對話框按鈕加 "Confirm" 字串對應 |
