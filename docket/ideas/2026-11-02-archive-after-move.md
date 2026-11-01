---
title: Recover from a failure after the archive move
---

Recorded 2026-11-02, from two codex reviews: the review of
`adopt-existing-projects` (a high finding, ruled out of scope since `main`
had the same order) and an earlier review that found the same in the spec
title step.

## Problem

`dkt archive` runs `openspec archive` before the spec title step and the
evidence step (`internal/archive/run.go:219-246`). When either fails, the
change has already moved to `openspec/changes/archive/` and OpenSpec may
have merged its deltas into the main specs:

- The spec title step writes each capability's `docket.yaml` in turn and
  stops at the first failure, so earlier capabilities keep the new title.
- `evidence.md` is not written.
- The active change folder no longer exists, so running
  `dkt archive <slug>` again cannot finish the run.

The spec says only that the run stops at the failing step. Nothing says what
the user gets back, so this is an idea, not an issue.

Today the user can recover by hand: `dkt archive` refuses unless the working
tree is clean and commits nothing, so restoring the working tree with git
returns it to the state before the run.

## Directions, not decided

- Check before the move what the later steps need: every delta title, every
  target `docket.yaml`, and that `evidence.md` can be written.
- On a failure after the move, restore what the run changed, as `dkt issue`
  removes a partial file.
- Let a second run finish an archived change that has no `evidence.md`.
- Say in the failure how to restore the working tree, and change nothing
  else.

## Open questions

- Whether restoring by hand with git is enough, given that the tree was
  clean before the run.
