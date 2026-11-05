# setup-inspection Specification

## Purpose
Defines what `dkt inspect` checks and reports about a project's Docket setup: docket-init uses it to confirm the setup is complete, and docket-doctor to learn what to fix. It writes nothing.

## Requirements

### Requirement: Inspect writes nothing
`dkt inspect` SHALL change no file, inside the repository or outside it, and the only processes it SHALL start are git and the OpenSpec CLI's version query. It SHALL exit with 0 when it reports no finding and with 1 when it reports any.

#### Scenario: Nothing written
- **WHEN** a user runs `dkt inspect` in a repository where `dkt init` never ran
- **THEN** no file inside or outside the repository is created, changed or removed

#### Scenario: Clean setup
- **WHEN** the repository was set up by this release's `dkt init` and not changed since, and the OpenSpec CLI reports 1.7.0
- **THEN** `dkt inspect --json` reports no finding and exits with 0

### Requirement: Installed files inspected
`dkt inspect` SHALL check every file `dkt init` writes, by the rules `dkt init` uses, and list each in `result.files` with its `path`, `kind` and a `status`: `same` (equal to this release's file), `missing` (`dkt init` writes it), `outdated` (equal to an earlier release's file; `dkt init` updates it) or `differs` (`dkt init` keeps it; `dkt init --force` replaces it).

#### Scenario: Statuses listed
- **WHEN** a user runs `dkt inspect --json` in a repository set up by this release's `dkt init`
- **THEN** `result.files` lists every file `dkt init` writes, each with status `same`

### Requirement: Installed file findings
For each file that is not `same`, `dkt inspect` SHALL report a finding named after its kind and status: `schema-missing`, `skill-missing`, `schema-outdated`, `skill-outdated`, `schema-differs` or `skill-differs`, naming the file.

#### Scenario: Before init
- **WHEN** a user runs `dkt inspect --json` in a repository without `openspec/`
- **THEN** the findings hold `schema-missing` and `skill-missing`, and dkt exits with 1

#### Scenario: Edited skill
- **WHEN** `.claude/skills/docket/SKILL.md` differs from every shipped version
- **THEN** `result.files` lists it as `differs` and the finding `skill-differs` names it

#### Scenario: File from an earlier release
- **WHEN** `openspec/schemas/docket/templates/plan.md` equals the file an earlier release shipped
- **THEN** `result.files` lists it as `outdated` and the finding `schema-outdated` names it

### Requirement: config.yaml inspected
`dkt inspect` SHALL check `openspec/config.yaml` by the rules `dkt init` uses and report `config-schema-default` when the file is missing, has no top-level `schema` or has `schema: spec-driven`; `config-schema-other` when it names another schema; and `guidance-missing`, `guidance-outdated` or `guidance-differs` when its `operations` section is missing, equals an earlier release's, or holds anything else.

#### Scenario: Before init config
- **WHEN** a user runs `dkt inspect --json` in a repository without `openspec/`
- **THEN** the findings hold `config-schema-default` and `guidance-missing` naming `openspec/config.yaml`

#### Scenario: Other schema reported
- **WHEN** `openspec/config.yaml` has `schema: team-flow`
- **THEN** the finding `config-schema-other` names the file

### Requirement: config.yaml status
In `result.files`, `openspec/config.yaml` SHALL be `missing` when it does not exist, `differs` when `dkt init` would keep a part of it (another schema, or other guidance), `outdated` when `dkt init` would otherwise change it, and `same` when both parts are as this release ships them.

#### Scenario: Default schema is outdated
- **WHEN** `openspec/config.yaml` is the one `openspec init` wrote, with `schema: spec-driven` and no `operations`
- **THEN** `result.files` lists it as `outdated`

#### Scenario: Other schema differs
- **WHEN** `openspec/config.yaml` has `schema: team-flow` and the shipped `operations` section
- **THEN** `result.files` lists it as `differs`

### Requirement: Symbolic links inspected
`dkt inspect` SHALL report `symlink-refused`, naming the link, for each path `dkt init` writes that passes through a symbolic link inside the repository, and SHALL read nothing through that link.

#### Scenario: Linked skill folder inspected
- **WHEN** `.claude/skills/docket` is a symbolic link to another directory
- **THEN** the finding `symlink-refused` names `.claude/skills/docket`, and dkt exits with 1

### Requirement: OpenSpec CLI inspected
`dkt inspect` SHALL locate the OpenSpec CLI as openspec-runtime says, read its version, and report in `result.openspec` the `version` (null when the CLI cannot be started), the `minimum` (`1.7.0`) and `supported`. It SHALL report `openspec-missing` when the CLI cannot be started and `openspec-unsupported` when the version is too old or cannot be read; neither SHALL make it exit with 4.

#### Scenario: Old OpenSpec
- **WHEN** the OpenSpec CLI reports 1.6.2 and a user runs `dkt inspect --json`
- **THEN** `result.openspec.version` is `1.6.2`, `result.openspec.supported` is false, the findings hold `openspec-unsupported`, and dkt exits with 1

#### Scenario: No OpenSpec
- **WHEN** no OpenSpec CLI can be started
- **THEN** `result.openspec.version` is null and the findings hold `openspec-missing`

### Requirement: Ignored files inspected
`dkt inspect` SHALL report `install-ignored` for each file `dkt init` writes or keeps that git ignores and does not track, whether or not the file exists yet, naming the ignore file, the line and the pattern. It SHALL change no ignore file.

#### Scenario: Ignored OpenSpec folder
- **WHEN** `.git/info/exclude` holds `openspec/` on its seventh line and a user runs `dkt inspect --json`
- **THEN** the finding `install-ignored` names `openspec/config.yaml` with `.git/info/exclude`, line 7 and `openspec/`, and `.git/info/exclude` is byte-for-byte unchanged
