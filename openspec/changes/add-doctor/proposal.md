# Proposal

## Why

A project's Docket setup can drift without anyone noticing: an installed
skill edited or out of date, an OpenSpec CLI outside
the supported range, a workflow file that no longer matches the user's
tools or instructions, a pull request squashed so its trailers are gone.
Each surfaces only when some later command fails. The user needs one thing
to ask, when they choose, that finds these and helps fix them
(`docket/ideas/2026-10-29-framework-skill.md`, the third of its three
changes).

## What Changes

- Docket's shipped agent files fall into three kinds, and one item may be
  more than one: a **skill** is loaded by the agent when a task matches
  its description; a **command** runs only when the user invokes it; the
  **CLI** (`dkt`) is a program that involves no agent. Work that involves
  the agent (judging, asking, guiding) is never in the CLI.
- New CLI command **`dkt inspect [--json]`**: reads, never writes, and
  reports as findings:
  - every file `dkt init` installs that is missing or differs from the
    running release;
  - whether an OpenSpec CLI can be started and its version is supported;
  - the workflow file's status, as `dkt workflow` reports it;
  - active changes on the default branch whose commits lost their
    `Change`/`Task` trailers, as a squashed pull request does.
- New command **`docket-doctor`**, installed by `dkt init`, which runs only
  when the user invokes it (`/docket-doctor`, `$docket-doctor`). It runs
  `dkt inspect`, then checks what needs judgment: whether this environment
  has what each workflow step names, and so which review fallback applies;
  where the workflow file and the user's other instructions disagree;
  instructions that conflict with the framework, such as squash merges;
  whether superpowers is available. It lists every problem with its reason
  first, then guides the user through the fixes one at a time.
- **Setup becomes both a skill and a command**: it moves from the `docket`
  skill's `setup.md` into its own `docket-setup` skill. The `docket` skill
  still sends the agent there when the workflow file is missing or lacks a
  path, and the user can now invoke it to revise the file at any time.
- `dkt init` installs the three skill folders (`docket`, `docket-setup`,
  `docket-doctor`) under both `.claude/skills/` and `.agents/skills/`.
  **BREAKING**: `docket/setup.md` is no longer shipped. No project outside
  this repository uses Docket yet, so nothing handles the upgrade; this
  repository deletes its own copies.

## Non-Goals

