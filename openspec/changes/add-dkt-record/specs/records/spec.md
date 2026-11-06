# Spec Delta

## Purpose
Defines records: one file per piece of work on the light review path, saying what was done, why, how it was checked and what its review found; how `dkt record` writes one, how `dkt lint` checks it and how `dkt sync` ticks its tasks.

## ADDED Requirements

### Requirement: Record file
A record SHALL be `docket/records/<slug>.md`: front matter with `title` and `source` of the form `change-metadata` defines and `kind` one of `fix`, `refactor`, `test`, `prompt`, `checks`, then the level-two sections `## What`, `## Why`, `## Verification` and `## Review`, in that order, and optionally `## Tasks` after them.

#### Scenario: Record written
- **WHEN** a user runs `dkt record fix-lint-timeout --kind fix --title "Lint timeout" --source "review of #40"` on 2026-10-30
- **THEN** `docket/records/2026-10-30-fix-lint-timeout.md` holds that title, kind and source in its front matter and the shipped record template after it, `result.slug` is `2026-10-30-fix-lint-timeout`, `result.path` is that file, and dkt exits with 0

### Requirement: Record flags
`dkt record` SHALL take one name and require `--kind`, one of the five kinds, and `--source` and `--title`, each of the form `change-metadata` defines. When the name or a flag is missing, or a value does not have that form, it SHALL write nothing and exit with 2; for an unknown kind it SHALL list the five kinds.

#### Scenario: Unknown record kind
- **WHEN** a user runs `dkt record fix-x --kind chore --title X --source x`
- **THEN** dkt exits with 2, the message lists the five kinds, and nothing is written

#### Scenario: Missing record source
- **WHEN** a user runs `dkt record fix-x --kind fix --title X`
- **THEN** dkt exits with 2 and nothing is written

#### Scenario: Record title over the limit
- **WHEN** a user runs `dkt record fix-x --kind fix --source x` with a `--title` of 121 bytes
- **THEN** dkt exits with 2, the message gives 121 and 120, and nothing is written

### Requirement: Record names
A record name SHALL be lowercase letters and digits in runs separated by single hyphens, at most 200 characters, and SHALL NOT start with a date of the form `YYYY-MM-DD`; any other name SHALL report `slug-invalid`, write nothing and exit with 1. The record's slug SHALL be the local date as `YYYY-MM-DD`, a hyphen and the name.

#### Scenario: Name with a date
- **WHEN** a user runs `dkt record 2026-10-30-fix-x --kind fix --title X --source x`
- **THEN** the finding `slug-invalid` says that `dkt record` adds the date, nothing is written, and dkt exits with 1

#### Scenario: Name that is not kebab case
- **WHEN** a user runs `dkt record Fix_X --kind fix --title X --source x`
- **THEN** the finding `slug-invalid` is reported, nothing is written, and dkt exits with 1

### Requirement: Record slug never reused
`dkt record` SHALL NOT write a record whose slug any scanned source uses for a record. When the slug is used, it SHALL write under the first of `<slug>-2`, `<slug>-3`, ... that no scanned source uses and that does not exist in the worktree, with `result.slug` the slug written, `result.requested` the slug without a suffix, and `result.used` each place the requested slug is used with its sources. When any source is unreadable it SHALL write nothing and exit with 3.

#### Scenario: Same name same day
- **WHEN** a branch holds `docket/records/2026-10-30-fix-x.md` and a user runs `dkt record fix-x --kind fix --title X --source x` on 2026-10-30 in a worktree without it
- **THEN** dkt writes `docket/records/2026-10-30-fix-x-2.md`, `result.slug` is `2026-10-30-fix-x-2`, `result.requested` is `2026-10-30-fix-x`, `result.used` names that branch, and dkt exits with 0

#### Scenario: Unreadable source blocks the record
- **WHEN** a listed worktree cannot be read and a user runs `dkt record fix-x --kind fix --title X --source x`
- **THEN** nothing is written, `unreadable` lists that worktree, and dkt exits with 3

#### Scenario: Concurrent records
- **WHEN** two runs of `dkt record fix-x --kind fix --title X --source x` start at the same time in one worktree on 2026-10-30
- **THEN** one writes `2026-10-30-fix-x.md`, the other writes `2026-10-30-fix-x-2.md`, and both exit with 0

### Requirement: Records folder not written through a link
`dkt record` SHALL NOT write through a symbolic link: when `docket`, `docket/records` or the record's path is one, it SHALL write nothing, report `symlink-refused` naming the link and exit with 1.

