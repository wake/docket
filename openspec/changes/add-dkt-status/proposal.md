---
title: "Name the next idea to work on with dkt status"
kind: feature
source: "Rebuild plan, step 4: dkt status, minimum version"
specs:
  ideas: Ideas
---

# Proposal

## Why

The docket skill tells an agent picking up work to run `dkt status --json` and do the one `result.next` names, but the command does not exist, so that step ends in an unknown command. This is the smallest version that makes the step work, and it belongs in the first release.

## What Changes

- New command `dkt status`: reads `docket/order.md` and the ideas it names from the default branch tip, and reports in `result.next` the first idea still to do, with `result.reason` saying why when there is none.
- `docket/order.md` gets a format: one `- docket/ideas/<name>.md` line per idea, in order.
- An idea's front matter gets a settled form: `title`, `dropped: true` and a one-line `needs` list.
- An idea stops being next once a change or record anywhere names it as its source, and an idea that needs another waits until that work is on the default branch.
- This repository's `AGENTS.md` drops `dkt status` from its stopgaps, and the README lists the command.

## Non-Goals

- Issues: their state and list.
- A list of ideas, or of their states (pending, in progress, done). `dkt status` names one idea.
- A list of changes or records, or any flag such as `--merged`.
- `dkt lint` checking the order or the ideas: `dkt status` reports only what it needs to read, for the ideas the order reaches.
- A shipped template or prompt text for `docket/order.md` or ideas.
- Fetching from a remote: the default branch tip is what is already on the machine.

## Capabilities

### New Capabilities
- `ideas`: what an idea file and `docket/order.md` hold, and how `dkt status` names the next idea from them.

### Modified Capabilities
- `cli-output`: `dkt status` is a command that needs a repository. `add-dkt-issue` changes the same requirement, so this delta is written once that change has merged (plan Task 3).

## Impact

- New: `internal/idea`, `internal/cli/cmd_status.go`.
- Changed: `internal/cli/command.go` (the command list), `AGENTS.md`, `README.md`.
- A project that has no `docket/order.md` is unaffected: `dkt status` says there is nothing ordered.
- No new dependency, and OpenSpec is not needed to run the command.
