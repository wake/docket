# plan-sync Specification

## Purpose
Defines how `dkt sync` ticks plan.md from commit trailers, so that plan.md can
be the only task list even though superpowers records its progress elsewhere
and never ticks the plan.

## Requirements

### Requirement: Ticks from trailers
`dkt sync <slug>`, for a slug that does not start with a date, SHALL collect the commits reachable from HEAD and not from the change's base whose trailers include `Change: <slug>`, take every `Task: N` trailer of those commits, and tick every unticked `- [ ]` step in the `### Task N` section of plan.md for each N taken. Trailers SHALL be read the way git reads them. A slug that starts with a date names a record, which `records` covers.

#### Scenario: Task with commits ticked
- **WHEN** two commits carry `Change: add-x` and `Task: 3`, and Task 3 has four unticked steps
- **THEN** after `dkt sync add-x` all four steps of Task 3 are ticked

#### Scenario: Task without commits left alone
- **WHEN** no commit carries `Task: 4` for `add-x`
- **THEN** after `dkt sync add-x` the steps of Task 4 are as they were

#### Scenario: Two tasks in one commit
- **WHEN** one commit carries `Change: add-x`, `Task: 1` and `Task: 2`
- **THEN** `dkt sync add-x` ticks the steps of both Task 1 and Task 2

#### Scenario: Commits counted from the base
- **WHEN** `add-x`'s `docket.yaml` records a base, and a commit carrying `Change: add-x` and `Task: 1` is reachable from that base
- **THEN** `dkt sync add-x` does not tick Task 1 for that commit

### Requirement: Only checkboxes change
`dkt sync` SHALL change nothing but `[ ]` to `[x]` in the step checkboxes of plan.md, or, for a slug that names a record as `records` defines, of that record's file. It SHALL never untick a step, never touch another file, and never commit. It SHALL NOT require the OpenSpec CLI.

#### Scenario: Nothing else changes
- **WHEN** `dkt sync add-x` ticks steps in plan.md
- **THEN** plan.md differs from before only in those checkbox characters, and `git status` shows no other modified file

#### Scenario: Record file only
- **WHEN** `dkt sync 2026-11-01-fix-x` ticks steps in that record
- **THEN** the record differs from before only in those checkbox characters, and `git status` shows no other modified file

### Requirement: Trailer problems
`dkt sync` SHALL report as a finding every `Task: N` trailer, on a commit for this change, whose N is not a `### Task` heading in plan.md, including tasks moved to Cancelled Tasks, and SHALL still tick the tasks it can.

#### Scenario: Trailer for an unknown task
- **WHEN** a commit carries `Change: add-x` and `Task: 9`, and plan.md has no Task 9
- **THEN** `dkt sync add-x` reports Task 9 and the commit, ticks the other tasks, and exits with 1

### Requirement: Check mode
`dkt sync <slug> --check` SHALL write nothing, and SHALL exit with 1, listing the tasks, when any step would be ticked.

#### Scenario: Check finds unticked work
- **WHEN** Task 2 has commits and unticked steps, and a user runs `dkt sync add-x --check`
- **THEN** plan.md is unchanged, Task 2 is listed, and the command exits with 1

#### Scenario: Check on a synced plan
- **WHEN** `dkt sync add-x` has just run and a user runs `dkt sync add-x --check`
- **THEN** the command exits with 0
