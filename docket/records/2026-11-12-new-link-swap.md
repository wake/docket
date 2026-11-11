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

Codex adversarial review (another model) raised two points, both present before this fix and outside it: OpenSpec, a subprocess, still writes by path, so `openspec/changes` swapped for a link while it runs can send its write outside (closing it means staging OpenSpec's output, against "OpenSpec still creates the change folder"); and cleanup removes by name, so a plain folder swapped in at that path with the same names is removed, as `dkt record` already accepts (a `shortcut:` comment now says so). Both go to a separate idea; `safefile.Install` goes to a separate issue.
