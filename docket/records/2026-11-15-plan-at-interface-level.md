---
title: "Plan what to build, not how to type it, and let one implementer do the whole change"
kind: prompt
source: "Maintainer's review of the workflow on 2026-11-15: plans written step by step and one implementer per task cost more than they give with current models"
---

## What

The plan instruction and template in the schema: a task says what to build and how it is accepted, with an `**Interfaces:**` block and checkboxes as acceptance items, not step-by-step code, commands or commit messages. The apply instruction keeps only what dkt needs and points to `docket/workflow.md`. In the workflow template, now version 3, "Task execution" has one implementer do the whole change by default, and "Reviewers" has a reviewer mark its must-fix findings.

## Why

Step-level plans ran longer than the code they produced and left the implementer nothing to decide; a fresh agent per task reread the code for small changes. The apply instruction repeated the workflow's rules and still put the user's instructions first. Nothing asked reviewers for the must-fix mark that the arbitration rule depends on.

## Verification

The shipped schema and plan template are identical to the copies installed here; `dkt inspect` of this build reports nothing; `gofmt -l .`, `go vet ./...` and `go test ./...` pass. Not verified: an agent writing a plan by the new instruction, and docket-doctor merging template 3 into a project.

## Review

