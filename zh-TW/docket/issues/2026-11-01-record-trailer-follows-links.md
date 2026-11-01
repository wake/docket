---
title: record-trailer 只把一般檔案算作 record
source: review of add-record-tracking, 2026-11-01
---

## Current

`dkt lint` 的 `record-trailer` 檢查（internal/lint/record.go:237）使用 `fs.Stat`，會跟隨符號連結。`Record:` trailer 指名 `docket/records/<slug>.md`，而該路徑是指向一般檔案的連結時，檢查會通過，但 `dkt status` 並不列出這樣的 record。

## Expected

record 是「每個一般檔案 `docket/records/<slug>.md`」（workspace-scan，Records found，spec 第 74 行），而 lint 的其他 record 檢查會跳過連結（`dkt lint <slug>` 將它回報為 `record-missing`，internal/lint/record.go:39）。指名連結的 trailer 並未指名任何 record，所以 `record-trailer` 應該回報它。

## Unchanged

指名一般 record 檔案的 trailer 通過；每個 commit 與值一個 finding，訊息相同；`dkt record` 與 `dkt sync` 仍拒絕透過連結寫入。

## Found

`add-record-tracking` 的審查認為可以等。以 dkt v0.13.0 試用：在一個分支上，`docket/records/2026-11-01-fix-x.md` 是指向一般檔案的連結，帶有 `Record: 2026-11-01-fix-x` 的 commit 沒有任何 finding，而 `dkt status --json` 也沒有列出 record。
