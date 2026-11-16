---
title: "Take the next suffix on Windows when a folder holds a record's or issue's path"
kind: fix
source: "docket/issues/2026-11-16-record-dir-slug-windows.md"
---

## What

`writeDated` in `internal/cli/cmd_record.go`, shared by `dkt record` and `dkt issue`, now moves on to the next suffix on Windows when a folder holds the path. A new `taken(windows, err)` decides whether the exclusive open's error means the name is taken: `fs.ErrExist` on every system, and on Windows also "is a directory". The Windows CI job runs `TestTakenFolderOnWindows` and `TestRecordOpenMovesToNextSuffix`.

`dkt new` (`openNewFile` in `cmd_new.go`) needs no change. It never takes a suffix after the open: any error from it stops the run and cleans up. A folder at `proposal.md` fails the run on every system; the message differs on Windows.

## Why

On Windows, Go's exclusive create reports a folder at the name as `EISDIR`, not as existing. `writeDated` took only `fs.ErrExist` as "taken", so it stopped with an `internal` finding.

## Verification

- `TestTakenFolderOnWindows` tests `taken` with `windows` true and false on any system. It failed before the fix (`windows=true ... is a directory: false, want true`) and passes after.
- macOS and Linux keep `fs.ErrExist` alone; the `windows=false` cases pin that.
- `TestRecordOpenMovesToNextSuffix` failed on `windows-latest` in #187; the pull request's Windows job runs it with the fix.
- `go test ./...`, `go vet ./...`, `gofmt -l .` pass on macOS.

## Review

Codex (gpt-5.6-sol, adversarial review) approved with no findings. It checked these points:

- `EISDIR` counts as taken only on Windows, and only for the final name's exclusive create.
- Failures at folders on the path, refused links and other open errors still stop the run and clean up.
- `dkt new` does not use `taken`.
- `safefile`'s temporary-file loop has the same shape, but it only matters if a random name collides with a folder, which is not a real risk.
