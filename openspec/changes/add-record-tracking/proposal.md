# Proposal

## Why

`add-records` made records exist, but nothing tracks them: `dkt status`
does not list them, `dkt sync` does not tick their tasks, and nothing
reads the `Record:` trailers that light-review commits carry. A record in
progress on another worktree is invisible, and an agent that writes the
bare name instead of the slug in `Record:` goes unnoticed. This is the
second of the two changes that take `docket/ideas/2026-10-29-records.md`.
It also takes the follow-ups from the review of `add-records`, since three
of the four touch the same code.

## What Changes

- **`dkt status` lists records**, in `result.records` beside
  `result.changes`. By default it lists the records not yet merged: those
  the default branch tip does not hold. Each gives its title, kind,
  source, sources, state (`open` or `merged`), tasks and progress.
- **`dkt status --merged`** adds the merged records, newest merge first,
  each with the default branch commit that merged it, that commit's time,
  and the first tag that holds it. `--since <rev>` keeps the ones merged
  after a revision; `--limit <n>` caps how many merged records are listed,
  and `result.records_total` always says how many the filters matched.
- **`dkt status <record-slug>`** lists that record whatever its state. A
  name that starts with a date names a record, as in `dkt lint`.
- **`dkt sync <record-slug>`** ticks the steps of the record's
  `## Tasks` from the commits on HEAD, not on the default branch tip, that
  carry `Record: <slug>` and `Task: <N>`, with `--check` as for changes.
- **`dkt lint`** reports `record-trailer` for a commit the branch adds
  whose `Record:` names no record in the worktree (my call: the risk
  `add-records` left for this change).
- The `docket` skill's light review path runs `dkt sync <slug>` for a
  record with tasks before `dkt lint`.
- Follow-ups from the review of `add-records`:
  - the word count counts the Katakana prolonged sound mark `ー` as half
    a word, like the characters around it;
  - `dkt lint` resolves the default branch and merge base once and lists
    one unreadable entry, not two, when it cannot;
  - the scan lists a worktree once when both its records folder and its
    ancestry cannot be read;
  - a failed `dkt record` write leaves no empty folder, and names what it
    could not remove.

## Non-Goals

- No pagination beyond `--limit`: a cursor such as `--before <commit>`
  waits for the workbench change that needs it.
- No last activity, idle days or uncommitted count for records.
- No new front matter key: a record still has no base.
- Records are never archived, and a record never grows into a change.
- `dkt archive` and `dkt inspect` do not check `Record:` trailers.
- How a workbench shows records is decided elsewhere.

## Builds On

- `scan.Scan`, `Inventory.Records` and `Inventory.RecordSources`
  (`internal/scan`): the scan already finds every record file by name in
  every source, ranked as changes are; this change reads their content.
- `scan.ReadFiles` and the batched tree reads in `internal/scan/trees.go`:
  reading many files and trees in one git process.
- `status.Summarize` and `status.Task`/`status.Progress`
  (`internal/status`): task states and progress, reused for a record's
  tasks.
- `change.NamedTasks` (`internal/change/trailers.go`): matching `Task:`
  trailers to tasks; it keys on `Change:` today.
- `gitx.Git.Commits` (`internal/gitx/log.go`): reads `Change:` and `Task:`
  trailers; it grows to read `Record:`.
- `archive.Sync` and its tick writer (`internal/archive/sync.go`), and
  `safepath.Check`: ticking checkboxes without writing through a link.
- `record.Parse`, `record.Words`, `record.Create`, `record.SlugOf`
  (`internal/record`).
- `addedFiles` in `internal/cli/cmd_lint.go` and `lint.Input`: the merge
  base and the branch's added files.
- `scan.ResolveDefaultBranch`: the default branch and its tip.
- Capabilities: `records`, `workspace-scan`, `change-status`, `plan-sync`,
  `change-lint`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `records`: records listed by `dkt status` with filters, merge and tag;
  record tasks ticked by `dkt sync`; `record-trailer` in `dkt lint`; the
  prolonged sound mark in the word count; a failed write leaves nothing;
  the skill syncs a record with tasks.
