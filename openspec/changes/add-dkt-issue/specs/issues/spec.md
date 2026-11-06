# Spec Delta

## Purpose

Defines issue files: one file per request whose expected behavior is already settled but that is put off; how `dkt issue` writes one, what `dkt issue template` prints for a project that keeps its issues in another tool, and how `dkt lint` checks them.

## ADDED Requirements

### Requirement: Issue file
An issue SHALL be `docket/issues/<slug>.md`: front matter with `title` and `source` of the form `change-metadata` defines, then the level-two sections `## Current`, `## Expected`, `## Unchanged` and `## Evidence`, in that order.

#### Scenario: Issue written
- **WHEN** a user runs `dkt issue lint-timeout --title "Lint times out" --source "review of #40"` on 2026-10-30
- **THEN** `docket/issues/2026-10-30-lint-timeout.md` holds that title and source in its front matter and the shipped issue template after it, `result.slug` is `2026-10-30-lint-timeout`, `result.path` is that file, and dkt exits with 0

### Requirement: Issue flags
`dkt issue` SHALL take one name and require `--source` and `--title`, each of the form `change-metadata` defines, with the title within the limit it defines. When the name or a flag is missing, or a value does not have that form, it SHALL write nothing and exit with 2.

#### Scenario: Missing issue source
- **WHEN** a user runs `dkt issue lint-timeout --title X`
- **THEN** dkt exits with 2 and nothing is written

#### Scenario: Issue title over the limit
- **WHEN** a user runs `dkt issue lint-timeout --source x` with a `--title` of 121 bytes
- **THEN** dkt exits with 2, the message gives 121 and 120, and nothing is written

### Requirement: Issue names
An issue name SHALL follow the rules `records` gives a record name, and the issue's slug SHALL be the local date as `YYYY-MM-DD`, a hyphen and the name. Another name SHALL report `slug-invalid`, saying that `dkt issue` adds the date when the name starts with one, write nothing and exit with 1.

#### Scenario: Issue name with a date
- **WHEN** a user runs `dkt issue 2026-10-30-lint-timeout --title X --source x`
- **THEN** the finding `slug-invalid` says that `dkt issue` adds the date, nothing is written, and dkt exits with 1

### Requirement: Issue slug never reused
`dkt issue` SHALL NOT write an issue whose slug any scanned source uses for an issue, and SHALL choose a suffixed slug and report `result.slug`, `result.requested` and `result.used` as `records` says `dkt record` does. When any source is unreadable it SHALL write nothing and exit with 3.

#### Scenario: Same issue name same day
- **WHEN** a branch holds `docket/issues/2026-10-30-lint-timeout.md` and a user runs `dkt issue lint-timeout --title X --source x` on 2026-10-30 in a worktree without it
- **THEN** dkt writes `docket/issues/2026-10-30-lint-timeout-2.md`, `result.requested` is `2026-10-30-lint-timeout`, `result.used` names that branch, and dkt exits with 0

#### Scenario: Concurrent issues
- **WHEN** two runs of `dkt issue lint-timeout --title X --source x` start at the same time in one worktree on 2026-10-30
- **THEN** one writes `2026-10-30-lint-timeout.md`, the other `2026-10-30-lint-timeout-2.md`, and both exit with 0

### Requirement: Issues folder not written through a link
`dkt issue` SHALL NOT write through a symbolic link: when `docket`, `docket/issues` or the issue's path is one, it SHALL write nothing, report `symlink-refused` naming the link and exit with 1.

#### Scenario: Linked issues folder refused
- **WHEN** `docket/issues` is a symbolic link to a directory outside the repository and a user runs `dkt issue lint-timeout --title X --source x`
- **THEN** nothing is written on either side of the link, the finding `symlink-refused` names `docket/issues`, and dkt exits with 1

### Requirement: Failed issue write leaves nothing
When `dkt issue` fails to write the issue after it began, it SHALL remove the partial file and any folder the run created, and SHALL report the error; when it cannot remove one of them, the error SHALL name what is left.

#### Scenario: Failed issue write leaves nothing
- **WHEN** `docket/issues` does not exist and writing the new issue fails
- **THEN** neither the issue file nor `docket/issues` exists afterwards, and the command reports the error

### Requirement: Issue template printed
`dkt issue template` SHALL print the shipped template for an issue kept in another tool, which has the four sections of an issue file, and SHALL write nothing; with `--json`, `result.template` SHALL hold it. Any other argument or flag after `template` SHALL be a usage error, and `template` is not an issue name.

#### Scenario: Template for another tool
- **WHEN** a user runs `dkt issue template --json`
- **THEN** `result.template` has the level-two headings `## Current`, `## Expected`, `## Unchanged` and `## Evidence` in that order, nothing is written, and dkt exits with 0

#### Scenario: Template with a flag
- **WHEN** a user runs `dkt issue template --title X --source x`
- **THEN** dkt exits with 2 and nothing is written

### Requirement: Issues linted
`dkt lint` with no name SHALL lint every issue the branch adds, chosen as `records` chooses the records the branch adds, under `docket/issues/`. With no default branch or no merge base it SHALL lint no issue this way.

#### Scenario: Issue the branch adds
- **WHEN** the current branch adds `docket/issues/2026-11-06-lint-timeout.md`, not yet committed, without a source, and a user runs `dkt lint`
- **THEN** the finding `issue-front-matter` names that file and dkt exits with 1

#### Scenario: Merged issue left alone
- **WHEN** an issue that breaks a rule exists at the merge base and a user runs `dkt lint`
- **THEN** no finding is about that issue

### Requirement: Issue front matter checked
`dkt lint` SHALL report `issue-front-matter` for an issue whose front matter is missing or cannot be read, or whose `title` or `source` is missing or not of the form `change-metadata` defines, and `title-too-long` for a title over the limit.

#### Scenario: Issue without front matter
- **WHEN** an issue the branch adds starts with `## Current`
- **THEN** `dkt lint` reports `issue-front-matter` for that file

#### Scenario: Issue title too long
- **WHEN** an issue the branch adds has a title of 121 bytes
- **THEN** `dkt lint` reports `title-too-long` giving 121 and 120, and no `issue-front-matter`

### Requirement: Issue sections checked
`dkt lint` SHALL report `issue-section` for an issue that lacks one of the four sections, has them out of order, has another level-two heading, has text outside its sections, or has one of them with no text beyond whitespace and HTML comments. It SHALL report `placeholder` for text left from the shipped issue template.

#### Scenario: Issue section empty
- **WHEN** an issue the branch adds has `## Unchanged` holding only the template's comment
- **THEN** `dkt lint` reports `issue-section` naming `Unchanged` and exits with 1

#### Scenario: Record section in an issue
- **WHEN** an issue the branch adds has a `## What` section
- **THEN** `dkt lint` reports `issue-section` naming `What`

### Requirement: Issue length
The four sections of an issue SHALL together hold at most 1,440 bytes, measured as `records` measures a record's sections. `dkt lint` SHALL report `issue-too-long` giving the length and the limit.

#### Scenario: Issue over the limit
- **WHEN** the four sections of an issue the branch adds hold 1,441 bytes
- **THEN** `dkt lint` reports `issue-too-long` giving 1441 and 1440
