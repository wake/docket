## MODIFIED Requirements

### Requirement: Reuse table
plan.md SHALL have a `## Reuse` section with a table whose columns are Need, Existing (path:symbol), Decision and Reason; the table MAY have no rows. Decision SHALL be `extend` or `new`; for `extend`, the path before the first `:` of Existing SHALL exist in the worktree unless Existing starts with `external:`; for `new`, Reason SHALL not be empty. A break SHALL be reported as `reuse-table`.

#### Scenario: Reuse row with a missing path
- **WHEN** an `extend` row names `internal/gitx/run.go:Run` and that file does not exist
- **THEN** lint reports `reuse-table` on that row

#### Scenario: External dependency row
- **WHEN** an `extend` row's Existing is `external: git CLI`
- **THEN** lint reports nothing about that row

#### Scenario: Empty reuse table
- **WHEN** the Reuse table has its header and no rows
- **THEN** lint reports nothing about the Reuse section

#### Scenario: Reuse decision in the table
- **WHEN** a row of the four-column table has the Decision `reuse`
- **THEN** lint reports `reuse-table` on that row

## ADDED Requirements

### Requirement: Reuse table in the earlier format
A Reuse table whose columns are Need, Existing (path:symbol), Decision, Reason and Evidence SHALL be accepted under the rules of the Reuse table requirement, except that Evidence SHALL not be checked and Decision MAY also be `reuse`, whose Existing SHALL be checked as for `extend`.

#### Scenario: Earlier table with an empty Evidence
- **WHEN** the Reuse table has the five columns and a `reuse` row names an existing file and has no Evidence
- **THEN** lint reports nothing about that row

#### Scenario: Reuse row in the earlier table with a missing path
- **WHEN** a `reuse` row of the five-column table names `internal/gitx/run.go:Run` and that file does not exist
- **THEN** lint reports `reuse-table` on that row
