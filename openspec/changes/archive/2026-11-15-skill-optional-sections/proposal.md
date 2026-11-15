---
title: "The docket skill keeps the sections for optional features in files of their own"
kind: change-request
source: "docket/ideas/skill-optional-sections.md"
---

## Why

The docket skill is read at the start of every piece of development work. Its sections "Issues in another tool" and "Domain tree" apply only to a project whose `AGENTS.md` has a `## Issues` section, or that has `docket/domains.md`; together they are about a fifth of the skill, and every other project reads them each time.

## What Changes

- The text of each section moves, word for word, into a file of its own next to `SKILL.md`: `issues-in-another-tool.md` and `domain-tree.md`. Each heading stays in `SKILL.md` with one line saying when to read the file, so the skill's own reference, see "Issues in another tool", still leads there.
- `dkt init` writes every file of the shipped skill folder, not only `SKILL.md`, to both `.claude/skills/docket/` and `.agents/skills/docket/`, and `dkt inspect` checks each of them. A project set up by an earlier release gets `skill-outdated` for `SKILL.md` and `skill-missing` for each new file, and `dkt init` updates and creates them.
- The shipped hashes record the new files and the new `SKILL.md`.

## Non-Goals

- No rewording of the moved text; only the two pointer lines are new.
- No change to how a file the user edited is kept, or to the next steps `dkt init` prints: the reload step still follows only a created `SKILL.md`.
- No removal of a skill file that a later release stops shipping; nothing ships fewer files yet.
- No change to the plugin's own skills under `skills/`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `project-setup`: "Files init writes" names every file of the skill folder, and an earlier release's skill folder is brought up to date.
- `setup-inspection`: "Installed file findings" reports an earlier release's skill folder.

## Impact

- `internal/setup/setup.go` and its tests; `internal/cli` tests that list installed files.
- `shipped/skill/`: `SKILL.md` and the two new files, with their installed copies in this repository and their hashes in `internal/setup/history.go`.
