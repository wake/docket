---
title: "Say in AGENTS.md to check gofmt before pushing"
kind: prompt
source: "Maintainer agreed on 2026-11-07: CI's gofmt step failed pull request 106 although the listed checks passed"
---

## What

The Development section of `AGENTS.md` now lists `gofmt -l .` with `go vet ./...` and `go test ./...`. This file only serves this repository and is not shipped.

## Why

CI runs `gofmt -l .` first, but `AGENTS.md` listed only the other two checks. Pull request 106 passed them locally, then failed in CI on one missing space, and the `go vet` and `go test` steps in the `test` jobs never ran.

## Verification

Mechanical: `.github/workflows/ci.yml` runs the same three commands, so the sentence holds. `gofmt -l .` printed nothing here. `AGENTS.md` still loads through `CLAUDE.md`. No code changed.

## Review

Not reviewed yet.
