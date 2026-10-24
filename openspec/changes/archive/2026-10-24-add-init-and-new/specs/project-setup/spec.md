# Spec Delta

## Purpose

Defines how `dkt init` prepares a repository for the Docket workflow by
installing the `docket` OpenSpec schema, without touching the user's own
agent or OpenSpec configuration.

## ADDED Requirements

### Requirement: Install the schema
`dkt init` SHALL write the `docket` schema shipped with the running Docket release to `openspec/schemas/docket/`. When `openspec/` does not exist, it SHALL first run `openspec init --tools none`.

#### Scenario: Fresh repository
- **WHEN** a user runs `dkt init` in a git repository without `openspec/`
- **THEN** `openspec/` exists, `openspec/schemas/docket/` matches the shipped schema, and the command exits with 0

#### Scenario: Existing OpenSpec project
- **WHEN** a user runs `dkt init` where `openspec/` already exists
- **THEN** Docket does not run `openspec init` and only adds the schema

### Requirement: Existing schema files
For each schema file that already exists, `dkt init` SHALL leave an identical file untouched. It SHALL leave a differing file in place and report it, unless `--force` is given, in which case it SHALL overwrite the file and report that it did.

#### Scenario: Locally edited template
- **WHEN** `openspec/schemas/docket/templates/proposal.md` differs from the shipped file and `--force` is absent
- **THEN** the file is unchanged, the result names it, and the command exits with 1

#### Scenario: Re-run with no differences
- **WHEN** a user runs `dkt init` a second time with nothing changed in between
- **THEN** the second run writes nothing and exits with 0

### Requirement: User configuration untouched
Apart from what `openspec init --tools none` itself creates, `dkt init` SHALL NOT create or modify AGENTS.md, CLAUDE.md, anything under `.claude/`, or `openspec/config.yaml`. It SHALL NOT write through a symbolic link: when any path it would write passes through one under `openspec/`, it SHALL write no schema file, name the link, and exit with 1. It SHALL print the recommended AGENTS.md routing section, and include it in the JSON result, for the user to add.

#### Scenario: Routing section printed
- **WHEN** `dkt init` succeeds in a repository with an existing AGENTS.md
- **THEN** the output contains the routing section and AGENTS.md is byte-for-byte unchanged

#### Scenario: Symlinked schema file refused
- **WHEN** `openspec/schemas/docket/templates/proposal.md` is a symbolic link to AGENTS.md and a user runs `dkt init --force`
- **THEN** AGENTS.md is byte-for-byte unchanged, no schema file is written, the result names the link, and the command exits with 1
