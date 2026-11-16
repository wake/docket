---
title: "Run the whole test suite on Windows"
kind: test
source: "docket/ideas/windows-suite.md"
---

## What

The tests that assumed Unix now skip or adapt on Windows, and the Windows CI job runs `go test ./...`. Skipped, with the reason in the message: mode 000 (`testutil.SkipUnlessModesDeny`, also for root), a folder renamed while dkt holds it open, and two product bugs, each naming its issue. Adapted: names Windows forbids, permission bits (`testutil.ModeBits`), Unix test paths (`C:` on Windows), paths written into JSON, an OpenSpec named without `.exe`, and file times read from the folder listing.

## Why

After `windows-runtime`, `windows-paths` and `windows-user-folders`, about 36 tests still failed on Windows because they assumed Unix, so the job could run only chosen tests. The job keeps its name, which the `platform-support` spec fixes, until `windows-release` changes that spec; a comment on the job says so.

## Verification

Throwaway probes (#187) ran `go test ./...` on `windows-latest`: 36 failures before, every package `ok` after. `go test ./...` passes on macOS. Two failures were product bugs, opened as `docket/issues/2026-11-16-record-dir-slug-windows.md` and `docket/issues/2026-11-16-file-as-folder-windows.md` (#188); their tests skip on Windows until fixed.

## Review

codex (gpt-5.6-sol), attacking the whole diff: whether a check weakened on macOS or Linux, skips broader than needed, adapted Windows values that miss their point, the two helpers, and coverage lost from the old CI steps. No findings.
