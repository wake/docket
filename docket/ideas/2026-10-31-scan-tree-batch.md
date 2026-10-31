---
title: Read the trees of unmerged refs in a batch
---

Recorded 2026-10-31, from a read-only trial of dkt v0.9.1 on a copy of the
Purdex repository. It follows `2026-10-24-scan-git-calls.md`, whose Status
left one `ls-tree` per unmerged ref.

## Measured

- The copy has 3,625 refs and 13,608 commits. `dkt status` took 21.9 s and
  started 322 git processes, 308 of them `ls-tree`, one per unmerged ref;
  `dkt conflicts` took the same. Before v0.9.1, `dkt status` had not
  finished after five minutes.
- The scan reads `refs/heads` and `refs/remotes`. 307 of those refs are
  unmerged (146 local, 161 remote-tracking), at 228 distinct commits, and
  none of them holds `openspec/changes` or `docket/records`: the 308 calls
  find nothing.
- dkt does not fetch: it reads the remote-tracking refs as of the last
  fetch. 158 of the 163 unmerged remote-tracking branches still exist
  upstream, so `git fetch --prune` would not help. The cost comes from
  branches that are not merged, not only from squash merges.

## Leads

1. Read the trees of every unmerged ref in a fixed number of git
   processes: one `git cat-file --batch` per level (each ref's root tree;
   then `openspec` and `docket`; then `changes` and `records`; then the
   archive), each tree read once however many refs share it. A ref without
   either folder then costs nothing more.
2. Keep the results exactly: an object missing at any level falls back to
   today's `ls-tree` for that ref, so it gets today's error.
3. `cat-file --batch-check` on `<rev>:openspec/changes` alone is simpler,
   but it reports an absent path and a missing object the same way, so it
   would change what a damaged repository reports.

## Related

- `2026-10-24-scan-git-calls.md`: the first part.
- `2026-10-31-existing-project-adoption.md`: the same trial.
