# Proposal

## Why

Two gaps remain once lint and conflict detection exist. First, superpowers
never ticks plan.md: it keeps progress in its own ledger and deletes it when
done, so OpenSpec sees an unfinished plan and archive warns about it.
Second, archiving is the only step that writes the current specs, and
OpenSpec 1.14 lets a later archive replace a requirement that another change
edited after this one began, without a warning. Docket ticks the plan from
commit trailers, and archives only after checking that nothing it is about
to replace changed underneath it.

## What Changes

- `dkt sync <slug>`: ticks the steps of every task that has commits carrying
  `Change: <slug>` and `Task: <N>`; `--check` only reports.
- `dkt archive <slug>...`: on the default branch, runs the plan check, lint,
  a base comparison, `openspec validate --strict` and a reading cross-check,
  then `openspec archive`, then writes `evidence.md`.
- `dkt rebase <slug>`: moves a change's `x-base` forward after its author has
  brought the delta up to date, and lists what changed in between.
- A reader for a change's `.openspec.yaml` (`x-base`, `x-kind`, `x-source`).

## Non-Goals

- Never commits, pushes or opens branches; results stay in the working tree.
- Does not merge specs itself; `openspec archive` does.
- No three-way merge of a requirement that drifted; the author updates the
  delta by hand.
- No review or test evidence from outside tools in `evidence.md` yet; only
  what git and OpenSpec show.
- Does not decide when a change is accepted; that stays with people.
- `sync` never unticks a step.

## Builds On

- From `add-init-and-new`: the scan, the OpenSpec runner, the default branch
  and merge base.
- From `add-lint`: the delta and plan readers, `dkt lint` itself as the lint
  step of archiving, and the contract test's comparison with
  `openspec show --json --deltas-only`, which becomes the reading
  cross-check.
- OpenSpec does the validation and the merge (`validate --strict`,
  `show --json`, `archive --json`); git parses commit trailers
  (`%(trailers)`).

## Capabilities

### New Capabilities

- `plan-sync`: ticking plan.md from commit trailers.
- `change-archive`: the checks and steps of `dkt archive`, the evidence it
  writes, and `dkt rebase`.

### Modified Capabilities

None.

## Impact

- Uses the scan from `add-init-and-new` and the readers from `add-lint`.
- `archive` needs the OpenSpec CLI; `sync` and `rebase` do not.
- Pull requests must keep per-commit trailers, so they merge with merge
  commits rather than squash.

## Implementation Contract

### Observable Behavior

- After `dkt sync add-x`, every step of every task that has an `add-x`
  commit is ticked, and nothing else in the repository has changed.
- `dkt archive add-x` either leaves `add-x` archived, specs merged and
  `evidence.md` written in the working tree, or changes nothing for `add-x`
  and says which check stopped it.
- `dkt rebase add-x` rewrites only the `x-base` line of the change's
  `.openspec.yaml`.

### Interfaces

- `dkt sync <slug> [--check] [--json]`
- `dkt archive <slug>... [--json]`
- `dkt rebase <slug> [--json]`
- Commit trailers `Change: <slug>` and `Task: <N>`, read as git reads trailers.
- `evidence.md` in the archived change folder.
- Rule identifiers in findings: `change-missing`, `metadata-invalid`,
  `base-missing`, `no-commits`, `plan-missing`, `task-unknown` and
  `plan-unsynced` (sync, and the plan check of archive); `plan-unticked`,
  `uncommitted-changes`, `behind-default-branch`, `change-not-merged`,
  `base-drift`, `name-taken`, `readings-disagree`, `openspec-failed`,
  `evidence-failed`, and the rules of `dkt lint` (archive); `no-merge-base`
  (rebase).

### Failure Modes

- Uncommitted changes, HEAD behind the default branch tip, or a change not
  yet on the default branch: `archive` refuses, exit 1.
- A requirement the change replaces changed since `x-base`: `archive`
  refuses and shows the difference.
- Docket's and OpenSpec's readings of a delta differ: `archive` refuses.
- `openspec validate --strict` or `openspec archive` fails: `archive` stops
  and reports OpenSpec's output.
- A `Task:` trailer naming a task the plan does not have: a finding, exit 1.

### Acceptance Criteria

- Sync: `plan-sync#task-with-commits-ticked`,
  `plan-sync#task-without-commits-left-alone`,
  `plan-sync#two-tasks-in-one-commit`, `plan-sync#nothing-else-changes`,
  `plan-sync#trailer-for-an-unknown-task`, `plan-sync#check-finds-unticked-work`,
  `plan-sync#check-on-a-synced-plan`.
- Archive: `change-archive#uncommitted-changes`, `change-archive#branch-behind-the-default-branch`,
  `change-archive#change-not-merged-yet`, `change-archive#second-of-two-fails`,
  `change-archive#unticked-task`, `change-archive#requirement-changed-since-base`,
  `change-archive#two-changes-in-one-run-touch-one-requirement`,
  `change-archive#other-requirement-in-the-same-capability`,
  `change-archive#added-name-now-taken`, `change-archive#readings-disagree`,
  `change-archive#evidence-written`, `change-archive#nothing-committed`.
- Rebase: `change-archive#rebase-after-merging-the-default-branch`,
  `change-archive#nothing-to-move`.

## Change Log

| Date | Source | Change | Impact |
|---|---|---|---|
| 2026-10-24 | self: writing the add-archive plan | The plan check also fails on a `Task:` trailer for an unknown task; `evidence.md` names the repository by its `origin` URL, not its local path; blocks are compared as OpenSpec compares them; the `.openspec.yaml` reader and the finding rules are named | Two requirement sentences in `change-archive` (Plan check, Evidence); no task exists yet |
