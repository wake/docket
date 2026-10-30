# Design

## Context

See proposal.md for why. What exists today:

- `setup.Init` decides the status of every file it installs (schema files,
  skill files, the block in `AGENTS.md` and `CLAUDE.md`) and checks every
  path for symbolic links before it writes anything. Only `dkt init`
  reports `skill-differs` and the other installation findings, and it
  writes while doing so.
- The skill is one embedded folder (`docket.Skill`), installed under
  `.claude/skills/docket/` and `.agents/skills/docket/`; `setup.md` is a
  second file in it that `SKILL.md` sends the agent to.
- `openspec.New` locates the OpenSpec CLI and checks its version; commands
  that need OpenSpec exit with 4 when it is missing or out of range.
- `dkt workflow` reports the workflow file; with no home directory it exits
  with 4.
- `dkt archive`'s plan check reports `no-task-commits` when no commit since
  a change's base carries `Change: <slug>` with a `Task:` trailer: the sign
  of a squashed pull request, found only at release time.

## Goals / Non-Goals

**Goals:**

- One read-only command that reports every mechanical problem with the
  setup, and one user-invoked command that adds what needs judgment and
  walks the user through the fixes.
- Setup that the user can start, not only the `docket` skill.
- Nothing in `dkt` judges, asks or guides.

**Non-Goals:**

- Running anything automatically (hooks, a check at the start of work).
- Reading anything outside the repository, the OpenSpec CLI's version and
  the user's workflow file; no forge, no agent configuration.
- Repairing anything from `dkt`.

## Approaches

Three ways were considered; the maintainer chose A.

- **A. `dkt inspect` plus a `docket-doctor` command (chosen).** The CLI
  answers every mechanical check in one call; the command does the
  judgment, the listing and the guided fixes. Costs: `dkt init` installs
  two more skill folders, and the prompt path list and the READMEs grow.
- **B. No new CLI command; the command calls existing ones.** A read-only
  `dkt init --check` and `dkt workflow` would cover two checks, but the
  OpenSpec version and the trailers on the default branch have no command,
  so new code is needed anyway, spread over several commands, and the
  squash check would fall to the agent reading `git log`, which is not
  repeatable.
- **C. The doctor as a third file of the `docket` skill, like `setup.md`.**
  It would load passively, against the requirement that the doctor runs
  only when asked, and the `docket` description, verified to load for
  development work and not for questions, would have to cover "check the
  setup" as well.

## Decisions

### Three kinds, and an item may be more than one

- A **skill** is loaded by the agent when a task matches its description;
  a **command** runs only when the user invokes it; the **CLI** is `dkt`,
  which involves no agent. Work that involves the agent never goes in the
  CLI.
- `docket` is a skill (and, by default, the user can invoke it too).
  `docket-setup` is a skill and a command: the `docket` skill sends the
  agent to it, and the user can invoke it. `docket-doctor` is a command
  only. `dkt inspect` is CLI.

### Commands are skill folders with model invocation turned off

- `docket-doctor/SKILL.md` has `disable-model-invocation: true` (Claude
  Code: the agent cannot invoke it on its own, the user can with
  `/docket-doctor`), and `docket-doctor/agents/openai.yaml` has
  `policy: allow_implicit_invocation: false` (Codex: only an explicit
  `$docket-doctor` runs it).
- Not `.claude/commands/`: Claude Code's documentation prefers the skill
  format for new work (a command file and a skill make the same `/name`),
  and Codex has no project-level command file. One format serves both.
- Both copies of every skill folder stay byte-for-byte identical, the
  `agents/` folder included: Claude Code treats it as a supporting file,
  and Codex reads `SKILL.md`'s `name` and `description`.
- Checked with codex-cli 0.153.4's `codex debug prompt-input`, which renders
  what the model sees without calling it: a skill whose frontmatter has
  `disable-model-invocation: true` and no `agents/openai.yaml` still loads
  and is listed (the unknown key is tolerated, and does not stop implicit
  use); with `agents/openai.yaml` setting
  `allow_implicit_invocation: false` it is not listed to the model. Codex's
  documentation says an explicit `$name` still runs it; the trial confirms
  that once.

### The CLI is named `inspect`

`dkt doctor` would read as if the CLI were the doctor. `check` is a triage
path, `setup` is the skill that writes the workflow file, and `status`
reports changes' progress.

### Inspect reuses init's decisions

`setup.Init` is split into deciding and writing, as a structural commit of
its own: `dkt init`'s output, findings and exit codes do not change, and
its existing tests pass unchanged. A new read-only entry point returns the
decided targets and problems. `dkt inspect` runs it without `--force` and
maps the statuses: a file init would
write is `missing`, one it would keep is `same`, one it would leave
because it differs is `differs`. Symbolic-link problems become
`symlink-refused` and malformed blocks `agents-block-malformed`, exactly as
`dkt init` reports them; unreadable paths go to `unreadable`. A
`CLAUDE.md` that imports `AGENTS.md` is no target, so it gives no finding.
Missing files get their own rules (`schema-missing`, `skill-missing`,
`agents-block-missing`) because `dkt init` never reports a missing file:
it writes it.

