## MODIFIED Requirements

### Requirement: Changes in focus
With slugs, `dkt status` SHALL list only those changes and records, in slug order. A slug that starts with a date SHALL name a record, listed as `records` defines; any other slug SHALL be an active change, and a slug that is not SHALL be reported, and the others still listed.

#### Scenario: One change in focus
- **WHEN** `add-x` and `add-y` are active and a user runs `dkt status add-y`
- **THEN** only `add-y` is listed and the command exits with 0

#### Scenario: Slug not active
- **WHEN** a user runs `dkt status add-q` and no active change is named `add-q`
- **THEN** the command reports that `add-q` is not active and exits with 1

#### Scenario: Record in focus
- **WHEN** `add-x` and `add-y` are active, the record `2026-11-01-fix-x` is open, and a user runs `dkt status add-y 2026-11-01-fix-x --json`
- **THEN** `result.changes` lists only `add-y`, `result.records` lists only `2026-11-01-fix-x`, and the command exits with 0
