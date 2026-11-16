---
title: "Read a file where a folder should be as unreadable on Windows, as elsewhere"
kind: fix
source: "docket/issues/2026-11-16-file-as-folder-windows.md"
---

## What

`internal/missing` decides whether a failed read means "not there". On Windows it looks up the deepest existing part of the path: missing only when that is a folder above it. Fifteen checks use it: the index, settings, user template (×3), sessions folder, and the `domains`, `lint` and `conflicts` reads. The table of all 46 such checks is in the pull request.

## Why

Windows gives a file in the way ERROR_PATH_NOT_FOUND, which matches `fs.ErrNotExist` (and is `syscall.ENOTDIR` there), so those reads took a file in a folder's place for a missing folder. macOS and Linux give ENOTDIR.

## Verification

- `TestIsOnWindows` runs the Windows rule on any system; it failed before the fix.
- Eleven new CLI tests and `TestDomainsChangesIsAFile` failed on `windows-latest` before the fix (#190) with the issue's symptom. Mutating `missing` makes each fail on macOS.
- `go test ./...`, `go vet ./...`, `gofmt -l .` pass on macOS.

## Review

Codex (gpt-5.6-sol, adversarial) raised two must-fixes; I turned both down, and a second codex (high effort) agreed:
- `dkt sync`'s `Stat` reports a file in the way as `change-missing`: `safepath.Check` stops first, on every system.
- A path created or removed between the read and `missing`'s `Stat` can flip one answer: a passing race, not a lasting error.
