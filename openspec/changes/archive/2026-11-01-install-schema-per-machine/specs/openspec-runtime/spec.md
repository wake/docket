# Spec Delta

## ADDED Requirements

### Requirement: Schema location
Docket SHALL look for the `docket` schema where OpenSpec 1.14 looks: the project's `openspec/schemas/docket/` first, then the per-machine directory, `openspec/schemas/docket/` under `$XDG_DATA_HOME`, or under `<home>/.local/share/` when `XDG_DATA_HOME` is unset or empty. The home directory SHALL be found as for the workflow file. When `XDG_DATA_HOME` is a relative path or no home directory is found, a command that would read or write the per-machine directory SHALL exit with 4 naming the reason.

#### Scenario: Schema from the data directory
- **WHEN** `XDG_DATA_HOME` is `/d` and a user runs `dkt init`
- **THEN** `/d/openspec/schemas/docket/schema.yaml` matches the shipped file

#### Scenario: Default under home
- **WHEN** `XDG_DATA_HOME` is unset, the home directory is `/h`, and a user runs `dkt init`
- **THEN** `/h/.local/share/openspec/schemas/docket/schema.yaml` matches the shipped file

#### Scenario: Project copy first
- **WHEN** both the project's `openspec/schemas/docket/` and the per-machine directory hold the schema and a user runs `dkt new add-x --kind feature --source x --title X`
- **THEN** `result.schema.path` is the project's `openspec/schemas/docket` as an absolute path

#### Scenario: Relative data directory
- **WHEN** `XDG_DATA_HOME` is `data` and a user runs `dkt init`
- **THEN** the command exits with 4 naming `XDG_DATA_HOME`, and nothing is written in the repository or outside it
