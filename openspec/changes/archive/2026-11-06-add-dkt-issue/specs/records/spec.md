# Spec Delta

## MODIFIED Requirements

### Requirement: Records named
A name given to `dkt lint` that starts with a date of the form `YYYY-MM-DD` SHALL name a record, `docket/records/<name>.md`, or an issue, `docket/issues/<name>.md`, and `dkt lint` SHALL lint each of the two that is a regular file. When neither is, it SHALL report `record-missing` naming it. Naming only records and issues SHALL lint no change and no main spec title.

#### Scenario: Named record linted
- **WHEN** the default branch holds `docket/records/2026-10-30-fix-a.md` with an unknown kind and a user runs `dkt lint 2026-10-30-fix-a`
- **THEN** the finding `record-kind` names that file, no change is linted, and dkt exits with 1

#### Scenario: Named record missing
- **WHEN** no such file exists and a user runs `dkt lint 2026-11-06-fix-q`
- **THEN** the finding `record-missing` names `2026-11-06-fix-q` and dkt exits with 1

#### Scenario: Named issue linted
- **WHEN** the default branch holds `docket/issues/2026-10-30-lint-timeout.md` without a source, no record has that slug, and a user runs `dkt lint 2026-10-30-lint-timeout`
- **THEN** the finding `issue-front-matter` names that file, no `record-missing` is reported, and dkt exits with 1
