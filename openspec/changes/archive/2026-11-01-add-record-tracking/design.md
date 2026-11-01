# Design

## Context

See proposal.md for why. The scan already lists every record file by name
in every source, ranked as changes are (`scan.Inventory.Records`), so that
`dkt record` can pick a free slug; nothing reads a record's content or its
commits. A change's commits are counted from the base its `docket.yaml`
records; a record's front matter has no base. Changes leave `dkt status`
when they are archived; records are never archived, so the merged ones only
grow.

## Goals / Non-Goals

**Goals:**
- `dkt status` stays as fast as today when nothing is asked about merged
  records, however many records the default branch holds.
- Asking about merged records costs a fixed number of git processes, not
  some per record.

**Non-Goals:**
- No new data in record files, and no migration of the three records
  already merged here.

## Decisions

### A record's commits are the ones the default branch tip does not reach

A record has no base, and the commit that adds its file is not its first
commit: in this repository, `2026-10-31-scan-tree-batch`'s first commit
with `Record:` comes before the one that adds the record. So `dkt sync`
counts the commits reachable from HEAD and not from the default branch
tip, and `dkt status` the ones reachable from the record's first source's
commit and not from the tip. Each is one `git log` that walks only the
branch's own commits and those the tip gained since they split.

- Alternative: a `base` key written by `dkt record`. Same cost, but it
  changes the file format and the merged records have none.
- Alternative: the parent of the commit that adds the file. Wrong for the
  record above.

A merged record's range is empty, so `dkt status` runs no `git log` for it
and lists its tasks from the ticks alone. Several open records on one
source share one `git log` (my call).

### Default lists open records; merged ones are asked for

The maintainer's own large project merged about 1,800 pull requests in
three weeks and makes releases without tags. With records for half of
them, a list of every record would run to thousands of lines and several
megabytes of JSON, and "not yet in a tag" would match all of them. So:

- by default, only open records: their number is bounded like active
  changes';
- `--merged` adds merged ones, newest merge first, with `--since <rev>`
  for "merged after the last release" (a tag, or the commit of a version
  bump) and `--limit <n>` with `records_total` so a cut is never silent;
- a named record is listed whatever its state.

Pagination with a cursor (`--before <commit>`) suits browsing in a
workbench, not the agents and release steps that read `dkt status` today,
which ask questions with one answer; an agent handed page one may take it
for all. It can be added on `--merged` later without changing this.

The filters with slugs are a usage error rather than ignored (my call):
fewer combinations, and no silent surprise.

### Merge commits and tags in a fixed number of git processes

Only when merged records are listed:

- one `git log --first-parent --diff-filter=A --name-only <tip> -- docket/records/`
  (with `^<rev>` for `--since`) gives each merged record's merge commit and
  time; the newest addition wins when a file was added twice;
- the tags, listed once by creation date, have their trees read in the
  scan's existing batched tree reads (`internal/scan/trees.go`), and a
  record's tag is the first tree that holds its file.

The log passes `--root`, so a record added in the root commit is found
even when `log.showRoot` is off. A record the log does not show (a shallow
clone) has `merge` null, sorts last, and is left out by `--since`. A tag
whose tree cannot be read is listed as unreadable (exit 3), as any
unreadable source is. git refuses to list the tags at all, sorted by
date, when one tag's object is missing; then `refs/tags` is listed as
unreadable once, with git's reason naming the tag, and every `tag` is null
(my call: listing and peeling each tag on its own would cost a process per
tag for a damaged repository).

- Alternative: `git log` and `git tag --contains` per record. Two
  history walks per record; thousands of processes in a large project.

Merged records' files are read from the tip with one `scan.ReadFiles`
batch, and only after `--since` and `--limit` have picked them.

### `records` beside `changes` in the JSON

The idea left open whether records join `result.changes` with a field
telling them apart. They do not: a change has a phase, documents, metadata
and activity; a record has a kind, a state, a merge and a tag. A separate
`result.records` keeps every reader of `changes` working unchanged.
`result.records_total` counts what the filters matched before `--limit`.

### One rule tells a record slug from a change slug

A slug that starts with a date names a record, as in `dkt lint` and as
`change.ValidateSlug` already refuses dated change slugs. `dkt status` and
`dkt sync` route by it; a dated name is checked with `record.SlugOf`, an
undated one with `change.ValidateSlug`.

### Matching trailers to tasks

`change.NamedTasks` keeps its signature; the matching of `Task:` values to
a plan's tasks moves into a function that takes commits already chosen,
which `NamedTasks` calls after choosing by `Change:` and the record code
calls after choosing by `Record:`. The move is a structural commit of its
own inside this change: it changes no behavior, serves only this change,
and is a few lines (my call not to split it into a pull request of its
own).

`gitx.Commit` gains `Record`, read with `%(trailers:key=Record,...)` like
`Change`.

### Record sync

Record sync lives beside `archive.Sync`, with the same tick writer and
`safepath.Check`. It takes no lock (my call): the ticks come from commits,
so two runs write the same bytes.

### Lint reads the branch once

`dkt lint` resolves the default branch and merge base once. The added
routed files, the added records and the branch's commits (for
`record-trailer`) all come from that one base; when it cannot be found,
one unreadable entry says why, where there were two. `record-trailer`
checks that `docket/records/<value>.md` exists in the worktree, which also
catches the bare name an agent may write for the slug.

### Small fixes

- `record.Words` counts `ー` (U+30FC) and its halfwidth form `ｰ` (U+FF70)
  as Katakana; both are in Unicode's Common script, which is why they were
  missed (the halfwidth form is my call).
- The scan settles a worktree's ancestry before it files its records
  error, so a worktree unreadable as a whole is listed once.
- `record.Create` notes which of `docket` and `docket/records` it made, and
  on a failed write removes the file and then those folders; the error
  names any it could not remove.

## Risks / Trade-offs

- [A rebase-merged record's merge time is its own commit's, which may
  predate the merge] → documented; squash and merge commits give the
  merge's time.
- [A merged record edited again on a branch shows only with `--merged`]
  → rare; the edit still shows in that branch's own `dkt lint`.
- [A project without release tags gets `tag` null everywhere] →
  `--since <rev>` works with any commit.
- [The first-parent log walks the whole default history] → one
  path-limited process, run only when merged records are asked for.

## Migration Plan

Nothing migrates: the three merged records need no new key, and the JSON
only gains fields (`result.change` is null only for a record). Rollback is
reverting the change.
