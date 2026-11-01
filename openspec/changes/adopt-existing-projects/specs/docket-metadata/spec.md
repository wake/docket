## ADDED Requirements

### Requirement: Change schema
Docket SHALL read a change's schema from the `schema` key of its `.openspec.yaml`, as OpenSpec writes it. A change whose `.openspec.yaml` has no `schema` key SHALL have the schema `docket`. A change whose schema is not `docket` is a change of another schema; `dkt` SHALL NOT write to its `.openspec.yaml`.

#### Scenario: Schema read
- **WHEN** `openspec/changes/add-x/.openspec.yaml` holds `schema: spec-driven`
- **THEN** Docket reads the schema of `add-x` as `spec-driven`, a change of another schema

#### Scenario: Schema missing
- **WHEN** `openspec/changes/add-x/.openspec.yaml` holds only `created: 2026-01-02`
- **THEN** Docket reads the schema of `add-x` as `docket`
