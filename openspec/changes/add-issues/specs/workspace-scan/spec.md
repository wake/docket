## ADDED Requirements

### Requirement: Issues found
A scan SHALL find file issues in the sources it reads for changes: every regular file `docket/issues/<slug>.md` whose slug starts with a date. An issue SHALL be identified by its slug, reported once with every source that holds it, and read from the first source in the order `One change per slug` defines. A missing `docket/issues` SHALL hold no issue and SHALL NOT make the source unreadable.

#### Scenario: Issue on a branch
- **WHEN** `refs/heads/fix-x` holds `docket/issues/2026-11-02-x.md`, worktree B on that branch holds it too, and a scan runs from worktree A
- **THEN** the scan reports `2026-11-02-x` once, with worktree B first and the branch second

#### Scenario: No issues folder
- **WHEN** a source has no `docket/issues` folder
- **THEN** the scan reports no issue for it and the source is readable
