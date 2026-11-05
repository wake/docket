# Default Branch Specification

## Purpose
Defines which branch dkt treats as the default branch, the one changes are merged into, and which commit is its tip, so that every check about what reaches that branch looks at the same commit.

## Requirements

### Requirement: Default branch name
dkt SHALL take as the default branch the branch that `refs/remotes/origin/HEAD` points to, when it points to a ref under `refs/remotes/origin/` and that branch exists on `origin` or locally. Otherwise it SHALL take `main`, and then `master`, the first that exists on `origin` or locally. Otherwise there SHALL be no default branch.

#### Scenario: origin/HEAD names the branch
- **WHEN** `refs/remotes/origin/HEAD` points to `refs/remotes/origin/trunk`, which exists, and a local `main` also exists
- **THEN** the default branch is `trunk`

#### Scenario: No remote
- **WHEN** the repository has no remote and has the local branches `main` and `master`
- **THEN** the default branch is `main`

#### Scenario: origin/HEAD points nowhere
- **WHEN** `refs/remotes/origin/HEAD` points to `refs/remotes/origin/gone`, which exists neither on `origin` nor locally, and only a local `master` exists
- **THEN** the default branch is `master`

#### Scenario: No default branch
- **WHEN** there is no `refs/remotes/origin/HEAD` and no branch named `main` or `master`, on `origin` or locally
- **THEN** there is no default branch

### Requirement: Default branch tip
The tip of the default branch `<name>` SHALL be the commit of `refs/remotes/origin/<name>` when that ref exists, and otherwise the commit of `refs/heads/<name>`.

#### Scenario: Remote-tracking ref first
- **WHEN** `refs/remotes/origin/main` and `refs/heads/main` both exist and point to different commits
- **THEN** the tip is the commit of `refs/remotes/origin/main`

#### Scenario: Local branch only
- **WHEN** only `refs/heads/main` exists
- **THEN** the tip is the commit of `refs/heads/main`
