# Docket

English | [繁體中文](README.zh-TW.md)

Docket is a small command-line tool that checks [OpenSpec](https://github.com/Fission-AI/OpenSpec) change folders and keeps their books. It is meant for projects that plan and execute work with [superpowers](https://github.com/obra/superpowers). Docket keeps no state of its own, and the command is `dkt`.

> **Status: early.** `init`, `new`, `lint`, `conflicts`, `sync`, `archive`, `rebase` and `status` work; the first seven of them completed the first milestone. The other commands below are a draft and will change. Docket is built with the workflow this README describes.

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
| Check (read-only; exits non-zero on failure) | `lint`, `conflicts`, `check` | Required sections, leftover placeholders, task heading format, reuse-table paths that do not exist, requirements touched by more than one active change, scenarios without tests |
| Bookkeeping | `status`, `sync`, `log` | Progress from commit trailers, idle days, and how a requirement changed over time. `sync` changes nothing but checkboxes in `plan.md` |
| Assemble | `brief`, `accept --export` | Gathers pieces from several files into a brief for an implementing agent, or into a page a client can review |
| Wrap | `init`, `new`, `archive`, `rebase` | Installs the schema, runs the OpenSpec CLI and the checks above in a fixed order, and records the base commit and the evidence |

The commands that write files are `init` (the schema), `new` (the change folder), `sync` (checkboxes), `archive` (through OpenSpec, plus `evidence.md`) and `rebase` (one line of `.openspec.yaml`). None of them commits.

## What Docket is not

Docket does **not**:

- **Judge content.** It never calls a language model or decides whether content is correct. That is the reviewer's job.
- **Store state.** Everything it reports can be recomputed from the repository and git.
- **Edit agent configuration.** It never touches `CLAUDE.md`, `AGENTS.md` or `.claude/settings.json`.
- **Merge specs itself.** It does not reimplement OpenSpec's merge engine; it calls a pinned `openspec` CLI.

## The workflow it assumes

superpowers starts the work. OpenSpec never starts a step on its own; it is called at three points: creating the change folder, validating it, and archiving it.

1. **Triage with brainstorming.** superpowers brainstorming reads the project and talks the idea through. When several paths apply, take the heaviest:
   - It changes a sentence in `openspec/specs/`: open a change and follow the steps below. So does a bug fix whose correct behavior no spec states and no existing behavior shows, since it defines behavior.
   - A bug fix that restores the specified behavior, or a restructuring that changes no behavior: write a regression test, see it fail before the fix, and have the diff reviewed once before merging. OpenSpec is not involved.
   - Prompt text (the text agents act on, such as `AGENTS.md` and the schema's instructions) or tests alone: check prompts mechanically and tests by mutating the code they cover, and have the diff reviewed once before merging. OpenSpec is not involved.
   - Docs for people, version bumps and pure moves: no review beyond the automated checks. OpenSpec is not involved.
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
dkt init [--force]                # install the docket schema
dkt new <slug> --kind feature|change-request|fix|baseline --source "<origin>"
dkt lint [<slug>...]              # document rules
dkt conflicts [<slug>]            # requirements touched by more than one active change
dkt sync <slug> [--check]         # tick plan.md from commit trailers
dkt archive <slug>...             # release step, on a branch made from the default branch
dkt rebase <slug>                 # accept the newer specs as the change's base
dkt status [<slug>...] [--json]   # phase, tasks and their commits, progress, last activity, uncommitted work
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
