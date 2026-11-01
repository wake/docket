# issues Specification

## Purpose
Defines issues: work whose expected behavior is already settled and that is
not done now. Covers the template Docket ships for them, file issues under
`docket/issues/` for a project that names no outside tool, how `dkt issue`
writes one, how `dkt lint` checks it, how `dkt status` lists it, and how the
`docket` skill uses issues.

## Requirements

### Requirement: Issue template
Docket SHALL ship an issue template with the level-two sections `## Current`, `## Expected`, `## Unchanged` and `## Found`, in that order, each holding an HTML comment that says what goes in it; the comment under `## Expected` SHALL ask where the expected behavior is settled. `dkt issue template` SHALL print the sections, and with `--json` give them in `result.text`.

#### Scenario: Template printed
- **WHEN** a user runs `dkt issue template --json`
- **THEN** `result.text` holds `## Current`, `## Expected`, `## Unchanged` and `## Found` in that order, and the command exits with 0

### Requirement: Issue file
A file issue SHALL be `docket/issues/<slug>.md`: YAML front matter with `title` (of the form `docket-metadata` defines) and a non-empty `source`, optionally `dropped: true` and `idea` naming a file under `docket/ideas/`, then the four sections of the template.

#### Scenario: Issue written
- **WHEN** a user runs `dkt issue lint-timeout --title "Lint timeout" --source "review of #40"` on 2026-11-02
- **THEN** `docket/issues/2026-11-02-lint-timeout.md` holds that title and source in its front matter and the four sections from the template, `result.slug` is `2026-11-02-lint-timeout`, and the command exits with 0

### Requirement: Issue creation
`dkt issue <name>` SHALL require a `--title` and a non-empty `--source`, and SHALL exit with 2, writing nothing, when one is missing or invalid. The name SHALL follow the slug rules of `dkt new`; the slug SHALL be today's date as `YYYY-MM-DD`, a hyphen and the name, and a name that starts with a date SHALL exit with 1. The name `template` SHALL print the template instead.

#### Scenario: Missing source writes nothing
- **WHEN** a user runs `dkt issue lint-timeout --title "Lint timeout"`
- **THEN** the command exits with 2 and writes nothing

#### Scenario: Issue name with a date
- **WHEN** a user runs `dkt issue 2026-11-02-x --title X --source x`
- **THEN** the command exits with 1, explains that `dkt issue` adds the date, and writes nothing

### Requirement: Issue slug never reused
`dkt issue` SHALL NOT write an issue whose slug any scanned source already uses for an issue. When the slug is used, it SHALL write the issue under the first of `<slug>-2`, `<slug>-3`, ... that no scanned source uses and that does not exist in the worktree, and SHALL report that slug. When any source is unreadable it SHALL write nothing and exit with 3.

#### Scenario: Same name same day
- **WHEN** `docket/issues/2026-11-02-x.md` exists on a branch and a user runs `dkt issue x --title X --source x` on 2026-11-02 in another worktree
- **THEN** the command writes `docket/issues/2026-11-02-x-2.md`, `result.slug` is `2026-11-02-x-2`, and it exits with 0

#### Scenario: Unreadable source blocks the issue
- **WHEN** a listed worktree cannot be read and a user runs `dkt issue x --title X --source x`
- **THEN** nothing is written and the command exits with 3

### Requirement: Issues folder not written through a link
`dkt issue` SHALL NOT write through a symbolic link: when `docket`, `docket/issues` or the issue's path is one, it SHALL write nothing, name the link and exit with 1.

#### Scenario: Symlinked issues folder refused
- **WHEN** `docket/issues` is a symbolic link to a directory outside the repository and a user runs `dkt issue x --title X --source x`
- **THEN** nothing is written on either side of the link, the result names the link, and the command exits with 1

### Requirement: Failed issue write leaves nothing
When `dkt issue` fails to write the issue after it began, it SHALL remove the partial file and any folder the run created, and SHALL report the error; when it cannot remove one of them, the error SHALL name what is left.

#### Scenario: Failed write leaves nothing
- **WHEN** `docket/issues` does not exist and writing the new issue fails because the disk is full
- **THEN** neither the issue file nor `docket/issues` exists afterwards, and the command reports the error

### Requirement: Issue front matter checked
`dkt lint` SHALL report `issue-front-matter` for a file issue whose front matter is missing, is not a YAML mapping, lacks a valid `title` or a non-empty `source`, has `dropped` with a value other than `true`, or has an `idea` that names no file under `docket/ideas/` in the worktree.

#### Scenario: Issue front matter invalid
- **WHEN** a file issue the branch adds has `idea: docket/ideas/2026-11-02-none.md` and the worktree has no such file
- **THEN** `dkt lint` reports `issue-front-matter` for that issue and exits with 1

### Requirement: Issue sections checked
`dkt lint` SHALL report `issue-section` for a file issue that lacks one of the four sections, has them out of order, has another level-two heading, or has a section with no text beyond whitespace and HTML comments. It SHALL report `placeholder` for text left from the shipped issue template.

#### Scenario: Issue section empty
- **WHEN** a file issue the branch adds has `## Unchanged` with nothing under it
- **THEN** `dkt lint` reports `issue-section` naming `Unchanged` and exits with 1

#### Scenario: Issue template text left
- **WHEN** a file issue still holds the template's comment under `## Found`
- **THEN** `dkt lint` reports `placeholder` on that line

