---
title: "dkt init and dkt sync can write outside the repository when a folder is swapped for a link during the run"
source: "review of #131"
---

## Current

`safefile.Install` (`internal/safefile/safefile.go`) creates a temporary file beside the target with `os.OpenFile(... O_NOFOLLOW ...)` and renames it over the target, both by path, and removes the temporary file by path on failure. `O_NOFOLLOW` guards only the last component, so a parent folder swapped for a link after the caller's check sends the write and the rename outside the repository. Callers in a repository: `dkt init` (`internal/setup/setup.go:231`, `:233`) and `dkt sync` (`internal/cli/cmd_sync.go:197`). Found by reading the code; not reproduced, as it takes another process swapping the folder at that moment.

## Expected

`project-setup`, "Symbolic links refused": `dkt init` SHALL write nothing when a path it would write passes through a link inside the repository. `plan-sync`: with a link on the path to plan.md, `dkt sync` SHALL read and write nothing there. `dkt new` and `dkt record` hold to this by writing through `os.Root` (#131, #88).

## Unchanged

- A link already on the path at the start still gives `symlink-refused` and exits with 1.
- A failed write leaves the target as it was, or missing.
- An existing file keeps its permission bits, and a new one gets its mode under the umask.
- The projects index (`internal/projects/files.go:131`) also uses `Install` outside any repository and keeps working.

## Evidence

Seen at `e99dfe7` (v0.22.1). The same shape as `docket/issues/2026-11-06-new-link-swap.md`, fixed in #131.
