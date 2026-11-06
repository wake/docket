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
- Every rule name `SKILL.md` mentions exists under `internal/`: literal in `cmd_inspect.go`, `cmd_init.go` or `runner.go` for install-ignored, title-missing, trailer-missing, workflow-missing, workflow-outdated, workflow-template-unknown, workflow-section-missing, config-schema-default, config-schema-other, guidance-differs, symlink-refused, openspec-missing, openspec-unsupported. Built from kind and status in `cmd_inspect.go` for schema-/skill-missing, -outdated, -differs, and from `"guidance-" + state` for guidance-missing, guidance-outdated.
- Result fields `files` (with `kind`, `status`), `openspec` (`version`, `minimum`, `supported`) and `default_branch` exist in `cmd_inspect.go`; `result.changes[].schema` in `cmd_lint.go`.
- `go vet ./...` and `go test ./...` pass.
- Not verified: the exit codes 2 and 4 the skill names.

## Review

One adversarial review by codex on the whole diff. Two findings, both about the approved prompt text, so left for the maintainer:
- high: when `dkt inspect` reports `openspec-missing` or `openspec-unsupported`, step 1 still runs `dkt lint --json`, which exits 4 with no usable `result`.
- medium: step 1 says `files` holds every file `dkt init` writes, but inspect omits the files `openspec init` generates.
