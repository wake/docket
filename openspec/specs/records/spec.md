# records Specification

## Purpose
Defines records: one file per piece of work on the light review path,
saying what was done, why, how it was checked and what its review found;
how `dkt record` writes one, how `dkt lint` checks it, and how the `docket`
skill uses it.

## Requirements

### Requirement: Record file
A record SHALL be `docket/records/<slug>.md`: YAML front matter with `title` (of the form `docket-metadata` defines), `kind` (one of `fix`, `refactor`, `test`, `prompt`, `checks`) and a non-empty `source`, then the level-two sections `## What`, `## Why`, `## Verification` and `## Review`, in that order, and optionally `## Tasks` after them.

#### Scenario: Record written
- **WHEN** a user runs `dkt record fix-lint-timeout --kind fix --title "Lint timeout" --source "review of #40"` on 2026-10-30
- **THEN** `docket/records/2026-10-30-fix-lint-timeout.md` holds that title, kind and source in its front matter and the four sections from the shipped template, `result.slug` is `2026-10-30-fix-lint-timeout`, and the command exits with 0

### Requirement: Record creation
`dkt record <name>` SHALL require `--kind` with one of the five kinds, a non-empty `--source` and a `--title`, and SHALL exit with 2, writing nothing, when one is missing or invalid. The name SHALL follow the slug rules of `dkt new`; the record's slug SHALL be today's date as `YYYY-MM-DD`, a hyphen and the name.

#### Scenario: Unknown kind
- **WHEN** a user runs `dkt record fix-x --kind chore --title X --source x`
- **THEN** the command exits with 2, lists the five kinds, and writes nothing

#### Scenario: Name with a date
- **WHEN** a user runs `dkt record 2026-10-30-fix-x --kind fix --title X --source x`
- **THEN** the command exits with 1, explains that `dkt record` adds the date, and writes nothing

### Requirement: Record slug never reused
`dkt record` SHALL NOT write a record whose slug any scanned source already uses for a record. When the slug is used, it SHALL write the record under the first of `<slug>-2`, `<slug>-3`, ... that no scanned source uses and that does not exist in the worktree, and SHALL report that slug. When any source is unreadable it SHALL write nothing and exit with 3.

#### Scenario: Same name same day
- **WHEN** `docket/records/2026-10-30-fix-x.md` exists on a branch and a user runs `dkt record fix-x --kind fix --title X --source x` on 2026-10-30 in another worktree
- **THEN** the command writes `docket/records/2026-10-30-fix-x-2.md`, `result.slug` is `2026-10-30-fix-x-2`, and it exits with 0

#### Scenario: Unreadable source blocks the record
- **WHEN** a listed worktree cannot be read and a user runs `dkt record fix-x --kind fix --title X --source x`
- **THEN** nothing is written and the command exits with 3

#### Scenario: Concurrent records
- **WHEN** two runs of `dkt record fix-x --kind fix --title X --source x` start at the same time in one worktree on 2026-10-30
- **THEN** one writes `2026-10-30-fix-x.md`, the other writes `2026-10-30-fix-x-2.md`, and both exit with 0

### Requirement: Records folder not written through a link
`dkt record` SHALL NOT write through a symbolic link: when `docket`, `docket/records` or the record's path is one, it SHALL write nothing, name the link and exit with 1.

#### Scenario: Symlinked records folder refused
- **WHEN** `docket/records` is a symbolic link to a directory outside the repository and a user runs `dkt record fix-x --kind fix --title X --source x`
- **THEN** nothing is written on either side of the link, the result names the link, and the command exits with 1

### Requirement: Record front matter checked
`dkt lint` SHALL report `record-front-matter` for a record whose front matter is missing, is not a YAML mapping, or lacks a valid `title` or a non-empty `source`, and `record-kind` for a `kind` that is not one of the five.

#### Scenario: Record kind not known
- **WHEN** a record the branch adds has `kind: chore`
- **THEN** `dkt lint` reports `record-kind` for that record and exits with 1

### Requirement: Record sections checked
`dkt lint` SHALL report `record-section` for a record that lacks one of the four sections, has them out of order, has another level-two heading, or has a section with no text beyond whitespace and HTML comments. It SHALL report `placeholder` for text left from the shipped record template.

#### Scenario: Record section empty
- **WHEN** a record the branch adds has `## Review` with nothing under it
- **THEN** `dkt lint` reports `record-section` naming `Review` and exits with 1

#### Scenario: Template text left
- **WHEN** a record still holds the template's comment under `## Why`
- **THEN** `dkt lint` reports `placeholder` on that line

### Requirement: Section word limits
Each of the four sections SHALL hold at most 60 words, outside HTML comments. A run of letters or digits SHALL count as one word, except that each Chinese, Japanese or Korean character SHALL count as half a word. `dkt lint` SHALL report `record-too-long` naming the section and its count.

#### Scenario: Section over the limit
- **WHEN** a record's `## What` holds 61 English words
- **THEN** `dkt lint` reports `record-too-long` naming `What` and 61

#### Scenario: Chinese text counted
- **WHEN** a record's `## Why` holds 110 Chinese characters and 4 English words
- **THEN** it counts 59 words and `dkt lint` reports no `record-too-long` for it

### Requirement: Record tasks
A record's `## Tasks` section, when present, SHALL hold tasks in the plan's format, and `dkt lint` SHALL check them by the plan's task heading rules, without requiring a `**Scenarios:**` or `**Review:**` line.

#### Scenario: Record tasks checked
- **WHEN** a record's `## Tasks` holds `### Task 1: Move` and another `### Task 1: Rename`
- **THEN** `dkt lint` reports `task-heading` naming both lines

#### Scenario: Record without tasks
- **WHEN** a record has no `## Tasks` section
- **THEN** `dkt lint` reports no task finding for it

### Requirement: Light review keeps a record
The `docket` skill SHALL have the agent, on the light review path, open a record with `dkt record` before its first commit, fill its sections from the work and the review, give every commit for it the trailer `Record: <slug>` (and `Task: <N>` for a commit that does a record's task) in the message's final paragraph, and run `dkt lint` before opening the pull request.

#### Scenario: Light review opens a record
- **WHEN** an agent in a repository set up by `dkt init`, with an `ok` workflow file, is asked to fix a typo in a prompt file
- **THEN** it states the light review path, runs `dkt record` before its first commit, and its commits carry `Record: <slug>` naming that record
