## ADDED Requirements

### Requirement: Repository without a default branch
`dkt inspect` SHALL report in `result.default_branch` the branch whose trailers it checked. When no default branch can be found, as in a repository with no commit, it SHALL skip the trailer check, report `result.default_branch` as null and an empty `result.changes`, report everything else as usual, and SHALL NOT exit with 4 for that reason.

#### Scenario: Repository without commits
- **WHEN** a user runs `dkt inspect --json` in a git repository with no commit
- **THEN** `result.default_branch` is null, `result.changes` is empty, `result.openspec` and `result.files` are reported, no finding is about the default branch, and the command does not exit with 4

#### Scenario: Default branch found
- **WHEN** a user runs `dkt inspect --json` in a repository whose default branch is `main`
- **THEN** `result.default_branch` is `main`

## MODIFIED Requirements

### Requirement: Inspect writes nothing
`dkt inspect` SHALL change no file, in the repository or anywhere else. The only processes it SHALL start are git and the OpenSpec CLI's version query. It SHALL exit with 0 when it reports no finding and with 1 when it reports any.

#### Scenario: Nothing written
- **WHEN** a user runs `dkt inspect` in a repository where every installed file differs from the shipped one, the project holds a copy of a shipped schema, `openspec/config.yaml` names `spec-driven`, the workflow file is missing, and neither user-level `docket-init` folder exists
- **THEN** no file in the repository, under the user's config directory, under the per-machine schema directory, or under `~/.claude/` or `~/.agents/` is created, changed or removed

#### Scenario: Clean setup
- **WHEN** a repository was set up by the running release's `dkt init`, the OpenSpec CLI reports 1.14.3, and `dkt workflow` would report `ok`
- **THEN** `dkt inspect --json` reports no finding and exits with 0

### Requirement: Installed files inspected
`dkt inspect` SHALL check every file `dkt init` would install or keep, the per-machine schema, the user-level `docket-init` files and `config.yaml` too, by the rules `dkt init` uses, and SHALL report each one as `same`, `missing` or `differs` in `result.files`. A missing file SHALL give `schema-missing`, `user-skill-missing`, `skill-missing` or `agents-block-missing`; a differing one `schema-differs`, `user-skill-differs`, `skill-differs` or `agents-block-differs`; a malformed block `agents-block-malformed`; a path `dkt init` would refuse, `symlink-refused`.

#### Scenario: Before init
- **WHEN** a user runs `dkt inspect --json` in a git repository where `dkt init` never ran, on a machine without the per-machine schema
- **THEN** the findings hold `schema-missing` naming per-machine files, `skill-missing`, `agents-block-missing` and `config-schema-default`, and the command exits with 1

#### Scenario: Edited skill
- **WHEN** `.claude/skills/docket/SKILL.md` differs from the shipped file
- **THEN** `result.files` lists it as `differs` and the findings hold `skill-differs` naming it

#### Scenario: Machine copy differs
- **WHEN** the per-machine `schema.yaml` differs from the shipped file
- **THEN** `result.files` lists its absolute path as `differs` with kind `schema`, and the findings hold `schema-differs` naming it and saying that `dkt init --force` overwrites it

#### Scenario: Command not installed
- **WHEN** `~/.agents/skills/docket-init/` does not exist
- **THEN** `result.files` lists the absolute path of each of its shipped files as `missing` with kind `user-skill`, and the findings hold `user-skill-missing` naming each one

#### Scenario: CLAUDE.md imports AGENTS.md
- **WHEN** `CLAUDE.md` holds the line `@AGENTS.md` and `AGENTS.md` holds the shipped block
- **THEN** no finding names `CLAUDE.md`

#### Scenario: No home directory
- **WHEN** `XDG_DATA_HOME` and `CLAUDE_CONFIG_DIR` are unset and no home directory can be found
- **THEN** the findings hold `schema-missing` and `user-skill-missing` with that reason, and the command does not exit with 4
