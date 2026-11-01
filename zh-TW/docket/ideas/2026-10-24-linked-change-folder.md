---
title: 回報屬於連結的 change 資料夾
---

2026-10-24 審查 `add-init-and-new` 時發現。在動到同一段程式碼的 change 裡一併處理，
或在不改任何 spec 句子時直接修。

在磁碟上，屬於符號連結的 change 資料夾不會被列出
（對它而言 `DirEntry.IsDir` 為 false）。要決定是否該回報它。
