# Spec Delta

## Purpose

Defines how `dkt init` prepares a project for Docket: which files it writes, what it does with files that already exist, and what its result tells the user, while leaving the user's other coding agent and OpenSpec configuration alone.

## ADDED Requirements

### Requirement: Repository root
`dkt init` SHALL run inside a git repository and SHALL write under the repository root, whichever subdirectory it runs from. It SHALL run as usual in a repository with no commit yet.

#### Scenario: Run from a subdirectory
- **WHEN** a user runs `dkt init --tools claude` in the repository's `src/`
- **THEN** the files are written under the repository root, and no file is added under `src/`

#### Scenario: Repository without commits
- **WHEN** a user runs `dkt init --tools claude` in a repository just made with `git init`, with no commit
- **THEN** the files are written as usual and dkt exits with 0

### Requirement: Checks before writing
`dkt init` SHALL finish every check before it writes anything or runs `openspec init`: the repository, `--tools`, the OpenSpec version, symbolic links on the paths it writes, and whether the existing files can be read. When any check fails, it SHALL write nothing and SHALL NOT run `openspec init`.

#### Scenario: Unreadable file stops everything
- **WHEN** `.claude/skills/docket/SKILL.md` exists but cannot be read and a user runs `dkt init`
- **THEN** no file is written, `unreadable` lists that file, and dkt exits with 3

### Requirement: OpenSpec project first
When the project has no `openspec/`, `dkt init` SHALL first run `openspec init --tools <value>` at the repository root, `<value>` being the string given to `--tools`, passed through for OpenSpec to check. With no `openspec/` and no `--tools`, it SHALL do nothing, say to add `--tools` or run `openspec init` first, and exit with 2. When `openspec/` exists, it SHALL NOT run `openspec init`, and ignores `--tools`.

#### Scenario: Fresh repository
- **WHEN** a user runs `dkt init --tools claude` in a repository without `openspec/`
- **THEN** dkt runs `openspec init --tools claude`, then writes Docket's files, and exits with 0

#### Scenario: Missing --tools
- **WHEN** a user runs `dkt init` in a repository without `openspec/`
- **THEN** no file is written, the message says to add `--tools` or run `openspec init` first, and dkt exits with 2

#### Scenario: Existing OpenSpec project
- **WHEN** a user runs `dkt init` in a repository that has `openspec/`, with or without `--tools`
- **THEN** dkt does not run `openspec init`

### Requirement: Files init writes
`dkt init` SHALL write these files shipped with the running dkt release:
- the schema under `openspec/schemas/docket/`: `schema.yaml` and the templates under `templates/`;
- the `schema` line and the `operations` section of `openspec/config.yaml`;
- `.claude/skills/docket/SKILL.md` and `.agents/skills/docket/SKILL.md`, which are the same.

#### Scenario: Fresh repository gets the files
- **WHEN** a user runs `dkt init --tools claude` in a repository without `openspec/`
- **THEN** each file listed above matches the shipped one, and the two `SKILL.md` files are byte-for-byte identical

### Requirement: Nothing else written
Apart from the files it ships and what `openspec init` creates, `dkt init` SHALL write or remove no file. It SHALL NOT write `AGENTS.md` or `CLAUDE.md`, and SHALL leave alone any other file in the folders it writes to.

#### Scenario: Other configuration unchanged
- **WHEN** the repository has `.claude/settings.json`, a skill of the user's own under `.claude/skills/`, `AGENTS.md` and `CLAUDE.md`, and a user runs `dkt init`
- **THEN** those files are byte-for-byte unchanged

### Requirement: Existing files
For each shipped file that already exists, apart from `openspec/config.yaml`, `dkt init` SHALL:
- leave it alone and list it as `same` when it equals the file this release ships;
- replace it with this release's file and list it as `updated` when it equals the file an earlier release, v0.16.0 or later, shipped;
- otherwise keep it, list it as `kept`, report `schema-differs` or `skill-differs`, and exit with 1; with `--force`, replace it instead and list it as `overwritten`.

#### Scenario: Re-run with nothing changed
- **WHEN** a user runs `dkt init`, changes nothing, and runs it again
- **THEN** the second run lists every file as `same`, writes nothing, and exits with 0

#### Scenario: File from an earlier release
- **WHEN** `openspec/schemas/docket/templates/plan.md` equals the file an earlier release shipped
- **THEN** it is replaced with this release's file and listed as `updated`, and dkt exits with 0

#### Scenario: Edited skill kept
- **WHEN** `.claude/skills/docket/SKILL.md` differs from every shipped version and `--force` is absent
- **THEN** the file is unchanged and listed as `kept`, the finding `skill-differs` names it, and dkt exits with 1

#### Scenario: Edited skill forced
- **WHEN** the same file differs and a user runs `dkt init --force`
- **THEN** the file matches the shipped one, is listed as `overwritten`, and dkt exits with 0

### Requirement: Schema line in config.yaml
`dkt init` SHALL make `openspec/config.yaml` name the `docket` schema, leaving every other byte unchanged: add `schema: docket` first when the file has no top-level `schema`, and replace `schema: spec-driven`, OpenSpec's default. Another schema SHALL be kept, with or without `--force`, and reported as `config-schema-other`.

