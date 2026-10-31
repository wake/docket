---
title: The shipped schema stops pointing to files only Docket has
kind: prompt
source: docket/ideas/2026-10-31-existing-project-adoption.md, the read-only trial on a copy of the Purdex repository
---

## What

The shipped schema's Builds On instruction and the proposal template's
comment no longer send agents to `docs/ARCHITECTURE.md` and the capability
index: they search the code and `openspec/specs/`, and read the project's
own architecture notes if its instructions name any. The Reuse table's
Evidence example drops `dkt index`.

## Why

In a trial on a copy of the Purdex repository, the agent writing a baseline
looked for `docs/ARCHITECTURE.md` and `dkt index`. Only Docket's own
repository has the first, and the second is still planned; it reported both
missing in its proposal.

## Verification

<!-- How it was checked and the result, by the light review steps of the
workflow file: the regression test seen failing, then passing; the
mutation that made a test fail; the mechanical check of the prompt; the
pull request's own check run. -->

## Review

<!-- Who reviewed the diff (a model, or a person), what the review found,
and how each finding was handled. -->
