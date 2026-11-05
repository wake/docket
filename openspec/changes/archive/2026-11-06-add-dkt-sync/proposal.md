# Proposal

## Why

The `docket` skill has the agent run `dkt sync <slug>` before opening a pull request, so that plan.md shows every task with commits as done, and the archive guidance runs `dkt sync <change> --check` to stop an archive while a task has no commit or a task with commits is not ticked. superpowers never ticks the plan, so without `dkt sync` plan.md does not show progress. It is the second of the three commands this repository needs before it switches to Docket's own workflow.

## What Changes

- Add `dkt sync <slug> [--check]` for a change:
  - It reads the trailer `Change: <slug>/<N>` (several tasks: `/<N>,<M>`) of every commit reachable from HEAD, as git reads trailers, and ticks every unticked `- [ ]` step of `### Task N` in `openspec/changes/<slug>/plan.md`. The whole history is read: slugs are never reused, so no base is needed.
  - It changes only checkboxes, never unticks, never touches another file and never commits; it does not need the OpenSpec CLI.
  - It reports a trailer that names a task plan.md does not have, or that it cannot read, and still ticks the others.
  - With `--check` it writes nothing and reports each task that has commits and unticked steps, and each unfinished task that has no commit, as the archive guidance asks.
  - It refuses a change whose `.openspec.yaml` names another schema.
- change-metadata: a change's schema is the `schema` key of its `.openspec.yaml`.
- cli-output: `dkt sync` joins the commands that need a repository.
- The no-follow read and the mode-keeping write that `dkt init` uses move to a shared package, so `dkt sync` writes plan.md the same way.

## Non-Goals

- Records: `dkt sync` for a record (`Record: <slug>/<N>`) comes with `dkt record`.
- Plan structure checks beyond what ticking needs (task headings, Reuse table, scenario references): `dkt lint`.
- No lock: two runs compute the same ticks from the same commits.
- A task finished without any commit is ticked by hand, as the plan rules say; `--check` does not report a task whose steps are all ticked.

## Capabilities

### New Capabilities
- `plan-sync`: how `dkt sync` ticks plan.md from commit trailers, what it reports, and its check mode.

### Modified Capabilities
- `change-metadata`: adds the change's schema.
- `cli-output`: `dkt sync` needs a repository.

## Impact

- Adds `dkt sync`, plan parsing and trailer reading in `internal/`, and their tests; moves two file helpers out of `internal/setup`.
- Dependencies: still only the Go standard library and the system `git`.
