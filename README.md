# Docket

English | [繁體中文](README.zh-TW.md)

Docket is a small command-line tool that checks [OpenSpec](https://github.com/Fission-AI/OpenSpec) change folders and keeps their books. It is meant for projects that plan and execute work with [superpowers](https://github.com/obra/superpowers). Docket keeps no state of its own, and the command is `dkt`.

> **Status: early.** `init`, `new`, `lint`, `conflicts`, `sync`, `archive`, `rebase`, `status`, `record`, `issue` and `workflow` work; the first seven of them completed the first milestone. The other commands below are a draft and will change. Docket is built with the workflow this README describes.

## Why

A common way to pair the two tools:

- **OpenSpec owns *what*:** current specs, change proposals, deltas and the archive.
- **superpowers owns *how*:** brainstorming, writing-plans, subagent-driven development and TDD.

The pairing leaves gaps that prompts alone do not close:

- **Two task lists.** OpenSpec tracks `tasks.md`, while superpowers executes `plan.md`. Keeping the two in step is left to the agent.
- **Rules only in prompts.** OpenSpec schema instructions and config guidance are advisory. Nothing fails when an agent ignores them.
- **Silent overwrites.** Two changes may modify the same requirement. Whichever is archived later replaces the earlier text, with no warning.
- **Untested scenarios.** Nothing checks that every scenario in a delta has a test.
- **Forgotten archives.** Finished changes sit unarchived, and the next session redoes the work.

Docket closes these gaps with checks a program can run and with bookkeeping. It does not add a third planning system.

## What Docket is

Docket reads the files in the repository and the git history. It does four kinds of work:

| Role | Commands | What it does |
|---|---|---|
| Check (read-only; exits non-zero on failure) | `lint`, `conflicts`, `check` | Required sections (records: front matter, sections, word limits), leftover placeholders, task heading format, reuse-table paths that do not exist, requirements touched by more than one active change, scenarios without tests |
| Bookkeeping | `status`, `sync`, `log` | Progress from commit trailers, idle days, and how a requirement changed over time. `sync` changes nothing but checkboxes in `plan.md` or a record |
| Assemble | `brief`, `accept --export` | Gathers pieces from several files into a brief for an implementing agent, or into a page a client can review |
| Wrap | `init`, `new`, `archive`, `rebase` | Installs the schema, runs the OpenSpec CLI and the checks above in a fixed order, and records the base commit and the evidence |

The commands that write files are `init` (the schema, the skills, the `docket-init` command and the Docket block), `new` (the change folder), `record` (a record under `docket/records/`), `sync` (checkboxes), `archive` (through OpenSpec, plus spec titles and `evidence.md`) and `rebase` (the base line of `docket.yaml`, or of `.openspec.yaml` in a change made by an older dkt). None of them commits.

## What Docket is not

Docket does **not**:

- **Judge content.** It never calls a language model or decides whether content is correct. That is the reviewer's job.
- **Store state.** Everything it reports can be recomputed from the repository and git. The one file it reads outside the repository is the user's workflow file, and it never writes it.
- **Edit agent configuration.** It never touches `.claude/settings.json`. In `AGENTS.md` and `CLAUDE.md` it writes only the marked Docket block, creating either file when missing.
- **Merge specs itself.** It does not reimplement OpenSpec's merge engine; it calls a pinned `openspec` CLI.

## The workflow it assumes

superpowers starts the work. OpenSpec never starts a step on its own; it is called at three points: creating the change folder, validating it, and archiving it. The `docket` skill carries these rules, and `dkt init` installs it. The review and verification steps of each path come from the user's workflow file (`~/.config/docket/workflow.md`, or under `XDG_CONFIG_HOME`). The agent writes it with the user from Docket's template at the first piece of work, and `dkt workflow` reports whether it is missing, lacks a path or was made from an older template. In any project the user can type `/docket-init` (`$docket-init` in Codex) to check the environment and set the project up. Setup is its own skill, `docket-setup`, which the user can also invoke to revise the file (`/docket-setup` in Claude Code, `$docket-setup` in Codex). To check a project's setup, the user invokes `docket-doctor`: it runs `dkt inspect`, checks what needs judgment, lists every problem and then fixes them with the user one at a time.

1. **Triage with brainstorming.** superpowers brainstorming reads the project and talks the idea through. When several paths apply, take the heaviest:
   - It changes a sentence in `openspec/specs/`: open a change and follow the steps below. So does any edit that sets an expectation no spec states and no existing behavior shows, such as a bug fix with no specified answer or a test that asserts new behavior, since it defines behavior.
   - A bug fix that restores the specified behavior, or a restructuring that changes no behavior (moving code is one): for a fix, write a regression test and see it fail before the fix; for a restructuring, the tests that cover the code pass before and after, and mutating the changed lines makes one fail. Have the diff reviewed once before merging. A comment that changes how the code builds or runs, such as a compiler directive, is code. OpenSpec is not involved.
   - Prompt text (the text agents act on, such as `AGENTS.md`, the `docket` skill and the schema's instructions; list the paths in `AGENTS.md`), tests that check behavior already specified or already shown, and check settings such as CI configuration: check prompts mechanically, tests by mutating the code they cover, and check settings by the pull request's own CI run, and have the diff reviewed once before merging. OpenSpec is not involved.
   - Docs for people, edits that only change comments or whitespace in code or test files (these are not edits to code or tests), version bumps, and dependency upgrades that touch only the dependency manifests: no review, and only the checks that read what changed (none for docs that nothing builds, embeds or tests). When such an edit touches a code or test file, a fresh agent that reads only the diff first confirms that every changed line is a comment or whitespace and that no comment changes how the code builds or runs. Sort by what is touched: a typo in a prompt is prompt text, and moving code is a restructuring. OpenSpec is not involved.

   Work on the second and third paths above opens no change but keeps a record in `docket/records/`, written by `dkt record` and checked by `dkt lint`; its commits carry `Record: <slug>`, and `dkt lint` reports a trailer that names no record. `dkt status` lists the open records and `dkt sync` ticks their tasks. `dkt new` and `dkt record` take a free `-2` suffix (then `-3`, and so on) when the slug is used.

   Work whose expected behavior is already settled and that is not done now is an issue, not an idea. An issue has four sections (`## Current`, `## Expected`, `## Unchanged`, `## Found`; `dkt issue template` prints them). A project that keeps issues in an outside tool names it in a `## Issues` section of its `AGENTS.md`, and the agent files the issue there once the user agrees; otherwise `dkt issue` writes `docket/issues/<date>-<name>.md`. A record or change whose source is the issue's path handles it, and `dkt status` lists file issues that are pending or in progress from those records and changes. `dkt lint` checks the file issues a branch adds (front matter, the four sections, word limits) and reports a record or change whose source names a file issue that does not exist; `dkt status --json` lists the open issues in `result.issues`. `dkt` never calls an outside tool.
2. **Open a change** in a git worktree with `dkt new <slug>`, which wraps `openspec new change`.
3. **Write the proposal and the delta.**
   - brainstorming writes `proposal.md` and `specs/**/spec.md` straight into the change folder, plus `design.md` when needed.
   - It gets the templates from `openspec instructions`.
4. **Validate** with `openspec validate --strict` and `dkt lint`.
5. **Plan.** superpowers writing-plans writes `plan.md` into the change folder.
   - This is the only task list. There is no `tasks.md`.
   - Each task says `**Review:** judgment` when it needs a reviewer's judgment, and `**Review:** checks` otherwise.
   - Before any code, the plan is reviewed once together with the proposal and the delta.
6. **Implement and review.**
   - Use superpowers subagent-driven development with TDD, but only `**Review:** judgment` tasks get their own reviewer.
   - Every task that changes code gets the mechanical checks instead: its new tests fail before the code and pass after, and mutating the lines it changed makes a test fail. A task that only edits docs is checked by its own verification steps.
   - Every task's commit carries trailers:

     ```
     Change: add-waitlist
     Task: 3
     ```

   - A change is one pull request. Before merging, it is reviewed once, by two reviewers in parallel: one on a different model attacks it, one in a fresh context checks it against the proposal and the delta. `dkt check` must pass.
   - Merge with a merge commit, not squash, so every commit keeps its trailers.
7. **Archive** in the release step, on a branch made from the default branch tip, with `dkt archive <slug>...`.

### One task list

A forked OpenSpec schema replaces the `tasks` artifact with `plan` and points `apply.tracks` at `plan.md`. In a sandbox with OpenSpec 1.14.3, `status`, `list`, `validate --strict` and `archive` all accepted this layout.

Two caveats:

- **OpenSpec counts steps, not tasks.** It counts every checkbox, so its progress figure is per step. `dkt status` reports per task.
- **superpowers does not tick `plan.md`.** superpowers 7.0 records progress in the agent harness's todo list and in a temporary ledger. Without `dkt sync`, which ticks the boxes from commit trailers, OpenSpec would always show zero progress.

We found no public project that uses a single list this way; most keep both files. Treat it as an experiment until the pilot shows otherwise.

## Planned commands (draft)

```
dkt init [--force]                # install the docket schema, the skills and the Docket block
dkt new <slug> --kind feature|change-request|baseline --source "<origin>" --title "<title>"
dkt record <name> --kind fix|refactor|test|prompt|checks --source "<origin>" --title "<title>"   # a record for light-review work
dkt issue <name> --title "<title>" --source "<origin>"   # a file issue for settled work that is not done now; `dkt issue template` prints the sections
dkt lint [<slug>|<record>...]     # document rules
dkt conflicts [<slug>]            # requirements touched by more than one active change
dkt sync <slug>|<record> [--check]   # tick plan.md or a record's tasks from commit trailers
dkt archive <slug>...             # release step, on a branch made from the default branch
dkt rebase <slug>                 # accept the newer specs as the change's base
dkt status [<slug>|<record>...] [--merged [--since <rev>] [--limit <n>]] [--json]   # phase, tasks and their commits, progress, last activity, uncommitted work; open records, and merged ones with --merged
dkt workflow [template] [--json]  # the user's workflow file, or the template
dkt inspect [--json]              # the project's Docket setup; writes nothing
dkt brief <slug> --task N         # brief for an implementing agent
dkt check <slug>                  # scenario coverage and checkbox consistency (pull request gate)
dkt accept <slug> --round N [--export html]
dkt log <capability>
dkt index                         # reuse index of existing capabilities
```

The first milestone is `init`, `new`, `lint`, `conflicts`, `sync`, `archive` and `rebase`, built as four OpenSpec changes, now archived in `openspec/changes/archive/`.

## Design rules

- **The repository is the only source of truth.** If Docket is deleted, nothing is lost.
- **Worktree-aware.** Active changes are found across every git worktree and unmerged branch, not only the current one.
- **"None" and "unreadable" are different results.** An unmounted disk or a missing checkout is reported as unavailable, never as zero changes.
- **Output names the commit.** Every result carries the repository and commit SHA, so runs on different machines can be compared.
- **`--json` everywhere.** Every command can produce JSON, so Docket can run in CI, a test runner or an agent session manager.

## Install

```
go install github.com/wake/docket/cmd/dkt@latest
```

`dkt init` and `dkt new` need the OpenSpec CLI 1.14.x on `PATH`, or set `DOCKET_OPENSPEC` to the command to run.

`dkt init` writes the `docket` schema once per machine, to `$XDG_DATA_HOME/openspec/schemas/docket/` (by default `~/.local/share/openspec/schemas/docket/`), where OpenSpec looks for it. It sets `schema: docket` in `openspec/config.yaml`. A wrapper that gives OpenSpec another `XDG_DATA_HOME` or `HOME` needs `dkt` run with the same values. CI that runs `openspec` on a change runs `dkt init` first. It also writes the `docket-init` command to `~/.claude/skills/docket-init/` (under `CLAUDE_CONFIG_DIR` when set) and `~/.agents/skills/docket-init/`.

### Upgrading

Changes made by an older dkt still work: their kind, source and base are read from `x-kind`, `x-source` and `x-base` in `.openspec.yaml`. `dkt lint` asks for a `docket.yaml` with a title for each of them, and each main spec needs a `docket.yaml` with its title too.

In a project set up by an older dkt, run `dkt init` again. It adds the `docket` skill and the Docket block. Delete any "Change workflow" section pasted into `AGENTS.md` from an earlier release, and commit `.claude/skills/docket/`, `.agents/skills/docket/`, `AGENTS.md` and `CLAUDE.md`.

In a project set up by dkt v0.6.0, run `dkt init --force` to take the new Docket block and skill (until then `dkt init` reports `agents-block-differs`), and commit them. The agent then sets up your workflow file at the first piece of development work.

In a project set up by dkt v0.9.2 or earlier, the schema is a copy in `openspec/schemas/docket/`. Run `dkt init`. Check that it removed `openspec/schemas/docket/` and set `schema: docket` in `openspec/config.yaml`, and commit both. Everyone who works on the project upgrades dkt and runs `dkt init`, once per machine. A copy that you edited is left alone, and `dkt init` reports it as `schema-edited`.

## Requirements (planned)

- git
- The [OpenSpec](https://github.com/Fission-AI/OpenSpec) CLI at a pinned version. 1.14.3 is the version tested so far.
- Go, to build from source.
- [superpowers](https://github.com/obra/superpowers) 7.0.1 or later, if you follow the workflow above.

## Related projects

Docket is not affiliated with OpenSpec, superpowers or any OpenSpec companion app.

Several community projects connect OpenSpec and superpowers through custom schemas or bridge skills, for example [openspec-schemas](https://github.com/JiangWay/openspec-schemas) and [superspec](https://github.com/danielhanold/superspec). Docket differs in moving every rule that a program can check out of the prompts and into code.

Docket is designed to work alongside [Purdex](https://github.com/wake/purdex), an agent session manager, but does not require it.

## License

[MIT](LICENSE)
