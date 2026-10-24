# change-status Specification

## Purpose
Defines how Docket reports the state of every active change, wherever it
lives, so that people and tools such as a web workbench can see each
change's phase, task progress and recent activity without opening each
worktree.

## Requirements

### Requirement: Every active change listed
`dkt status` SHALL list, sorted by slug and once each, every active change the scan defined by `workspace-scan` finds, with every source it was found in and that source's commit. It SHALL read each change's documents and commits from the source the scan lists first. It SHALL NOT list archived changes, nor a slug that has an archived use in any source.

#### Scenario: Changes in worktrees and branches
- **WHEN** worktree A holds `add-x`, `add-y` exists in worktree B and on `refs/remotes/origin/add-y`, and the command runs in worktree A
- **THEN** it lists `add-x` and `add-y` in that order, and `add-y` with both sources and their commits

#### Scenario: Archived change not listed
- **WHEN** `add-x` is archived on the default branch tip and an unmerged branch still holds `openspec/changes/add-x/`
- **THEN** `add-x` is not listed

#### Scenario: No active changes
- **WHEN** no source holds an active change
- **THEN** the command lists no change and exits with 0

### Requirement: Phase
Each change SHALL have one phase: `design` when it has no `plan.md` or its plan has no task; `ready` when no task has started; `done` when every task that is not cancelled is done; `implementing` otherwise. A task is done when it has steps and all of them are ticked, and started when a step is ticked or a commit for the change names it in a `Task` trailer.

#### Scenario: Plan not written
- **WHEN** `add-x` has `proposal.md` and a delta spec but no `plan.md`
- **THEN** its phase is `design`

#### Scenario: Nothing started
- **WHEN** `add-x` has a plan with three tasks, no step ticked and no commit naming a task
- **THEN** its phase is `ready`

#### Scenario: Commit names a task
- **WHEN** no step of `add-x`'s plan is ticked and a commit for `add-x` carries `Task: 1`
- **THEN** its phase is `implementing` and Task 1 is `started`

#### Scenario: Every task done
- **WHEN** every step of every task of `add-x`'s plan is ticked
- **THEN** its phase is `done`

#### Scenario: Cancelled task left out
- **WHEN** `add-x`'s plan moves Task 3 to Cancelled Tasks and every step of Tasks 1 and 2 is ticked
- **THEN** its phase is `done`, Task 3 is `cancelled`, and the task progress counts two of two tasks

### Requirement: Task progress
Each change SHALL list the tasks of its `plan.md` in plan order, then its cancelled tasks, each with its number, name, state (`pending`, `started`, `done` or `cancelled`), its steps and ticked steps, and the commits that name it, taken as `plan-sync` takes them but from the first source's commit instead of HEAD. A task with such commits and an unticked step SHALL be marked unsynced. The change SHALL also give tasks done and total, and steps ticked and total, without cancelled tasks.

#### Scenario: Commits not yet synced
- **WHEN** a commit for `add-x` carries `Task: 2` and Task 2 has unticked steps
- **THEN** Task 2 lists that commit and is marked unsynced, and `plan.md` is unchanged

### Requirement: Activity
Each change SHALL give its last activity: the newest commit, reachable from the first source's commit and not from its `x-base`, that names the change in a `Change` trailer or changes a file in its folder, with that commit's time and the whole days from it to the start of the run. When the first source is a worktree, the change SHALL also give how many files in its folder have uncommitted changes.

#### Scenario: Idle change
- **WHEN** the newest commit for `add-x` was made ten and a half days before the run
- **THEN** its last activity names that commit and gives 10 idle days

#### Scenario: Uncommitted edits
- **WHEN** `add-x` is read from a worktree where its `plan.md` and `design.md` have uncommitted edits
- **THEN** it gives 2 uncommitted files

#### Scenario: No commit yet
- **WHEN** `add-x` exists only as untracked files in a worktree
- **THEN** it has no last activity and gives its uncommitted files

### Requirement: Changes in focus
With slugs, `dkt status` SHALL list only those changes, in slug order. Each slug SHALL be an active change; a slug that is not SHALL be reported, and the others still listed.

#### Scenario: One change in focus
- **WHEN** `add-x` and `add-y` are active and a user runs `dkt status add-y`
- **THEN** only `add-y` is listed and the command exits with 0

#### Scenario: Slug not active
- **WHEN** a user runs `dkt status add-q` and no active change is named `add-q`
- **THEN** the command reports that `add-q` is not active and exits with 1

### Requirement: Read-only report
`dkt status` SHALL write nothing, take no lock, never fetch, and SHALL NOT require the OpenSpec CLI. It SHALL exit with 0 when it reports no finding, except as `cli-output` requires for unreadable sources.

#### Scenario: Unreadable worktree
- **WHEN** one listed worktree cannot be read and another worktree holds `add-x`
- **THEN** `add-x` is listed, the worktree is listed as unreadable, and the command exits with 3

#### Scenario: Writes nothing
- **WHEN** a user runs `dkt status` in a repository with active changes
- **THEN** no file in any worktree changes and no ref moves
