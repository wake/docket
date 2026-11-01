---
title: 內部錯誤加上無法讀取的 source 時結束碼為 3
source: review of add-init-and-new, 2026-10-24; moved from docket/ideas/2026-10-24-internal-error-exit-code.md
---

## Current

`ExitCode`（internal/cli/envelope.go:106）在檢查 `Unreadable` 之前，對任何非 usage、非環境的錯誤都回傳 1，所以內部錯誤加上無法讀取的 source 時結束碼是 1。

## Expected

`cli-output`（spec 第 50 行）：無法讀取某個 source 的命令「SHALL exit with 3 even when it also found problems elsewhere」。

## Unchanged

usage 錯誤結束碼為 2，環境錯誤為 4；沒有無法讀取的 source 的錯誤結束碼為 1。

## Found

`add-init-and-new` 的審查，2026-10-24。main 上仍存在。
