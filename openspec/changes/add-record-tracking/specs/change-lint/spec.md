## MODIFIED Requirements

### Requirement: Routing
lint SHALL report every file under `docs/superpowers/specs/` or `docs/superpowers/plans/` that the current branch adds after its merge base with the default branch tip, whether committed or present uncommitted in the worktree, because designs and plans belong in the change folder. When no default branch or merge base can be found, lint SHALL list the branch's added files as unreadable once, for routing, records and `Record` trailers alike, and still run its other rules.

#### Scenario: Design written to the superpowers folder
- **WHEN** the worktree has a new untracked file `docs/superpowers/specs/2026-10-25-x-design.md`
- **THEN** lint reports a `routing` finding naming that file

#### Scenario: No default branch for routing
- **WHEN** the repository has no default branch Docket can find and a change has a `proposal-section` problem
- **THEN** lint reports the `proposal-section` finding, lists the branch's added files as unreadable, and exits with 3

#### Scenario: One unreadable entry
- **WHEN** the repository has no default branch Docket can find and a user runs `dkt lint`
- **THEN** lint lists exactly one unreadable entry
