# change-creation Specification

## Purpose
Defines how `dkt new` opens a change: a slug never used before, the `docket` schema, and the change's title, kind and source written where Docket reads them.

## Requirements

### Requirement: Metadata flags
`dkt new` SHALL take one slug and require `--kind`, `--source` and `--title`, each of the form `change-metadata` defines. When the slug or a flag is missing, or a value does not have that form, it SHALL create nothing, run nothing and exit with 2; for an unknown kind it SHALL list the three kinds.

#### Scenario: Missing source
- **WHEN** a user runs `dkt new add-x --kind feature --title X`
- **THEN** nothing is created, OpenSpec is not run, and dkt exits with 2

#### Scenario: Unknown kind
- **WHEN** a user runs `dkt new add-x --kind fix --source x --title X`
- **THEN** dkt exits with 2 and the message lists `feature`, `change-request` and `baseline`

#### Scenario: Title on two lines
- **WHEN** a user runs `dkt new add-x --kind feature --source x --title "Line one` followed by a line break and `line two"`
- **THEN** nothing is created and dkt exits with 2

### Requirement: Slug rules
`dkt new` SHALL refuse a slug that starts with a date of the form `YYYY-MM-DD`, since archiving adds the date, or that is `archive`, the name of OpenSpec's archive folder: it SHALL report `slug-invalid`, create nothing, run nothing and exit with 1. The rest of the slug's form is OpenSpec's to check.

#### Scenario: Date prefix refused
- **WHEN** a user runs `dkt new 2026-10-24-add-x --kind feature --source x --title X`
- **THEN** the finding `slug-invalid` says that archiving adds the date, nothing is created, and dkt exits with 1

#### Scenario: Reserved name refused
- **WHEN** the project has archived changes and a user runs `dkt new archive --kind feature --source x --title X`
- **THEN** the finding `slug-invalid` names `archive`, `openspec/changes/archive/` is unchanged, and dkt exits with 1

#### Scenario: Form left to OpenSpec
- **WHEN** a user runs `dkt new Add-X --kind feature --source x --title X`
- **THEN** OpenSpec refuses the name, the finding `openspec-failed` holds its message, no folder `openspec/changes/Add-X/` is left, and dkt exits with 1

### Requirement: Schema required
Before it runs OpenSpec, `dkt new` SHALL check that `openspec/schemas/docket/schema.yaml` exists in the project. When it does not, it SHALL report `schema-missing`, telling the user to run `dkt init`, create nothing and exit with 1. It SHALL NOT compare the schema with the shipped one.

#### Scenario: Schema not installed
- **WHEN** the project has no `openspec/schemas/docket/schema.yaml` and a user runs `dkt new add-x --kind feature --source x --title X`
- **THEN** OpenSpec is not run, nothing is created, the finding `schema-missing` names `dkt init`, and dkt exits with 1

#### Scenario: Edited schema accepted
- **WHEN** the project's `schema.yaml` differs from the shipped one and a user runs `dkt new add-x --kind feature --source x --title X`
- **THEN** the change is created and dkt exits with 0

### Requirement: Slug never reused
`dkt new` SHALL NOT create a change under a slug that the workspace scan finds in use, active or archived. It SHALL create it under the first of `<slug>-2`, `<slug>-3`, … that the scan does not find, and SHALL report every place the requested slug is used.

#### Scenario: Slug active in another worktree
- **WHEN** worktree B holds an uncommitted `openspec/changes/add-x/` and a user runs `dkt new add-x --kind feature --source x --title X` in worktree A
- **THEN** dkt creates `openspec/changes/add-x-2/`, `result.slug` is `add-x-2`, `result.requested` is `add-x`, `result.used` names worktree B and `openspec/changes/add-x`, and dkt exits with 0

#### Scenario: Slug archived on a branch
- **WHEN** a local branch holds `openspec/changes/archive/2026-01-02-add-x/` and a user runs `dkt new add-x --kind feature --source x --title X`
- **THEN** dkt creates `openspec/changes/add-x-2/` and `result.used` names that branch and the archived folder

