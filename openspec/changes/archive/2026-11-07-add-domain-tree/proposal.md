---
title: "Add a domain tree for viewing specs by area"
kind: feature
source: "Maintainer request: classify specs in a tree for viewing, apart from where they are stored"
specs:
  domain-tree: Domain Tree
---

# Proposal

## Why

A project can only browse its specs and changes by path. OpenSpec does not allow a change folder to be nested. A spec can be nested, but then its path decides both where it is stored and how it is classified: reclassifying it means moving the file, and a change still holding only ADDED requirements on the old path brings the old spec back as a second copy.

How a project is viewed should not depend on how its files are stored. A tree kept apart from the specs lets a project lay out its areas first, even before anything is in them, name them in any language, and reclassify a spec by editing one line.

## What Changes

- New optional file `docket/domains.md`: a nested list of nodes, with capability paths under them. A node may be empty, and its name may be in any language.
- New read-only command `dkt domains`: reports the tree, with each capability's title, whether its main spec exists and the active changes touching it, and lists the capabilities no node holds as unclassified.
- `dkt lint`, when no change is named, reports problems in the tree: `domain-format`, `domain-duplicate` and `domain-unknown`.
- The docket skill says when a coding agent places a new capability in the tree, that it updates or deletes the line when a capability's path changes or the capability is removed, that it may suggest a new node, and that it adds, renames or moves a node only once the user agrees. Editing only the tree is a direct merge, when `dkt domains` reports no problem.
- The README lists the command.

## Non-Goals

- Changing where specs are stored, or moving any spec. The tree says nothing about paths on disk.
- Front matter in main specs: OpenSpec does not carry a delta's front matter into the main spec, and `dkt lint` reads a main spec's title from its first line. See `design.md`.
- A `domain` field on a change: a change can touch capabilities in several nodes, so where it belongs follows from the capabilities it touches.
- Requiring a tree, or requiring every capability to be in it: a capability no node holds is reported, not a finding.
- `dkt` writing the tree, and a template or `dkt init` step for it: the file is edited by hand, or by a coding agent the user asked.
- A capability under more than one node.
- Changes in other worktrees and branches, and archived changes: `dkt domains` reads the worktree it runs in.
- `dkt lint <slug>` checking the tree, as it does not check main spec titles either.
- A user interface: `dkt domains --json` is the data a viewer reads.

## Capabilities

### New Capabilities
- `domain-tree`: what `docket/domains.md` holds and how `dkt domains` reports it.

### Modified Capabilities
- `change-lint`: `dkt lint` with no change named also lints the domain tree.
- `cli-output`: `dkt domains` is a command that needs a repository.

## Impact

- New: `internal/domain`, `internal/cli/cmd_domains.go`.
- Changed: `internal/cli/command.go` (the command list), `internal/lint/lint.go`, `shipped/skill/SKILL.md` and its installed copies, `README.md`.
- A project without `docket/domains.md` is unaffected: `dkt lint` reports nothing about it, and `dkt domains` lists every capability as unclassified.
- No new dependency, and OpenSpec is not needed to run the command.
