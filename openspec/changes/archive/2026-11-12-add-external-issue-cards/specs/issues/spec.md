## MODIFIED Requirements

### Requirement: Issue file
An issue SHALL be `docket/issues/<slug>.md`: front matter with `title` and `source` of the form `change-metadata` defines, then the level-two sections `## Current`, `## Expected`, `## Unchanged` and `## Evidence`, in that order. A card, an issue whose front matter has `external`, SHALL have no section but `## Dropped`, as its content is kept in the other tool. Any issue MAY end with `## Dropped`.

#### Scenario: Issue written
- **WHEN** a user runs `dkt issue lint-timeout --title "Lint times out" --source "review of #40"` on 2026-10-30
- **THEN** `docket/issues/2026-10-30-lint-timeout.md` holds that title and source in its front matter and the shipped issue template after it, `result.slug` is `2026-10-30-lint-timeout`, `result.path` is that file, and dkt exits with 0

#### Scenario: Card file
- **WHEN** `docket/issues/2026-11-12-login-loop.md` has front matter with `title`, `source`, `external: "https://git.example.com/app/issues/92"` and `external-id: "#92"`, and nothing after it
- **THEN** when the branch adds it, `dkt lint` reports nothing for it

### Requirement: Issue flags
`dkt issue` SHALL take one name and require `--source` and `--title`, each of the form `change-metadata` defines, with the title within the limit it defines. `--external` and `--external-id` SHALL be given together or not at all, each of the form of a source. When the name or a flag is missing, or a value does not have that form, it SHALL write nothing and exit with 2.

#### Scenario: Missing issue source
- **WHEN** a user runs `dkt issue lint-timeout --title X`
- **THEN** dkt exits with 2 and nothing is written

#### Scenario: Issue title over the limit
- **WHEN** a user runs `dkt issue lint-timeout --source x` with a `--title` of 121 bytes
- **THEN** dkt exits with 2, the message gives 121 and 120, and nothing is written

#### Scenario: Address without its number
- **WHEN** a user runs `dkt issue login-loop --title X --source x --external https://git.example.com/app/issues/92`
- **THEN** dkt exits with 2, the message names `--external-id`, and nothing is written

#### Scenario: Blank external number
- **WHEN** a user runs `dkt issue login-loop --title X --source x --external https://git.example.com/app/issues/92 --external-id " "`
- **THEN** dkt exits with 2 and nothing is written

### Requirement: Issue front matter checked
`dkt lint` SHALL report `issue-front-matter` for an issue whose front matter is missing or cannot be read; whose `title` or `source` is missing or not of the form `change-metadata` defines; that has only one of `external` and `external-id`, or one not of the form of a source; or whose `dropped` is other than `true` or `false`. It SHALL report `title-too-long` for a title over the limit.

#### Scenario: Issue without front matter
- **WHEN** an issue the branch adds starts with `## Current`
- **THEN** `dkt lint` reports `issue-front-matter` for that file

#### Scenario: Issue title too long
- **WHEN** an issue the branch adds has a title of 121 bytes
- **THEN** `dkt lint` reports `title-too-long` giving 121 and 120, and no `issue-front-matter`

#### Scenario: Card without its number
- **WHEN** an issue the branch adds has `external` and no `external-id`
- **THEN** `dkt lint` reports `issue-front-matter` naming `external-id`

#### Scenario: Dropped that is not a boolean
- **WHEN** an issue the branch adds has `dropped: yes`
- **THEN** `dkt lint` reports `issue-front-matter` on that line

### Requirement: Issue sections checked
`dkt lint` SHALL report `issue-section` for an issue that is not a card and lacks one of the four sections, has them out of order, has another level-two heading, has text outside its sections, or has one of them with no text beyond whitespace and HTML comments; and for a card with a section other than `## Dropped`, or with text outside it. It SHALL report `placeholder` for text left from the shipped issue template.

#### Scenario: Issue section empty
- **WHEN** an issue the branch adds has `## Unchanged` holding only the template's comment
- **THEN** `dkt lint` reports `issue-section` naming `Unchanged` and exits with 1

