## ADDED Requirements

### Requirement: Sync of a change of another schema
`dkt sync <slug>` for a change of another schema SHALL report `other-schema`, naming the change's `.openspec.yaml` and its schema, change nothing, and exit with 1, with or without `--check`.

#### Scenario: Sync refused for another schema
- **WHEN** `add-x` has `schema: spec-driven` and a user runs `dkt sync add-x`
- **THEN** the findings hold `other-schema` naming `openspec/changes/add-x/.openspec.yaml` and `spec-driven`, no file changes, and the command exits with 1
