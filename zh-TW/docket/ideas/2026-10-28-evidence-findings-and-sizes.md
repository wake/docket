---
title: 在 evidence.md 記錄審查 finding 與規模
---

2026-10-28 隨 `update-workflow` 記下。該 change 只改了 Docket 出貨的指引：
plan 把每個 task 標成 `**Review:** judgment` 或 `**Review:**
checks`，並在 `**Files:**` 區塊列出每個檔案，平行的 session 開工前先比對
`**Files:**` 區塊。目前還沒有任何東西檢查這點。這會改到 spec，所以需要一個 change。

流程是否站得住腳，只能從每個 change 保存的數字判斷：
change 範圍內的產品程式碼與測試行數、依嚴重度與審查者分類的審查 finding，
以及其中哪些已修正。`dkt archive` 本來就從該範圍寫出 `evidence.md`（`change-archive`），
所以規模可以機械取得。finding 則需要一個輸入來源。

## 待釐清問題

- 輸入來源：change 資料夾裡固定格式的 `review.md`，
  還是每個修正 commit 上標明所修 finding 的 trailer。
- `dkt status` 是否該在 change 封存前就顯示同樣的數字。
