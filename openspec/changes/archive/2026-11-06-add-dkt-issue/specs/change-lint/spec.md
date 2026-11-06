# Spec Delta

## MODIFIED Requirements

### Requirement: What is linted
`dkt lint` SHALL lint each change named on the command line, or, when none is named, every active change in the worktree's `openspec/changes/` then the title of every main spec under `openspec/specs/`. It SHALL also lint records and issues, as their specs define. It SHALL read the worktree's files as they are, uncommitted edits included, and SHALL report `change-missing` for a named change that does not exist. A name that is empty, holds `/` or `\`, or is `.` or `..` SHALL be a usage error.

#### Scenario: Lint every active change
- **WHEN** the worktree has the active changes `add-x` and `add-y` and a user runs `dkt lint`
- **THEN** both changes are linted, the archive folder is not, and the main specs' titles are checked

#### Scenario: Named change missing
- **WHEN** a user runs `dkt lint add-z` and the worktree has no change `add-z`
- **THEN** the finding `change-missing` names `add-z` and dkt exits with 1

#### Scenario: Named change skips main specs
- **WHEN** a main spec has no title and a user runs `dkt lint add-x`
- **THEN** no finding is about that main spec

#### Scenario: Named change skips records
- **WHEN** the branch adds a record that breaks a rule and a user runs `dkt lint add-x`
- **THEN** no finding is about that record

### Requirement: Lint result
The `result` of `dkt lint` SHALL hold `changes`, the slugs linted with their schema (`{"slug", "schema"}`), `records` and `issues`, the record and issue slugs linted, `specs`, the number of main spec titles checked (0 when a name was given), and `default_branch`, the default branch the routing rule used, or null when it was skipped. Every finding SHALL name its change in `change` when it is about one, and its file and line when it has them. Without `--json`, the output SHALL list the findings.

#### Scenario: Clean change
- **WHEN** `add-x` breaks no rule, the default branch is `main`, and a user runs `dkt lint add-x --json`
- **THEN** `result.changes` is `[{"slug": "add-x", "schema": "docket"}]`, `result.records` and `result.issues` are `[]`, `result.specs` is 0, `result.default_branch` is `main`, there is no finding, and dkt exits with 0

#### Scenario: Records in the result
- **WHEN** the branch adds the clean record `2026-11-06-fix-x` and a user runs `dkt lint --json`
- **THEN** `result.records` is `["2026-11-06-fix-x"]`

#### Scenario: Issues in the result
- **WHEN** the branch adds the clean issue `2026-11-06-lint-timeout` and a user runs `dkt lint --json`
- **THEN** `result.issues` is `["2026-11-06-lint-timeout"]`
