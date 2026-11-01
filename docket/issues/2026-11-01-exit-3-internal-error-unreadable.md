---
title: Exit 3 for an internal error with unreadable sources
source: review of add-init-and-new, 2026-10-24; moved from docket/ideas/2026-10-24-internal-error-exit-code.md
---

## Current

`ExitCode` (internal/cli/envelope.go:106) returns 1 for any non-usage, non-environment error before it looks at `Unreadable`, so an internal error plus unreadable sources exits 1.

## Expected

`cli-output` (spec line 50): a command that could not read a source "SHALL exit with 3 even when it also found problems elsewhere".

## Unchanged

Usage errors exit 2, environment errors 4; an error with no unreadable source exits 1.

## Found

Review of `add-init-and-new`, 2026-10-24. Still present on main.
