---
title: Ideas with state, and an order file
needs:
  - docket/ideas/2026-10-29-records.md
---

Recorded 2026-10-29. The points below were decided with the maintainer; the
change that takes this idea turns them into a proposal and deltas.

## Problem

Ideas lived in follow-up notes holding many items each, and the order of
upcoming work was a list inside one idea. Nothing said which idea became
which change, which were queued and which were given up, so the state had to
be read by hand.

## Decided

- One idea per file: `docket/ideas/<date>-<slug>.md`. Its front matter holds
  only what people decide: `title`; `to: change:<slug>` or
  `to: record:<slug>` once work starts; `dropped: true` for an idea that
  will not be done; `needs:` for the ideas it depends on. The body says what
  the idea is and why.
- `docket/order.md` holds the order of upcoming work and nothing else: the
  heading `# Order`, then a numbered list from 1 with no gaps, each line a
  checkbox and one repository-root path in backticks to an idea. No idea
  titles, versions, dependencies or notes.
- The first unticked line is the next piece of work. `dkt sync` ticks a
  line when the change or record its idea points to merges, and
  `dkt sync --check` reports a line out of step. Finished lines stay.
- `dkt lint` checks each line's format, that the path exists under
  `docket/ideas/`, that numbers run from 1 with no gaps, no duplicates, and
  no dropped idea in the order.
- State is derived, not written: pool (not in the order, no `to`), queued
  (in the order, no `to`), in progress (its `to` exists), done (merged or
  archived), dropped. The version that shipped an idea comes from
  `git tag --contains`.

## Parallel work

Recorded 2026-10-31, from a discussion with the maintainer. This is a
proposal, not decided; it would replace two of the decided points above, to
be settled when this idea is taken up.

Two decided points have each change's worktree write files that every
change shares:

- `dkt sync` ticks the order line. Sync runs in the change's worktree
  before its pull request opens, so two changes made in parallel edit
  adjacent lines of `docket/order.md`, which git reports as a conflict, and
  a line is ticked before its pull request merges.
- `to:` is written when work starts, in that worktree. An idea that becomes
  several pieces of work, as records did (`add-records`, then the change
  that reads records), has no single value to write.

A plan has neither problem, since its ticks live in the change's own folder.
The proposal applies the same rule to ideas:

- A worktree writes only the files of its own work: its change folder or
  its record. `docket/order.md` and the idea files change only by the
  maintainer's decisions, written from one place.
- `docket/order.md` holds the order. Nothing ticks it; `dkt` derives
  whether each line is done, from the change or record whose `source`
  names the idea having merged or been archived. The next piece of work is
  the first line not done, reported by a command (the open question below
  on which command reports the state of ideas). AGENTS.md's "the first
  unticked line" changes with it.
- `to:` is derived from the `source` of changes and records, not written.
- One idea is one piece of work. When an idea has a further part, that
  part becomes an idea of its own, whose `needs:` names the first.

With no worktree writing them, ideas and the order have one writer. As a
trial from 2026-10-31, edits to ideas in this repository are pushed straight
to the default branch, without a pull request.

## Open questions

- Which command reports the state of ideas.
- The exact form of `needs:`.
- How ideas are classified, so that the order reads as a priority list.
  Recorded 2026-10-30. The maintainer finds the queue mixes the user's
  requests, fixes found during development and optimizations. Two
  proposed axes, to settle when this idea is taken up:
  - origin: the user's request, found during development (a review
    finding, a trial, a bug hit while building), maintenance;
  - nature: new capability, fix, optimization, docs.
  A `domain:` field is designed with them
  (`2026-10-30-spec-domains.md`). `dkt` lists and groups by these fields;
  the order itself stays the maintainer's decision.
