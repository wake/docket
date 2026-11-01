---
title: 拆分 status.LoadRecords
source: review of add-record-tracking, 2026-11-01; moved from docket/ideas/2026-11-01-split-load-records.md
---

## Current

`status.LoadRecords`（internal/status/records.go:54-155）是一個約 100 行的函式，做五件事：挑選、merge commit 與限制、分組與讀取、排序、tag。

## Expected

行為相同，依這五件事拆分。行為已出貨，並由 `internal/status` 與 `internal/cli` 的測試涵蓋；這是重構（kind 為 `refactor` 的 record）。

## Unchanged

`LoadRecords` 的簽章與 git 行程數量：每個 source 一次 `Additions`、一次 `ReadFiles`、最多一次 `Commits`，另加一次 `Tags`。

## Found

`add-record-tracking` 的審查認為可以等。