#### Scenario: Suffix also used
- **WHEN** `add-x` and `add-x-2` are both in use and a user runs `dkt new add-x --kind feature --source x --title X`
- **THEN** dkt creates `openspec/changes/add-x-3/`

### Requirement: Unreadable source blocks creation
When the workspace scan reports any source as unreadable, `dkt new` SHALL create nothing, SHALL NOT run `openspec new change`, SHALL list the source as `cli-output` requires, and SHALL exit with 3, because it cannot show that the slug is unused.

#### Scenario: Worktree directory missing
- **WHEN** a worktree that git lists has no directory on disk and a user runs `dkt new add-x --kind feature --source x --title X`
- **THEN** nothing is created, `unreadable` lists that worktree, and dkt exits with 3

### Requirement: Change created
`dkt new` SHALL create the change by running `openspec new change <slug> --schema docket`, then write `openspec/changes/<slug>/proposal.md` holding only the front matter `change-metadata` defines. It SHALL leave `.openspec.yaml` as OpenSpec wrote it.

#### Scenario: Change created
- **WHEN** a user runs `dkt new add-x --kind feature --source "issue 12" --title "Booking waitlist"`
- **THEN** `openspec/changes/add-x/.openspec.yaml` names the schema `docket`, `proposal.md` holds only front matter with that title, kind and source, and dkt exits with 0

### Requirement: New result
The `result` of `dkt new` SHALL hold `slug` (the slug created), `requested` (the slug given), `path` (the change folder relative to the repository root), `title`, `kind`, `source`, and `used`: a list, empty when the slug was free, of each folder where the requested slug is in use, with its `path` and the `sources` that hold it. Without `--json`, the output SHALL name the folder created and, when a suffix was taken, where the requested slug is used.

#### Scenario: Free slug
- **WHEN** no source uses `add-x` and a user runs `dkt new add-x --json --kind feature --source x --title X`
- **THEN** `result.slug` and `result.requested` are `add-x`, `result.path` is `openspec/changes/add-x`, and `result.used` is an empty list

### Requirement: OpenSpec failure cleaned up
When `openspec new change` fails, `dkt new` SHALL report `openspec-failed` as `openspec-runtime` says and write no `proposal.md`. It SHALL remove `openspec/changes/<slug>/` when that folder holds nothing but `.openspec.yaml`; otherwise it SHALL leave the folder as it is and say so in the finding.

#### Scenario: OpenSpec fails
- **WHEN** `openspec new change` exits with a non-zero status after creating `openspec/changes/add-x/`
- **THEN** `openspec/changes/add-x/` does not exist, the finding `openspec-failed` names the command, and dkt exits with 1

#### Scenario: Other files kept
- **WHEN** `openspec new change` fails after `openspec/changes/add-x/` gained a file other than `.openspec.yaml`
- **THEN** the folder and that file remain, the finding says the folder was left in place, and dkt exits with 1

### Requirement: Symbolic links refused
When a symbolic link is on the path from the repository root to `openspec/changes/<slug>/proposal.md`, `dkt new` SHALL run nothing, write nothing on either side of the link, report `symlink-refused` naming the link, and exit with 1.

#### Scenario: Changes folder is a link
- **WHEN** `openspec/changes` is a symbolic link to a directory outside the repository and a user runs `dkt new add-x --kind feature --source x --title X`
- **THEN** OpenSpec is not run, nothing is written on either side of the link, the finding `symlink-refused` names `openspec/changes`, and dkt exits with 1

### Requirement: Concurrent runs
Of two runs of `dkt new` with the same slug in one worktree at the same time, one SHALL create the slug and the other SHALL take the next suffix, without touching what the first created.

#### Scenario: Two runs at once
- **WHEN** two runs of `dkt new add-x --kind feature --source x --title X` start at the same time in one worktree
- **THEN** one creates `openspec/changes/add-x/` and the other `openspec/changes/add-x-2/`, both exit with 0, and `add-x` holds what the first run wrote
