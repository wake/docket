---
title: "Have dkt status scan the workspace once to find the work on each idea"
kind: refactor
source: "docket/issues/2026-11-08-findwork-one-scan.md"
---

## What

`idea.FindWork` takes the change and record inventories and the refs from one `scan.Scan(g)`, as `dkt snapshot` does, instead of `scan.Changes`, `scan.Records` and `Git.Refs`. The fallback for a ref use with no commit stays.

## Why

Each of the two scans was a whole `scan.Scan`, so `dkt status` listed the worktrees twice, the refs three times and every ref commit twice.

## Verification

- Git runs of one `dkt status`, counted by a logging git wrapper first on PATH, in a repository with 4 worktrees and 7 refs (6 distinct commits): 31 before, 22 after (worktree list 2 to 1, ref list 4 to 2, `ls-tree` 12 to 6). Result, findings and unreadable entries the same.
- The `TestStatus*` tests and `internal/idea` pass before and after.
- Mutation: reading records from the changes inventory fails `TestStatusRecordOnAnotherBranch` and four more. Not caught: emptying the refs map, which only the kept fallback reads; `scan.Scan` gives every ref use its commit, so no test reaches it through `FindWork`.

## Review