#### Scenario: Linked records folder refused
- **WHEN** `docket/records` is a symbolic link to a directory outside the repository and a user runs `dkt record fix-x --kind fix --title X --source x`
- **THEN** nothing is written on either side of the link, the finding `symlink-refused` names `docket/records`, and dkt exits with 1

### Requirement: Failed record write leaves nothing
When `dkt record` fails to write the record after it began, it SHALL remove the partial file and any folder the run created, and SHALL report the error; when it cannot remove one of them, the error SHALL name what is left.

#### Scenario: Failed write leaves nothing
- **WHEN** `docket/records` does not exist and writing the new record fails
- **THEN** neither the record file nor `docket/records` exists afterwards, and the command reports the error

### Requirement: Records linted
`dkt lint` with no name SHALL lint every record the branch adds: each regular `.md` file directly under `docket/records/` that is in the worktree but not at the merge base of HEAD and the default branch's tip, committed, staged or untracked and not ignored. With no default branch or no merge base it SHALL lint no record this way. A named record is linted whatever its state.

#### Scenario: Record the branch adds
- **WHEN** the current branch adds `docket/records/2026-11-06-fix-x.md`, not yet committed, with an unknown kind, and a user runs `dkt lint`
- **THEN** the finding `record-kind` names that file and dkt exits with 1

#### Scenario: Merged record left alone
- **WHEN** a record that breaks a rule exists at the merge base and a user runs `dkt lint`
- **THEN** no finding is about that record

#### Scenario: Records without a default branch
- **WHEN** no default branch can be found, the worktree holds a record that breaks a rule, and a user runs `dkt lint --json`
- **THEN** no finding is about that record and `result.records` is empty

### Requirement: Records named
A name given to `dkt lint` that starts with a date of the form `YYYY-MM-DD` SHALL name a record, `docket/records/<name>.md`. When no regular file is there, `dkt lint` SHALL report `record-missing` naming it. Naming only records SHALL lint no change and no main spec title.

#### Scenario: Named record linted
- **WHEN** the default branch holds `docket/records/2026-10-30-fix-a.md` with an unknown kind and a user runs `dkt lint 2026-10-30-fix-a`
- **THEN** the finding `record-kind` names that file, no change is linted, and dkt exits with 1

#### Scenario: Named record missing
- **WHEN** no such file exists and a user runs `dkt lint 2026-11-06-fix-q`
- **THEN** the finding `record-missing` names `2026-11-06-fix-q` and dkt exits with 1

### Requirement: Record front matter checked
`dkt lint` SHALL report `record-front-matter` for a record whose front matter is missing or cannot be read, or whose `title` or `source` is missing or not of the form `change-metadata` defines, `record-kind` for a `kind` that is missing or not one of the five, and `title-too-long` for a title over the limit.

#### Scenario: Record kind not known
- **WHEN** a record the branch adds has `kind: chore`
- **THEN** `dkt lint` reports `record-kind` on that line and exits with 1

#### Scenario: Record without front matter
- **WHEN** a record the branch adds starts with `## What`
- **THEN** `dkt lint` reports `record-front-matter` for that file

#### Scenario: Record title too long
- **WHEN** a record the branch adds has a title of 121 bytes
- **THEN** `dkt lint` reports `title-too-long` giving 121 and 120, and no `record-front-matter`

### Requirement: Record sections checked
`dkt lint` SHALL report `record-section` for a record that lacks one of the four sections, has them out of order, has another level-two heading, has text outside its sections, or has one of the four with no text beyond whitespace and HTML comments. It SHALL report `placeholder` for text left from the shipped record template.

#### Scenario: Record section empty
- **WHEN** a record the branch adds has `## Review` with nothing under it
- **THEN** `dkt lint` reports `record-section` naming `Review` and exits with 1

#### Scenario: Record sections out of order
- **WHEN** a record the branch adds has `## Why` before `## What`
- **THEN** `dkt lint` reports `record-section` saying the order is wrong

#### Scenario: Other heading in a record
- **WHEN** a record the branch adds has a `## Notes` section
- **THEN** `dkt lint` reports `record-section` naming `Notes`

#### Scenario: Text outside the sections
- **WHEN** a record the branch adds has a paragraph between its front matter and `## What`
- **THEN** `dkt lint` reports `record-section` on that line

#### Scenario: Template text left in a record
- **WHEN** a record still holds the template's comment under `## Why`
- **THEN** `dkt lint` reports `placeholder` on that line

### Requirement: Record length
The sections `## What`, `## Why`, `## Verification` and `## Review` of a record SHALL together hold at most 1,440 bytes, measured as `change-metadata` defines, outside HTML comments and without their headings. `## Tasks` SHALL NOT count. `dkt lint` SHALL report `record-too-long` giving the length and the limit.

