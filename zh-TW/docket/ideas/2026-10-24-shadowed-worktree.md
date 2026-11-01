---
title: 重新檢視被未合併分支遮蔽的 worktree
---

2026-10-24 審查 `add-init-and-new` 時發現。在動到同一段程式碼的 change 裡一併處理，
或在不改任何 spec 句子時直接修。

某個 worktree 的 HEAD 已被預設分支包含，且對某個 change 有未提交的修改，
它會被持有相同 slug 的未合併分支遮蔽：這是 `workspace-scan` spec 裡的優先順序。
既然 `status` 現在會顯示內容的來源，就重新檢視這點。
