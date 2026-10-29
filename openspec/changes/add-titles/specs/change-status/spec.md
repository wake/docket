# Spec Delta

## ADDED Requirements

### Requirement: Title
Each change SHALL give its title, read as `docket-metadata` defines from the first source, or none when it has no title.

#### Scenario: Title shown
- **WHEN** `add-x`'s `docket.yaml` holds `title: Booking waitlist` and a user runs `dkt status`
- **THEN** the line for `add-x` shows "Booking waitlist", and with `--json` its `title` is `"Booking waitlist"`

#### Scenario: Change without title
- **WHEN** `add-x` has no `docket.yaml` and a user runs `dkt status --json`
- **THEN** `add-x` is listed with `title` null and the command exits with 0

## MODIFIED Requirements

### Requirement: Activity
Each change SHALL give its last activity: the newest commit, reachable from the first source's commit and not from the change's base, that names the change in a `Change` trailer or changes a file in its folder, with that commit's time and the whole days from it to the start of the run. When the first source is a worktree, the change SHALL also give how many files in its folder have uncommitted changes.

#### Scenario: Idle change
- **WHEN** the newest commit for `add-x` was made ten and a half days before the run
- **THEN** its last activity names that commit and gives 10 idle days

#### Scenario: Uncommitted edits
- **WHEN** `add-x` is read from a worktree where its `plan.md` and `design.md` have uncommitted edits
- **THEN** it gives 2 uncommitted files

#### Scenario: No commit yet
- **WHEN** `add-x` exists only as untracked files in a worktree
- **THEN** it has no last activity and gives its uncommitted files
