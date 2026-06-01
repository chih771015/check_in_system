# 更動報告 — 2026-06-01：E2E seed MustChangePW=false 沒生效

## Commit

| Hash | 說明 |
|---|---|
| (本 commit) | fix(e2e): seed users 加 Select("*") 修 GORM 零值跳過 |

## 變更摘要

E2E 跑 16 個測試 fail，根因：admin 登入後被導到 `/change-password` 而不是 `/admin/translators`。

GORM 經典坑：

- `model.User.MustChangePW bool` 帶 `gorm:"default:true"`
- seed 程式碼寫 `MustChangePW: false`，但 false 是 bool 零值
- GORM 預設**跳過零值欄位**，DB default `true` 接手
- 三個 seeded user 全部 `must_change_pw=true` → 一登入就被導到改密碼頁

修法：`db.Select("*").Create(&users)` 強制寫所有欄位（包含零值）。

注意：`MustChangePW: false` 寫法本身是正確意圖；只是 GORM 行為與直覺相反，需要 `Select("*")` 才會照單全收。

## 影響檔案

- `backend/internal/handler/test_reset_handler.go`：seed users Create 加 `.Select("*")`，並加註解解釋為何需要
