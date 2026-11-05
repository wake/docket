# Proposal

## Why

Two of Docket's rules are about what reaches the default branch, and nothing checks them yet:

- Designs and plans go into the change folder, not superpowers' default `docs/superpowers/specs/` and `docs/superpowers/plans/`.
- Pull requests are not squashed, because `dkt sync` reads each commit's `Change:` trailer and a squash merge drops it.

`dkt lint` and `dkt inspect` left these for this change, which settles what the default branch is. `dkt inspect` also still lacks the main spec title check that `dkt lint` now has. This is the last change before this repository switches to Docket's own workflow.

## What Changes

- dkt finds the default branch: the branch `origin/HEAD` points to, else `main`, else `master`. It takes the branch's tip from `origin` when that branch has been fetched, else from the local branch.
- `dkt lint` reports `routing` for each file the current branch adds under `docs/superpowers/specs/` or `docs/superpowers/plans/`, compared with where it left the default branch. Uncommitted and untracked files count. Its result names the default branch.
- `dkt inspect` reports `trailer-missing` for each active `docket` change on the default branch whose plan has a task but whose name no commit on that branch carries in a `Change:` trailer. That is what a squash merge leaves. Its result names the default branch.
- `dkt inspect` reports `title-missing` for each main spec without a title, as `dkt lint` does.
- When no default branch can be found, both checks are skipped. The result says so, and dkt does not fail.

## Non-Goals

- Choosing the default branch by a flag or a setting: the three names cover the projects Docket targets, and a later change can add one.
- Other folders a coding agent might write designs into: only superpowers' two defaults are known.
- Changes made with another schema on the default branch: docket-doctor's change reports them.
- Fixing anything: both commands only report.

## Capabilities

### New Capabilities
- `default-branch`: how dkt decides which branch is the default branch and which commit is its tip.

### Modified Capabilities
- `change-lint`: the `routing` rule, and `default_branch` in the result.
- `setup-inspection`: the trailer check on the default branch, the main spec title check, and `default_branch` in the result.

## Impact

- `internal/gitx` gains the git calls needed: the target of `origin/HEAD`, a ref's commit, the merge base, a file from a commit, the files a branch adds, and the trailers of a given commit's history.
- `internal/lint` gains the routing rule and shares its main spec title check with `dkt inspect`.
- `dkt lint` and `dkt inspect` gain `default_branch` in their results.
- Dependencies: still only the Go standard library, the system `git` and the OpenSpec CLI.
