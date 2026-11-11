---
title: "Write the change of dkt new through the repository root"
kind: fix
source: "docket/issues/2026-11-06-new-link-swap.md"
---

## What

`dkt new` now creates `proposal.md` and cleans up through an `os.Root` opened at the repository root (`internal/cli/cmd_new.go`), using the `openNewFile` seam it shares with `dkt record`.

## Why

A parent folder swapped for a symbolic link between the check and the open sent the write, or the cleanup removal, outside the repository; `O_NOFOLLOW` guards only the last component.

## Verification

`TestNewFolderSwappedAfterTheChecks` and `TestNewCleanupAfterFolderSwapped` failed before the fix and pass after; gofmt, go vet and go test pass. The race itself was exercised only through the seam.

## Review

The review is pending.
