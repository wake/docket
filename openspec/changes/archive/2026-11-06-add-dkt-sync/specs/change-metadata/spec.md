## ADDED Requirements

### Requirement: Change schema
A change's schema SHALL be the value of the top-level `schema` key of its `.openspec.yaml`, as OpenSpec writes it, plain or quoted. A change whose `.openspec.yaml` is missing or has no `schema` key SHALL have the schema `docket`. A change whose schema is not `docket` is a change of another schema.

#### Scenario: Schema read
- **WHEN** `openspec/changes/add-x/.openspec.yaml` holds `schema: spec-driven` and `created: 2026-01-01`
- **THEN** the schema of `add-x` is `spec-driven`

#### Scenario: No schema key
- **WHEN** `openspec/changes/add-x/.openspec.yaml` holds only `created: 2026-01-01`
- **THEN** the schema of `add-x` is `docket`
