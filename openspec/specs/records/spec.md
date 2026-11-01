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
Each of the four sections SHALL hold at most 60 words, outside HTML comments. A run of letters or digits SHALL count as one word, except that each Chinese, Japanese or Korean character, the Japanese prolonged sound mark `ー` included, SHALL count as half a word. `dkt lint` SHALL report `record-too-long` naming the section and its count.

#### Scenario: Section over the limit
- **WHEN** a record's `## What` holds 61 English words
- **THEN** `dkt lint` reports `record-too-long` naming `What` and 61

#### Scenario: Chinese text counted
- **WHEN** a record's `## Why` holds 110 Chinese characters and 4 English words
- **THEN** it counts 59 words and `dkt lint` reports no `record-too-long` for it

#### Scenario: Prolonged sound mark counted
- **WHEN** a record's section holds only `コーヒー`
- **THEN** it counts 2 words

### Requirement: Record tasks
A record's `## Tasks` section, when present, SHALL hold tasks in the plan's format, and `dkt lint` SHALL check them by the plan's task heading rules, without requiring a `**Scenarios:**` or `**Review:**` line.

#### Scenario: Record tasks checked
- **WHEN** a record's `## Tasks` holds `### Task 1: Move` and another `### Task 1: Rename`
- **THEN** `dkt lint` reports `task-heading` naming both lines

#### Scenario: Record without tasks
- **WHEN** a record has no `## Tasks` section
- **THEN** `dkt lint` reports no task finding for it

