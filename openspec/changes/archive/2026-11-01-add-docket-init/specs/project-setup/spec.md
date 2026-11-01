## ADDED Requirements

### Requirement: Install the init command
`dkt init` SHALL write the `docket-init` skill folder shipped with the running Docket release to `<claude>/skills/docket-init/` and `<home>/.agents/skills/docket-init/`, where `<claude>` is `CLAUDE_CONFIG_DIR` when it is set and not empty and `<home>/.claude` otherwise, and SHALL list each file by its absolute path. A relative `CLAUDE_CONFIG_DIR` SHALL be an error, as a relative `XDG_DATA_HOME` is. A differing file SHALL be reported as `user-skill-differs`.

#### Scenario: Fresh machine gets the command
- **WHEN** neither `~/.claude/skills/docket-init/` nor `~/.agents/skills/docket-init/` exists and a user runs `dkt init` in a project
- **THEN** `SKILL.md` and `agents/openai.yaml` under each folder match the shipped files, the result lists the four as `written` with their absolute paths, and the command exits with 0

#### Scenario: Claude config dir
- **WHEN** `CLAUDE_CONFIG_DIR` is `/srv/claude-work` and a user runs `dkt init`
- **THEN** the Claude Code copy is written under `/srv/claude-work/skills/docket-init/`, nothing is written under `~/.claude/`, and the Codex copy is still written under `~/.agents/skills/docket-init/`

#### Scenario: Edited command copy
- **WHEN** `~/.agents/skills/docket-init/SKILL.md` differs from the shipped file and `--force` is absent
- **THEN** the file is unchanged, the result reports `user-skill-differs` naming its absolute path, and the command exits with 1; with `--force`, it matches the shipped file and is listed as `overwritten`

#### Scenario: Copies agree
- **WHEN** `dkt init` has run
- **THEN** the two `docket-init` folders hold byte-for-byte identical files

## MODIFIED Requirements

### Requirement: Files init writes
Apart from what `openspec init --tools none` creates, `dkt init` SHALL write only the per-machine schema, the user-level `docket-init` folders, the `schema` key of `openspec/config.yaml`, the skill folders, the Docket block of `AGENTS.md` and `CLAUDE.md`, and a missing `AGENTS.md` or `CLAUDE.md`, and SHALL remove only a project copy of a shipped schema. When a path it would change passes through a symbolic link in the repository, or at or below the per-machine `docket` folder or a user-level `docket-init` folder, it SHALL change nothing, name the link, and exit with 1.

#### Scenario: Other configuration unchanged
- **WHEN** `dkt init` succeeds in a repository with `.claude/settings.json`, a skill of its own under `.claude/skills/`, and an `openspec/config.yaml` that names `docket` and has a `context`
- **THEN** those files are byte-for-byte unchanged, and the output holds no section to paste

#### Scenario: Symlinked schema file refused
- **WHEN** the per-machine `templates/proposal.md` is a symbolic link to another file and a user runs `dkt init --force`
- **THEN** the file the link points to is byte-for-byte unchanged, no schema file is written, the result names the link, and the command exits with 1

#### Scenario: Symlinked skill directory refused
- **WHEN** `.claude/skills/docket` is a symbolic link to another directory and a user runs `dkt init --force`
- **THEN** nothing in that directory or anywhere else is written, the result names the link, and the command exits with 1

#### Scenario: Symlinked command folder refused
- **WHEN** `~/.claude/skills/docket-init` is a symbolic link to another directory and a user runs `dkt init --force`
- **THEN** nothing in that directory or anywhere else is written, the result names the link, and the command exits with 1

#### Scenario: User skills directory linked
- **WHEN** `~/.claude/skills` is a symbolic link to another directory and a user runs `dkt init`
- **THEN** the Claude Code copy is written under the directory the link points to, and the command exits with 0

#### Scenario: Personal skills unchanged
- **WHEN** `~/.claude/skills/` holds a skill of the user's own and `~/.claude/settings.json` exists, and a user runs `dkt init`
- **THEN** both are byte-for-byte unchanged
