---
title: Match --json only as a flag of its own
source: review of add-init-and-new, 2026-10-24; moved from docket/ideas/2026-10-24-json-requested-match.md
---

## Current

`jsonRequested` (internal/cli/run.go:166) matches `--json` given as another flag's value (`--default-branch --json --bogus`) and a bare `json` positional, so such a failing call answers in JSON.

## Expected

`cli-output` (spec line 10): `--json` is a flag; only it selects JSON output. A value or positional named `json` should not.

## Unchanged

`--json`, `-json` and `--json=true` after a parse failure or for an unknown command.

## Found

Review of `add-init-and-new`, 2026-10-24. Still present on main.
