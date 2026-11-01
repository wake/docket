---
title: Say that dkt conflicts reads each change's first source
source: review of add-conflicts, 2026-10-24; moved from docket/ideas/2026-10-24-conflicts-first-source.md
---

## Current

`dkt conflicts` compares only the content of each change's first source (internal/conflicts/load.go:10). Its usage line (internal/cli/cmd_conflicts.go:23) and docs do not say so.

## Expected

Behavior as shipped, and as `change-status` (spec line 12) states for `dkt status`: content comes from the source the scan lists first. The command's documentation should say the same.

## Unchanged

Which source is read, and the overlap results.

## Found

Review of `add-conflicts`, 2026-10-24, after `dkt status` began showing where content comes from.
