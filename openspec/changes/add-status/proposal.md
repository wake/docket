# Proposal

## Why

Changes in progress live in several worktrees and on branches. Knowing how
far each one has got means opening every worktree's `plan.md` and reading
`git log` by hand, and the counts OpenSpec shows are per step, not per task.
The web workbench planned next (see
`docs/ideas/2026-10-24-web-workbench.md`) needs the same facts as data.
`dkt status` gives both: one listing of every active change, where it
lives, its phase and its task progress, in text for people and in JSON for
the workbench.

## What Changes

- `dkt status [<slug>...]`: lists every active change the scan finds,
  sorted by slug, with every source it was found in; with slugs, only
  those changes.
- For each change: its recorded kind, source, base and creation date;
  which documents it has; its phase (`design`, `ready`, `implementing`,
  `done`); each task of `plan.md` with its state, step counts and the
  commits whose trailers name it; its last activity and the whole days
  since; and, for a change read from a worktree, how many of its files
  have uncommitted changes.
- The run's start time becomes part of the CLI's process boundary, so
  idle days can be computed and tested.

## Non-Goals

- No owner, pull request or acceptance round, though the README draft
  lists them: an owner has no recorded source yet, a pull request needs a
  forge and the network, and acceptance rounds come with `dkt accept`.
- No check that the main specs moved since `x-base`; that is the base
  check of `dkt archive`.
- No document checks; that is `dkt lint`.
- Does not tick `plan.md`; it shows which tasks `dkt sync` would tick.
- Does not compare a change's copies in different sources; content comes
  from the first source, as in `dkt conflicts`.
- Does not list archived changes.
- No watch mode or server; the workbench is a later change.

## Builds On

- `workspace-scan`: `scan.Scan` for the active changes and their sources,
  `scan.ReadFiles` to read a change folder from any source, and the
  precedence that makes `Sources[0]` the content source
  (`conflicts.Load` reads changes the same way).
- `change.ParseMetadata` for `.openspec.yaml` and `change.ParsePlan` for
  tasks, cancelled tasks and steps.
- `plan-sync`: the reading of `Change` and `Task` trailers in
  `internal/archive/sync.go` (`computeSync`), which this change shares
  instead of copying; `gitx.Git.Commits` lists the commits.
- `delta.CapabilityPath` to name the delta specs a change has.
- `gitx.Git.Uncommitted` for uncommitted files.
- In `internal/cli`: `scanDefaultBranch`, `scanUnreadable`, and the
  `slug-invalid` and `change-not-active` findings of `dkt conflicts`.

## Capabilities

### New Capabilities

- `change-status`: how `dkt status` reports where each active change
  lives, its phase, its tasks and its recent activity.

### Modified Capabilities

None.

## Impact

- New command in `internal/cli`; a status package beside `conflicts`.
- `cli.Options` gains the run's start time, set by `cmd/dkt`.
- The trailer reading moves out of `dkt sync` into a shared function; sync
  behaves as before.
- Needs no OpenSpec CLI at run time.

## Implementation Contract

### Observable Behavior

- `dkt status` prints one entry per active change: its slug, phase, task
  progress, idle days and uncommitted files, then its sources with their
  commits. With `--json`, the same as data. It exits 0 unless a named slug
  is not active or invalid (1) or a source could not be read (3).

### Interfaces

- `dkt status [<slug>...] [--json] [--default-branch <name>]`
- JSON `result.changes[]`: `slug`; `sources[]` (`kind`, `name`, `commit`);
  `metadata` (`kind`, `source`, `base`, `created`) or null;
  `documents` (`proposal`, `design`, `plan` as booleans, `specs[]` as
  capability paths); `phase`; `tasks[]` (`number`, `name`, `state`:
  `pending`, `started`, `done` or `cancelled`; `steps`, `ticked`,
  `commits[]` as full SHAs, `unsynced`); `progress` (`tasks_done`,
  `tasks_total`, `steps_ticked`, `steps_total`); `last_activity`
  (`commit`, `time` in RFC 3339, `idle_days`) or null; `uncommitted` (a
  count) or null.
- Finding rules: `change-not-active`, `slug-invalid`.

### Failure Modes

- A source that cannot be read, or a file in a change folder: listed as
  unreadable, exit 3; the other changes are still reported, and a change
  with an unreadable file is reported from the files that could be read.
- `.openspec.yaml` missing or unreadable as metadata: `metadata` is null,
  and so are the commit-derived fields; reporting it is `dkt lint`'s job.
- `x-base` that is not a commit: commits and last activity are left out
  (empty and null), the rest is reported.
- A named slug that is not active: `change-not-active`, exit 1; an invalid
  one: `slug-invalid`, exit 1.

### Acceptance Criteria

- `change-status#changes-in-worktrees-and-branches`,
  `change-status#archived-change-not-listed`,
  `change-status#no-active-changes`,
  `change-status#plan-not-written`, `change-status#nothing-started`,
  `change-status#commit-names-a-task`, `change-status#every-task-done`,
  `change-status#cancelled-task-left-out`,
  `change-status#commits-not-yet-synced`,
  `change-status#idle-change`, `change-status#uncommitted-edits`,
  `change-status#no-commit-yet`,
  `change-status#one-change-in-focus`, `change-status#slug-not-active`,
  `change-status#unreadable-worktree`, `change-status#writes-nothing`.

## Change Log

| Date | Source | Change | Impact |
|---|---|---|---|
