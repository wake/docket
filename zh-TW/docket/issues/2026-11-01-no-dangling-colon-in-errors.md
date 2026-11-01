---
title: 工具沒有輸出任何內容時，去掉懸空的冒號
source: review of add-init-and-new, 2026-10-24; moved from docket/ideas/2026-10-24-dangling-colon.md
---

## Current

工具沒有對標準錯誤輸出任何內容時，`gitx.ExitError.Error`（internal/gitx/git.go:34）與 `openspec.FailedError.Error`（internal/openspec/runner.go:59）都以 `: ` 結尾。

## Expected

`openspec-runtime`（spec 第 33 行）：回報命令、結束狀態與工具的標準錯誤。沒有標準錯誤時，訊息在狀態之後結束。

## Unchanged

有標準錯誤文字的訊息，以及結束碼。

## Found

`add-init-and-new` 的審查，2026-10-24。main 上仍存在。