### Requirement: Issue section word limits
Each of the four sections of a file issue SHALL hold at most 60 words outside HTML comments, counted as `records` counts a record's words. `dkt lint` SHALL report `issue-too-long` naming the section and its count.

#### Scenario: Issue section over the limit
- **WHEN** a file issue's `## Current` holds 61 English words
- **THEN** `dkt lint` reports `issue-too-long` naming `Current` and 61

### Requirement: Issue sources checked
`dkt lint` SHALL report `issue-source` for each change it lints and each record the branch adds whose source names a path under `docket/issues/`, as `docket-metadata` defines, that is no file issue in the worktree, naming the source.

#### Scenario: Source names no issue
- **WHEN** a record the branch adds has `source: docket/issues/2026-11-02-x.md` and the worktree has no such file
- **THEN** `dkt lint` reports `issue-source` naming that source and exits with 1

#### Scenario: Source names an issue
- **WHEN** a record the branch adds has `source: docket/issues/2026-11-02-x.md` and the worktree holds that file
- **THEN** `dkt lint` reports no `issue-source`

#### Scenario: Source with a note names no issue
- **WHEN** a record the branch adds has `source: docket/issues/2026-11-02-x.md, from review of #40` and the worktree has no such file
- **THEN** `dkt lint` reports `issue-source` naming that source

### Requirement: Issue state
A file issue's handlers SHALL be the records and changes, active or archived, whose source names the issue's path as `docket-metadata` defines, each read from the default branch tip when the tip holds it and from its first source otherwise. Its state SHALL be `dropped` with `dropped: true`, `moved` with an `idea`, and otherwise `pending` with no handler, `in-progress` when the tip does not hold a handler, and `handled` when it holds every handler.

#### Scenario: Handler on a branch
- **WHEN** the default branch tip holds `docket/issues/2026-11-02-x.md` and an unmerged branch holds a record whose source is that path
- **THEN** the issue's state is `in-progress`

#### Scenario: Handler merged
- **WHEN** the default branch tip holds the issue and a record whose source is its path, and no other source holds a handler the tip does not
- **THEN** the issue's state is `handled`

#### Scenario: Merged change not yet archived
- **WHEN** the default branch tip holds the active change `add-x`, whose `docket.yaml` has the issue's path as its source
- **THEN** the issue's state is `handled`

#### Scenario: Merged record edited elsewhere
- **WHEN** the default branch tip holds a record whose source is `review of #40`, and worktree B changes that record's source to the issue's path
- **THEN** the record is not a handler of the issue

#### Scenario: Handler with a note
- **WHEN** the default branch tip holds the issue and a record whose source is its path followed by `, found while fixing y`
- **THEN** the issue's state is `handled`

### Requirement: Issues listed by status
`dkt status` SHALL list in `result.issues`, sorted by slug, every file issue the scan finds that is `pending` or `in-progress`, each with its slug, title and source from the front matter of its first source (null when unreadable), its state, its handlers (kind `record` or `change`, slug, and whether the default branch tip holds it) and its sources with their commits, and a line per issue after the records. With slugs it SHALL list no issue.

#### Scenario: Pending issue listed
- **WHEN** the default branch tip holds `docket/issues/2026-11-02-x.md` and no record or change names it, and a user runs `dkt status --json`
- **THEN** `result.issues` lists `2026-11-02-x` with state `pending`, its title and source, and no handler

#### Scenario: Issue in progress
- **WHEN** worktree B holds an uncommitted record whose source is `docket/issues/2026-11-02-x.md`, and a user runs `dkt status --json` in worktree A
- **THEN** `result.issues` lists `2026-11-02-x` with state `in-progress` and that record as an unmerged handler

#### Scenario: Handled issue hidden
- **WHEN** the default branch tip holds the issue and a merged record that names it, and no other source holds a handler
- **THEN** `result.issues` does not list it

#### Scenario: Dropped and moved hidden
- **WHEN** one file issue has `dropped: true`, another has `idea: docket/ideas/2026-11-02-y.md`, and neither has a handler
- **THEN** `result.issues` lists neither

#### Scenario: Issues not listed with slugs
- **WHEN** a file issue is pending and a user runs `dkt status add-x --json`
- **THEN** `result.issues` is empty

### Requirement: Issues for settled work
The `docket` skill SHALL have the agent turn work whose expected behavior is settled and that it does not do now into an issue: in the tool named by a `## Issues` section of the project's `AGENTS.md`, with the sections `dkt issue template` prints, once the user agrees to file it there; otherwise with `dkt issue`. Work whose expected behavior it cannot point to SHALL go to an idea.

#### Scenario: Bug found out of scope
- **WHEN** an agent in a project whose `AGENTS.md` has no `## Issues` section finds, during other work, a bug whose expected behavior a spec states
- **THEN** it opens a file issue with `dkt issue` instead of writing an idea

#### Scenario: Issue in the project tool
- **WHEN** the project's `AGENTS.md` has `## Issues` naming GitHub Issues and the agent finds such a bug
- **THEN** it shows the user an issue made from the sections `dkt issue template` prints, files it there only once the user agrees, and runs no `dkt issue`

### Requirement: Work names its issue
When work handles an issue, the `docket` skill SHALL have the agent pass the issue as the `--source` of the record or the change: the path of a file issue, or the issue's address in the outside tool.

#### Scenario: Record names its issue
- **WHEN** an agent on the light review path fixes the bug of `docket/issues/2026-11-02-x.md`
- **THEN** it runs `dkt record` with `--source docket/issues/2026-11-02-x.md`
