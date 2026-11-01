---
title: Drop the dangling colon when a tool wrote nothing
source: review of add-init-and-new, 2026-10-24; moved from docket/ideas/2026-10-24-dangling-colon.md
---

## Current

`gitx.ExitError.Error` (internal/gitx/git.go:34) and `openspec.FailedError.Error` (internal/openspec/runner.go:59) end in `: ` when the tool wrote nothing to standard error.

## Expected

`openspec-runtime` (spec line 33): report the command, exit status and the tool's standard error. With no standard error, the message ends after the status.

## Unchanged

Messages that have standard error text, and the exit codes.

## Found

Review of `add-init-and-new`, 2026-10-24. Still present on main.