### Requirement: Light review keeps a record
The `docket` skill SHALL have the agent, on the light review path, open a record with `dkt record` before its first commit, fill its sections from the work and the review, give every commit for it the trailer `Record: <slug>` (and `Task: <N>` for a commit that does a record's task) in the message's final paragraph, run `dkt sync <slug>` when the record has tasks, and run `dkt lint` before opening the pull request.

#### Scenario: Light review opens a record
- **WHEN** an agent in a repository set up by `dkt init`, with an `ok` workflow file, is asked to fix a typo in a prompt file
- **THEN** it states the light review path, runs `dkt record` before its first commit, and its commits carry `Record: <slug>` naming that record

#### Scenario: Light review syncs record tasks
- **WHEN** an agent on the light review path has listed tasks under the record's `## Tasks` and committed them with `Task` trailers
- **THEN** it runs `dkt sync <slug>` before `dkt lint`, and the record's steps for those tasks are ticked in the pull request

### Requirement: Records listed by status
`dkt status` SHALL list in `result.records` every record the scan defined by `workspace-scan` finds that the default branch tip does not hold, with state `open`, sorted by slug. Each record SHALL give its slug, its title, kind and source read from the front matter of its first source (each null when the front matter cannot be read), and every source it was found in with that source's commit. The human output SHALL give a line per record, after the changes.

#### Scenario: Open records listed
- **WHEN** worktree B holds the uncommitted record `2026-11-01-fix-x` and branch `refs/heads/fix-y` holds `2026-11-01-fix-y`, and a user runs `dkt status --json` in worktree A
- **THEN** `result.records` lists `2026-11-01-fix-x` and `2026-11-01-fix-y` in that order, each with state `open`, its title, kind and source, and its source

#### Scenario: Merged records hidden
- **WHEN** the default branch tip holds the record `2026-10-31-fix-a` and no other source holds a record, and a user runs `dkt status --json`
- **THEN** `result.records` is empty and `result.records_total` is 0

#### Scenario: Record front matter unreadable
- **WHEN** an open record's file has no front matter
- **THEN** it is listed with `title`, `kind` and `source` null, and the command exits with 0

### Requirement: Record task progress
Each listed record SHALL give the tasks of its `## Tasks` and their progress as `change-status` gives a change's, with each task's commits taken from the commits reachable from the first source's commit and not from the default branch tip that carry `Record: <slug>` and `Task: <N>`. A record without `## Tasks` SHALL give no task and zero progress.

#### Scenario: Record tasks from trailers
- **WHEN** an open record on a branch has two tasks with no step ticked, and a commit on that branch carries `Record: <slug>` and `Task: 1`
- **THEN** Task 1 is `started`, lists that commit and is marked unsynced, and Task 2 is `pending`

### Requirement: Merged records
With `--merged`, `dkt status` SHALL also list every record the default branch tip holds, with state `merged`, after the open ones and newest merge first, ties by slug. Its `merge` SHALL be the newest commit on the first-parent history of the default branch tip that added the record's file, with its committer time, or null when that history does not show one. Its `tag` SHALL be the first tag, by creation date then name, whose commit holds the file, or null.

#### Scenario: Merged records with merge and tag
- **WHEN** `2026-10-30-fix-a` merged in a merge commit on 2026-10-30, tag `v0.9.1` was made after it, and a user runs `dkt status --merged --json`
- **THEN** `2026-10-30-fix-a` is listed with state `merged`, `merge.commit` that merge commit, `merge.time` its committer time, and `tag` `v0.9.1`

### Requirement: Merged record filters
`--since <rev>` SHALL keep only the merged records whose `merge` commit `<rev>` does not reach, leaving out those with a null `merge`. `--limit <n>` SHALL list at most `n` merged records, the newest first. `result.records_total` SHALL count the records the filters matched before `--limit`, and the human output SHALL say how many `--limit` left out. `--since` and `--limit` without `--merged`, or any of the three with slugs, SHALL be a usage error.

#### Scenario: Merged since
- **WHEN** `2026-10-29-fix-a` merged before tag `v0.9.1` and `2026-10-31-fix-b` after it, and a user runs `dkt status --merged --since v0.9.1 --json`
- **THEN** only `2026-10-31-fix-b` is listed among the merged records

#### Scenario: Limit reports total
- **WHEN** the default branch holds five records, no other record exists, and a user runs `dkt status --merged --limit 2 --json`
- **THEN** the two newest merged records are listed, `result.records_total` is 5, and the human output says three more were left out

#### Scenario: Filter flags need merged
- **WHEN** a user runs `dkt status --limit 2`
- **THEN** the command exits with 2

### Requirement: Record in focus
With slugs, `dkt status` SHALL list each named record that any scanned source holds, open or merged, with its `merge` and `tag` when merged, and SHALL report `record-missing` for a named record no source holds.

#### Scenario: Named record listed
- **WHEN** the default branch tip holds `2026-10-30-fix-a` and a user runs `dkt status 2026-10-30-fix-a --json`
- **THEN** `result.records` lists it with state `merged`, its `merge` and its `tag`, and the command exits with 0

#### Scenario: Named record missing
- **WHEN** no source holds `2026-11-01-fix-q` and a user runs `dkt status 2026-11-01-fix-q`
- **THEN** the command reports `record-missing` naming it and exits with 1

### Requirement: Record tasks ticked by sync
`dkt sync <slug>` with a slug that starts with a date SHALL tick, in `docket/records/<slug>.md` of the current worktree, the steps of each task under `## Tasks` that a commit names in a `Task` trailer, taking the commits reachable from HEAD and not from the default branch tip that carry `Record: <slug>`. `result.record` SHALL be the slug and `result.change` null. `--check` SHALL behave as for a change.

#### Scenario: Record tasks ticked
- **WHEN** a commit on the branch carries `Record: 2026-11-01-fix-x` and `Task: 2`, and Task 2 of that record has two unticked steps
- **THEN** after `dkt sync 2026-11-01-fix-x` both steps are ticked and `result.record` is `2026-11-01-fix-x`

#### Scenario: Record sync ignores merged commits
- **WHEN** a commit the default branch tip reaches carries `Record: 2026-11-01-fix-x` and `Task: 1`
- **THEN** `dkt sync 2026-11-01-fix-x` does not tick Task 1 for that commit

#### Scenario: Record sync check
- **WHEN** Task 2 of a record has a commit and unticked steps and a user runs `dkt sync <slug> --check`
- **THEN** the record is unchanged, Task 2 is listed, and the command exits with 1

### Requirement: Record sync problems
Record sync SHALL report `task-unknown` for a `Task` trailer on a commit for the record that names no task of the record, and still tick the others; `record-missing` with exit 1 when the worktree has no such record; and SHALL write nothing and exit with 1, naming the link, when `docket`, `docket/records` or the record is a symbolic link. Without a default branch it SHALL exit with 4.

#### Scenario: Record sync unknown task
- **WHEN** a commit carries `Record: <slug>` and `Task: 9` and the record has no Task 9
- **THEN** `dkt sync <slug>` reports `task-unknown` naming the commit, ticks the other tasks, and exits with 1

#### Scenario: Record sync needs default branch
- **WHEN** no default branch can be found and a user runs `dkt sync 2026-11-01-fix-x`
- **THEN** the command writes nothing and exits with 4

#### Scenario: Record sync through link
- **WHEN** `docket/records` is a symbolic link and a user runs `dkt sync 2026-11-01-fix-x`
- **THEN** nothing is written on either side of the link, the result names the link, and the command exits with 1

### Requirement: Record trailers checked
`dkt lint`, when no name is given, SHALL report `record-trailer` for each commit since the merge base with the default branch whose `Record` trailer names no file under `docket/records/` in the worktree, naming the commit and the value.

#### Scenario: Trailer names no record
- **WHEN** the record is `2026-11-01-fix-x` and a commit the branch adds carries `Record: fix-x`
- **THEN** `dkt lint` reports `record-trailer` naming that commit and `fix-x`, and exits with 1

#### Scenario: Trailer names a record
- **WHEN** every commit the branch adds with a `Record` trailer names `2026-11-01-fix-x`, which the worktree holds
- **THEN** `dkt lint` reports no `record-trailer`

### Requirement: Failed record write leaves nothing
When `dkt record` fails to write the record after it began, it SHALL remove the partial file and any folder the run created, and SHALL report the error; when it cannot remove one of them, the error SHALL name what is left.

#### Scenario: Failed write leaves nothing
- **WHEN** `docket/records` does not exist and writing the new record fails because the disk is full
- **THEN** neither the record file nor `docket/records` exists afterwards, and the command reports the error
