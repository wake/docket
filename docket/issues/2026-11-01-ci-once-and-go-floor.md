---
title: Run CI once per pull request and test the Go floor
source: review of add-init-and-new, 2026-10-24; moved from docket/ideas/2026-10-24-ci-triggers-and-go-floor.md
---

## Current

.github/workflows/ci.yml runs on `push` and `pull_request` (lines 3-5), so a pull request branch runs twice. Every job uses `go-version: stable` (lines 45, 57, 88).

## Expected

go.mod declares `go 1.24` as the floor, so CI should test it. A branch with a pull request should run once.

## Unchanged

The docs-only skip, the macOS and Linux matrix, and the checks that run.

## Found

Review of `add-init-and-new`, 2026-10-24. Still present on main. This is a check setting.
