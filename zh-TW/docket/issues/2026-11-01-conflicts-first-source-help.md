---
title: 說明 dkt conflicts 讀的是每個 change 的第一個 source
source: review of add-conflicts, 2026-10-24; moved from docket/ideas/2026-10-24-conflicts-first-source.md
---

## Current

`dkt conflicts` 只比對每個 change 第一個 source 的內容（internal/conflicts/load.go:10）。它的用法列（internal/cli/cmd_conflicts.go:23）與文件都沒有說明這點。

## Expected

行為維持現狀，並如 `change-status`（spec 第 12 行）對 `dkt status` 的描述：內容來自掃描列在最前面的 source。命令的文件應該說同一件事。

## Unchanged

讀的是哪個 source，以及重疊的結果。

## Found

`add-conflicts` 的審查，2026-10-24，在 `dkt status` 開始顯示內容來源之後。