#### Scenario: Record over the limit
- **WHEN** the four sections of a record the branch adds hold 1,441 bytes
- **THEN** `dkt lint` reports `record-too-long` giving 1441 and 1440, and exits with 1

#### Scenario: Chinese text at the limit
- **WHEN** the four sections of a record hold 480 Chinese characters and no other text
- **THEN** `dkt lint` reports no `record-too-long`

#### Scenario: Comments and tasks not counted
- **WHEN** a record's four sections hold 100 bytes of text and a long HTML comment, and its `## Tasks` holds 2,000 bytes
- **THEN** `dkt lint` reports no `record-too-long`

### Requirement: Record tasks
The tasks of a record SHALL be the `### Task N: <name>` sections under its `## Tasks`, in the plan's format, and `dkt lint` SHALL check them by the plan's task heading rules, without requiring a `**Scenarios:**` or `**Review:**` line. A heading under another section SHALL NOT be a task.

#### Scenario: Record tasks checked
- **WHEN** a record's `## Tasks` holds `### Task 1: Move` and another `### Task 1: Rename`
- **THEN** `dkt lint` reports `task-heading` naming both lines

#### Scenario: Record without tasks
- **WHEN** a record has no `## Tasks` section
- **THEN** `dkt lint` reports no task finding for it

#### Scenario: Task heading outside Tasks
- **WHEN** a record's `## What` holds `### Task 1: Move` and its `## Tasks` holds `### Task 1: Rename`
- **THEN** `dkt lint` reports no `task-heading` for them

### Requirement: Record tasks ticked by sync
`dkt sync <slug>` with a slug that starts with a date of the form `YYYY-MM-DD` SHALL treat `docket/records/<slug>.md` as it treats a change's plan.md, with its tasks as defined above and the `Record` trailer in place of `Change`. `result.slug` SHALL be the slug, and `--check` SHALL behave as for a change.

#### Scenario: Record tasks ticked
- **WHEN** a commit carries `Record: 2026-11-01-fix-x/2`, and Task 2 of that record has two unticked steps
- **THEN** after `dkt sync 2026-11-01-fix-x` both steps are ticked and nothing else in the file changes

#### Scenario: Bare record trailer names no task
- **WHEN** the only commit for a record carries `Record: 2026-11-01-fix-x`
- **THEN** `dkt sync 2026-11-01-fix-x` ticks nothing and reports nothing

#### Scenario: Record sync check
- **WHEN** Task 2 of a record has a commit and unticked steps and a user runs `dkt sync 2026-11-01-fix-x --check`
- **THEN** the record is unchanged, the finding `task-unticked` names Task 2, and dkt exits with 1

### Requirement: Record sync problems
`dkt sync` on a record SHALL report `record-missing` and exit with 1 when the worktree has no such file, and SHALL write nothing, report `symlink-refused` and exit with 1 when `docket`, `docket/records` or the record is a symbolic link.

#### Scenario: Record sync missing
- **WHEN** no file `docket/records/2026-11-01-fix-q.md` exists and a user runs `dkt sync 2026-11-01-fix-q`
- **THEN** the finding `record-missing` names that file and dkt exits with 1

#### Scenario: Record sync through a link
- **WHEN** `docket/records` is a symbolic link and a user runs `dkt sync 2026-11-01-fix-x`
- **THEN** nothing is written on either side of the link, the finding `symlink-refused` names `docket/records`, and dkt exits with 1

### Requirement: Record trailers checked
`dkt lint` with no name SHALL report `record-trailer` for each commit between the merge base of HEAD and the default branch's tip, and HEAD, whose `Record` trailer names no regular file under `docket/records/` in the worktree, naming the commit and the value. A value is `<slug>` or `<slug>/<tasks>`. Without a default branch or a merge base it SHALL skip this rule.

#### Scenario: Trailer names no record
- **WHEN** the record is `2026-11-01-fix-x` and a commit the branch adds carries `Record: fix-x`
- **THEN** `dkt lint` reports `record-trailer` naming that commit and `fix-x`, and exits with 1

#### Scenario: Trailer names a record
- **WHEN** the commits the branch adds carry `Record: 2026-11-01-fix-x` and `Record: 2026-11-01-fix-x/2`, and the worktree holds that record
- **THEN** `dkt lint` reports no `record-trailer`

#### Scenario: Merged trailer ignored
- **WHEN** a commit at the merge base carries `Record: gone`
- **THEN** `dkt lint` reports no `record-trailer` for it
