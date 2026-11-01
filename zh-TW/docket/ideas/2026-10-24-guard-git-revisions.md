---
title: 防範以連字號開頭的 git revision
---

2026-10-24 審查 `add-conflicts` 時發現。

`gitx` 中接收 revision 的方法（`ListFiles`、`ListDirs`、`ShowIfExists`、`Show`），
把 revision 放在 git 仍會解析選項的位置，所以以 `-` 開頭的 revision 會被當成選項。
目前每個呼叫端傳入的都是掃描得到的 SHA；用一次修正把它們全部防住。
