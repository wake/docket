---
title: "A read-only command that outputs a whole project at once"
kind: feature
source: "docket/ideas/dkt-project-snapshot.md"
specs:
  project-snapshot: Project Snapshot
---

# Proposal

## Why

A viewer of a project's work needs what dkt knows about every change, record, idea, issue and spec, and today it would have to run `status`, `lint`, `domains` and `sync --check` one by one, each scanning the project again, and still miss what no command reports: where each piece of work is, which stages it has reached, its merge and release, and what its deltas change. One read-only command can scan once and report it all in Docket's own terms.

## What Changes

- New command `dkt snapshot [--json]`: one result holding the default branch's tip, every change and record found by `workspace-scan` with its locations, tasks and their progress, documents, deltas, stages, merge and release, every idea and issue with its state and position in the order, every main spec with its requirements, and the domain tree.
- A chain of stages for a piece of work (a change: `proposal`, `design`, `plan`, `dev`, `merged`, `archived`; a record: `record`, `dev`, `merged`), a task that is `done` or `pending`, the date of an idea, and the state of an idea (`queued`, `pool`, `dropped`, `taken`) and of an issue (`open`, `in-progress`, `done`).
- dkt reads delta specs and main specs itself, so the command needs no OpenSpec CLI.
- `cli-output` lists `dkt snapshot` among the commands that need a repository.

## Non-Goals

- Writing anything, apart from the project index registration that `project-index` adds to every command.
- Contacting a remote: remote-tracking branches are read as they were at the last fetch.
- Labels, sorting for display, filtering and counts: the viewer does those from the fields.
- Worktree and file listings for browsing, and file contents: the viewer's server reads git and the file system for those.
- Replacing `dkt status`, `dkt lint`, `dkt domains` or `dkt sync --check`: they keep their own output, and `dkt lint` stays the authority on whether a delta is valid.
- Caching between runs.

## Capabilities

### New Capabilities
- `project-snapshot`: what `dkt snapshot` reports, from which sources, and the stage and state rules it reports.

### Modified Capabilities
- `cli-output`: `dkt snapshot` needs a repository.

## Impact

- New: `internal/snapshot`, `internal/cli/cmd_snapshot.go`, a delta and main spec reader.
- Changed: `internal/cli/command.go` (the command list), `README.md`.
- Reuses `internal/scan`, `internal/idea`, `internal/change`, `internal/record`, `internal/domain` and `internal/gitx`, with batched git reads.
- No new dependency; OpenSpec is not needed to run the command.
