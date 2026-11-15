---
title: "Settle four cases the plan instruction left open: wording-only MODIFIED, review fixes, who found it, plain dependencies"
kind: prompt
source: "PR #163 (comment-in-code-span) and PR #165 (init-reuse-workflow)"
---

## What

The plan instruction in `shipped/schema/schema.yaml` and its installed copy gain a sentence or two beside four existing rules: a wording-only MODIFIED requirement is still claimed, with an acceptance item that existing tests pass unchanged; a review's fix in a completed task's own code changes that task's acceptance items with a Revision, and Compensates stays for overturning requirements; a Revision says who found the change when a reviewer or the implementer did; a dependency used as it is goes in the task's Interfaces block. `internal/setup/history.go` gains the v0.29.0 hash.

## Why

The third and fourth plans written to the instruction met these cases and had to guess.

## Verification

`dkt init --force` from this branch reinstalled the copy, which matches `shipped/`; `openspec schema validate docket` passes; `go test ./internal/setup/` passes with the new hash.

## Review

codex (gpt-5.6-sol) reviewed the whole diff and marked two findings must-fix. Taken up: the Interfaces rule now names a dependency used as it is, which the Reuse rule sends there (ed9a3a1). Turned down: that `dkt sync` would tick an acceptance item added to a completed task from its old commits; plan-sync's "Ticks from trailers" ticks a task's steps as a whole by design, and the fix's commit and the Revision record it. A second codex reviewer (high effort) decided it: not valid.
