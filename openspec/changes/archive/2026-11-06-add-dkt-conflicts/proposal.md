---
title: "Find requirements that parallel changes both touch"
kind: feature
source: "Rebuild plan, step 4: dkt conflicts"
specs:
  conflict-detection: Conflict Detection
---

# Proposal

## Why

Changes made in parallel, in other worktrees or on other branches, can modify, remove or rename the same requirement. Nothing reports it until archiving, when the second change to land overwrites the first or fails to apply. The docket skill already tells agents to run `dkt conflicts` before parallel work, and the command does not exist yet.

## What Changes

- New command `dkt conflicts [<slug>]`: for every active change in every worktree and branch the workspace scan reads, it finds the requirements that two or more changes touch (add, modify, remove, or rename from or to), and reports each with the changes, their operations and the sources holding them. With a slug, it reports only the overlaps involving that change.
- Delta specs are parsed by OpenSpec, from a copy that does not depend on the rest of the change being complete, so a change still being proposed is compared too.
- It exits with 1 when it reports an overlap, so a script or an agent can stop before starting work that would collide.
- `dkt conflicts` joins the commands that need a git repository.

## Non-Goals

- Comparing plans' `**Files:**` blocks: the docket skill leaves that to the agent, and a requirement overlap is what archiving breaks on.
- Overlaps between an active change and the main specs on the default branch, or with archived changes.
- Merging or resolving overlaps, or deciding which change goes first.
- Fetching from remotes: remote-tracking branches are read as of the last fetch, as `workspace-scan` defines.

## Capabilities

### New Capabilities
- `conflict-detection`: which active changes touch the same requirement, where they are found, and how `dkt conflicts` reports it.

### Modified Capabilities
- `cli-output`: `dkt conflicts` is added to the commands that need a git repository.

## Impact

- New `internal/conflicts` package and `internal/cli/cmd_conflicts.go`; `dkt --help` lists the command.
- `internal/scan`: a use found on a branch carries the commit it was read from.
- `internal/gitx`: lists the files under a folder of a commit.
- Runs OpenSpec (`openspec show --json --deltas-only`) once per distinct copy of a change when two or more active changes exist.
