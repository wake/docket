---
title: "Report capabilities missing from the domain tree, and suggest starting one"
kind: feature
source: "docket/ideas/domain-tree-upkeep.md"
---

# Proposal

## Why

The docket skill covers the domain tree only once `docket/domains.md` exists, so in a project without one, specs pile up as a flat list: in one project 27 capabilities built up before the user asked for a tree by hand. And with the file present, a capability missing from it is only listed as unclassified by `dkt domains`; nothing fails, so a change can merge with its new capability left out. The step in the change should be checked, not remembered, and the agent should know when to suggest starting a tree.

## What Changes

- `dkt lint` reports `domain-unlisted`, naming `docket/domains.md`, for each capability `dkt domains` reports as unclassified, when the tree lists at least one item. With no change named, it reports every such capability; with changes named, those with a delta in a named change. The message names the capability and says to list it under a node in `docket/domains.md` and run `dkt domains`. A capability that a change retires, its delta removing every requirement of the main spec and nothing else, is not reported, since the skill has the change drop its line. Like every lint finding, it stops archiving, so a project with a tree that left out a capability sees it on its first lint after upgrading.
- A `docket/domains.md` with no items records that the project keeps no tree: `dkt lint` reports no `domain-unlisted` for it, and the skill does not suggest starting one.
- The docket skill: when a change adds a capability to a project that has no `docket/domains.md` and ten or more main specs, the agent suggests starting the tree, as `domain-tree.md` in the skill's folder says. It writes the tree only once the user agrees; when the user declines and agrees to record it, it writes the file with no items.
- `dkt domains` reports unclassified capabilities as before, with no finding.

## Non-Goals

- No new setting or configuration file: the record of a declined tree is the tree file itself.
- No suggestion based on how many areas the specs span: agents cannot judge that the same way across projects, and a count can be checked.
- No `domain-unlisted` for format problems, duplicates or unreadable sources: those findings come first, and what is unlisted is only known once the tree and the capabilities are read.
- No change to `dkt snapshot`, which reports the tree as it does today.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `change-lint`: a new requirement, Capability missing from the domain tree, adds the `domain-unlisted` rule.
- `domain-tree`: Unclassified capabilities says that `dkt domains` reports no finding for them while `dkt lint` does; a new requirement, Tree with no items, says what such a file records.

## Impact

- `internal/domain` gains the list of capabilities no node holds, which `dkt domains` and `dkt lint` share.
- `internal/lint` reports `domain-unlisted`, with or without changes named.
- `shipped/skill/SKILL.md` and `shipped/skill/domain-tree.md`, their installed copies, and `internal/setup/history.go`.
- Tests in `internal/domain`, `internal/lint` and `internal/cli`.
