---
title: "Ship the docket-doctor skill"
kind: prompt
source: "Rebuild plan, step 5: docket-doctor"
---

## What

Added `skills/docket-doctor/` (`SKILL.md`, `convert.md`, `agents/openai.yaml`), copied from the approved drafts. In `AGENTS.md`, docket-doctor is no longer listed as missing, and its fallback sub-bullet is removed.

## Why

The docket skill, docket-init and `dkt init` send agents to docket-doctor, and until now it did not exist.

## Verification

- `cmp` of `SKILL.md` and `convert.md` against the drafts: identical. Front matter `name` is `docket-doctor`.
- Every rule name `SKILL.md` mentions exists under `internal/`: literal in `cmd_inspect.go`, `cmd_init.go` or `runner.go` (install-ignored, title-missing, trailer-missing, workflow-*, config-schema-*, guidance-differs, symlink-refused, openspec-*); built as kind-status for schema-/skill-missing, -outdated, -differs, and as guidance-state for guidance-missing, -outdated.
- Result fields `files`, `openspec`, `default_branch` (`cmd_inspect.go`) and `result.changes[].schema` (`cmd_lint.go`) exist.
- `go vet ./...` and `go test ./...` pass.
- Not verified: the exit codes 2 and 4 the skill names.

## Review

One adversarial review by codex on the whole diff. Two findings, both about the prompt text: lint run after a missing OpenSpec (exit 4), and `files` not covering `openspec init` files. Both were fixed in the approved text, which was copied again and checked with `cmp`.
