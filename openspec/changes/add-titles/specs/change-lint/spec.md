# Spec Delta

## ADDED Requirements

### Requirement: Titles
lint SHALL report `title-missing` for a linted change without a title; for a linted change's delta of a capability that has no main spec in the worktree, when no title sits beside that delta; and, when no change is named, for every main spec without a title. A `docket.yaml` that is not a YAML mapping SHALL be reported as `metadata-invalid`.

#### Scenario: Change without title
- **WHEN** `add-x` has no `docket.yaml` and a user runs `dkt lint add-x`
- **THEN** lint reports a `title-missing` finding naming `openspec/changes/add-x/docket.yaml`

#### Scenario: New capability without title
- **WHEN** `add-x` has a delta for `booking`, the worktree has no `openspec/specs/booking/spec.md`, and `openspec/changes/add-x/specs/booking/docket.yaml` does not exist
- **THEN** lint reports a `title-missing` finding for `booking`

#### Scenario: Existing capability needs no delta title
- **WHEN** `add-x` has a delta for `booking`, the main spec `booking` exists, and no `docket.yaml` sits beside the delta
- **THEN** lint reports no `title-missing` finding for `booking`

#### Scenario: Main spec without title
- **WHEN** `openspec/specs/booking/spec.md` exists without a `docket.yaml` beside it and a user runs `dkt lint`
- **THEN** lint reports a `title-missing` finding naming `openspec/specs/booking/docket.yaml`

#### Scenario: Named change skips main specs
- **WHEN** a main spec has no title and a user runs `dkt lint add-x`, where `add-x` has a title
- **THEN** lint reports no finding about that main spec

#### Scenario: Broken docket.yaml
- **WHEN** `openspec/changes/add-x/docket.yaml` holds a YAML list
- **THEN** lint reports a `metadata-invalid` finding on that file

## MODIFIED Requirements

### Requirement: What is linted
`dkt lint` SHALL lint the changes named on the command line, or, when none is named, every active change in the current worktree's `openspec/changes/` and the titles of the main specs under `openspec/specs/`. It SHALL read the worktree's files as they are, uncommitted edits included, and SHALL NOT require the OpenSpec CLI.

#### Scenario: Lint every active change
- **WHEN** the worktree has the active changes `add-x` and `add-y` and a user runs `dkt lint`
- **THEN** both changes are linted and the archive folder is not

#### Scenario: Named change missing
- **WHEN** a user runs `dkt lint add-z` and the worktree has no change `add-z`
- **THEN** the command reports that `add-z` does not exist and exits with 1

#### Scenario: No OpenSpec installed
- **WHEN** no OpenSpec CLI is installed and a user runs `dkt lint`
- **THEN** the command lints normally
