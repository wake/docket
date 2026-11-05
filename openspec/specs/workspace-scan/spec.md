# Workspace Scan Specification

## Purpose
Defines where Docket looks for changes, so that a change in progress in another worktree, on another branch or on another machine is seen, and a place Docket could not read is never taken for an empty one.

## Requirements

### Requirement: Sources scanned
A scan SHALL read every worktree that `git worktree list` reports, from its files on disk, uncommitted and untracked ones included, and the committed tree of every local branch and every remote-tracking branch. It SHALL skip a bare repository's entry and symbolic refs.

#### Scenario: Change only in another worktree
- **WHEN** worktree B holds an uncommitted, untracked `openspec/changes/add-x/` and a scan runs from worktree A
- **THEN** the scan reports `add-x` with worktree B's path as its source

#### Scenario: Change only on a remote-tracking branch
- **WHEN** `refs/remotes/origin/feature-y` holds `openspec/changes/add-y/` and no worktree or local branch does
- **THEN** the scan reports `add-y` with `refs/remotes/origin/feature-y` as its source

### Requirement: Slugs found
In each source, every directory `openspec/changes/<slug>/` other than `archive` SHALL count as an active use of `<slug>`, and every directory `openspec/changes/archive/<YYYY-MM-DD>-<slug>/` as an archived use of `<slug>`. Each use SHALL be reported with its source and the folder's path. A source without `openspec/changes/` holds no use.

#### Scenario: Archived slug found
- **WHEN** a branch holds `openspec/changes/archive/2026-10-24-add-x/`
- **THEN** the scan reports an archived use of `add-x` with that branch and that folder

#### Scenario: Project without changes
- **WHEN** a branch has no `openspec/` folder
- **THEN** the scan reports no use from that branch and does not list it as unreadable

### Requirement: Unreadable sources
A worktree that git lists but whose directory is missing or cannot be read, a worktree whose `openspec/changes` path passes through a symbolic link, and a branch whose tree cannot be read, SHALL be reported as unreadable with the reason, and the command SHALL exit as `cli-output` requires for unreadable sources.

#### Scenario: Worktree on an unmounted volume
- **WHEN** a listed worktree's directory is missing because its volume is not mounted
- **THEN** the scan lists that worktree as unreadable, with a reason that names `git worktree prune` for a worktree that was deleted, and the command exits with 3

#### Scenario: Linked changes folder in another worktree
- **WHEN** worktree B's `openspec/changes` is a symbolic link to a directory outside it
- **THEN** the scan lists worktree B as unreadable, naming the link, and reports no use from what the link points to

### Requirement: No network
A scan SHALL NOT fetch from or otherwise contact a remote. Remote-tracking branches SHALL be read as they were at the last fetch.

#### Scenario: Scan offline
- **WHEN** the network is unavailable and a scan runs
- **THEN** the scan completes from the existing remote-tracking refs
