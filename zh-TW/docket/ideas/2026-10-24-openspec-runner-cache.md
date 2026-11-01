---
title: 安全地快取失敗的 OpenSpec 版本檢查
---

2026-10-24 審查 `add-init-and-new` 時發現。在動到同一段程式碼的 change 裡一併處理，
或在不改任何 spec 句子時直接修。

`openspec.Runner` 不會快取失敗的版本檢查，而且它的快取在並行使用下不安全。