#### Scenario: Record section in an issue
- **WHEN** an issue the branch adds has a `## What` section
- **THEN** `dkt lint` reports `issue-section` naming `What`

#### Scenario: Empty card
- **WHEN** a card the branch adds has nothing after its front matter
- **THEN** `dkt lint` reports nothing for it

#### Scenario: Card with content
- **WHEN** a card the branch adds has a `## Current` section
- **THEN** `dkt lint` reports `issue-section` naming `Current`, saying that a card keeps its content in the other tool

## ADDED Requirements

### Requirement: Card written
With `--external` and `--external-id`, `dkt issue` SHALL write a card: front matter with `title`, `source`, `external` and `external-id`, in that order, and nothing after it. `result.external` and `result.external_id` SHALL hold the two values; without them, both SHALL be absent.

#### Scenario: Card from an address
- **WHEN** a user runs `dkt issue login-loop --title "Login loops" --source "https://git.example.com/app/issues/92" --external https://git.example.com/app/issues/92 --external-id "#92" --json` on 2026-11-12
- **THEN** `docket/issues/2026-11-12-login-loop.md` holds the four fields in that order and nothing after them, `result.external_id` is `#92`, and dkt exits with 0

### Requirement: Card address not reused
`dkt issue` SHALL NOT write a card whose `external` is the `external` of an issue in the worktree's `docket/issues/`. It SHALL report `external-taken` naming that issue, write nothing and exit with 1. An issue it cannot read SHALL be listed in `unreadable`, and dkt SHALL write nothing and exit with 3.

#### Scenario: Address already carded
- **WHEN** `docket/issues/2026-11-10-login-loop.md` has `external: "https://git.example.com/app/issues/92"` and a user runs `dkt issue login --title X --source x --external https://git.example.com/app/issues/92 --external-id "#92"`
- **THEN** the finding `external-taken` names that issue, nothing is written, and dkt exits with 1

### Requirement: Card address checked
`dkt lint` SHALL report `issue-external` for an issue it lints whose `external` is also the `external` of another issue in the worktree's `docket/issues/`, naming that issue.

#### Scenario: Two cards for one address
- **WHEN** the branch adds a card whose `external` is the `external` of a card already in `docket/issues/`
- **THEN** `dkt lint` reports `issue-external` for the added card, naming the other, and exits with 1

### Requirement: Dropped issue
An issue SHALL be dropped when its front matter has `dropped: true`, and SHALL then end with `## Dropped`, saying why. `dkt lint` SHALL report `issue-section` for a dropped issue without `## Dropped` or with one that has no text beyond whitespace and HTML comments, and for `## Dropped` in an issue that is not dropped.

#### Scenario: Dropped card
- **WHEN** a card the branch adds has `dropped: true` and `## Dropped` holding `Closed in the tracker as a duplicate.`
- **THEN** `dkt lint` reports nothing for it

#### Scenario: Dropped without a reason
- **WHEN** an issue the branch adds has `dropped: true` and no `## Dropped`
- **THEN** `dkt lint` reports `issue-section` naming `Dropped`

### Requirement: Unsynced writes
An issue MAY list in `unsynced: [<step>, ...]` the writes to the other tool that could not be made: `open`, an issue not filed there yet; `taken`, `pr`, `merged` and `dropped`, the notes for work taking the issue, its pull request, its merge, and the issue being dropped. `dkt lint` SHALL report `issue-unsynced` for another form, an unknown or repeated step, `open` on a card, or another step on an issue that is not a card.

#### Scenario: Notes not written
- **WHEN** a card the branch adds has `unsynced: [taken, pr]`
- **THEN** `dkt lint` reports nothing for it

#### Scenario: Issue not filed yet
- **WHEN** an issue the branch adds has the four sections and `unsynced: [open]`, and no `external`
- **THEN** `dkt lint` reports nothing for it

#### Scenario: Open on a card
- **WHEN** a card the branch adds has `unsynced: [open]`
- **THEN** `dkt lint` reports `issue-unsynced` on that line

#### Scenario: Unsynced as a block
- **WHEN** an issue the branch adds writes `unsynced:` with the steps on the lines under it
- **THEN** `dkt lint` reports `issue-front-matter` for that file
