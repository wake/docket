## ADDED Requirements

### Requirement: Records found
A scan SHALL find records in the sources it reads for changes: every regular file `docket/records/<slug>.md` whose slug starts with a date. A record SHALL be identified by its slug, reported once with every source that holds it, and read from the first source in the order `One change per slug` defines. A missing `docket/records` SHALL hold no record and SHALL NOT make the source unreadable.

#### Scenario: Record on a branch
- **WHEN** `refs/heads/fix-x` holds `docket/records/2026-11-01-fix-x.md`, worktree B on that branch holds it too, and a scan runs from worktree A
- **THEN** the scan reports `2026-11-01-fix-x` once, with worktree B first and the branch second

## MODIFIED Requirements

### Requirement: Unreadable sources
A worktree that git lists but whose directory is missing or cannot be read, and a branch whose tree cannot be read, SHALL be reported as unreadable with the reason, and the command SHALL exit as `cli-output` requires for unreadable sources. A source unreadable as a whole SHALL be listed once, without a separate entry for a folder in it.

#### Scenario: Worktree on an unmounted volume
- **WHEN** a listed worktree's directory is missing because its volume is not mounted
- **THEN** the scan lists that worktree as unreadable and the command exits with 3

#### Scenario: Worktree listed once
- **WHEN** a worktree's `docket/records` cannot be listed and git cannot tell whether the default branch tip contains the worktree's HEAD
- **THEN** the scan lists that worktree as unreadable once, and the command exits with 3