`result.files` has one entry per decided target: every embedded file under
each of the two skill roots, every schema file, `AGENTS.md`, and
`CLAUDE.md` when it is a target. `kind` is the installer's kind (`schema`,
`skill`, `block`); `path` is relative to the repository root.

### No upgrade handling

The maintainer's call: no project outside this repository uses Docket yet,
so nothing reports or removes files an earlier release shipped and this
one drops (`docket/setup.md`). This repository deletes its own two copies
in this change.

### OpenSpec is reported, not required

Inspect calls the same lookup and version check other commands use
(`openspec.New`, `Runner.CheckVersion`) and turns their typed errors into
findings with exit 1: `UnavailableError` gives `openspec-missing` and a
null version; `UnsupportedError` gives `openspec-unsupported`, with its
`Found` as the version. `supported` is true only when the check passes. It
runs no other OpenSpec command; the runner needs no change.

### The workflow part is `dkt workflow`'s

The code behind `dkt workflow --json` is shared, so `result.workflow` and
its findings are the same. With no home directory, inspect reports
`workflow-missing` with the reason and a null path, so the other checks
still show; `dkt workflow` itself keeps exiting with 4.

### The trailer check is archive's, run early

For each active change folder in the default branch's tree
(`Git.ListDirs`, as archive's preconditions list them; not `scan.Scan`,
which reads every worktree and branch), inspect reads its `docket.yaml`
(base) and `plan.md` from that tree; a
task is a `### Task N` section, as `change.ParsePlan` reads it. With at
least one task, it collects the commits reachable from the default
branch's tip and not from the base and applies the same test as
`dkt archive` (`change.NamedTasks`): no commit with `Change: <slug>` and a
`Task:` trailer gives `no-task-commits`. A change merged with a merge
commit and waiting for release has those commits, so the normal state
between merge and release gives no finding. A change with no recorded
base, or a base that is not an ancestor of the tip (a rewritten history),
is skipped and listed with the reason; `dkt archive`'s base check is where
such a base is reported. Archived changes passed the test at release and
are not checked again.

The default branch is resolved as for every command
(`scan.ResolveDefaultBranch`, `--default-branch`); with none, inspect
exits with 4 like the others.

### The doctor's steps

1. Run `dkt inspect --json`.
2. Check what needs judgment: for each step of the workflow file, whether
   this environment has what it names (another model to review with,
   subagents), and so which level of the review fallback applies (the
   levels are the template's `## Reviews`: a reviewer on a different
   model, else the same model in a fresh context, else a note in the pull
   request that the work had no independent review); where the
   workflow file and the user's instructions already in context disagree,
   quoting both; instructions that conflict with the framework, such as
   squash merges or rules that open a change or a release beyond the
   triage; whether superpowers' skills are available. No other user-level
   file is read without the user's agreement, as in setup.
3. List every problem, with its reason and the proposed fix, before fixing
   any.
4. Take them one at a time, each only after the user agrees:
   - workflow file: run `docket-setup`;
   - installed file missing or differing: show the difference, then
     `dkt init` or `dkt init --force`;
   - OpenSpec: say how to install a supported CLI or set
     `DOCKET_OPENSPEC`; install nothing;
   - the user's own instructions: propose wording; never edit them;
   - `no-task-commits`: explain that the pull request lost its trailers
     and that `dkt archive` will refuse the change.
5. Run `dkt inspect` again and report what is left.

### Setup moves into its own skill

`setup.md`'s text becomes `docket-setup/SKILL.md`, with a description that
says when to use it (the user asks to set up or revise the workflow file,
or the `docket` skill sends the agent there). `docket/SKILL.md` names the
`docket-setup` skill where it now names `setup.md`. Its steps stay as they
are, plus the case of a user revising an `ok` file.

### Embedding three skill folders

`schema.go` embeds `.agents/skills/docket`, `.agents/skills/docket-setup`
and `.agents/skills/docket-doctor`; the installer takes the skills by name
and installs each under both skill roots. The `.claude/skills/` copies in
this repository are the output of `dkt init`, as today.

## Risks / Trade-offs

- [Codex's handling of skill files may change between releases] → checked
  with codex-cli 0.153.4 before the plan (see Commands are skill folders);
  the trial confirms an explicit `$docket-doctor` once.
- [The doctor's judgment varies by model] → trial runs with two models, as
  for setup, on a project with known problems.
- [`no-task-commits` for a change merged with no trailers on purpose] →
  the doctor explains the consequence and leaves the decision to the user;
  only active changes are checked.
- [`docket` itself stays invocable by the user, so `/docket` exists] →
  harmless: it is the same rules the agent would load.

## Migration Plan

- No other project uses Docket yet, so there is no upgrade path (see No
  upgrade handling).
- This repository: its two `docket/setup.md` files are removed, and its
  `.claude/skills/` copies are regenerated with the new `dkt init`. The
  version bump and the release stay in the release step.
- The prompts are verified in the plan, by a trial task the lead runs after
  they land, as in `add-workflow`: a scratch project, headless Claude Code
  with two models, prompts that should and should not run the doctor, the
  doctor on a project with known problems, setup invoked by the user.
- Rollback: revert the merge.
