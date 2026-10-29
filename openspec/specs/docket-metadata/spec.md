# docket-metadata Specification

## Purpose
Defines where Docket keeps the data it adds to OpenSpec's changes and
specs, how it reads changes made before that, and what a title is.

## Requirements

### Requirement: Change metadata file
Docket SHALL read a change's title, kind, source and base from `docket.yaml` in the change folder: a YAML mapping with the keys `title`, `kind`, `source` and `base`. The creation date SHALL stay in `.openspec.yaml`, as OpenSpec writes it.

#### Scenario: Metadata read from docket.yaml
- **WHEN** `openspec/changes/add-x/docket.yaml` holds `title: Booking waitlist`, `kind: feature`, `source: issue 12` and a base SHA
- **THEN** Docket reads that title, kind, source and base for `add-x`

### Requirement: Changes made before docket.yaml
For a change folder with no `docket.yaml`, Docket SHALL read the kind, source and base from the `x-kind`, `x-source` and `x-base` keys of `.openspec.yaml`, and the change SHALL have no title. When `docket.yaml` exists, Docket SHALL ignore those keys. Docket SHALL NOT rewrite an archived change's files to move them.

#### Scenario: Change made before docket.yaml
- **WHEN** `add-x` has no `docket.yaml` and its `.openspec.yaml` holds `x-kind`, `x-source` and `x-base`
- **THEN** Docket reads its kind, source and base from those keys and gives it no title

#### Scenario: Both files present
- **WHEN** `add-x` has a `docket.yaml` with one base and an `.openspec.yaml` whose `x-base` is another
- **THEN** Docket uses the base in `docket.yaml`

### Requirement: Spec titles
A main spec's title SHALL be `title` in `openspec/specs/<capability>/docket.yaml`. A change SHALL give a capability its title in `openspec/changes/<slug>/specs/<capability>/docket.yaml`, beside the capability's delta. Docket SHALL NOT take a title from a heading.

#### Scenario: Title beside a main spec
- **WHEN** `openspec/specs/booking/docket.yaml` holds `title: Booking`
- **THEN** the title of the `booking` spec is "Booking", whatever the H1 of its `spec.md` says

### Requirement: Title form
A title SHALL be one line of text that is not empty and not only whitespace.

#### Scenario: Title on two lines
- **WHEN** a `docket.yaml` holds a `title` with a line break in it
- **THEN** Docket treats the title as missing
