---
title: Make TestOriginHEADNotASource able to fail
source: review of add-init-and-new, 2026-10-24; moved from docket/ideas/2026-10-24-origin-head-test.md
---

## Current

`TestOriginHEADNotASource` (internal/scan/scan_test.go:120) cannot fail on its own: `origin/HEAD` points at the default tip, so it is never listed as an extra source; only the guard in `internal/gitx` is tested.

## Expected

`workspace-scan` (spec line 38) lists the sources a scan reads; `origin/HEAD` is not one. The test should fail if the scan listed it.

## Unchanged

Production code; the gitx guard test.

## Found

Review of `add-init-and-new`, 2026-10-24. Mutate the scan to list it and see the test fail.
