---
title: Refuse a --source made of spaces in dkt new
source: review of add-init-and-new, 2026-10-24; moved from docket/ideas/2026-10-24-blank-source.md
---

## Current

`dkt new --source " "` is accepted: internal/cli/cmd_new.go:45 refuses only `source == ""`.

## Expected

`change-creation` (spec line 45) requires a non-empty `--source`. `dkt record` and `dkt issue` already refuse a blank one with `strings.TrimSpace` (cmd_record.go:35, cmd_issue.go:41); `dkt new` should match, exit 2.

## Unchanged

A real source is stored as given; the other usage errors.

## Found

Review of `add-init-and-new`, 2026-10-24. Still present on main.
