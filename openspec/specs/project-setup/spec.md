# project-setup Specification

## Purpose
Defines how `dkt init` prepares a repository for the Docket workflow by
installing the `docket` OpenSpec schema, without touching the user's own
agent or OpenSpec configuration.

## Requirements

### Requirement: Install the schema
`dkt init` SHALL write the `docket` schema shipped with the running Docket release to `openspec/schemas/docket/`. When `openspec/` does not exist, it SHALL first run `openspec init --tools none`.

#### Scenario: Fresh repository
- **WHEN** a user runs `dkt init` in a git repository without `openspec/`
- **THEN** `openspec/` exists, `openspec/schemas/docket/` matches the shipped schema, and the command exits with 0

#### Scenario: Existing OpenSpec project
- **WHEN** a user runs `dkt init` where `openspec/` already exists
- **THEN** Docket does not run `openspec init` and only adds the schema

### Requirement: Existing installed files
For each schema or skill file that already exists, `dkt init` SHALL leave an identical file untouched. It SHALL leave a differing file in place and report it, unless `--force` is given, in which case it SHALL overwrite the file and report that it did.

#### Scenario: Locally edited template
- **WHEN** `openspec/schemas/docket/templates/proposal.md` differs from the shipped file and `--force` is absent
- **THEN** the file is unchanged, the result names it, and the command exits with 1

#### Scenario: Edited skill file
- **WHEN** `.agents/skills/docket/SKILL.md` differs from the shipped file and `--force` is absent
- **THEN** the file is unchanged, the result reports `skill-differs` for it, and the command exits with 1

#### Scenario: Re-run with no differences
- **WHEN** a user runs `dkt init` a second time with nothing changed in between
- **THEN** the second run writes nothing and exits with 0

### Requirement: Files init writes
Apart from what `openspec init --tools none` creates, `dkt init` SHALL write only under `openspec/schemas/docket/`, `.claude/skills/docket/` and `.agents/skills/docket/`, the Docket block of `AGENTS.md` and `CLAUDE.md`, and a missing `AGENTS.md` or `CLAUDE.md`. It SHALL NOT write through a symbolic link: when any path it would write passes through one, it SHALL write nothing, name the link, and exit with 1.

#### Scenario: Other configuration unchanged
- **WHEN** `dkt init` succeeds in a repository with `.claude/settings.json`, a skill of its own under `.claude/skills/`, and `openspec/config.yaml`
- **THEN** those files are byte-for-byte unchanged, and the output holds no section to paste

#### Scenario: Symlinked schema file refused
- **WHEN** `openspec/schemas/docket/templates/proposal.md` is a symbolic link to AGENTS.md and a user runs `dkt init --force`
- **THEN** AGENTS.md is byte-for-byte unchanged, no schema file is written, the result names the link, and the command exits with 1

#### Scenario: Symlinked skill directory refused
- **WHEN** `.claude/skills/docket` is a symbolic link to another directory and a user runs `dkt init --force`
- **THEN** nothing in that directory or anywhere else is written, the result names the link, and the command exits with 1

### Requirement: Install the framework skill
`dkt init` SHALL write the `docket` skill shipped with the running Docket release to both `.claude/skills/docket/` and `.agents/skills/docket/`. Files a user added under those directories SHALL be left alone.

#### Scenario: Fresh repository gets the skill
- **WHEN** a user runs `dkt init` in a git repository without the skill
- **THEN** `.claude/skills/docket/SKILL.md` and `.agents/skills/docket/SKILL.md` match the shipped skill, and the command exits with 0

#### Scenario: Skill copies agree
- **WHEN** `dkt init` has run
- **THEN** the two `SKILL.md` files are byte-for-byte identical, and each starts with frontmatter whose `name` is `docket` and whose `description` is at most 1,024 characters

### Requirement: Framework skill content
The shipped skill SHALL hold the triage of work into four paths (full review, light review, check, direct merge) and each path's steps, naming review steps by capability, such as a cross-model review or a fresh agent, never by product. A required review SHALL fall back from a cross-model review to a same-model reviewer in a fresh context, then to a note in the pull request that the work had no independent review. Its description SHALL say only when to use it.

#### Scenario: Skill loads for development work
- **WHEN** an agent with superpowers installed, in a repository set up by `dkt init`, is asked to add a feature
- **THEN** it loads the `docket` skill before it brainstorms or writes a file

#### Scenario: Skill not loaded for a question
- **WHEN** the same agent is asked what a function does
- **THEN** it answers without loading the `docket` skill

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
The shipped block SHALL tell an agent to use the `docket` skill at the start of development work, before brainstorming, planning or editing; to follow the skill where it disagrees with another skill, naming each superpowers default the skill replaces; and that the block does not override the user's own instructions.

#### Scenario: Skill wins over superpowers
- **WHEN** an agent with superpowers installed, in a repository set up by `dkt init`, takes a feature that needs a change through brainstorming
- **THEN** the design is written into the change folder, not `docs/superpowers/specs/`

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
