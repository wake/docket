# Spec Delta

## Purpose

Defines where Docket looks for changes and how it reports them, so that a
change in progress in another worktree, on another branch or on another
machine is seen, and a place Docket could not read is never taken for an
empty one.

## ADDED Requirements

### Requirement: Sources scanned
A scan SHALL find changes in every worktree that `git worktree list` reports, reading its files including uncommitted and untracked ones; in every local branch and every remote-tracking branch whose tip the default branch tip does not contain; and in the default branch tip, including its archive.

#### Scenario: Change only in another worktree
- **WHEN** worktree B holds an uncommitted `openspec/changes/add-x/` and a scan runs from worktree A
- **THEN** the scan reports `add-x` with worktree B as its source

#### Scenario: Change only on a remote-tracking branch
- **WHEN** `refs/remotes/origin/feature-y` holds `openspec/changes/add-y/` and no worktree or local branch does
- **THEN** the scan reports `add-y` with `refs/remotes/origin/feature-y` as its source

#### Scenario: Merged branch skipped
- **WHEN** a local branch's tip is contained in the default branch tip
- **THEN** the scan does not read that branch as a separate source

### Requirement: Default branch
Docket SHALL take the default branch from `--default-branch` when given; otherwise from the target of `refs/remotes/origin/HEAD`; otherwise `main` if it exists, then `master`. Its tip SHALL be the remote-tracking ref of that branch when one exists, and the local branch otherwise. When no default branch can be found, the command SHALL exit with 4.

#### Scenario: origin HEAD set
- **WHEN** `refs/remotes/origin/HEAD` points to `refs/remotes/origin/trunk` and no flag is given
- **THEN** Docket uses `trunk` as the default branch and `refs/remotes/origin/trunk` as its tip

#### Scenario: No default branch
- **WHEN** no flag is given, `refs/remotes/origin/HEAD` is unset, and neither `main` nor `master` exists
- **THEN** the command exits with 4 and suggests `--default-branch`

### Requirement: One change per slug
A change SHALL be identified by its slug. When the same slug appears in several sources, Docket SHALL report it once with every source listed, and SHALL use the content from the first source in this order: a worktree whose HEAD the default branch tip does not contain, a local branch, a remote-tracking branch, a worktree whose HEAD the default branch tip contains, the default branch tip.

#### Scenario: Merged change also on a branch
- **WHEN** `add-x` exists on the default branch tip, in the main worktree checked out on the default branch, and on an unmerged branch that has edited it
- **THEN** the scan reports `add-x` once, lists all three sources, and uses the branch's content

#### Scenario: Worktree with unmerged work comes first
- **WHEN** worktree B, on a branch with commits the default branch tip does not contain, has uncommitted edits to `add-x`, which also exists on B's branch and on the default branch tip
- **THEN** the scan uses worktree B's content

### Requirement: Archived slugs
Every directory `openspec/changes/archive/<YYYY-MM-DD>-<slug>/` in a scanned source SHALL count as an archived use of `<slug>`.

#### Scenario: Archived slug found
- **WHEN** the default branch tip holds `openspec/changes/archive/2026-10-24-add-x/`
- **THEN** the scan reports `add-x` as archived

### Requirement: Unreadable sources
A worktree that git lists but whose directory is missing or cannot be read, and a branch whose tree cannot be read, SHALL be reported as unreadable with the reason, and the command SHALL exit as `cli-output` requires for unreadable sources.

#### Scenario: Worktree on an unmounted volume
- **WHEN** a listed worktree's directory is missing because its volume is not mounted
- **THEN** the scan lists that worktree as unreadable and the command exits with 3

### Requirement: No network
A scan SHALL NOT fetch from or otherwise contact a remote. Remote-tracking branches SHALL be read as they were at the last fetch.

#### Scenario: Scan offline
- **WHEN** the network is unavailable and a scan runs
- **THEN** the scan completes from the existing remote-tracking refs
