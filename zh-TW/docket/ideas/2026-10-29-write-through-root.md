---
title: 透過 os.Root 寫入
---

2026-10-29 審查 `add-titles` 時發現。在動到同一段程式碼的 change 裡處理，或在不改任何 spec 句子時直接修。

每個寫入端都用 `safepath.Check` 檢查路徑，然後以絕對路徑開啟。這個檢查會拒絕提交進 repository 的連結，但若有行程在檢查與寫入之間把某個上層目錄換成連結，寫入就會順著它跑出 repository：`dkt new`
寫 `docket.yaml`、封存標題步驟的 `MkdirAll`、`dkt record` 建立它的資料夾與檔案（`record.Create`），以及其他所有寫入端皆然。這個競爭需要同一個使用者的行程在 `dkt` 執行時改動 repository，而那個行程本來就能自己寫到那裡，所以這是縱深防禦。

只開啟 repository 一次成 `os.Root`，所有寫入都透過它，這樣任何路徑都無法離開 repository。`Root.MkdirAll` 與 `Root.WriteFile` 需要 Go 1.25；`go.mod` 寫的是 1.24（見
`docket/issues/2026-11-01-ci-once-and-go-floor.md`）。

## 待釐清問題

- `os.Root` 會跟隨留在 root 內的連結；這些連結是否仍像 `safepath.Check` 現在這樣逐段拒絕。
- 如何測試檢查與寫入之間的競爭。
