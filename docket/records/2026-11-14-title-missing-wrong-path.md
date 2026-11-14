---
title: "Stop calling a delta with no main spec a new capability"
kind: fix
source: "docket/issues/2026-11-07-title-missing-wrong-path.md"
---

## What

`title-missing` for a delta with no main spec at its path no longer calls the capability new. It names the path where dkt found no main spec and gives both ways out: name the capability if it is new, or move the delta to the capability's existing path. The rule, its line and the exit status are unchanged (`internal/lint/document.go`).

## Why

dkt only knows there is no main spec at the delta's path, and a mistyped path looks the same. Told the capability was new, an author named it, lint passed, and archiving created a second spec beside the real one.

## Verification

A regression test with a main spec at `billing/refund` and a delta at `refund` failed before the fix and passes after it. `gofmt -l .`, `go vet ./...` and `go test ./...` pass. A dkt built from this branch, run on that layout in a scratch repository, prints the new message and exits with 1.

## Review

Codex (gpt-5.6-sol), a different model, reviewed the whole diff and found nothing: the message holds in every case that reaches it, its two ways out match the issue, and nothing else quotes the old text.
