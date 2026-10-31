# Spec Delta

## MODIFIED Requirements

### Requirement: Change created with metadata
`dkt new` SHALL create the change by running `openspec new change <slug> --schema docket`, then write the change's `docket.yaml` with the title, kind, source and base, leaving `.openspec.yaml` as OpenSpec wrote it. The base SHALL be the full SHA of the merge base of HEAD and the default branch tip. When the change folder's path passes through a symbolic link under `openspec/`, `dkt new` SHALL neither run OpenSpec nor write anything, and SHALL exit with 1 naming the link.

#### Scenario: Change created
- **WHEN** a user runs `dkt new add-x --kind feature --source "issue 12" --title "Booking waitlist"` on a branch made from the default branch
- **THEN** `openspec/changes/add-x/docket.yaml` holds `title: Booking waitlist`, `kind: feature`, `source: issue 12` and `base` equal to that merge base, and `.openspec.yaml` holds only what OpenSpec wrote

#### Scenario: Schema not installed
- **WHEN** neither the project's `openspec/schemas/docket/` nor the per-machine directory holds the schema and a user runs `dkt new add-x --kind feature --source x --title X`
- **THEN** OpenSpec is not run, nothing is created, the findings hold `schema-missing` naming `dkt init`, and the command exits with 1

#### Scenario: Symlinked changes folder refused
- **WHEN** `openspec/changes` is a symbolic link to a directory outside the repository and a user runs `dkt new add-x --kind feature --source x --title X`
- **THEN** OpenSpec is not run, nothing is written on either side of the link, and the command exits with 1 naming the link

## ADDED Requirements

### Requirement: Schema found before creating
Before it runs OpenSpec, `dkt new` SHALL find the `docket` schema as `openspec-runtime` says. When neither place holds it, `dkt new` SHALL create nothing, report `schema-missing` telling the user to run `dkt init`, and exit with 1. Otherwise it SHALL report in `result.schema` the directory found, as an absolute path, and whether it matches the shipped schema byte for byte; a copy that differs SHALL be reported and SHALL NOT stop the change from being created.

#### Scenario: Schema from the machine
- **WHEN** only the per-machine directory holds the schema, matching the shipped one, and a user runs `dkt new add-x --kind feature --source x --title X`
- **THEN** the change is created, `result.schema.path` is the per-machine directory, `result.schema.matches` is true, and the command exits with 0

#### Scenario: Schema differs
- **WHEN** the per-machine `schema.yaml` differs from the shipped file and a user runs `dkt new add-x --kind feature --source x --title X`
- **THEN** the change is created, `result.schema.matches` is false, the output says the copy differs and that `dkt init --force` overwrites it, and the command exits with 0
