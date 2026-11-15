---
title: "Say how fine a Reuse row is, cover must-not lines, and skip the task reviewer in a one-task plan"
kind: prompt
source: "Second plan written by the new instruction (fence-rule-drift, #158): three places it left unclear"
---

## What

The plan instruction in `shipped/schema/schema.yaml` and its installed copy: a Reuse row is one named thing the change builds or changes (a function, a type, a command, a file); a Review Focus line saying that something must not happen or change becomes an acceptance item too; a plan with a single task writes `**Review:** checks`.

## Why

Writing the plan of `fence-rule-drift`: nothing said whether two parts of one function were one Reuse row or two, or whether a guarantee that something stays unreported needed its own item. With one task, the task's reviewer and the reviewers before merging read the same diff, so the first added a review without adding a reader.

## Verification

`openspec schema validate docket` passes; the installed copy, written by this branch's `init --force`, equals `shipped/`; `gofmt -l .`, `go vet ./...` and `go test ./...` pass. Not verified: an agent writing a plan by the new wording.

## Review

Codex (gpt-5.6-sol), a different model, reviewed the whole diff: one must-fix finding, a row "for every capability" and "parts of one function are one row" gave two units for a row. Fixed by naming one unit; a small fix, not reviewed again.
