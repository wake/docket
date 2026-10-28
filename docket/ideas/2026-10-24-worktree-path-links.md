---
title: Do not follow a link at or above a worktree path
---

Found while reviewing `add-conflicts`, 2026-10-24.

A worktree is opened by the path git lists (`os.OpenRoot` in `scan.ReadFile`
and `scan.ReadFiles`), so a link at or above that path is followed. Changing
it needs another process to swap the folder between the scan and the read.
