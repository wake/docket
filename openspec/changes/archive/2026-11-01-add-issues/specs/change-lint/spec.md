## ADDED Requirements

### Requirement: Issues linted
When no name is given, `dkt lint` SHALL also lint every file issue under `docket/issues/` that the branch adds, committed since the merge base with the default branch or not yet committed, reading it from the worktree as it is. When the branch's added files cannot be read, the one unreadable entry `Routing` lists SHALL stand for the issues too. A name given on the command line SHALL NOT name an issue.

#### Scenario: Lint issues the branch adds
- **WHEN** the default branch holds the file issue `2026-10-01-a`, the branch adds the file issue `2026-11-02-b`, and a user runs `dkt lint`
- **THEN** `2026-11-02-b` is linted and `2026-10-01-a` is not

#### Scenario: Issues without a default branch
- **WHEN** the repository has no default branch Docket can find and the worktree holds an uncommitted file issue
- **THEN** lint lists exactly one unreadable entry and reports no finding for the issue
