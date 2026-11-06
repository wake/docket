---
title: "dkt new can write outside the repository when a folder is swapped for a link during the run"
source: "https://github.com/wake/docket/issues/92"
---

## Current

`dkt new` checks the path with `safepath.Check`, then opens `proposal.md` by path (`internal/cli/cmd_new.go:173-177`). `O_NOFOLLOW` guards only the last component, so a parent folder swapped for a link between the two sends the write through the link. `cleanup` in `runNew` also removes by path after its check. Found by reading the code: reproducing it takes another process swapping the folder at that moment, and `dkt new` has no seam for a test yet.

## Expected

`openspec/specs/change-creation/spec.md`, "Symbolic links refused": `dkt new` SHALL "write nothing on either side of the link". `dkt record` holds to this since #88 by writing through `os.Root` (`TestRecordFolderSwappedAfterTheChecks`).

## Unchanged

- A link already on the path at the start still gives `symlink-refused` and exits with 1.
- OpenSpec still creates the change folder.
- Slug suffixes and two concurrent runs work as now.
- `cleanup` removes only what this run made.

## Evidence

Seen at `6360d01`. The same shape was reproduced on `dkt record` with a failing test before #88. `safefile.Install` (`dkt init`, `dkt sync`) also opens and renames by path; not checked for this race.