- `workspace-scan`: records found in every source the way changes are; a
  source unreadable as a whole is listed once.
- `change-status`: a slug that starts with a date names a record.
- `plan-sync`: `dkt sync` may tick a record instead of `plan.md`.
- `change-lint`: one unreadable entry when the branch's added files cannot
  be found.

## Impact

- `internal/gitx`: `Commit` gains the `Record` trailer values.
- `internal/scan`: the unreadable-once fix.
- `internal/status`: loads records, their merge commits and tags.
- `internal/archive`: sync ticks a record's tasks.
- `internal/lint`, `internal/cli/cmd_lint.go`: one default branch lookup,
  the `record-trailer` rule.
- `internal/cli`: `status` flags and result fields; `sync` result field.
- `internal/record`: word count, write cleanup.
- Prompts: `.claude/skills/docket/SKILL.md` and `.agents/skills/docket/SKILL.md`
  (shipped by `dkt init`).
- Docs: `README.md`, `docs/ARCHITECTURE.md`.
- JSON: `status` gains `result.records` and `result.records_total`;
  `sync` gains `result.record`, and `result.change` is null for a record.

## Implementation Contract

### Observable Behavior

- `dkt status` prints a line per open record after the changes, with its
  sources; `--merged` adds the merged ones with merge date and tag, and
  says how many `--limit` left out.
- `dkt sync 2026-11-01-fix-x` ticks the record's task steps that its
  commits name, and nothing else.
- `dkt lint` on a branch whose commit says `Record: fix-x` when the record
  is `2026-11-01-fix-x` reports `record-trailer`.
- `コーヒー` counts as two words.
- `dkt lint` with no default branch lists one unreadable entry.

### Interfaces

- `dkt status [<slug>...] [--merged [--since <rev>] [--limit <n>]]`.
- `result.records[]`: `slug`, `title`, `kind`, `source` (each null when
  the front matter cannot be read), `sources`, `state`, `merge` (null, or
  `commit` and `time`), `tag` (null or a name), `tasks`, `progress` (as
  for changes).
- `result.records_total`: records the filters matched before `--limit`.
- `dkt sync <record-slug> [--check]`; `result.record` (string or null),
  `result.change` null for a record.
- Finding rules: `record-missing` (status, sync), `record-trailer` (lint),
  `task-unknown` and `plan-unsynced` as for changes.
- Trailer `Record: <slug>`, read like `Change:`.

### Failure Modes

- `--since` or `--limit` without `--merged`, `--merged` with slugs, a
  `--since` that names no commit, or a `--limit` that is not a positive
  integer: usage error, exit 2.
- A named record no source holds: `record-missing`, exit 1.
- `dkt sync` of a record without a default branch: exit 4; of a record
  missing from the worktree: `record-missing`, exit 1; through a link:
  refused, exit 1.
- A tag or record that cannot be read: listed as unreadable, exit 3.
- A merged record whose merge commit history does not show (a shallow
  clone): listed with `merge` null, and left out by `--since`.

### Acceptance Criteria

- `records#open-records-listed`, `records#merged-records-hidden`
- `records#merged-records-with-merge-and-tag`, `records#merged-since`,
  `records#limit-reports-total`, `records#filter-flags-need-merged`
- `records#named-record-listed`, `records#named-record-missing`
- `records#record-front-matter-unreadable`
- `records#record-tasks-from-trailers`
- `records#record-tasks-ticked`, `records#record-sync-ignores-merged-commits`,
  `records#record-sync-check`, `records#record-sync-unknown-task`,
  `records#record-sync-needs-default-branch`, `records#record-sync-through-link`
- `records#trailer-names-no-record`, `records#trailer-names-a-record`
- `records#prolonged-sound-mark-counted`
- `records#failed-write-leaves-nothing`
- `records#light-review-syncs-record-tasks`
- `workspace-scan#record-on-a-branch`,
  `workspace-scan#worktree-listed-once`
- `change-status#record-in-focus`
- `plan-sync#record-file-only`
- `change-lint#one-unreadable-entry`

## Change Log

| Date | Source | Change | Impact |
|---|---|---|---|
