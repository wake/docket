---
title: A web workbench in Docket (dkt serve)
---

Recorded 2026-10-24.

## Problem

Changes in progress live in many worktrees, on branches and on other
machines. Reading or editing a proposal, a delta or a plan means finding the
right worktree directory by hand. Teammates, and people who do not use a
terminal, cannot do it at all.

## Direction chosen

Docket serves a web UI from the same binary: `dkt serve`, which can also run
as a long-lived daemon (a launchd or systemd unit). Docket's scan and readers
are the engine, so the UI sees exactly what the CLI sees: every change, from
every worktree, local branch and remote-tracking branch, with its stage and
task progress.

First thoughts on scope, all to be settled in brainstorming:

- List all changes and where each one lives; open any document rendered
  (tables, Mermaid) or as source.
- Edit documents in worktrees, with a compare-before-write check so an edit
  never overwrites what an agent changed a moment earlier. Changes that only
  exist on a branch are read-only.
- Run Docket commands (lint, conflicts, sync, archive) from the UI and show
  their findings.
- Identity comes from a proxy in front (an access gateway or a private
  network that passes the user's identity); Docket only distinguishes
  readers from writers.
- Optionally other markdown in the repository (`docs/`), not only
  `openspec/`.

## Alternatives considered and set aside

Surveyed 2026-10-24; details are in the maintainer's research notes.

- Spectra 3.x (closed source): a viewer and progress panel for the spec
  directory only; does not see manually created worktrees; single local user.
- openspecui (MIT): edits with conflict detection and follows
  `apply.tracks`, but serves one project root, is tied to each OpenSpec
  minor version, and has no users. A fork would have to track it.
- spek (MIT): the only viewer that aggregates worktrees, but read-only and
  hardcoded to `tasks.md`.
- SilverBullet (MIT): mature multi-user markdown editing, but knows nothing
  of OpenSpec, needs one space per worktree, and lets every writer run
  scripts.
- Building it into an agent session manager: possible for one developer,
  but such tools usually have a single admin credential, which rules out
  "everyone".

Ideas worth borrowing: openspecui's compare-and-write file writer, spek's
handling of the same change diverging across worktrees.

## Open questions

- How far editing goes in the first version: documents only, or also
  creating changes and ticking tasks.
- Work that is not pushed on another machine: one server per machine with
  an aggregating view, or pushed branches only.
- Whether the UI ships in this repository or as a separate one.
