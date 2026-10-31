---
title: Start fewer git processes per ref in the scan
---

Found while reviewing `add-init-and-new`, 2026-10-24. Pick it up in the
change that touches the same code, or fix it directly when it changes no
spec sentence.

The scan starts several git processes per ref (`merge-base --is-ancestor`
and two `ls-tree`); `git for-each-ref --no-merged` could drop merged refs in
one call. `Toplevel` runs git twice.

## Measured, 2026-10-31

Reported by the session building the workbench, on `main` at 68294e2:

- In this repository (22 refs: 4 local, 18 remote-tracking; 4 worktrees),
  one `dkt status` took 2.7 seconds and started 44 git processes. Starting
  a git process costs about 30 to 45 ms on that machine.
  - `merge-base --is-ancestor`: 24 calls, 1.09 s, about 40%.
  - `ls-tree -d`: 6 calls, 0.27 s.
  - `rev-parse`, `log`, `symbolic-ref`, `status` and others: 14 calls,
    about 0.6 s.
- In a repository estimated at thousands of refs, `dkt status` had not
  finished after two minutes. A repository that merges by squash never sees its old
  branches as merged, so every ref also pays the `ls-tree` calls.
- Since v0.9.0 the scan also lists `docket/records` in each unmerged ref,
  one more `ls-tree` per ref.
- `status.Load` starts one `git show` per file of a change on a branch.

## Leads, by expected effect

1. One `git for-each-ref --no-merged=<default tip> refs/heads refs/remotes`
   instead of one `merge-base --is-ancestor` per ref.
2. One listing per ref for `openspec/changes`, its archive and
   `docket/records`, or one `git cat-file --batch` for every unmerged ref.
3. Batch reads in `status.Load` (`cat-file --batch`, or a whole change
   folder at once).
4. Bounded parallelism for the calls that stay one per ref.

Caching scan results by ref SHA belongs to a long-running workbench, not to
`dkt`. The results must stay the same, including a single ref that cannot be
read being reported alone as unreadable; a promised speed would be a new
expectation and a spec change.

## Status

Done by the record `docket/records/2026-10-31-scan-git-calls.md` (leads 1
to 3): `dkt status` here went from 46 git processes and 3.3 s to 21 and
1.6 s, and settling refs no longer grows with their number. Left for later:
each unmerged ref still costs one `ls-tree`, which matters in a repository
that merges by squash (its old branches stay unmerged); each worktree still
costs one `merge-base`; `Toplevel` still runs git twice; and lead 4,
bounded parallelism. The `ls-tree` per unmerged ref is measured and taken up
in `2026-10-31-scan-tree-batch.md`.
