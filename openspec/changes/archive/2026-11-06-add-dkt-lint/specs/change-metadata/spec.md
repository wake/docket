## ADDED Requirements

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
