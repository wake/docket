# change-metadata Specification

## Purpose
Defines where a change's title, kind and source are kept, in the front matter of its `proposal.md`, and what form each takes, so that every command reads them the same way.

## Requirements

### Requirement: Front matter of proposal.md
A change's title, kind and source SHALL be the keys `title`, `kind` and `source` of a YAML front matter at the start of its `proposal.md`: a first line `---`, the keys, and a closing line `---`. When `dkt new` writes them, `title` and `source` SHALL be YAML double-quoted strings and `kind` a plain value, one key per line in that order.

#### Scenario: Front matter written
- **WHEN** `dkt new add-x --kind feature --source "issue 12" --title 'Booking: waitlist "v2"'` creates a change
- **THEN** `proposal.md` reads `---`, `title: "Booking: waitlist \"v2\""`, `kind: feature`, `source: "issue 12"`, `---`, one per line

### Requirement: Kinds
A change's kind SHALL be one of `feature` (something new), `change-request` (a change to existing behavior) and `baseline` (specifying behavior that is already released).

#### Scenario: Kind outside the list
- **WHEN** a user runs `dkt new add-x --kind refactor --source x --title X`
- **THEN** dkt exits with 2 and lists the three kinds

### Requirement: Title and source form
A title, and a source, SHALL be one line of valid UTF-8 text that is not empty and not only whitespace.

#### Scenario: Blank title
- **WHEN** a user runs `dkt new add-x --kind feature --source x --title "   "`
- **THEN** nothing is created and dkt exits with 2

#### Scenario: Source on two lines
- **WHEN** a user runs `dkt new add-x --kind feature --title X --source` with a value holding a line break
- **THEN** nothing is created and dkt exits with 2

### Requirement: Change schema
A change's schema SHALL be the value of the top-level `schema` key of its `.openspec.yaml`, as OpenSpec writes it, plain or quoted. A change whose `.openspec.yaml` is missing or has no `schema` key SHALL have the schema `docket`. A change whose schema is not `docket` is a change of another schema.

#### Scenario: Schema read
- **WHEN** `openspec/changes/add-x/.openspec.yaml` holds `schema: spec-driven` and `created: 2026-01-01`
- **THEN** the schema of `add-x` is `spec-driven`

#### Scenario: No schema key
- **WHEN** `openspec/changes/add-x/.openspec.yaml` holds only `created: 2026-01-01`
- **THEN** the schema of `add-x` is `docket`
