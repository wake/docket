---
title: 在 dkt conflicts 回報檔案重疊
---

2026-10-28 隨 `update-workflow` 記下。該 change 只改了 Docket 出貨的指引：
plan 把每個 task 標成 `**Review:** judgment` 或 `**Review:**
checks`，並在 `**Files:**` 區塊列出每個檔案，平行的 session 開工前先比對
`**Files:**` 區塊。目前還沒有任何東西檢查這點。這會改到 spec，所以需要一個 change。

`dkt conflicts` 會回報動到同一個需求的 change（`conflict-
detection`）。兩個 change 沒有動到共同的需求，卻編輯同樣的檔案，現在會通過，
而 session 得靠手動比對 plan。請回報出現在兩個以上 active change 的
`**Files:**` 區塊中的每個路徑，找法與需求重疊相同。

## 待釐清問題

- 路徑後面帶說明（`Modify: AGENTS.md (the "Change
  workflow" section)`）或行號範圍時怎麼讀；大概是取第一對反引號裡的文字。
- 檔案重疊是像需求重疊那樣算失敗，還是另外當作警告回報。
