# Spec Delta

## MODIFIED Requirements

### Requirement: Install the schema
`dkt init` SHALL write the `docket` schema shipped with the running Docket release to the per-machine directory that `openspec-runtime` names, and SHALL list each file it writes there by its absolute path. It SHALL NOT write the schema into the project. When `openspec/` does not exist, it SHALL first run `openspec init --tools none`.

#### Scenario: Fresh repository
- **WHEN** a user runs `dkt init` in a git repository without `openspec/`
- **THEN** `openspec/` exists, `openspec/schemas/docket/` does not, the per-machine copy matches the shipped schema, and the command exits with 0

#### Scenario: Existing OpenSpec project
- **WHEN** a user runs `dkt init` where `openspec/` already exists
- **THEN** Docket does not run `openspec init`

#### Scenario: Fresh machine
- **WHEN** the per-machine directory has no `docket` schema and a user runs `dkt init` in a project
- **THEN** every shipped schema file is written there, the result lists each one as `written` with its absolute path, and the command exits with 0

#### Scenario: Linked data directory followed
- **WHEN** `XDG_DATA_HOME` is a symbolic link to another directory and a user runs `dkt init`
- **THEN** the schema is written under the directory the link points to, and the command exits with 0

### Requirement: Existing installed files
For each schema or skill file that already exists, `dkt init` SHALL leave an identical file untouched. It SHALL leave a differing file in place and report it, unless `--force` is given, in which case it SHALL overwrite the file and report that it did.

#### Scenario: Locally edited template
- **WHEN** the per-machine `templates/proposal.md` differs from the shipped file and `--force` is absent
- **THEN** the file is unchanged, the result reports `schema-differs` naming its absolute path, and the command exits with 1

#### Scenario: Machine copy forced
- **WHEN** the per-machine `templates/proposal.md` and `.agents/skills/docket/SKILL.md` both differ from the shipped files and a user runs `dkt init --force`
- **THEN** both files match the shipped ones, the result lists both as `overwritten`, and the command exits with 0

#### Scenario: Edited skill file
- **WHEN** `.agents/skills/docket/SKILL.md` differs from the shipped file and `--force` is absent
- **THEN** the file is unchanged, the result reports `skill-differs` for it, and the command exits with 1

#### Scenario: Re-run with no differences
- **WHEN** a user runs `dkt init` a second time with nothing changed in between
- **THEN** the second run writes nothing and exits with 0

### Requirement: Files init writes
Apart from what `openspec init --tools none` creates, `dkt init` SHALL write only the per-machine schema, the `schema` key of `openspec/config.yaml`, the skill folders, the Docket block of `AGENTS.md` and `CLAUDE.md`, and a missing `AGENTS.md` or `CLAUDE.md`, and SHALL remove only a project copy of a shipped schema. When a path it would change passes through a symbolic link in the repository or at or below the per-machine `docket` folder, it SHALL change nothing, name the link, and exit with 1.

#### Scenario: Other configuration unchanged
- **WHEN** `dkt init` succeeds in a repository with `.claude/settings.json`, a skill of its own under `.claude/skills/`, and an `openspec/config.yaml` that names `docket` and has a `context`
- **THEN** those files are byte-for-byte unchanged, and the output holds no section to paste

#### Scenario: Symlinked schema file refused
- **WHEN** the per-machine `templates/proposal.md` is a symbolic link to another file and a user runs `dkt init --force`
- **THEN** the file the link points to is byte-for-byte unchanged, no schema file is written, the result names the link, and the command exits with 1

#### Scenario: Symlinked skill directory refused
- **WHEN** `.claude/skills/docket` is a symbolic link to another directory and a user runs `dkt init --force`
- **THEN** nothing in that directory or anywhere else is written, the result names the link, and the command exits with 1

## ADDED Requirements

### Requirement: Project copy of the schema
`dkt init` SHALL compare the project's `openspec/schemas/docket/` with each schema a Docket release up to the running one shipped. When it holds exactly the files of one of them, byte for byte, and every per-machine file matches the shipped one after this run, it SHALL remove those files and the folders they leave empty, listing each file as `removed`. Otherwise it SHALL keep the copy and report `schema-project-copy` when it equals a shipped schema, `schema-edited` when it equals none.

#### Scenario: Shipped project copy removed
- **WHEN** `openspec/schemas/docket/` holds the schema dkt v0.9.1 shipped, the per-machine copy is missing, and a user runs `dkt init`
- **THEN** the per-machine copy is written, each file of the project copy is listed as `removed`, `openspec/schemas/docket/` no longer exists, and the command exits with 0

#### Scenario: Edited project copy kept
- **WHEN** the project's `openspec/schemas/docket/templates/proposal.md` differs from that file in every shipped schema and a user runs `dkt init --force`
- **THEN** the project copy is byte-for-byte unchanged, the result reports `schema-edited` naming `openspec/schemas/docket`, and the command exits with 1

#### Scenario: Folder without a schema
- **WHEN** the project's `openspec/schemas/docket/` holds `templates/proposal.md` but no `schema.yaml`, which OpenSpec does not read as a schema, and a user runs `dkt init`
- **THEN** the folder is byte-for-byte unchanged and no finding names it

#### Scenario: Removal waits for the machine copy
- **WHEN** the project copy equals the shipped schema, the per-machine `schema.yaml` differs from it, and `--force` is absent
- **THEN** the project copy is unchanged, the result reports `schema-differs` and `schema-project-copy`, and the command exits with 1

### Requirement: Schema key in config.yaml
`dkt init` SHALL make `openspec/config.yaml` name `docket`: create the file holding `schema: docket` when it is missing, add that line first when the file has no top-level `schema` key, and replace a `schema: spec-driven` line, leaving every other byte unchanged. Another schema name SHALL be kept, with or without `--force`, and reported as `config-schema-other`. A `schema` key that is not one plain top-level line SHALL make it write nothing and list the file as unreadable.

#### Scenario: Default schema replaced
- **WHEN** `openspec/config.yaml` is what `openspec init --tools none` writes, starting with `schema: spec-driven`, and a user runs `dkt init`
- **THEN** its first line is `schema: docket`, every other line is byte-for-byte unchanged, and the result lists it as `written`

#### Scenario: Config file created
- **WHEN** `openspec/` exists without `config.yaml` and a user runs `dkt init`
- **THEN** `openspec/config.yaml` holds `schema: docket` and nothing else

#### Scenario: Other schema left
- **WHEN** `openspec/config.yaml` has `schema: team-flow` and a user runs `dkt init --force`
- **THEN** the file is byte-for-byte unchanged, the result reports `config-schema-other` for it, and the command exits with 1

#### Scenario: Unreadable schema key
- **WHEN** `openspec/config.yaml` has `schema:` followed by an indented value on the next line and a user runs `dkt init`
- **THEN** no file is written or removed, `unreadable` lists `openspec/config.yaml`, and the command exits with 3
