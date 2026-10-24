# change-creation Specification

## Purpose
Defines how `dkt new` opens a change: a valid name that was never used before,
the `docket` schema, and a record of what kind of change it is, where the
request came from, and which commit it started from.

## Requirements

### Requirement: Slug rules
`dkt new` SHALL accept a slug only if it is lowercase kebab-case (letters a-z and digits, in groups joined by single hyphens), at most 200 characters long, does not start with a date of the form `YYYY-MM-DD`, and is not `archive`, the name of OpenSpec's archive folder.

#### Scenario: Date prefix rejected
- **WHEN** a user runs `dkt new 2026-10-24-add-x --kind feature --source x`
- **THEN** the command exits with 1 and explains that archiving adds the date

#### Scenario: Uppercase rejected
- **WHEN** a user runs `dkt new Add-X --kind feature --source x`
- **THEN** the command exits with 1

#### Scenario: Reserved name rejected
- **WHEN** a repository has archived changes and a user runs `dkt new archive --kind feature --source x`
- **THEN** the command exits with 1 and `openspec/changes/archive/` is unchanged

### Requirement: Slug never reused
`dkt new` SHALL refuse a slug that any scanned source already uses, whether active or archived, and SHALL name every place it is used. When any source is unreadable it SHALL create nothing and exit with 3, because it cannot show that the slug is unused. Of two runs that create the same slug in one worktree at the same time, one SHALL create the change and the other SHALL refuse the slug without touching what the first created.

#### Scenario: Slug active in another worktree
- **WHEN** worktree B holds `add-x` and a user runs `dkt new add-x --kind feature --source x` in worktree A
- **THEN** the command exits with 1 and names worktree B

#### Scenario: Slug archived
- **WHEN** the default branch tip holds `openspec/changes/archive/2025-01-02-add-x/` and a user runs `dkt new add-x --kind feature --source x`
- **THEN** the command exits with 1 and names the archived folder

#### Scenario: Unreadable worktree blocks creation
- **WHEN** a listed worktree cannot be read and a user runs `dkt new add-x --kind feature --source x`
- **THEN** nothing is created and the command exits with 3

#### Scenario: Concurrent runs keep the first
- **WHEN** two runs of `dkt new add-x --kind feature --source x` start at the same time in one worktree
- **THEN** exactly one creates `openspec/changes/add-x/`, the other exits with 1 and names it, and the created folder keeps everything the first run wrote

### Requirement: Kind and source
`dkt new` SHALL require `--kind` with one of the values `feature`, `change-request`, `fix` or `baseline`, and SHALL require a non-empty `--source` saying where the request came from.

#### Scenario: Missing source
- **WHEN** a user runs `dkt new add-x --kind feature`
- **THEN** the command exits with 2

#### Scenario: Unknown kind
- **WHEN** a user runs `dkt new add-x --kind chore --source x`
- **THEN** the command exits with 2 and lists the four kinds

### Requirement: Change created with metadata
`dkt new` SHALL create the change by running `openspec new change <slug> --schema docket`, then add `x-kind`, `x-source` and `x-base` to the change's `.openspec.yaml`, keeping every key OpenSpec wrote. `x-base` SHALL be the full SHA of the merge base of HEAD and the default branch tip. When the change folder's path passes through a symbolic link under `openspec/`, `dkt new` SHALL neither run OpenSpec nor write anything, and SHALL exit with 1 naming the link.

#### Scenario: Change created
- **WHEN** a user runs `dkt new add-x --kind feature --source "issue 12"` on a branch made from the default branch
- **THEN** `openspec/changes/add-x/.openspec.yaml` holds `schema: docket`, the `created` date OpenSpec wrote, `x-kind: feature`, `x-source: issue 12`, and `x-base` equal to that merge base

#### Scenario: Schema not installed
- **WHEN** `openspec/schemas/docket/` does not exist and a user runs `dkt new add-x --kind feature --source x`
- **THEN** nothing is created, the command exits with 1, and it tells the user to run `dkt init`

#### Scenario: Symlinked changes folder refused
- **WHEN** `openspec/changes` is a symbolic link to a directory outside the repository and a user runs `dkt new add-x --kind feature --source x`
- **THEN** OpenSpec is not run, nothing is written on either side of the link, and the command exits with 1 naming the link