#### Scenario: Default schema replaced
- **WHEN** `openspec/config.yaml` is the one `openspec init` just wrote, with `schema: spec-driven`
- **THEN** that line reads `schema: docket`, every other line is unchanged, and the file is listed as `updated`

#### Scenario: Other schema kept
- **WHEN** `openspec/config.yaml` has `schema: team-flow` and a user runs `dkt init --force`
- **THEN** the `schema` line is unchanged, the finding `config-schema-other` names the file, and dkt exits with 1

### Requirement: Operations section in config.yaml
When `openspec/config.yaml` has no top-level `operations`, `dkt init` SHALL append the shipped `operations` section, which holds the archive guidance. A section equal to the one this release or an earlier release shipped SHALL be handled as an existing file is. Any other section SHALL be kept, with or without `--force`, since it may hold the user's own settings, and reported as `guidance-differs`.

#### Scenario: Section appended
- **WHEN** `openspec/config.yaml` has no `operations`
- **THEN** the shipped `operations` section is appended at its end, and every earlier byte is unchanged

#### Scenario: Edited guidance kept
- **WHEN** the `operations` section of `openspec/config.yaml` differs from every shipped version
- **THEN** the section is unchanged, the finding `guidance-differs` names the file, and dkt exits with 1

### Requirement: config.yaml created or unreadable
When `openspec/config.yaml` does not exist, `dkt init` SHALL create it holding only `schema: docket` and the shipped `operations` section. When its `schema` or `operations` is not a plain top-level entry that can be found line by line, such as a value on the next line, it SHALL write nothing, list the file in `unreadable`, and exit with 3.

#### Scenario: Config file created
- **WHEN** `openspec/` exists without `config.yaml` and a user runs `dkt init`
- **THEN** `openspec/config.yaml` holds `schema: docket` and the shipped `operations` section and nothing else

#### Scenario: Unreadable schema key
- **WHEN** `openspec/config.yaml` has `schema:` with its value indented on the next line and a user runs `dkt init`
- **THEN** no file is written, `unreadable` lists `openspec/config.yaml`, and dkt exits with 3

### Requirement: Symbolic links refused
When a path `dkt init` would write passes through a symbolic link inside the repository, it SHALL write nothing and SHALL NOT run `openspec init`; it SHALL report `symlink-refused` naming the link, and exit with 1.

#### Scenario: Linked skill folder
- **WHEN** `.claude/skills/docket` is a symbolic link to another directory and a user runs `dkt init --force`
- **THEN** nothing is written there or anywhere else, the finding `symlink-refused` names `.claude/skills/docket`, and dkt exits with 1

### Requirement: Ignored install paths
After writing, `dkt init` SHALL report `install-ignored` for each file it wrote or kept that git ignores and does not track, naming the ignore file, the line and the pattern, and SHALL exit with 1. It SHALL still write such a file, and SHALL NOT change any ignore file.

#### Scenario: Ignored skill reported
- **WHEN** `.gitignore` holds `.claude/` on its third line and a user runs `dkt init --tools claude`
- **THEN** `.claude/skills/docket/SKILL.md` is written, the finding `install-ignored` names it with `.gitignore`, line 3 and `.claude/`, `.gitignore` is byte-for-byte unchanged, and dkt exits with 1

### Requirement: Init result
The `result` of `dkt init` SHALL hold `files` and `openspec_init`. Each entry of `files` has a `path` relative to the repository root, a `kind` (`schema`, `config`, `skill`, or `openspec` for a file `openspec init` created) and a `status` (`created`, `updated`, `overwritten`, `same` or `kept`). `openspec_init` is the `--tools` value passed to `openspec init`, or null when it did not run. Without `--json`, the output SHALL list each file with its status.

#### Scenario: Files listed
- **WHEN** a user runs `dkt init --json --tools claude` in a repository without `openspec/`
- **THEN** `result.openspec_init` is `claude`, and `result.files` lists the files `openspec init` created with kind `openspec`, and Docket's files with status `created`

#### Scenario: Nothing to do
- **WHEN** a user runs `dkt init --json` in a repository already set up and unchanged since
- **THEN** `result.openspec_init` is null and every entry of `result.files` has status `same`

### Requirement: Next steps
The `result` of `dkt init` SHALL hold `next_steps`, a list in order, empty when there is nothing to do: commit the files listed as `created`, `updated` or `overwritten`, when there are any; reload the skills or start a new session, when this run created the `docket` skill; run docket-doctor, when there is any finding. Their wording is prompt text the user edits. Without `--json`, the output SHALL print them after the files.

#### Scenario: First setup
- **WHEN** a user runs `dkt init --json --tools claude` in a repository without `openspec/`
- **THEN** `result.next_steps` first says to commit the listed files, then to reload the skills

#### Scenario: Findings point to the doctor
- **WHEN** `dkt init` keeps an edited `SKILL.md` and reports `skill-differs`
- **THEN** the last entry of `result.next_steps` says to run docket-doctor
