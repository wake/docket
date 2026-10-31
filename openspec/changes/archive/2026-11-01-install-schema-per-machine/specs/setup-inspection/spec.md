# Spec Delta

## MODIFIED Requirements

### Requirement: Inspect writes nothing
`dkt inspect` SHALL change no file, in the repository or anywhere else. The only processes it SHALL start are git and the OpenSpec CLI's version query. It SHALL exit with 0 when it reports no finding and with 1 when it reports any.

#### Scenario: Nothing written
- **WHEN** a user runs `dkt inspect` in a repository where every installed file differs from the shipped one, the project holds a copy of a shipped schema, `openspec/config.yaml` names `spec-driven`, and the workflow file is missing
- **THEN** no file in the repository, under the user's config directory or under the per-machine schema directory is created, changed or removed

#### Scenario: Clean setup
- **WHEN** a repository was set up by the running release's `dkt init`, the OpenSpec CLI reports 1.14.3, and `dkt workflow` would report `ok`
- **THEN** `dkt inspect --json` reports no finding and exits with 0

### Requirement: Installed files inspected
`dkt inspect` SHALL check every file `dkt init` would install or keep, the per-machine schema and `config.yaml` too, by the rules `dkt init` uses, and SHALL report each one as `same`, `missing` or `differs` in `result.files`. A missing file SHALL give `schema-missing`, `skill-missing` or `agents-block-missing`; a differing one `schema-differs`, `skill-differs` or `agents-block-differs`; a malformed block `agents-block-malformed`; a path `dkt init` would refuse, `symlink-refused`.

#### Scenario: Before init
- **WHEN** a user runs `dkt inspect --json` in a git repository where `dkt init` never ran, on a machine without the per-machine schema
- **THEN** the findings hold `schema-missing` naming per-machine files, `skill-missing`, `agents-block-missing` and `config-schema-default`, and the command exits with 1

#### Scenario: Edited skill
- **WHEN** `.claude/skills/docket/SKILL.md` differs from the shipped file
- **THEN** `result.files` lists it as `differs` and the findings hold `skill-differs` naming it

#### Scenario: Machine copy differs
- **WHEN** the per-machine `schema.yaml` differs from the shipped file
- **THEN** `result.files` lists its absolute path as `differs` with kind `schema`, and the findings hold `schema-differs` naming it and saying that `dkt init --force` overwrites it

#### Scenario: CLAUDE.md imports AGENTS.md
- **WHEN** `CLAUDE.md` holds the line `@AGENTS.md` and `AGENTS.md` holds the shipped block
- **THEN** no finding names `CLAUDE.md`

#### Scenario: No home directory
- **WHEN** `XDG_DATA_HOME` is unset and no home directory can be found
- **THEN** the findings hold `schema-missing` with that reason, and the command does not exit with 4

## ADDED Requirements

### Requirement: Project copy and schema key inspected
`dkt inspect` SHALL report a project's `openspec/schemas/docket/` as `dkt init` would treat it: `schema-project-copy`, saying `dkt init` removes it, when it equals a shipped schema, and `schema-edited`, saying it is used instead of the per-machine copy, when it equals none. It SHALL report `config-schema-default` when `openspec/config.yaml` is missing, has no `schema` key or names `spec-driven`, and `config-schema-other` when it names another schema.

#### Scenario: Project copy reported
- **WHEN** the project's `openspec/schemas/docket/` holds the schema dkt v0.9.1 shipped
- **THEN** the findings hold `schema-project-copy` naming `openspec/schemas/docket`, and the command exits with 1

#### Scenario: Default schema key reported
- **WHEN** `openspec/config.yaml` starts with `schema: spec-driven`
- **THEN** `result.files` lists `openspec/config.yaml` as `differs` with kind `config`, and the findings hold `config-schema-default` naming it
