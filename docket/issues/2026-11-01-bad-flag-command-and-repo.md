---
title: Name the command and repository after a bad flag
source: review of add-init-and-new, 2026-10-24; moved from docket/ideas/2026-10-24-bad-flag-command-name.md
---

## Current

`dkt --bogus probe --json` prints `"command":""` and `"repo":null` even inside a repository (internal/cli/run.go:31 calls `usageFailure` with an empty name and no repo).

## Expected

`cli-output` (spec lines 21, 28): `repo` is null only when the repository could not be identified; every result identifies what it describes.

## Unchanged

Exit code 2, the usage finding, one JSON object on stdout.

## Found

Review of `add-init-and-new`, 2026-10-24; reproduced on main with a built `dkt`.
