# Spec Delta

## MODIFIED Requirements

### Requirement: Preconditions
`dkt archive` SHALL refuse with exit 1, changing nothing, unless the working tree has no uncommitted changes, HEAD contains the default branch tip, every named change exists at the default branch tip, and the `docket` schema is found as `openspec-runtime` says. When the schema is not found, it SHALL report `schema-missing` telling the user to run `dkt init`.

#### Scenario: Uncommitted changes
- **WHEN** the working tree has an uncommitted edit and a user runs `dkt archive add-x`
- **THEN** the command refuses and exits with 1

#### Scenario: Branch behind the default branch
- **WHEN** HEAD does not contain the default branch tip
- **THEN** `dkt archive add-x` refuses and exits with 1

#### Scenario: Change not merged yet
- **WHEN** `add-x` exists in the worktree but not at the default branch tip
- **THEN** `dkt archive add-x` refuses and exits with 1

#### Scenario: Schema not installed
- **WHEN** neither the project's `openspec/schemas/docket/` nor the per-machine directory holds the schema and a user runs `dkt archive add-x`
- **THEN** no OpenSpec command is run, nothing is changed, the findings hold `schema-missing` naming `dkt init`, and the command exits with 1

## ADDED Requirements

### Requirement: Schema reported on archive
`dkt archive` SHALL report in `result.schema` the directory it found the schema in, as an absolute path, and whether it matches the shipped schema byte for byte. A copy that differs SHALL be reported and SHALL NOT stop the archive.

#### Scenario: Archive with a differing schema
- **WHEN** the per-machine `schema.yaml` differs from the shipped file and `add-x` passes every step
- **THEN** `add-x` is archived, `result.schema.matches` is false, the output says the copy differs, and the command exits with 0
