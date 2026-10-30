# change-creation Specification

## Purpose
Defines how `dkt new` opens a change: a valid name that was never used before,
the `docket` schema, and a record of what kind of change it is, where the
request came from, and which commit it started from.

## Requirements

### Requirement: Slug rules
`dkt new` SHALL accept a slug only if it is lowercase kebab-case (letters a-z and digits, in groups joined by single hyphens), at most 200 characters long, does not start with a date of the form `YYYY-MM-DD`, and is not `archive`, the name of OpenSpec's archive folder.

#### Scenario: Date prefix rejected
- **WHEN** a user runs `dkt new 2026-10-24-add-x --kind feature --source x --title X`
- **THEN** the command exits with 1 and explains that archiving adds the date

#### Scenario: Uppercase rejected
- **WHEN** a user runs `dkt new Add-X --kind feature --source x --title X`
- **THEN** the command exits with 1

#### Scenario: Reserved name rejected
- **WHEN** a repository has archived changes and a user runs `dkt new archive --kind feature --source x --title X`
- **THEN** the command exits with 1 and `openspec/changes/archive/` is unchanged

### Requirement: Slug never reused
`dkt new` SHALL NOT create a change under a slug that any scanned source already uses, whether active or archived. When the slug is used, it SHALL create the change under the first of `<slug>-2`, `<slug>-3`, ... that no scanned source uses, and SHALL report that slug and every place the given slug is used. When any source is unreadable it SHALL create nothing and exit with 3, because it cannot show that the slug is unused. Of two runs that create the same slug in one worktree at the same time, one SHALL create the slug and the other SHALL take the next suffix without touching what the first created.

#### Scenario: Slug active in another worktree
- **WHEN** worktree B holds `add-x` and a user runs `dkt new add-x --kind feature --source x --title X` in worktree A
- **THEN** the command creates `openspec/changes/add-x-2/`, `result.slug` is `add-x-2`, the output names worktree B, and it exits with 0

#### Scenario: Slug archived
- **WHEN** the default branch tip holds `openspec/changes/archive/2025-01-02-add-x/` and a user runs `dkt new add-x --kind feature --source x --title X`
- **THEN** the command creates `openspec/changes/add-x-2/`, the output names the archived folder, and it exits with 0

#### Scenario: Unreadable worktree blocks creation
- **WHEN** a listed worktree cannot be read and a user runs `dkt new add-x --kind feature --source x --title X`
- **THEN** nothing is created and the command exits with 3

#### Scenario: Concurrent runs keep the first
- **WHEN** two runs of `dkt new add-x --kind feature --source x --title X` start at the same time in one worktree
- **THEN** one creates `openspec/changes/add-x/` and the other `openspec/changes/add-x-2/`, both exit with 0, and `add-x` keeps everything the first run wrote

### Requirement: Kind and source
`dkt new` SHALL require `--kind` with one of the values `feature`, `change-request` or `baseline`, and SHALL require a non-empty `--source` saying where the request came from. A fix that restores a known answer is a record, not a change.

#### Scenario: Missing source
- **WHEN** a user runs `dkt new add-x --kind feature`
- **THEN** the command exits with 2

#### Scenario: Unknown kind
- **WHEN** a user runs `dkt new add-x --kind fix --source x`
- **THEN** the command exits with 2 and lists the three kinds

### Requirement: Change created with metadata
`dkt new` SHALL create the change by running `openspec new change <slug> --schema docket`, then write the change's `docket.yaml` with the title, kind, source and base, leaving `.openspec.yaml` as OpenSpec wrote it. The base SHALL be the full SHA of the merge base of HEAD and the default branch tip. When the change folder's path passes through a symbolic link under `openspec/`, `dkt new` SHALL neither run OpenSpec nor write anything, and SHALL exit with 1 naming the link.

#### Scenario: Change created
- **WHEN** a user runs `dkt new add-x --kind feature --source "issue 12" --title "Booking waitlist"` on a branch made from the default branch
- **THEN** `openspec/changes/add-x/docket.yaml` holds `title: Booking waitlist`, `kind: feature`, `source: issue 12` and `base` equal to that merge base, and `.openspec.yaml` holds only what OpenSpec wrote

#### Scenario: Schema not installed
- **WHEN** `openspec/schemas/docket/` does not exist and a user runs `dkt new add-x --kind feature --source x --title X`
- **THEN** nothing is created, the command exits with 1, and it tells the user to run `dkt init`

#### Scenario: Symlinked changes folder refused
- **WHEN** `openspec/changes` is a symbolic link to a directory outside the repository and a user runs `dkt new add-x --kind feature --source x --title X`
- **THEN** OpenSpec is not run, nothing is written on either side of the link, and the command exits with 1 naming the link

### Requirement: Title
`dkt new` SHALL require `--title` with a title of the form `docket-metadata` defines, and SHALL create nothing when it is missing or not of that form.

#### Scenario: Missing title
- **WHEN** a user runs `dkt new add-x --kind feature --source x`
- **THEN** the command exits with 2 and nothing is created

#### Scenario: Blank title
- **WHEN** a user runs `dkt new add-x --kind feature --source x --title "   "`
- **THEN** the command exits with 2 and nothing is created
