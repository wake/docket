# Spec Delta

## MODIFIED Requirements

### Requirement: What is linted
`dkt lint` SHALL lint the changes and records named on the command line, or, when none is named, every active change in the current worktree's `openspec/changes/`, every record under `docket/records/` that the branch adds (committed since the merge base with the default branch, or not yet committed), and the titles of the main specs under `openspec/specs/`. A name that starts with a date names a record. It SHALL read the worktree's files as they are, uncommitted edits included, and SHALL NOT require the OpenSpec CLI.

#### Scenario: Lint every active change
- **WHEN** the worktree has the active changes `add-x` and `add-y` and a user runs `dkt lint`
- **THEN** both changes are linted and the archive folder is not

#### Scenario: Named change missing
- **WHEN** a user runs `dkt lint add-z` and the worktree has no change `add-z`
- **THEN** the command reports that `add-z` does not exist and exits with 1

#### Scenario: No OpenSpec installed
- **WHEN** no OpenSpec CLI is installed and a user runs `dkt lint`
- **THEN** the command lints normally

#### Scenario: Lint records the branch adds
- **WHEN** the default branch holds the record `2026-10-01-fix-a`, the branch adds the record `2026-10-30-fix-b`, and a user runs `dkt lint`
- **THEN** `2026-10-30-fix-b` is linted and `2026-10-01-fix-a` is not
