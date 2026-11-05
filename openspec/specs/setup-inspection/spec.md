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
- **WHEN** the repository was set up by this release's `dkt init` and not changed since, and the OpenSpec CLI reports 1.14.0
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
`dkt inspect` SHALL locate the OpenSpec CLI as openspec-runtime says, read its version, and report in `result.openspec` the `version` (null when the CLI cannot be started), the `minimum` (`1.14.0`) and `supported`. It SHALL report `openspec-missing` when the CLI cannot be started and `openspec-unsupported` when the version is too old or cannot be read; neither SHALL make it exit with 4.

#### Scenario: Old OpenSpec
- **WHEN** the OpenSpec CLI reports 1.13.2 and a user runs `dkt inspect --json`
- **THEN** `result.openspec.version` is `1.13.2`, `result.openspec.supported` is false, the findings hold `openspec-unsupported`, and dkt exits with 1

#### Scenario: No OpenSpec
- **WHEN** no OpenSpec CLI can be started
- **THEN** `result.openspec.version` is null and the findings hold `openspec-missing`

### Requirement: Ignored files inspected
`dkt inspect` SHALL report `install-ignored` for each file `dkt init` writes or keeps that git ignores and does not track, whether or not the file exists yet, naming the ignore file, the line and the pattern. It SHALL change no ignore file.

#### Scenario: Ignored OpenSpec folder
- **WHEN** `.git/info/exclude` holds `openspec/` on its seventh line and a user runs `dkt inspect --json`
- **THEN** the finding `install-ignored` names `openspec/config.yaml` with `.git/info/exclude`, line 7 and `openspec/`, and `.git/info/exclude` is byte-for-byte unchanged

### Requirement: Commit trailers on the default branch
`dkt inspect` SHALL read the default branch's tip, not the worktree. For each folder under its `openspec/changes/` other than `archive` whose schema is `docket` and whose `plan.md` has a task, some commit reachable from the tip SHALL carry a `Change` trailer whose value is the change's name or starts with the name and `/`, read as `dkt sync` reads trailers. Otherwise it SHALL report `trailer-missing`, naming the change and its `plan.md`.

#### Scenario: Squash merge
- **WHEN** the default branch's tip holds `openspec/changes/add-x/` with a plan.md that has a task, and no commit reachable from the tip carries a `Change` trailer naming `add-x`
- **THEN** `dkt inspect --json` reports `trailer-missing` for `add-x` and exits with 1

#### Scenario: Merged with its trailers
- **WHEN** a commit reachable from the default branch's tip carries `Change: add-x/1`
- **THEN** no `trailer-missing` finding names `add-x`

#### Scenario: Change not on the default branch
- **WHEN** `add-y` exists only in the worktree, on a branch that is not the default branch
- **THEN** no `trailer-missing` finding names `add-y`

#### Scenario: Plan without tasks
- **WHEN** the default branch's tip holds `add-z` whose plan.md has no task, and no commit names `add-z`
- **THEN** no `trailer-missing` finding names `add-z`

### Requirement: Default branch in the inspect result
`dkt inspect` SHALL report in `result.default_branch` the name of the default branch the trailer check read, or null when it did not run. It SHALL skip that check, say why in the text output, and not fail for it, when there is no default branch or when the repository is a shallow clone, whose history may not hold the commits that carry the trailers.

#### Scenario: Inspect without a default branch
- **WHEN** no default branch can be found and a user runs `dkt inspect --json`
- **THEN** `result.default_branch` is null and no finding is `trailer-missing`

#### Scenario: Shallow clone
- **WHEN** the repository was cloned with `--depth 1`, and the default branch's tip holds `add-x`, with a plan.md that has a task, whose `Change` trailers are on earlier commits only
- **THEN** `dkt inspect --json` reports no `trailer-missing`, `result.default_branch` is null, and the text output says the check was skipped for a shallow clone

### Requirement: Main spec titles inspected
`dkt inspect` SHALL report `title-missing` for each main spec in the worktree that has no title, as change-lint's Main spec titles requirement says, naming its `spec.md`.

#### Scenario: Main spec with OpenSpec's first line
- **WHEN** `openspec/specs/booking/spec.md` starts with `# booking Specification` and a user runs `dkt inspect --json`
- **THEN** the findings hold `title-missing` naming `openspec/specs/booking/spec.md`, and dkt exits with 1
