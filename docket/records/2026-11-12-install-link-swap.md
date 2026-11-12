---
title: "dkt init and dkt sync write through the repository root, so a folder swapped for a link cannot lead them outside"
kind: fix
source: "docket/issues/2026-11-12-install-link-swap.md"
---

## What

`safefile.InstallIn` writes the temporary file, renames it and removes it on failure through an `os.Root`; `Install` now opens a root at the target's folder and calls it. `dkt init` (`setup.Write`) makes folders, checks the mode and installs through a root at the repository root, and so does `dkt sync` when it ticks a plan or a record. The Go floor rises to 1.25, the first whose `os.Root` has `Rename` and `MkdirAll`; 1.24 is past its support window.

## Why

A parent folder swapped for a symbolic link after the checks sent the write and the rename outside the repository; `O_NOFOLLOW` guards only the last component.

## Verification

`TestWriteFolderSwappedAfterDecide` and `TestSyncFolderSwappedAfterTheRead` failed before the fix (the file outside was overwritten) and pass after; the sync test then passed the disk path, now the root. gofmt, go vet and go test pass. The race itself was exercised by swapping the folder between the steps, not by a concurrent process.

## Review
