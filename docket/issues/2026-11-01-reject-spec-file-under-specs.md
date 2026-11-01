---
title: Reject a spec file directly under specs/
source: review of add-conflicts, 2026-10-24; moved from docket/ideas/2026-10-24-spec-file-under-specs.md
---

## Current

`delta.CapabilityPath` (internal/delta/capability.go:9) accepts `specs/spec.md` and returns the capability `spec.md`. Lint, archive, status and conflicts all call it, so each treats that file as a delta spec.

## Expected

A capability path has the form `specs/<capability>/spec.md`. OpenSpec ignores a file directly under `specs/`, and `conflict-detection` compares "capability path" values; Docket should ignore it too.

## Unchanged

Nested capability paths such as `account/login` keep working; `specs/<name>/spec.md` is still accepted.

## Found

Review of `add-conflicts`, 2026-10-24. Still present on main: the function only checks the `specs/` prefix and the `/spec.md` suffix.
