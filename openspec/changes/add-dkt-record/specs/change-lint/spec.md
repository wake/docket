# Spec Delta

## MODIFIED Requirements

### Requirement: What is linted
`dkt lint` SHALL lint each change named on the command line, or, when none is named, every active change in the worktree's `openspec/changes/` and then the title of every main spec under `openspec/specs/`. It SHALL also lint records, as `records` defines. It SHALL read the worktree's files as they are, uncommitted edits included, and SHALL report `change-missing` for a named change that does not exist. A name that is empty, holds `/` or `\`, or is `.` or `..` SHALL be a usage error.

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

### Requirement: Front matter checked
For a change of the `docket` schema, `dkt lint` SHALL report `title-missing` when `proposal.md` has no front matter or no title that is one line of text, `title-too-long` giving the length and the limit for a title over the limit `change-metadata` defines, and `metadata-invalid` for a front matter it cannot read and for a `kind` or `source` that is missing or not of the form `change-metadata` defines.

#### Scenario: Proposal without front matter
- **WHEN** `openspec/changes/add-x/proposal.md` starts with `# Proposal`
- **THEN** `dkt lint add-x` reports `title-missing` naming that file

#### Scenario: Unknown kind
- **WHEN** the front matter of `add-x`'s proposal has `kind: fix`
- **THEN** `dkt lint add-x` reports `metadata-invalid` naming `kind` on that line

#### Scenario: Change title too long
- **WHEN** the front matter of `add-x`'s proposal has a title of 121 bytes
- **THEN** `dkt lint add-x` reports `title-too-long` on that line, giving 121 and 120, and no `title-missing`

### Requirement: Lint result
The `result` of `dkt lint` SHALL hold `changes`, the slugs linted with their schema (`{"slug", "schema"}`), `records`, the slugs of the records linted, `specs`, the number of main specs whose titles were checked (0 when a change was named), and `default_branch`, the name of the default branch the routing rule used or null when it was skipped. Every finding SHALL name its change in `change` when it is about one, and its file and line when it has them. Without `--json`, the output SHALL list the findings.

#### Scenario: Clean change
- **WHEN** `add-x` breaks no rule, the default branch is `main`, and a user runs `dkt lint add-x --json`
- **THEN** `result.changes` is `[{"slug": "add-x", "schema": "docket"}]`, `result.records` is `[]`, `result.specs` is 0, `result.default_branch` is `main`, there is no finding, and dkt exits with 0

#### Scenario: Records in the result
- **WHEN** the branch adds the clean record `2026-11-06-fix-x` and a user runs `dkt lint --json`
- **THEN** `result.records` is `["2026-11-06-fix-x"]`
