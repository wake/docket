---
title: Split status.LoadRecords
---

Recorded 2026-11-01, from the reviews of `add-record-tracking`, which judged
it able to wait. Nothing below is decided.

## Problem

`status.LoadRecords` (`internal/status/records.go`) is one function of
about 110 lines that does five jobs:

- picks the records the query asks for: named slugs, open records, merged
  ones under `--merged`;
- finds each merged record's merge commit, drops those `--since` excludes,
  sorts them newest first, counts the total and applies `--limit`;
- groups the records by first source, reads their files and commits once
  per source, parses each record and summarizes its tasks;
- sorts the result, open records by slug before merged ones;
- reads the tags and gives each merged record the first that holds it.

It is correct and tested, but every new filter or field lands in the same
function, and a reader has to hold all five jobs to change one.

## Direction

A restructuring that changes no behavior: light review, a record of kind
`refactor`. Split along the five jobs, keeping `LoadRecords`'s signature and
its git process count (one `Additions`, one `ReadFiles` and at most one
`Commits` per source, one `Tags`). The tests in `internal/status` and
`internal/cli` that cover it pass before and after, and mutating the moved
lines makes one fail.

It could be taken up on its own, or before the next change that touches
`LoadRecords`, such as a `--before` cursor for the workbench.
