# project-setup Specification

## Purpose
Defines how `dkt init` prepares a repository for the Docket workflow by
installing the `docket` OpenSpec schema and the `docket` skill and keeping
the Docket block in `AGENTS.md`, without touching the rest of the user's
agent or OpenSpec configuration.

## Requirements

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

### Requirement: Install the framework skill
`dkt init` SHALL write each skill folder shipped with the running Docket release, `docket`, `docket-setup` and `docket-doctor`, to both `.claude/skills/<name>/` and `.agents/skills/<name>/`. Files a user added under those directories SHALL be left alone.

#### Scenario: Fresh repository gets the skill
- **WHEN** a user runs `dkt init` in a git repository without the skills
- **THEN** `SKILL.md` under each of `.claude/skills/docket/`, `.claude/skills/docket-setup/`, `.claude/skills/docket-doctor/` and the same three folders under `.agents/skills/` matches the shipped file, and the command exits with 0

#### Scenario: Skill copies agree
- **WHEN** `dkt init` has run
- **THEN** for each skill the folder under `.claude/skills/` and the one under `.agents/skills/` hold byte-for-byte identical files, and each `SKILL.md` starts with frontmatter whose `name` is its folder's name and whose `description` is at most 1,024 characters

### Requirement: Framework skill content
The shipped skill SHALL hold the triage of work into four paths (full review, light review, check, direct merge) and the framework rules: what `dkt` reads or checks and where things go. For each path's review and verification steps it SHALL send the agent to the user's materialized workflow, located through `dkt workflow`. Its description SHALL say only when to use it.

#### Scenario: Skill loads for development work
- **WHEN** an agent with superpowers installed, in a repository set up by `dkt init`, is asked to add a feature
- **THEN** it loads the `docket` skill before it brainstorms or writes a file

#### Scenario: Skill not loaded for a question
- **WHEN** the same agent is asked what a function does
- **THEN** it answers without loading the `docket` skill

#### Scenario: Steps from the workflow
- **WHEN** `dkt workflow` reports `ok` and the agent takes the light review path
- **THEN** it does the steps under `## Light review` in the workflow file

### Requirement: Docket block
`dkt init` SHALL keep a Docket block in `AGENTS.md`: the lines from `<!-- docket:start -->` to `<!-- docket:end -->`, each marker alone on its line. It SHALL create `AGENTS.md` holding only the block when the file is missing, and append the block when the file has none, leaving text outside the markers byte-for-byte unchanged. An identical block SHALL be left alone; a differing one SHALL be reported and kept unless `--force` is given, which replaces it.

#### Scenario: Block added to an existing AGENTS.md
- **WHEN** a user runs `dkt init` where `AGENTS.md` exists without the block
- **THEN** the block is appended, the earlier content is byte-for-byte unchanged, and the command exits with 0

#### Scenario: AGENTS.md created
- **WHEN** a user runs `dkt init` where `AGENTS.md` does not exist
- **THEN** `AGENTS.md` holds the block and nothing else

#### Scenario: Edited block
- **WHEN** the block in `AGENTS.md` differs from the shipped block and `--force` is absent
- **THEN** `AGENTS.md` is unchanged, the result reports `agents-block-differs` for it, and the command exits with 1; with `--force`, only the lines from the start marker to the end marker are replaced

### Requirement: Docket block content
The shipped block SHALL tell an agent to use the `docket` skill at the start of development work, before brainstorming, planning or editing; to follow the skill where it disagrees with another skill, naming each superpowers default the skill replaces; that the steps of the four paths follow the user's materialized workflow even where the user's other instructions differ; and that otherwise the block does not override the user's own instructions.

#### Scenario: Skill wins over superpowers
- **WHEN** an agent with superpowers installed, in a repository set up by `dkt init`, takes a feature that needs a change through brainstorming
- **THEN** the design is written into the change folder, not `docs/superpowers/specs/`

#### Scenario: Workflow wins over user instructions
- **WHEN** the user's own instructions name one reviewer for light review and the materialized workflow names another
- **THEN** the agent on the light review path uses the one the workflow names

### Requirement: Malformed block
When `AGENTS.md` or `CLAUDE.md` has a start marker without an end marker, an end marker before its start marker, or more than one block, `dkt init` SHALL write no file, report `agents-block-malformed` for that file, and exit with 1.

#### Scenario: Malformed block
- **WHEN** `AGENTS.md` has `<!-- docket:start -->` and no end marker
- **THEN** no file is written, the result reports `agents-block-malformed` for `AGENTS.md`, and the command exits with 1

### Requirement: CLAUDE.md reaches the block
`dkt init` SHALL leave `CLAUDE.md` alone when it has a line `@AGENTS.md` or `@./AGENTS.md`, or is a symbolic link to `AGENTS.md`. When `CLAUDE.md` is missing, it SHALL create it holding `@AGENTS.md`. Otherwise it SHALL keep the Docket block in `CLAUDE.md` as well, by the rules for `AGENTS.md`.

#### Scenario: CLAUDE.md imports AGENTS.md
- **WHEN** `CLAUDE.md` holds the line `@AGENTS.md` and a user runs `dkt init`
- **THEN** `CLAUDE.md` is byte-for-byte unchanged and `AGENTS.md` holds the block

#### Scenario: CLAUDE.md created
- **WHEN** a user runs `dkt init` where `CLAUDE.md` does not exist
- **THEN** `CLAUDE.md` holds `@AGENTS.md` and nothing else

#### Scenario: CLAUDE.md without the import
- **WHEN** `CLAUDE.md` exists without importing `AGENTS.md` and a user runs `dkt init`
- **THEN** the block is appended to `CLAUDE.md`, and its earlier content is byte-for-byte unchanged

### Requirement: Workflow setup
Setup SHALL be the `docket-setup` skill, which the user can invoke at any time to write or revise the workflow file. When `dkt workflow` reports `missing` or `incomplete` at the start of development work, the `docket` skill SHALL have the agent run it before the work. Setup SHALL draft from `dkt workflow template`, the user's instructions the agent already has and the tools it can find, ask before reading other user-level files, show the whole draft, and write it, with the template's hash, only after the user agrees.

#### Scenario: Setup before the first work
- **WHEN** an agent in a repository set up by `dkt init`, with no workflow file, is asked to add a feature
- **THEN** it runs `dkt workflow` and `dkt workflow template` and shows a draft of the four sections before any other step of the work, and writes no workflow file until the user agrees

#### Scenario: Setup invoked by the user
- **WHEN** `dkt workflow` reports `ok` and the user invokes `docket-setup` to change the reviewer on the light review path
- **THEN** the agent shows the whole revised file and writes nothing until the user agrees

### Requirement: Stale workflow
When `dkt workflow` reports `stale`, the skill SHALL have the agent keep following the workflow file and tell the user once that it was made from an older template, what the current template does differently, and that setup can update it.

#### Scenario: Stale workflow kept
- **WHEN** `dkt workflow` reports `stale` and the agent takes the light review path
- **THEN** it does the steps under `## Light review` in the workflow file and tells the user once that the workflow is out of date

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
