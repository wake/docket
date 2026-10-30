---
title: Start fewer git processes in the scan
kind: refactor
source: docket/ideas/2026-10-24-scan-git-calls.md, with measurements from the workbench session
---

## What

Refs are settled by three `for-each-ref` listings and one `rev-list`, not a
`merge-base` each; each tree source is read with one `ls-tree`, not three;
a branch's change files with one `cat-file --batch`. Where batches report
damaged history or odd entries differently, the old calls run instead.

## Why

`dkt status` started 46 git processes here (3.3 s), one or more per ref, and
did not finish in two minutes in a repository with thousands of refs.

## Verification

Tests pinned each result first (merged and unmerged refs, damaged history,
tree entries, file reads); all pass before and after. Mutating the changed
lines failed a test, except four equivalent mutants and two fallbacks only
fault injection reaches. `dkt status` here: 46 git processes and 3.3 s
before, 21 and 1.6 s after.

## Review

Codex (gpt-5.6-sol) attacked the whole diff once: one medium finding,
`ShowAll` not checking its revision when no path is sent. Its doc now says
so, with tests. The implementers found two batch differences on damaged
history; fallbacks keep today's results.

## Tasks

### Task 1: Settle refs without a merge-base each

- [x] Pin the current behavior with tests where none covers it: a merged
  ref is skipped, an unmerged one is read, a ref whose tip or whose history
  cannot be read is unreadable alone.
- [x] List merged and unmerged refs with `for-each-ref --merged` and
  `--no-merged`, walk the unmerged history once with `rev-list`, and ask
  per ref, as before, for any ref the listings or the walk cannot settle.

### Task 2: One ls-tree per tree source

- [x] Pin the current behavior of `readTree` with tests where none covers
  it: active and archived change folders, records, nested and non-record
  files, `docket/records` that is not a folder.
- [x] Read the three folders with one `ls-tree`, and with the three old
  calls when it fails.

### Task 3: One cat-file for a branch's change files

- [x] Pin the current behavior of `scan.ReadFiles` on a branch with tests
  where none covers it.
- [x] Read the kept files with one `cat-file --batch`, and with `git show`
  any file it does not return.
