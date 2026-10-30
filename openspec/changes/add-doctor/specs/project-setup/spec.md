# Spec Delta

## MODIFIED Requirements

### Requirement: Files init writes
Apart from what `openspec init --tools none` creates, `dkt init` SHALL write only under `openspec/schemas/docket/` and under `.claude/skills/<name>/` and `.agents/skills/<name>/` for `docket`, `docket-setup` and `docket-doctor`, the Docket block of `AGENTS.md` and `CLAUDE.md`, and a missing `AGENTS.md` or `CLAUDE.md`. It SHALL NOT write through a symbolic link: when any path it would write passes through one, it SHALL write nothing, name the link, and exit with 1.

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
`dkt init` SHALL write each skill folder shipped with the running Docket release, `docket`, `docket-setup` and `docket-doctor`, to both `.claude/skills/<name>/` and `.agents/skills/<name>/`. Files a user added under those directories SHALL be left alone.

#### Scenario: Fresh repository gets the skill
- **WHEN** a user runs `dkt init` in a git repository without the skills
- **THEN** `SKILL.md` under each of `.claude/skills/docket/`, `.claude/skills/docket-setup/`, `.claude/skills/docket-doctor/` and the same three folders under `.agents/skills/` matches the shipped file, and the command exits with 0

#### Scenario: Skill copies agree
- **WHEN** `dkt init` has run
- **THEN** for each skill the folder under `.claude/skills/` and the one under `.agents/skills/` hold byte-for-byte identical files, and each `SKILL.md` starts with frontmatter whose `name` is its folder's name and whose `description` is at most 1,024 characters

### Requirement: Workflow setup
Setup SHALL be the `docket-setup` skill, which the user can invoke at any time to write or revise the workflow file. When `dkt workflow` reports `missing` or `incomplete` at the start of development work, the `docket` skill SHALL have the agent run it before the work. Setup SHALL draft from `dkt workflow template`, the user's instructions the agent already has and the tools it can find, ask before reading other user-level files, show the whole draft, and write it, with the template's hash, only after the user agrees.

#### Scenario: Setup before the first work
- **WHEN** an agent in a repository set up by `dkt init`, with no workflow file, is asked to add a feature
- **THEN** it runs `dkt workflow` and `dkt workflow template` and shows a draft of the four sections before any other step of the work, and writes no workflow file until the user agrees

#### Scenario: Setup invoked by the user
- **WHEN** `dkt workflow` reports `ok` and the user invokes `docket-setup` to change the reviewer on the light review path
- **THEN** the agent shows the whole revised file and writes nothing until the user agrees
