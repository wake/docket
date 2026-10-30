---
title: Start fewer git processes in the scan
kind: refactor
source: docket/ideas/2026-10-24-scan-git-calls.md, with measurements from the workbench session
---

## What

The scan lists unmerged refs with one `for-each-ref --no-merged` instead of
one `merge-base --is-ancestor` per ref, reads each tree source with one
`ls-tree` instead of three, and reads a branch's change files with one
`cat-file --batch`. Every command's results stay the same.

## Why

`dkt status` started 46 git processes here (3.3 s), one or more per ref, and
did not finish in two minutes in a repository with thousands of refs.

## Verification

<!-- How it was checked and the result, by the light review steps of the
workflow file: the regression test seen failing, then passing; the
mutation that made a test fail; the mechanical check of the prompt; the
pull request's own check run. -->

## Review

<!-- Who reviewed the diff (a model, or a person), what the review found,
and how each finding was handled. -->

## Tasks

### Task 1: One listing of the unmerged refs

- [ ] Pin the current behavior with tests where none covers it: a merged
  ref is skipped, an unmerged one is read, a ref whose ancestry cannot be
  computed is unreadable alone.
- [ ] Replace the per-ref `merge-base --is-ancestor` with one
  `for-each-ref --no-merged`, falling back to the per-ref check when it
  fails.

### Task 2: One ls-tree per tree source

- [ ] Pin the current behavior of `readTree` with tests where none covers
  it: active and archived change folders, records, nested and non-record
  files, `docket/records` that is not a folder.
- [ ] Read the three folders with one `ls-tree`.

### Task 3: One cat-file for a branch's change files

- [ ] Pin the current behavior of `scan.ReadFiles` on a branch with tests
  where none covers it.
- [ ] Read the kept files with one `cat-file --batch`.
