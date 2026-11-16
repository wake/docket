---
title: "On Windows dkt reads a file where a folder should be as a missing folder, not as unreadable"
source: "Windows probe of windows-suite (#187)"
---

## Current

On Windows, when `openspec/changes` is a file, `dkt domains` reports `domain-unknown` and exits with 1, instead of listing `openspec/changes` in `unreadable` and exiting with 3. `internal/domain/load.go:89` treats any `fs.ErrNotExist` from `fs.ReadDir` as "no changes"; reading a file as a folder fails with `ENOTDIR` on macOS and Linux, but on Windows with an error that `errors.Is` matches to `fs.ErrNotExist`. Reproduce with `TestDomainsChangesIsAFile` on Windows.

The shared shape: a folder read whose error is checked with `fs.ErrNotExist` alone. To check: `internal/lint/lint.go:310`, `deltas.go:35`, `document.go:67`, `:141`; `internal/activity/activity.go:105`, `:136`; `internal/cli/cmd_sync.go`, `cmd_workflow.go:128`, `:192`; `internal/conflicts/copies.go:105`. Places that also check `syscall.ENOTDIR` (`internal/scan/scan.go:215`) may still differ on Windows.

## Expected

`domain-tree`, "Unreadable sources": when `openspec/changes` cannot be read, `unreadable` SHALL list it, `domain-unknown` SHALL NOT be reported, and dkt SHALL exit with 3, as on macOS and Linux, where the test passes.

## Unchanged

- A missing folder still counts as empty wherever the specs say so.
- Where a spec says a file in a folder's place means "nothing here" (`workspace-scan`), it keeps meaning that.
- macOS and Linux behave as now.

## Evidence

Seen at `91f99c4` on `windows-latest`, run 43810020035 of #187.
