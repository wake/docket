---
title: git 停在掛載點時指出檔案系統邊界
---

2026-10-24 審查 `add-init-and-new` 時發現。在動到同一段程式碼的 change 裡一併處理，
或在不改任何 spec 句子時直接修。

`gitx.Toplevel` 沒有為檔案系統邊界建模（`GIT_DISCOVERY_ACROSS_FILESYSTEM`）：
掛載點之上的 `.git` 會保留 git 自己的錯誤，而不是「not inside a git work tree」。
兩者的結束碼都是 4。