- No external systems, optional or not: `dkt inspect` never asks a forge
  (such as GitHub's merge settings); it reads only the repository, the
  OpenSpec CLI's version and the user's workflow file.
- Doctor never runs on its own: no hook, no check at the start of work.
- `dkt inspect` writes nothing and judges no prose; the doctor never edits
  the user's own instruction files, it proposes wording.
- No review of how well the process pays off from development data; it is
  recorded as a separate idea (`docket/ideas/2026-10-30-process-review.md`).
- No upgrade path from v0.7.0: files an earlier release shipped and this
  one does not are neither reported nor removed.

## Builds On

- `setup.Init` and its decide phase (`internal/setup/setup.go`): the
  missing/differs/malformed rules for every installed file, `skill-differs`
  and `agents-block-*` findings; inspect reuses them without writing.
- `docket.Skill` (`schema.go`): the embedding of the skill, replaced by
  `docket.Skills`, which embeds the three skill folders.
- `openspec.New` (`internal/openspec`): locating the CLI and checking its
  version.
- `workflow.Locate`, `workflow.Read`, `workflow.Inspect` and the
  `WorkflowResult` of `dkt workflow` (`internal/cli/cmd_workflow.go`).
- The plan check of `dkt archive` (`internal/archive/run.go`, rule
  `no-task-commits`), `change.NamedTasks` and `Git.Commits`: the trailer
  test inspect applies on the default branch.
- `scan.ResolveDefaultBranch` and `Git.ListDirs`: the default branch and
  the active changes in its tree.
- `cli.Outcome`, findings, `unreadable` and exit codes (`envelope.go`).
- The `project-setup` capability (installed files, the skill, setup) and
  `materialized-workflow` (the workflow status).

## Capabilities

### New Capabilities

- `setup-inspection`: what `dkt inspect` reports about a project's Docket
  setup, and the `docket-doctor` command that runs it, checks what needs
  judgment and guides the fixes.

### Modified Capabilities

- `project-setup`: `dkt init` installs three skill folders instead of one;
  setup lives in the `docket-setup` skill and the user can invoke it.

## Impact

- New `internal/cli/cmd_inspect.go`; a read-only entry point in
  `internal/setup`; `schema.go` embeds three skill folders.
- Shipped prompt text: `.agents/skills/` and `.claude/skills/` gain
  `docket-setup/` (the setup text, moved) and `docket-doctor/` (with
  `agents/openai.yaml`); `docket/SKILL.md` points to `docket-setup`; the
  prompt path list in `AGENTS.md`.
- Both READMEs, `docs/ARCHITECTURE.md`, the framework-skill idea (status),
  a new idea file for the process review.
- This repository: `.claude/skills/docket/setup.md` and
  `.agents/skills/docket/setup.md` are deleted.

## Implementation Contract

### Observable Behavior

- In a project set up by the running release, with an OpenSpec CLI in
  range and an `ok` workflow file, `dkt inspect` prints no finding and
  exits with 0; any problem gives a finding and exit 1.
- `/docket-doctor` (or `$docket-doctor`) lists every problem it found, with
  its reason, before fixing any, then offers the fixes one at a time and
  writes nothing the user has not agreed to.
- An agent asked to add a feature, or asked a question, never runs the
  doctor on its own.
- The user can invoke `docket-setup` to revise their workflow file; the
  `docket` skill still runs it when the file is missing or lacks a path.

### Interfaces

- `dkt inspect [--json] [--default-branch <name>]`. `result` has `files`
  (`path`, `kind`, `status`: `same`, `missing`, `differs`),
  `openspec` (`version`, or null when no CLI could be started; `supported`),
  `workflow` (the `result` of `dkt workflow --json`) and `changes` (each
  active change on the default branch with a task: `slug`, and `skipped`
  with the reason when its base is missing or not an ancestor of the tip).
  A `files` entry is one file `dkt init` would install or keep; `kind` is
  `schema`, `skill` or `block`; `path` is relative to
  the repository root; a `CLAUDE.md` that imports or links to `AGENTS.md`
  is not listed.
- Findings: `schema-missing`, `skill-missing`, `agents-block-missing`,
  `schema-differs`, `skill-differs`, `agents-block-differs`,
  `agents-block-malformed`, `symlink-refused`, `openspec-missing`,
  `openspec-unsupported`, `workflow-missing`, `workflow-incomplete`,
  `workflow-stale`, `no-task-commits`.
- Shipped skill folders: `docket/`, `docket-setup/`, `docket-doctor/`, each
  under `.claude/skills/` and `.agents/skills/`. `docket-doctor/SKILL.md`
  has `disable-model-invocation: true`; `docket-doctor/agents/openai.yaml`
  has `policy: allow_implicit_invocation: false`.

### Failure Modes

- Not in a git repository, or no default branch can be found: exit 4, as
  for every command.
- The workflow file or an installed file cannot be read: listed in
  `unreadable`, exit 3, and the other checks still reported.
- No home directory: the workflow part is reported as `workflow-missing`
  with the reason, not exit 4, so the other checks still show.
- No OpenSpec CLI, or one out of range: a finding and exit 1, never exit 4:
  inspect reports the environment rather than needing it.
- A path `dkt init` would refuse because it passes through a symbolic
  link: `symlink-refused`, as `dkt init` reports it, since the fix runs
  `dkt init`.

### Acceptance Criteria

- `setup-inspection#nothing-written`
- `setup-inspection#clean-setup`
- `setup-inspection#before-init`
- `setup-inspection#edited-skill`
- `setup-inspection#claude-md-imports-agents-md`
- `setup-inspection#unsupported-openspec`
- `setup-inspection#no-openspec`
- `setup-inspection#workflow-reported`
- `setup-inspection#squashed-change`
- `setup-inspection#merged-with-a-merge-commit`
- `setup-inspection#base-not-an-ancestor`
- `setup-inspection#doctor-not-run-on-its-own`
- `setup-inspection#doctor-invoked`
- `setup-inspection#drift-found`
- `setup-inspection#problems-listed-first`
- `setup-inspection#edited-skill-fixed-with-consent`
- `project-setup#fresh-repository-gets-the-skill`
- `project-setup#skill-copies-agree`
- `project-setup#setup-before-the-first-work`
- `project-setup#setup-invoked-by-the-user`

## Change Log

| Date | Source | Change | Impact |
|---|---|---|---|
