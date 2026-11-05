# plan-sync Specification

## Purpose
Defines how `dkt sync` ticks a change's plan.md from commit trailers, so that plan.md stays the only task list even though superpowers records its progress elsewhere and never ticks the plan.

## Requirements

### Requirement: Trailers read
`dkt sync <slug>` SHALL read the values of the `Change` trailer, as git reads trailers, of every commit reachable from HEAD. A value `<slug>/<N>` names Task N of the change, and `<slug>/<N>,<M>` names several tasks; a value that is `<slug>` alone names no task. A value for any other slug SHALL be ignored, even when it starts with `<slug>`.

#### Scenario: Two tasks in one commit
- **WHEN** one commit carries `Change: add-x/1,2`
- **THEN** `dkt sync add-x` takes the commit for both Task 1 and Task 2

#### Scenario: Longer slug ignored
- **WHEN** a commit carries `Change: add-x-2/1` and no commit names `add-x` Task 1
- **THEN** `dkt sync add-x` leaves the steps of Task 1 as they were

#### Scenario: Trailer outside the last paragraph
- **WHEN** a commit's message holds `Change: add-x/1` in a paragraph that is not its last
- **THEN** `dkt sync add-x` does not take that commit for Task 1

### Requirement: Ticks from trailers
For each task a commit names, `dkt sync <slug>` SHALL tick every unticked step of that task in `openspec/changes/<slug>/plan.md`: each `- [ ]` checkbox line between the `### Task N:` heading and the next heading of level 1 to 3, outside fenced code and HTML comments. A heading inside fenced code or an HTML comment is not a heading. It SHALL leave the steps of every other task as they were.

#### Scenario: Task with commits ticked
- **WHEN** two commits carry `Change: add-x/3`, and Task 3 has four unticked steps
- **THEN** after `dkt sync add-x` all four steps of Task 3 are ticked

#### Scenario: Task without commits left alone
- **WHEN** no commit names Task 4 of `add-x`
- **THEN** after `dkt sync add-x` the steps of Task 4 are as they were

#### Scenario: Checkbox in a code block
- **WHEN** Task 1 has commits and a fenced code block in it holds a line `- [ ] example`
- **THEN** `dkt sync add-x` ticks Task 1's steps and leaves that line as it was

#### Scenario: Heading in a comment
- **WHEN** plan.md holds `### Task 1: hidden` inside an HTML comment, followed by a `- [ ]` line outside it, and a commit carries `Change: add-x/1`
- **THEN** `dkt sync add-x` reports `trailer-unknown-task` for Task 1 and leaves that line as it was

### Requirement: Only checkboxes change
`dkt sync` SHALL change nothing but `[ ]` to `[x]` in the steps it ticks. It SHALL never untick a step, never touch another file, never commit, and SHALL NOT require the OpenSpec CLI. It SHALL keep plan.md's line endings and mode. When a symbolic link is on the path to plan.md, it SHALL read and write nothing there, report `symlink-refused` naming the link, and exit with 1, with or without `--check`.

#### Scenario: Nothing else changes
- **WHEN** `dkt sync add-x` ticks steps in a plan.md with CRLF line endings
- **THEN** plan.md differs from before only in those checkbox characters, and `git status` shows no other modified file

#### Scenario: Linked plan refused
- **WHEN** `openspec/changes/add-x/plan.md` is a symbolic link and a user runs `dkt sync add-x`, with or without `--check`
- **THEN** nothing is written on either side of the link, the finding `symlink-refused` names the link, and dkt exits with 1

### Requirement: Trailer problems
`dkt sync` SHALL report `trailer-unknown-task` for each task number a trailer names that is not a `### Task` heading of plan.md, including a task moved to Cancelled Tasks, and `trailer-malformed` for a value `<slug>/…` whose task list is not one or more positive integers separated by commas. Each finding SHALL name the commit. It SHALL still tick the tasks it can, and exit with 1.

#### Scenario: Trailer for an unknown task
- **WHEN** a commit carries `Change: add-x/9`, and plan.md has no Task 9
- **THEN** `dkt sync add-x` reports `trailer-unknown-task` naming Task 9 and the commit, ticks the other tasks, and exits with 1

#### Scenario: Malformed task list
- **WHEN** a commit carries `Change: add-x/two`
- **THEN** `dkt sync add-x` reports `trailer-malformed` naming the commit and the value, and exits with 1

### Requirement: Check mode
`dkt sync <slug> --check` SHALL write nothing. It SHALL report `task-unticked` for each task that has commits and unticked steps, and `task-no-commit` for each task that no commit names and that has an unticked step; it SHALL exit with 1 when it reports anything, and with 0 otherwise.

#### Scenario: Check finds unticked work
- **WHEN** Task 2 has commits and unticked steps, and a user runs `dkt sync add-x --check`
- **THEN** plan.md is unchanged, `task-unticked` names Task 2, and dkt exits with 1

#### Scenario: Check finds a task without commits
- **WHEN** no commit names Task 5 and Task 5 has an unticked step
- **THEN** `dkt sync add-x --check` reports `task-no-commit` naming Task 5 and exits with 1

#### Scenario: Task ticked by hand
- **WHEN** no commit names Task 6 and every step of Task 6 was ticked by hand
- **THEN** `dkt sync add-x --check` reports nothing about Task 6

#### Scenario: Check on a synced plan
- **WHEN** every task has commits and `dkt sync add-x` has just run
- **THEN** `dkt sync add-x --check` exits with 0

### Requirement: Change and plan found
`dkt sync <slug>` SHALL read `openspec/changes/<slug>/plan.md` in the repository. A slug that is empty, holds `/` or `\`, or is `.` or `..`, names no change: it SHALL be a usage error (exit 2), and nothing is read. When the folder does not exist it SHALL report `change-missing`; when it has no plan.md, `plan-missing`; in both cases it SHALL write nothing and exit with 1. An unreadable plan.md SHALL be listed as `cli-output` requires.

#### Scenario: No such change
- **WHEN** `openspec/changes/add-y/` does not exist and a user runs `dkt sync add-y`
- **THEN** the finding `change-missing` names `add-y` and dkt exits with 1

#### Scenario: Path in place of a slug
- **WHEN** a user runs `dkt sync ../../docs`
- **THEN** dkt exits with 2 and no file is read or written

### Requirement: Other schema refused
`dkt sync <slug>` for a change whose schema, as `change-metadata` defines it, is not `docket` SHALL report `other-schema`, naming the change's `.openspec.yaml` and its schema, change nothing, and exit with 1, with or without `--check`.

#### Scenario: Sync refused for another schema
- **WHEN** `add-x` has `schema: spec-driven` and a user runs `dkt sync add-x`
- **THEN** the findings hold `other-schema` naming `openspec/changes/add-x/.openspec.yaml` and `spec-driven`, no file changes, and dkt exits with 1

### Requirement: Sync result
The `result` of `dkt sync` SHALL hold `slug`, `check`, and `tasks`: every `### Task` of plan.md in order, with its `number`, `name`, `commits` (the full SHAs that name it, oldest first), `steps` (its number of steps), `done` (its ticked steps after the run, or as the file is with `--check`) and `ticked` (the steps this run ticked, or would tick with `--check`). Without `--json`, the output SHALL list each task with its done and total steps and its number of commits.

#### Scenario: Result after ticking
- **WHEN** Task 1 has two steps, one ticked, and one commit names it, and a user runs `dkt sync add-x --json`
- **THEN** `result.tasks[0]` has `number` 1, one commit, `steps` 2, `done` 2 and `ticked` 1
