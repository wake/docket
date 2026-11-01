# setup-inspection Specification

## Purpose
Defines what `dkt inspect` reports about a project's Docket setup, and the
`docket-doctor` command that the user invokes to run it, check what needs
judgment, and fix what it finds.

## Requirements

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
For each active change whose folder is in the default branch's tree and whose plan there has a task, `dkt inspect` SHALL report `no-task-commits` when no commit reachable from the default branch's tip and not from the change's recorded base carries both `Change: <slug>` and a `Task:` trailer. A change whose base is missing or not an ancestor of the tip, or that is a change of another schema, SHALL be skipped. `result.changes` SHALL list each such change, with the reason when skipped.

#### Scenario: Squashed change
- **WHEN** `add-x` is in the default branch's tree with a plan of two tasks, and the only commit since its base on the default branch carries no `Change: add-x` trailer
- **THEN** the findings hold `no-task-commits` for `add-x`

#### Scenario: Merged with a merge commit
- **WHEN** `add-x` reached the default branch through a merge commit whose second parent's history has commits with `Change: add-x` and `Task: 1`
- **THEN** no finding names `add-x`

#### Scenario: Base not an ancestor
- **WHEN** `add-x` is in the default branch's tree and its recorded base is not an ancestor of the default branch's tip
- **THEN** no finding names `add-x`, and `result.changes` lists `add-x` as skipped with the reason

#### Scenario: Change of another schema skipped
- **WHEN** `add-x` is in the default branch's tree with `schema: spec-driven` and a `plan.md` with a task, and no commit carries `Change: add-x`
- **THEN** no finding names `add-x`, and `result.changes` lists `add-x` as skipped, saying it was made with the `spec-driven` schema

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

### Requirement: Repository without a default branch
`dkt inspect` SHALL report in `result.default_branch` the branch whose trailers it checked. When no default branch can be found, as in a repository with no commit, it SHALL skip the trailer check, report `result.default_branch` as null and an empty `result.changes`, report everything else as usual, and SHALL NOT exit with 4 for that reason.

#### Scenario: Repository without commits
- **WHEN** a user runs `dkt inspect --json` in a git repository with no commit
- **THEN** `result.default_branch` is null, `result.changes` is empty, `result.openspec` and `result.files` are reported, no finding is about the default branch, and the command does not exit with 4

#### Scenario: Default branch found
- **WHEN** a user runs `dkt inspect --json` in a repository whose default branch is `main`
- **THEN** `result.default_branch` is `main`

### Requirement: Ignored files inspected
`dkt inspect` SHALL report `install-ignored`, as `dkt init` would after writing, for each file in the repository that `dkt init` would install or keep and that git ignores and does not track, whether or not the file exists yet. It SHALL change no ignore file.

#### Scenario: Ignored file inspected
- **WHEN** `.git/info/exclude` holds `openspec/` on its seventh line and a user runs `dkt inspect --json`
- **THEN** the findings hold `install-ignored` naming `openspec/config.yaml` with `.git/info/exclude`, line 7 and `openspec/`, and `.git/info/exclude` is byte-for-byte unchanged

### Requirement: Main spec titles inspected
`dkt inspect` SHALL report `title-missing` for each main spec under the worktree's `openspec/specs/` that has no title, as `dkt lint` does when no change is named. When `openspec` itself is a symbolic link (`symlink-refused`), it SHALL read nothing under it for this check.

#### Scenario: Main spec title inspected
- **WHEN** `openspec/specs/booking/spec.md` exists without a `docket.yaml` beside it
- **THEN** `dkt inspect` reports `title-missing` naming `openspec/specs/booking/docket.yaml` and exits with 1

### Requirement: Changes of another schema inspected
`dkt inspect` SHALL report `other-schema` for each active change in the worktree's `openspec/changes/` that is a change of another schema, naming its `.openspec.yaml`, its schema, and how many checkboxes of its `tasks.md` are ticked out of how many. When `openspec` itself is a symbolic link (`symlink-refused`), it SHALL read nothing under it for this check.

#### Scenario: Change of another schema inspected
- **WHEN** `openspec/changes/add-x/.openspec.yaml` holds `schema: spec-driven` and its `tasks.md` has four checkboxes, none ticked
- **THEN** `dkt inspect` reports `other-schema` naming `openspec/changes/add-x/.openspec.yaml`, `spec-driven` and 0 of 4

### Requirement: Doctor fixes adoption problems
The doctor SHALL take each `install-ignored`, `title-missing` and `other-schema` problem as the steps it lists, and act only after the user agrees: for an ignore rule, first show which files the change would let git see and whether the ignore file is shared or local; for titles, list a title drafted from each spec's Purpose, then write them together; for a change of another schema, propose converting it only when no step is ticked.

#### Scenario: Ignore rule changed with consent
- **WHEN** the doctor reaches `install-ignored` for files under `.claude/skills/`, ignored by `.claude/` in `.gitignore`
- **THEN** it proposes a rule that lets in `.claude/skills/`, lists the files git would then see, says `.gitignore` is shared with everyone who clones the project, and edits it only after the user agrees

#### Scenario: Titles written with consent
- **WHEN** the doctor reaches `title-missing` for three main specs
- **THEN** it lists a drafted title for each of the three before writing any, and writes the three `docket.yaml` files only after the user agrees

#### Scenario: Unstarted change converted
- **WHEN** the doctor reaches `other-schema` for `add-x` with 0 of 4 steps ticked and the user agrees to convert it
- **THEN** the agent converts `add-x` to the `docket` schema by the conversion guide shipped with the doctor, and `dkt lint add-x` then reports no finding

#### Scenario: Started change left
- **WHEN** the doctor reaches `other-schema` for `add-x` with 2 of 4 steps ticked
- **THEN** it proposes no conversion, says to finish `add-x` with its own files and archive it with `dkt archive`, and lists any capability `add-x` adds that has no title
