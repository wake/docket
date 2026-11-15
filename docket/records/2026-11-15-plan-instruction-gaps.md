---
title: "Close four gaps the plan instruction left in the first plan written by it"
kind: prompt
source: "First plan written by the v0.26.0 instruction (reuse-table-slim, #154): four places it left unclear"
---

## What

The plan instruction in `shipped/schema/schema.yaml` and its installed copy: the writing-plans header leaves out the line naming a sub-skill to execute the plan, since "Task execution" in `docket/workflow.md` decides who implements it; an acceptance item names every case it covers instead of saying a rule applies the same way elsewhere; each line of Review Focus becomes an acceptance item of the task that owns it. The schema's hash for v0.27.0 in `internal/setup/history.go` is replaced, as that release is not out.

## Why

Writing the plan of `reuse-table-slim` by the v0.26.0 instruction: the copied header recommended one implementer per task, against "Task execution"; writing-plans asks for a Review Focus test in step style, which the new format has no place for; an item saying the earlier table follows "the same rules" let an implementation meet it in part, which plan review caught. A fourth gap, how to write a result that counts only when reached a certain way, is left out: one sentence could not say it without inviting steps back.

## Verification

`openspec schema validate docket` passes; the installed copy, written by this branch's `init --force`, equals `shipped/`; `TestEarlierHoldsShippedFiles` and `TestEarlierHoldsReleases` pass with the replaced hash.

## Review
