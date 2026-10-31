# setup-inspection Specification

## Purpose
Defines what `dkt inspect` reports about a project's Docket setup, and the
`docket-doctor` command that the user invokes to run it, check what needs
judgment, and fix what it finds.

## Requirements

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

### Requirement: OpenSpec CLI inspected
`dkt inspect` SHALL locate the OpenSpec CLI as other commands do and read its version. It SHALL report `openspec-missing` when no CLI can be started and `openspec-unsupported`, naming the version and the accepted range, when the version is outside it. Neither SHALL make it exit with 4.

#### Scenario: Unsupported OpenSpec
- **WHEN** the OpenSpec CLI reports version 1.3.0 and a user runs `dkt inspect --json`
- **THEN** `result.openspec.version` is `1.3.0`, `result.openspec.supported` is false, the findings hold `openspec-unsupported`, and the command exits with 1

#### Scenario: No OpenSpec
- **WHEN** no OpenSpec CLI can be started
- **THEN** `result.openspec.version` is null and the findings hold `openspec-missing`

### Requirement: Workflow file inspected
`dkt inspect` SHALL report in `result.workflow` what `dkt workflow --json` reports in its `result`, and SHALL give the same findings. When no home directory can be found, it SHALL report `workflow-missing` with that reason instead of exiting with 4. An unreadable workflow file SHALL be listed in `unreadable`.

#### Scenario: Workflow reported
- **WHEN** the workflow file lacks `## Check` and a user runs `dkt inspect --json`
- **THEN** `result.workflow.status` is `incomplete` and the findings hold `workflow-incomplete` naming `Check`

### Requirement: Trailers on the default branch
For each active change whose folder is in the default branch's tree and whose plan there has a task, `dkt inspect` SHALL report `no-task-commits` when no commit reachable from the default branch's tip and not from the change's recorded base carries both `Change: <slug>` and a `Task:` trailer. A change whose base is missing or not an ancestor of the tip SHALL be skipped. `result.changes` SHALL list each such change, with the reason when skipped.

#### Scenario: Squashed change
- **WHEN** `add-x` is in the default branch's tree with a plan of two tasks, and the only commit since its base on the default branch carries no `Change: add-x` trailer
- **THEN** the findings hold `no-task-commits` for `add-x`

#### Scenario: Merged with a merge commit
- **WHEN** `add-x` reached the default branch through a merge commit whose second parent's history has commits with `Change: add-x` and `Task: 1`
- **THEN** no finding names `add-x`

#### Scenario: Base not an ancestor
- **WHEN** `add-x` is in the default branch's tree and its recorded base is not an ancestor of the default branch's tip
- **THEN** no finding names `add-x`, and `result.changes` lists `add-x` as skipped with the reason

### Requirement: Doctor runs only when invoked
Docket SHALL ship a `docket-doctor` command that runs only when the user invokes it. Its `SKILL.md` SHALL have `disable-model-invocation: true` in its frontmatter, and its folder SHALL hold `agents/openai.yaml` with `policy: allow_implicit_invocation: false`.

#### Scenario: Doctor not run on its own
- **WHEN** an agent in a repository set up by `dkt init` is asked to add a feature, or asked what a function does
- **THEN** it does not load `docket-doctor` and does not run `dkt inspect`

#### Scenario: Doctor invoked
- **WHEN** the user invokes `docket-doctor`
- **THEN** the agent runs `dkt inspect --json` before any other step of the doctor

### Requirement: Doctor judges the setup
After `dkt inspect`, the doctor SHALL check what needs judgment: whether this environment has what each workflow step names, and so which review fallback applies; where the workflow file and the user's other instructions in its context disagree; instructions that conflict with the framework, such as squash merges; and whether superpowers is available. It SHALL not read other user-level files unless the user agrees first.

#### Scenario: Drift found
- **WHEN** the workflow file names one effort for the cross-model reviewer and the user's instructions in the agent's context name another
- **THEN** the doctor lists the two, quoting each, as a problem

### Requirement: Doctor lists, then guides
The doctor SHALL list every problem it found, each with its reason and proposed fix, before fixing any. It SHALL then take them one at a time and act only after the user agrees: the workflow file through `docket-setup`; installed files by showing the differences, then `dkt init` or `dkt init --force`; the user's own instruction files never edited, only wording proposed. It SHALL end by running `dkt inspect` again.

#### Scenario: Problems listed first
- **WHEN** the user invokes `docket-doctor` and `dkt inspect` reports `skill-differs` and `workflow-stale`
- **THEN** the agent lists both problems, with their reasons, before it changes anything

#### Scenario: Edited skill fixed with consent
- **WHEN** the doctor reaches a `skill-differs` problem
- **THEN** it shows how the file differs from the shipped one and runs `dkt init --force` only after the user agrees

### Requirement: Project copy and schema key inspected
`dkt inspect` SHALL report a project's `openspec/schemas/docket/` as `dkt init` would treat it: `schema-project-copy`, saying `dkt init` removes it, when it equals a shipped schema, and `schema-edited`, saying it is used instead of the per-machine copy, when it equals none. It SHALL report `config-schema-default` when `openspec/config.yaml` is missing, has no `schema` key or names `spec-driven`, and `config-schema-other` when it names another schema.

#### Scenario: Project copy reported
- **WHEN** the project's `openspec/schemas/docket/` holds the schema dkt v0.9.1 shipped
- **THEN** the findings hold `schema-project-copy` naming `openspec/schemas/docket`, and the command exits with 1

#### Scenario: Default schema key reported
- **WHEN** `openspec/config.yaml` starts with `schema: spec-driven`
- **THEN** `result.files` lists `openspec/config.yaml` as `differs` with kind `config`, and the findings hold `config-schema-default` naming it
