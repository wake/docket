# Proposal

## Why

Work on the light review path opens no change: a fix that restores known
behavior, a restructuring, tests for existing behavior, prompt text, check
settings. Nothing in the repository says what such work did, why, how it
was checked or what its review found. Later ideas need that record:
`docket/ideas/2026-10-29-commit-gate.md` names a commit's container with a
`Record:` trailer, and `docket/ideas/2026-10-30-process-review.md` counts
light-review work. This is the first of two changes that take
`docket/ideas/2026-10-29-records.md`; the second tracks records in
`dkt status` and `dkt sync`.

## What Changes

- A **record** is one Markdown file per piece of light-review work,
  `docket/records/<slug>.md`, with front matter `title`, `kind` and
  `source`. Kinds, one per kind of light-review edit: `fix`, `refactor`,
  `test`, `prompt`, `checks`.
- Every record has the same four sections, `## What`, `## Why`,
  `## Verification` and `## Review`, each with a word limit. A record may
  add a `## Tasks` section holding tasks in the plan's format.
- New command **`dkt record <name> --kind <kind> --title <text> --source <text>`**
  writes the record from the shipped template. The slug is today's date,
  then the name (`2026-10-30-fix-lint-timeout`); a slug any record already
  uses on any scanned source gets the first free suffix `-2`, `-3`, ...
  instead of being refused.
- **`dkt new` no longer refuses a used slug**: it takes the first free
  suffix the same way and reports the slug it created. The kind `fix`
  leaves `dkt new`, so the word means one thing: a fix restores a known
  answer and is a record; a bug whose right behavior cannot be stated is a
  `change-request`.
- **`dkt lint`** checks the records the branch adds: front matter, kind,
  every section filled with no template text left, each section within its
  word limit, and tasks by the plan's task rules.
- The `docket` skill's light review path opens a record with `dkt record`,
  fills it, and has every commit for it carry `Record: <slug>` (with
  `Task: <N>` for a record's task).

## Non-Goals

- `dkt status` and `dkt sync` do not read records yet, and nothing reads
  `Record:` trailers; the second change does both.
- No record for the check or direct merge paths.
- A record never grows into a change, and is never archived.
- No numbered or random slugs. The `-2` suffix is a stopgap the maintainer
  may replace.
- How a workbench shows records is decided elsewhere.

## Builds On

- `change.ValidateSlug`, `change.ValidTitle` and `change.Kinds`
  (`internal/change`): slug and title rules, the kinds `dkt new` accepts.
- `scan.Scan` and `Inventory.Uses` (`internal/scan`): every worktree,
  branch and the default branch, as `dkt new` uses them to find a slug in
  use; the scan grows to list records.
- `dkt new`'s lock on `openspec/` and its creation steps
  (`internal/cli/cmd_new.go`).
- `lint.Run` and its plan task rules, placeholder rule and `Added` files
  (`internal/lint`, `internal/cli/cmd_lint.go`).
- `change.ParsePlan` (`internal/change/plan.go`): tasks and steps.
- `markdown.Lines` and `markdown.Headings` (`internal/markdown`).
- `safepath.Check`: no write through a symbolic link.
- `Context.Now`: the clock, passed in through `cli.Options`.
- The `change-creation` and `change-lint` capabilities.

## Capabilities

### New Capabilities

- `records`: the record file, `dkt record`, the record lint rules, and the
  `docket` skill's use of records on the light review path.

### Modified Capabilities

- `change-creation`: a used slug gets a suffix instead of a refusal; the
  kind `fix` is gone.
- `change-lint`: `dkt lint` also lints records.

## Impact

- New `internal/record` (the record format, the template, word counting),
  `internal/cli/cmd_record.go`, and an embedded record template.
- `internal/cli/cmd_new.go`, `internal/change/new.go`, `internal/scan`,
  `internal/lint`, `internal/cli/cmd_lint.go`.
- Shipped prompt text: the `docket` skill (both copies) and the record
  template; the prompt path list in `AGENTS.md`.
- Both READMEs, `docs/ARCHITECTURE.md`, the records idea (status).

## Implementation Contract

### Observable Behavior

- `dkt record fix-lint-timeout --kind fix --title "..." --source "..."`
  run on 2026-10-30 writes `docket/records/2026-10-30-fix-lint-timeout.md`
  and reports that slug; a second run the same day writes
  `2026-10-30-fix-lint-timeout-2`.
- `dkt new add-x` where `add-x` is used anywhere creates `add-x-2` and
  reports it.
- `dkt lint` on a branch that adds a record reports a section left empty,
  template text left, a section over its limit, or a malformed task.
- On the light review path the agent opens a record before its first
  commit, and its commits carry `Record: <slug>`.

### Interfaces

- `dkt record <name> --kind fix|refactor|test|prompt|checks --title <text> --source <text>`.
  `result` has `slug`, `path` (repository-relative), `title`, `kind`,
  `source`, `requested` (the slug before any suffix) and `used` (each
  place the requested slug is used; empty when no suffix was needed).
- `dkt new`'s `result.slug` is the slug it created, which may carry a
  suffix, and its `result` gains `requested` and `used` the same way.
- Record file: front matter `title`, `kind`, `source`; sections
  `## What`, `## Why`, `## Verification`, `## Review`, in that order; an
  optional `## Tasks` after them.
- Word limit: 60 words per section, counting each run of letters or digits
  outside Chinese, Japanese and Korean script as one word and each such
  character as half a word.
- Lint rules: `record-front-matter`, `record-kind`, `record-section`,
  `record-too-long`, `record-missing` (a named record that does not
  exist), `placeholder` (the record template's text), and the plan's
  `task-heading` rule for tasks. `dkt lint`'s `result` gains `records`,
  the record slugs it linted.
- Commit trailers for a record: `Record: <slug>`, and `Task: <N>` for a
  task, in the message's final paragraph.

### Failure Modes

- A missing or invalid kind, title or source: exit 2, nothing written.
  An invalid name, such as one starting with a date: `slug-invalid`,
  exit 1, nothing written, as `dkt new` does.
- A source that cannot be scanned: nothing written, exit 3, as for
  `dkt new`.
- `docket/records` or the record's path passes through a symbolic link:
  nothing written, `symlink-refused`, exit 1.
- No default branch: exit 4, as for every command that scans.
- Two runs that pick the same slug at once in one worktree: one writes the
  slug, the other takes the next suffix.

### Acceptance Criteria

- `records#record-written`
- `records#same-name-same-day`
- `records#name-with-a-date`
- `records#unknown-kind`
- `records#unreadable-source-blocks-the-record`
- `records#symlinked-records-folder-refused`
- `records#concurrent-records`
- `records#record-kind-not-known`
- `records#record-section-empty`
- `records#template-text-left`
- `records#section-over-the-limit`
- `records#chinese-text-counted`
- `records#record-tasks-checked`
- `records#record-without-tasks`
- `records#light-review-opens-a-record`
- `change-creation#slug-active-in-another-worktree`
- `change-creation#slug-archived`
- `change-creation#concurrent-runs-keep-the-first`
- `change-creation#unknown-kind`
- `change-lint#lint-records-the-branch-adds`

## Change Log

| Date | Source | Change | Impact |
|---|---|---|---|
