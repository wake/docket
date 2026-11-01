## MODIFIED Requirements

### Requirement: Issue sources checked
`dkt lint` SHALL report `issue-source` for each change it lints and each record the branch adds whose source names a path under `docket/issues/`, as `docket-metadata` defines, that is no file issue in the worktree, naming the source.

#### Scenario: Source names no issue
- **WHEN** a record the branch adds has `source: docket/issues/2026-11-02-x.md` and the worktree has no such file
- **THEN** `dkt lint` reports `issue-source` naming that source and exits with 1

#### Scenario: Source names an issue
- **WHEN** a record the branch adds has `source: docket/issues/2026-11-02-x.md` and the worktree holds that file
- **THEN** `dkt lint` reports no `issue-source`

#### Scenario: Source with a note names no issue
- **WHEN** a record the branch adds has `source: docket/issues/2026-11-02-x.md, from review of #40` and the worktree has no such file
- **THEN** `dkt lint` reports `issue-source` naming that source

### Requirement: Issue state
A file issue's handlers SHALL be the records and changes, active or archived, whose source names the issue's path as `docket-metadata` defines, each read from the default branch tip when the tip holds it and from its first source otherwise. Its state SHALL be `dropped` with `dropped: true`, `moved` with an `idea`, and otherwise `pending` with no handler, `in-progress` when the tip does not hold a handler, and `handled` when it holds every handler.

#### Scenario: Handler on a branch
- **WHEN** the default branch tip holds `docket/issues/2026-11-02-x.md` and an unmerged branch holds a record whose source is that path
- **THEN** the issue's state is `in-progress`

#### Scenario: Handler merged
- **WHEN** the default branch tip holds the issue and a record whose source is its path, and no other source holds a handler the tip does not
- **THEN** the issue's state is `handled`

#### Scenario: Merged change not yet archived
- **WHEN** the default branch tip holds the active change `add-x`, whose `docket.yaml` has the issue's path as its source
- **THEN** the issue's state is `handled`

#### Scenario: Merged record edited elsewhere
- **WHEN** the default branch tip holds a record whose source is `review of #40`, and worktree B changes that record's source to the issue's path
- **THEN** the record is not a handler of the issue

#### Scenario: Handler with a note
- **WHEN** the default branch tip holds the issue and a record whose source is its path followed by `, found while fixing y`
- **THEN** the issue's state is `handled`
