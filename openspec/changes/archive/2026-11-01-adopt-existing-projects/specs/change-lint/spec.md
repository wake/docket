## ADDED Requirements

### Requirement: Changes of another schema
lint SHALL list each linted change of another schema in `result.other_schema`, with its slug and schema, and in the text output. It SHALL apply to such a change only the title rule for a capability it adds that has no main spec, and none of the other rules on its proposal, deltas, plan or metadata.

#### Scenario: Change of another schema
- **WHEN** `add-x` has `schema: spec-driven` in its `.openspec.yaml`, a `proposal.md` without Non-Goals, a `tasks.md`, no `docket.yaml`, and a delta for the existing capability `booking`, and a user runs `dkt lint --json`
- **THEN** `result.other_schema` holds `add-x` with `spec-driven`, no finding names `add-x`, and the command exits with 0

#### Scenario: New capability of another schema
- **WHEN** the same `add-x` has a delta for `waitlist`, the worktree has no `openspec/specs/waitlist/spec.md`, and no `docket.yaml` sits beside that delta
- **THEN** lint reports a `title-missing` finding for `waitlist` and no other finding for `add-x`

## MODIFIED Requirements

### Requirement: Titles
lint SHALL report `title-missing` for a linted change without a title, unless it is a change of another schema; for a linted change's delta of a capability that has no main spec in the worktree, when no title sits beside that delta; and, when no change is named, for every main spec without a title. A `docket.yaml` that is not a YAML mapping SHALL be reported as `metadata-invalid`.

#### Scenario: Change without title
- **WHEN** `add-x` has no `docket.yaml` and a user runs `dkt lint add-x`
- **THEN** lint reports a `title-missing` finding naming `openspec/changes/add-x/docket.yaml`

#### Scenario: New capability without title
- **WHEN** `add-x` has a delta for `booking`, the worktree has no `openspec/specs/booking/spec.md`, and `openspec/changes/add-x/specs/booking/docket.yaml` does not exist
- **THEN** lint reports a `title-missing` finding for `booking`

#### Scenario: Existing capability needs no delta title
- **WHEN** `add-x` has a delta for `booking`, the main spec `booking` exists, and no `docket.yaml` sits beside the delta
- **THEN** lint reports no `title-missing` finding for `booking`

#### Scenario: Main spec without title
- **WHEN** `openspec/specs/booking/spec.md` exists without a `docket.yaml` beside it and a user runs `dkt lint`
- **THEN** lint reports a `title-missing` finding naming `openspec/specs/booking/docket.yaml`

#### Scenario: Named change skips main specs
- **WHEN** a main spec has no title and a user runs `dkt lint add-x`, where `add-x` has a title
- **THEN** lint reports no finding about that main spec

#### Scenario: Broken docket.yaml
- **WHEN** `openspec/changes/add-x/docket.yaml` holds a YAML list
- **THEN** lint reports a `metadata-invalid` finding on that file

#### Scenario: Change of another schema has no title
- **WHEN** `add-x` is a change of another schema with no `docket.yaml`
- **THEN** lint reports no `title-missing` finding naming `openspec/changes/add-x/docket.yaml`
