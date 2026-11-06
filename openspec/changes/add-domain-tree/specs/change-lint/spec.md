# Spec Delta

## MODIFIED Requirements

### Requirement: What is linted
`dkt lint` SHALL lint each change named on the command line, or, when none is named, every active change in the worktree's `openspec/changes/`, then the title of every main spec under `openspec/specs/`, then the domain tree as `domain-tree` defines. It SHALL also lint records and issues, as their specs define. It SHALL read the worktree's files as they are, uncommitted edits included, and SHALL report `change-missing` for a named change that does not exist. A name that is empty, holds `/` or `\`, or is `.` or `..` SHALL be a usage error.

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

#### Scenario: Lint reports the domain tree
- **WHEN** `docket/domains.md` has a line that is not an item, lists a capability twice, or lists a capability that does not exist, and a user runs `dkt lint`
- **THEN** the findings hold `domain-format`, `domain-duplicate` or `domain-unknown` as `domain-tree` defines, naming `docket/domains.md` and the line, and dkt exits with 1

#### Scenario: Lint with no domain tree
- **WHEN** there is no `docket/domains.md` and a user runs `dkt lint`
- **THEN** no finding is about the domain tree

#### Scenario: Linked domain tree
- **WHEN** `docket/domains.md` is a symbolic link and a user runs `dkt lint`
- **THEN** `unreadable` lists it and dkt exits with 3

#### Scenario: Named change skips the domain tree
- **WHEN** `docket/domains.md` has a line that is not an item and a user runs `dkt lint add-x`
- **THEN** no finding is about the domain tree
