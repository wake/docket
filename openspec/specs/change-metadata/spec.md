# Change Metadata Specification

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
A title, and a source, SHALL be one line of valid UTF-8 text that is not empty and not only whitespace. A title SHALL hold at most 120 bytes, measured as defined under Text length; a source has no length limit.

#### Scenario: Blank title
- **WHEN** a user runs `dkt new add-x --kind feature --source x --title "   "`
- **THEN** nothing is created and dkt exits with 2

#### Scenario: Source on two lines
- **WHEN** a user runs `dkt new add-x --kind feature --title X --source` with a value holding a line break
- **THEN** nothing is created and dkt exits with 2

#### Scenario: Title over the limit
- **WHEN** a user runs `dkt new add-x --kind feature --source x` with a `--title` of 121 bytes
- **THEN** nothing is created, OpenSpec is not run, the message gives 121 and 120, and dkt exits with 2

#### Scenario: Chinese title at the limit
- **WHEN** a user runs `dkt new add-x --kind feature --source x` with a `--title` of 40 Chinese characters
- **THEN** the change is created

#### Scenario: Long source
- **WHEN** a user runs `dkt new add-x --kind feature --title X` with a `--source` of 500 bytes
- **THEN** the change is created

### Requirement: Change schema
A change's schema SHALL be the value of the top-level `schema` key of its `.openspec.yaml`, as OpenSpec writes it, plain or quoted. A change whose `.openspec.yaml` is missing or has no `schema` key SHALL have the schema `docket`. A change whose schema is not `docket` is a change of another schema.

#### Scenario: Schema read
- **WHEN** `openspec/changes/add-x/.openspec.yaml` holds `schema: spec-driven` and `created: 2026-01-01`
- **THEN** the schema of `add-x` is `spec-driven`

#### Scenario: No schema key
- **WHEN** `openspec/changes/add-x/.openspec.yaml` holds only `created: 2026-01-01`
- **THEN** the schema of `add-x` is `docket`

### Requirement: Front matter read
Docket SHALL read the front matter of `proposal.md` as the lines between a first line `---` and the next line `---`: each top-level `key: value`, the value plain or in double or single quotes, a double-quoted value unescaped as YAML does; and under `specs:`, each indented `<capability-path>: <name>` line. A front matter it cannot read this way, or a key given twice, is invalid.

#### Scenario: Quoted title read back
- **WHEN** the front matter holds `title: "Booking: waitlist \"v2\""`
- **THEN** the change's title is `Booking: waitlist "v2"`

#### Scenario: Capability name read
- **WHEN** the front matter holds `specs:` followed by `  booking: Booking Waitlist`
- **THEN** the name given to the capability `booking` is `Booking Waitlist`

#### Scenario: Unclosed front matter
- **WHEN** `proposal.md` starts with `---` and no later line is `---`
- **THEN** the front matter is invalid

### Requirement: Main spec title
A main spec's title SHALL be taken from the first line of its `spec.md`, of the form `# <name> Specification`, as `<name>`. A first line of another form, or a name equal to the capability's path, which is what OpenSpec writes when it creates the spec, SHALL mean the spec has no title.

#### Scenario: Title read
- **WHEN** `openspec/specs/booking/spec.md` starts with `# Booking Waitlist Specification`
- **THEN** the title of `booking` is `Booking Waitlist`

#### Scenario: OpenSpec's first line
- **WHEN** `openspec/specs/booking/spec.md` starts with `# booking Specification`
- **THEN** `booking` has no title

### Requirement: Text length
Docket SHALL measure the length of text in UTF-8 bytes, with no rule for any language: whitespace at either end SHALL NOT count, and each other run of whitespace SHALL count as one space.

#### Scenario: Whitespace collapsed
- **WHEN** a title holds 120 bytes once a run of three spaces in it is counted as one
- **THEN** the title is within the limit
