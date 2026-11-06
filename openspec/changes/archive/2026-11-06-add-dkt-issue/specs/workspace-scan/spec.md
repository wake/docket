# Spec Delta

## ADDED Requirements

### Requirement: Issue files found
A scan for issues SHALL read the sources a scan for records reads and find the uses of `docket/issues/<slug>.md` in each as it finds the uses of `docket/records/<slug>.md`, reporting a linked `docket/issues` in a worktree, and a source that cannot be read, as unreadable.

#### Scenario: Issue only on a branch
- **WHEN** `refs/remotes/origin/fix-y` holds `docket/issues/2026-11-06-lint-timeout.md` and no worktree or local branch does
- **THEN** the scan for issues reports `2026-11-06-lint-timeout` with `refs/remotes/origin/fix-y` as its source

#### Scenario: Records are not issues
- **WHEN** a branch holds `docket/records/2026-11-06-fix-x.md` and no issue file
- **THEN** the scan for issues reports no use from that branch
