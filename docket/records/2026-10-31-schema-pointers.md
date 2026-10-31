---
title: The shipped schema stops pointing to files only Docket has
kind: prompt
source: docket/ideas/2026-10-31-existing-project-adoption.md, the read-only trial on a copy of the Purdex repository
---

## What

The shipped schema's Builds On instruction and the proposal template's
comment no longer send agents to `docs/ARCHITECTURE.md` and the capability
index: they search the code and `openspec/specs/`, and read the project's
own architecture notes if its instructions name any. The Reuse table's
Evidence example drops `dkt index`.

## Why

In a trial on a copy of the Purdex repository, the agent writing a baseline
looked for `docs/ARCHITECTURE.md` and `dkt index`. Only Docket's own
repository has the first, and the second is still planned; it reported both
missing in its proposal.

## Verification

In a throwaway repository set up by v0.9.1, the branch build reports both
files as differing and `dkt init --force` updates them. OpenSpec 1.14.3
prints the new proposal and plan instructions and template; no shipped
schema file names `ARCHITECTURE` or `dkt index`. `go vet` and
`go test ./...` pass.

## Review

Codex (gpt-5.6-sol) attacked the whole diff once and approved with no
findings: the schema and template agree, no other shipped prompt points to
something only Docket's repository has, and Docket's own AGENTS.md still
names `docs/ARCHITECTURE.md`.
