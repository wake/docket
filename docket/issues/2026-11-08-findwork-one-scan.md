---
title: "dkt status scans changes and records separately"
source: "review of project-snapshot (#119)"
---

## Current

`idea.FindWork` (`internal/idea/work.go`), which `dkt status` uses to find which works take an idea, calls `scan.Changes`, then `scan.Records`, then `Git.Refs`. Each of the two scans is a whole `scan.Scan`, which lists the worktrees and the refs and runs one `ls-tree` per distinct ref commit. As a result, the worktrees are listed twice, the refs three times, and every ref commit twice. Running `dkt status` with a git wrapper first on PATH that logs each run shows the repeats. On macOS a git run costs about 44 ms, so this adds roughly (C + 3) × 44 ms to every `dkt status`, where C is the number of distinct ref commits.

## Expected

One `scan.Scan(g)` gives both inventories and the refs, as `dkt snapshot` uses it: one worktree list, one ref list, and one `ls-tree` per distinct ref commit. This is what the project-snapshot design settles in its "Batched git, bounded by commits, not by works" paragraph. The output of `dkt status` does not change.

## Unchanged

- `dkt status`'s result, findings and unreadable entries.
- `FindWork`'s fallback that looks up a ref use's commit when `Use.Commit` is empty.

## Evidence

Seen at 743873d (v0.21.0). In `dkt snapshot`, merging the three scans cut 42 git runs to 18 on a repository with 4 worktrees and 7 refs.
