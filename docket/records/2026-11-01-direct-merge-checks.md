---
title: Run only the checks that read what a direct merge changed
kind: prompt
source: 'maintainer, 2026-11-01: a direct merge runs only the checks related to what changed'
---

## What

The workflow template's direct merge step now runs only the checks that
read what changed: none for docs that nothing builds, embeds or tests;
the build and tests for code, tests or manifests. Both READMEs say the
same. CI gains a `changes` job: test, openspec and duplicates skip when a
diff touches only those docs; lint always runs.

## Why

"Pass the automated checks" made an agent run the whole test suite, about
five and a half minutes, for an edit to `docket/order.md` that no test
reads, and CI ran it again for every docs-only pull request.

## Verification

The template prints through `dkt workflow template`; the READMEs match it.
The CI decision, run from `ci.yml` against ten git diffs, gives docs-only
for ideas, the order and the READMEs, and everything for code, an embedded
template, code moved into `docs/`, lookalike paths, an empty diff and a
missing base. Dropping `--no-renames` makes one case fail.

## Review

<!-- Who reviewed the diff (a model, or a person), what the review found,
and how each finding was handled. -->
