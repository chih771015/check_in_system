# 更動報告 — 2026-06-01：E2E seed 改用 explicit UPDATE 強制 must_change_pw=false

## 背景

前一次 commit `0fff13e` 把 seed users 的 Create 加 `Select("*")`，期望強制送 zero value。實際 rebuild 後仍出現 admin 登入後被導到 `/change-password`。

根因：GORM v2 的 `db.Select("*").Create(&slice)` 在 batch insert + 欄位帶 `gorm:"default:true"` 的組合下，行為不一定能讓 zero-value false 送出去。

## 變更

`backend/internal/handler/test_reset_handler.go`：
- 移除 `MustChangePW: false` 在 struct 字面值（反正 bool 零值就是 false，寫不寫沒差）
- 移除 `.Select("*")`
- Create 後額外跑 `db.Exec("UPDATE users SET must_change_pw = false")` 強制覆寫整張表

直接 raw SQL 不靠 GORM magic，最不會出錯。GORM v2 forbids globalUpdate，所以不能用 `db.Model().Update()` 不帶 Where；改用 `db.Exec` 跳過此限制。

## Commit

| Hash | 說明 |
|---|---|
| (本 commit) | fix(e2e): 改用 explicit UPDATE 強制 must_change_pw=false |
