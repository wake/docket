---
title: Split status.LoadRecords
source: review of add-record-tracking, 2026-11-01; moved from docket/ideas/2026-11-01-split-load-records.md
---

## Current

`status.LoadRecords` (internal/status/records.go:54-155) is one function of about 100 lines doing five jobs: pick, merge-commit and limit, group and read, sort, tags.

## Expected

Same behavior, split along the five jobs. Behavior is shipped and covered by tests in `internal/status` and `internal/cli`; it is a restructuring (a record of kind `refactor`).

## Unchanged

`LoadRecords`'s signature and git process count: one `Additions`, one `ReadFiles` and at most one `Commits` per source, one `Tags`.

## Found

Reviews of `add-record-tracking` judged it able to wait.
