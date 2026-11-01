---
title: 檢查每個 task 的 Review 行與 Files 區塊
---

2026-10-28 記錄於 `update-workflow`，那次只改了 Docket 隨附的指引：plan 把每個 task 標成 `**Review:** judgment` 或 `**Review:**
checks`，並在 `**Files:**` 區塊列出每個檔案，平行的 session 開工前會比對彼此的 `**Files:**` 區塊。目前沒有任何東西檢查這件事。
這會改動 spec，所以需要一個 change。

`dkt lint` 已經要求每個 task 有一行 `**Scenarios:**`（`change-
lint`）。擴充它：每個未取消的 task 都有一行 `**Review:**`，值為 `judgment` 或 `checks`，並有一個 `**Files:**` 區塊，至少列出一個路徑。

## 待釐清問題

- 在這條規則之前寫好的 plan 是失敗還是只警告。
- 某些 task 是否應該強制要求 `judgment`，例如 `**Files:**` 區塊動到公開介面的 task。
