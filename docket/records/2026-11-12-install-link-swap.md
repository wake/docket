---
title: "dkt init, sync, new and record refuse a folder swapped for a link during the run"
kind: fix
source: "docket/issues/2026-11-12-install-link-swap.md"
---

## What

`safepath.OpenDir` opens a folder inside an `os.Root` one name at a time, refusing a link at any name and checking each folder opened is the one checked. `dkt init` and `dkt sync` write in the folder it returns through `safefile.InstallIn` (temporary file, rename and removal all there) and report `symlink-refused`; `dkt new` and `dkt record` create their file the same way. The Go floor rises to 1.25 for `os.Root.Rename`.

## Why

A parent folder swapped for a link after the checks sent the write and rename outside; `O_NOFOLLOW` guards only the last component. `os.Root` alone still follows a relative link that stays inside the repository, which the specs also refuse.

## Verification

`TestWriteFolderSwappedAfterDecide`, `TestWriteFailureAfterFolderMoved`, `TestSyncFolderSwappedAfterTheRead`, `TestNewFolderSwappedForLinkInside` and `TestRecordFolderSwappedForLinkInside` failed on the code before and pass after; gofmt, go vet and go test pass. Not exercised: a swap between `OpenDir`'s check and open, and `dkt init` turning the error into `symlink-refused`.

## Review

Codex (another model), first on `os.Root` alone: it followed a relative link inside the repository, and the removal on failure left the temporary file in a moved folder; both fixed with `OpenDir`. Then on the fix: new and record did not report `symlink-refused` (fixed); a link to the checked folder moved away passes, which is kept and documented, as it equals a move after the open.
