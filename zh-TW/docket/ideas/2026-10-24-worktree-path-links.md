---
title: 不跟隨 worktree 路徑上或其上層的連結
---

2026-10-24 審查 `add-conflicts` 時發現。

worktree 是以 git 列出的路徑開啟的（`scan.ReadFile` 與 `scan.ReadFiles` 中的 `os.OpenRoot`），
所以該路徑上或其上層的連結會被跟隨。要改它，
需要另一個行程在掃描與讀取之間把資料夾換掉。
