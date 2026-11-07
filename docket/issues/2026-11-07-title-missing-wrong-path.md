---
title: "dkt lint calls a delta on a wrong path a new capability"
source: "Found while designing add-domain-tree, 2026-11-07"
---

## Current

For a delta with no main spec at its path, `dkt lint <change>` says `new capability "refund" has no name: add ...` (`internal/lint/document.go:72`), even when the capability exists elsewhere. Doing what it says gives a clean lint, and archiving then creates a second spec, with no warning.

To reproduce: a main spec at `openspec/specs/billing/refund/`, and a change with an ADDED-only delta at `specs/refund/spec.md`. Run `dkt lint`, add the name, then `openspec archive`: both `billing/refund` and `refund` now exist.

## Expected

The message does not call it new, and gives both ways out: name it if it is new, or use the capability's existing path. `dkt` only knows there is no main spec there; `shipped/schema/schema.yaml` says a mistyped path "targets a capability that does not exist rather than the one you meant", and the `change-lint` requirement "New capabilities named" asks only that the finding name the capability.

## Unchanged

`title-missing` fires for the same deltas and still exits with 1. A delta with a main spec gets no finding.

## Evidence

Reproduced on `main@c110c22` (v0.19.0) with OpenSpec 1.16.0, validation on.
