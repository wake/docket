# Proposal

## Why

When two changes modify the same requirement, OpenSpec 1.14 lets the one
archived second replace the first one's text without a warning (reproduced
in a sandbox, and described in OpenSpec's own parallel-merge plan). Teams
that develop several changes at once in separate worktrees find out only at
archive time, when the work is already done. `dkt conflicts` shows the
overlap while there is still time to choose: wait, or write the change so
that it does not touch the shared requirement.

## What Changes

- `dkt conflicts [<slug>]`: lists every requirement that more than one
  active change touches, across all worktrees, unmerged local and
  remote-tracking branches, and the default branch tip.
- With a slug, only the overlaps that involve that change, to answer "can I
  start this one in parallel with what is in progress?"

## Non-Goals

- No conflicts in code; git handles those.
- No record of which change depends on which. Deciding what can run in
  parallel stays with the people planning the work.
- Does not block merges. The blocking check is the base comparison in
  `dkt archive`.
- Does not resolve overlaps or edit any file.
- Does not fetch; remote-tracking branches are read as last fetched.

## Builds On

- From `add-init-and-new`: the scan of worktrees and branches and the
  reading of git objects (`workspace-scan`).
- From `add-lint`: the delta reader. This change adds no parser of its own.

## Capabilities

### New Capabilities

- `conflict-detection`: which active changes touch the same requirement,
  and how `dkt conflicts` reports it.

### Modified Capabilities

None.

## Impact

- Uses the scan from `add-init-and-new` and the delta reader from
  `add-lint`; reads other branches' deltas from git objects.
- Needs no OpenSpec CLI at run time.

## Implementation Contract

### Observable Behavior

- `dkt conflicts` prints one entry per overlapping requirement, naming the
  capability, the requirement, and for each change its operation and where
  it was found. It exits 1 when there is any overlap and 0 when there is
  none.

### Interfaces

- `dkt conflicts [<slug>] [--json]`
- JSON `result.overlaps[]`: `capability`, `requirement`, and `changes[]`
  with `slug`, `operation`, `sources[]` (each a worktree path or ref, with
  its commit).

### Failure Modes

- A source that cannot be read: reported as unreadable, exit 3; overlaps
  found elsewhere are still listed.
- A named slug that is not an active change: a finding, exit 1.
- A delta that cannot be read: an `unparseable` finding naming the change
  and source.

### Acceptance Criteria

- `conflict-detection#both-modify-one-requirement`,
  `conflict-detection#rename-against-modify`,
  `conflict-detection#same-capability-different-requirements`,
  `conflict-detection#both-add-the-same-name`,
  `conflict-detection#overlap-with-another-worktree`,
  `conflict-detection#unrelated-overlap-hidden`,
  `conflict-detection#slug-not-active`,
  `conflict-detection#sources-listed`,
  `conflict-detection#no-overlaps`,
  `conflict-detection#unreadable-worktree`.

## Change Log

| Date | Source | Change | Impact |
|---|---|---|---|
