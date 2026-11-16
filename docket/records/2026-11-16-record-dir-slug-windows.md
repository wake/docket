---
title: "Take the next suffix on Windows when a folder holds a record's or issue's path"
kind: fix
source: "docket/issues/2026-11-16-record-dir-slug-windows.md"
---

## What

`writeDated` (`cmd_record.go`, shared by `dkt record` and `dkt issue`) asks `taken(windows, err)` whether the name is taken: `fs.ErrExist`, and on Windows also "is a directory". The Windows CI job runs the two tests below. `dkt new` needs no change: any error from its `openNewFile` stops the run, on every system.

## Why

On Windows, Go's exclusive create reports a folder at the name as `EISDIR`, not as existing, so `writeDated` stopped with an `internal` finding.

## Verification

- `TestTakenFolderOnWindows` tests `taken` with `windows` true and false on any system. It failed before the fix (`is a directory: false, want true`) and passes after; the `windows=false` cases pin macOS and Linux.
- `TestRecordOpenMovesToNextSuffix`, which failed on `windows-latest` in #187, runs in the Windows job.
- `go test ./...`, `go vet ./...` and `gofmt -l .` pass on macOS.

## Review

Codex (gpt-5.6-sol, adversarial) approved with no findings. `EISDIR` counts only on Windows and only at the final name. Other open errors still stop and clean up, and `dkt new` does not use `taken`. `safefile`'s temporary file loop has the same shape, but only a random name colliding with a folder would reach it.
