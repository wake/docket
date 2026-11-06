---
title: "Keep a short record of light-review work"
kind: feature
source: "Rebuild plan, step 4: dkt record"
specs:
  records: Records
---

# Proposal

## Why

The docket skill tells an agent on the light review path to open a record with `dkt record`, but the command does not exist, so light-review work leaves no trace in the repository beyond its pull request. Titles and record bodies also need a length rule that does not depend on the writer's language: the previous version counted words, with a separate weight for Chinese, Japanese and Korean, and that rule fails outright for languages written without spaces.

## What Changes

- New command `dkt record <name> --kind fix|refactor|test|prompt|checks --source <text> --title <text>`. It writes `docket/records/<date>-<name>.md` from a shipped template, takes the first free suffix when the slug is used anywhere the scan looks, and reports the slug it wrote.
- Shipped template `shipped/record/template.md`, the text the maintainer settled.
- `dkt lint` checks records: the ones the branch adds when no name is given, and any record named on the command line (a name starting with a date). It checks front matter, sections, leftover template text, task headings and length, and reports a `Record` trailer that names no record.
- `dkt sync <slug>` with a slug starting with a date ticks the steps under a record's `## Tasks` from `Record` trailers, as it does a plan's from `Change` trailers.
- Length is measured in UTF-8 bytes, with no rule per language: a title holds at most 120 bytes and a record's sections together at most 1,440. **BREAKING**: `dkt new` refuses, and `dkt lint` reports, a change title over 120 bytes.
- The scan lists record files in every worktree and branch.
- The docket skill says to keep a record's sections short rather than "a few sentences each", and to run `dkt lint` after filling a file issue. This repository's `AGENTS.md` drops its stopgap for records.

## Non-Goals

- `dkt status` listing records, and any way to see merged records: that belongs to `dkt status`.
- `dkt issue`, its template and its checks.
- `dkt inspect` checking records or their trailers on the default branch.
- Checking merged records: lint looks at what the branch adds, so a later, stricter rule does not fail history.
- A length rule for `source`, which stays one line of any length.
- Fairness across every script: languages that use more bytes per letter than English reach the limit sooner, and that is accepted until someone reports it.

## Capabilities

### New Capabilities
- `records`: what a record is, how `dkt record` writes one, how `dkt lint` checks it and how `dkt sync` ticks its tasks.

### Modified Capabilities
- `change-metadata`: a title has a length limit, and how text length is measured.
- `change-lint`: what is linted now includes records, a change title over the limit is reported, and the result lists the records linted.
- `cli-output`: `dkt record` is a command that needs a repository.
- `workspace-scan`: a scan also finds record files.

## Impact

- New: `internal/record`, `internal/scan/records.go`, `internal/lint/record.go`, `internal/cli/cmd_record.go`, `shipped/record/template.md`.
- Changed: `internal/change/metadata.go`, `internal/gitx` (trailers by key), `internal/cli` (`dkt new`, `dkt lint`, `dkt sync`, the command list, `Options.Now`), `internal/lint`, `shipped/skill/SKILL.md` and its installed copies, `AGENTS.md`.
- `dkt new` and `dkt lint` become stricter for change titles over 120 bytes. No archived or active change has one.
- No new dependency.
