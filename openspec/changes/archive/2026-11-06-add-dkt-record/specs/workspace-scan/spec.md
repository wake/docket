# Spec Delta

## ADDED Requirements

### Requirement: Record files found
A scan for records SHALL read the sources a scan for changes reads. In each, every entry `docket/records/<slug>.md` that is not a directory SHALL count as a use of `<slug>`, reported with its source and path. A source without `docket/records/` holds no use. A worktree whose `docket/records` path passes through a symbolic link, and a source that cannot be read, SHALL be reported as unreadable with the reason.

#### Scenario: Record only in another worktree
- **WHEN** worktree B holds an uncommitted `docket/records/2026-11-06-fix-x.md` and a scan for records runs from worktree A
- **THEN** the scan reports `2026-11-06-fix-x` with worktree B's path as its source

#### Scenario: Record only on a branch
- **WHEN** `refs/remotes/origin/fix-y` holds `docket/records/2026-11-06-fix-y.md` and no worktree or local branch does
- **THEN** the scan reports `2026-11-06-fix-y` with `refs/remotes/origin/fix-y` as its source

#### Scenario: Project without records
- **WHEN** a branch has no `docket/` folder
- **THEN** the scan reports no record from that branch and does not list it as unreadable

#### Scenario: Linked records folder in another worktree
- **WHEN** worktree B's `docket/records` is a symbolic link to a directory outside it
- **THEN** the scan lists worktree B as unreadable, naming the link, and reports no record from what the link points to
