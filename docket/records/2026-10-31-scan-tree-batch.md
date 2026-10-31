---
title: Read the trees of unmerged refs in a batch
kind: refactor
source: docket/ideas/2026-10-31-scan-tree-batch.md, measured on a copy of the Purdex repository
---

## What

The scan reads the trees of every unmerged ref and the default tip level
by level: one `cat-file --batch` for the root trees, then `openspec` and
`docket`, then `changes` and `records`, then the archive, each commit and
tree once. A commit with a missing object or a malformed tree goes to the
old `ls-tree`.

## Why

On a copy of the Purdex repository `dkt status` started 322 git processes,
308 of them one `ls-tree` per unmerged ref, and took 21.9 s; none of those
refs holds a change.

## Verification

Tests pinned first, passing on the old code: non-folder entries, record
modes, refs sharing a commit, one missing object per level, the default
tip, a SHA-256 repository. Mutating the changed lines failed a test except
two equivalent mutants. Purdex copy: 322 git processes and 22.1 s before,
18 and 1.7 s after; `dkt status --json` identical.

## Review

Codex (gpt-5.6-sol) attacked the whole diff once: two medium findings, raw
tree entries whose name holds a slash and duplicate names, which ls-tree
reads differently. A commit with such a tree now goes to the old `ls-tree`,
with tests built from literal trees.
