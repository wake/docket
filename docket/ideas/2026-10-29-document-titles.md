---
title: Titles for changes and specs in docket.yaml
---

Recorded 2026-10-29. The points below were decided with the maintainer; the
change that takes this idea turns them into a proposal and deltas.

## Problem

A change and a main spec have no title a reader can use: the slug is the
only name. OpenSpec ignores the H1 of a main spec, writes `# <id>
Specification` for a new one, and `openspec show` reports the id as the
title. `dkt status` and a workbench need a title for each.

## Decided

- A change's title is `title:` in `openspec/changes/<slug>/docket.yaml`.
  `dkt new` requires `--title`.
- A main spec's title is `title:` in
  `openspec/specs/<capability>/docket.yaml`. A change that adds a spec puts
  the title in `openspec/changes/<slug>/specs/<capability>/docket.yaml`, and
  `dkt archive` moves it next to the main spec, since OpenSpec does not.
- Documents inside a change (proposal, design, plan, deltas, evidence) get
  no titles of their own; they are named by role under the change's title.
- `x-kind`, `x-source` and `x-base` move from `.openspec.yaml` into the
  change's `docket.yaml` in the same change. Archived changes stay as they
  are, and readers accept both places.
- Titles are required: `dkt lint` reports an active change, a delta that
  adds a spec, and a main spec without one. The proposal drafts titles for
  the existing main specs.
- `dkt status --json` gives each change's title.
- Docket keeps its data in its own files, so OpenSpec's files stay as
  OpenSpec writes them. Checked with OpenSpec 1.14.3: a `docket.yaml` at a
  change root or in a spec folder leaves `validate`, `list`, `show`,
  `status`, `instructions` and `archive` working, and one at a change root
  moves with the change on archive.
