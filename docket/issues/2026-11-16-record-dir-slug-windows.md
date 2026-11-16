---
title: "dkt record fails on Windows when a folder holds the record's path, instead of taking the next suffix"
source: "Windows probe of windows-suite (#187)"
---

## Current

On Windows, when a folder already sits at `docket/records/<slug>.md`, `dkt record` stops with an `internal` finding, `openat <slug>.md: is a directory`, and exits with 1. `writeDated` (`internal/cli/cmd_record.go:123`) takes only `fs.ErrExist` from the exclusive open as "this slug is taken" and moves on; on Windows, Go's exclusive create on a folder reports "is a directory" instead. `dkt issue` writes through the same `writeDated`. Reproduce with `TestRecordOpenMovesToNextSuffix` on Windows. `dkt new` opens its proposal through the same `openNewFile` (`internal/cli/cmd_new.go:170`); check it too.

## Expected

`records`: when the slug is used, `dkt record` SHALL write under the first of `<slug>-2`, `<slug>-3`, ... that no scanned source uses and that does not exist in the worktree; `issues` says `dkt issue` does the same. A folder at the path exists in the worktree, as on macOS and Linux, where the test passes.

## Unchanged

- The exclusive open never writes through a link or over anything at the path.
- Any other failure of the open still stops the run and removes the folders this run created.
- macOS and Linux behave as now.

## Evidence

Seen at `91f99c4` on `windows-latest`, run 43810020035 of #187. The same Windows error shows in `internal/safefile/safefile_windows_test.go:46`, where only a failure is required.
