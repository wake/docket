# Design

## Context

See proposal.md for why. The pieces exist in separate places:
`scan.Scan` finds every active change and its sources, `scan.ReadFiles`
reads a change folder from any of them (added by `add-conflicts`),
`change.ParseMetadata` and `change.ParsePlan` read `.openspec.yaml` and
`plan.md`, and `dkt sync` (`computeSync` in `internal/archive/sync.go`)
maps the `Change` and `Task` trailers of `Git.Commits` to the tasks of a
plan. That mapping is unexported and tied to sync's own types. Nothing in
Docket reads a commit's time or knows the current time.

## Goals / Non-Goals

**Goals:**

- One report a person reads and the workbench parses, computed the same
  way for every source kind.
- The task states agree with `dkt sync`: a task status calls started
  because a commit names it is the task sync would tick.

**Non-Goals:**

- Speed beyond a few git calls per change; there are rarely more than a
  dozen active changes.

## Decisions

- **Content from the first source, read once.** Each change folder is
  read with one `scan.ReadFiles` call on `Sources[0]`, keeping
  `.openspec.yaml`, `proposal.md`, `design.md`, `plan.md` and the delta
  specs (`delta.CapabilityPath`). This is how `conflicts.Load` reads
  changes, and like it, a slug with an archived use in any source is left
  out. Commits are listed from `x-base` to `Sources[0].Commit`, so a
  change on a branch shows that branch's work, not HEAD's. The
  alternative, reading at HEAD like `dkt sync`, would show nothing for a
  change that lives in another worktree.
- **One trailer reading.** The part of `computeSync` that turns commits
  into "which tasks of this plan are named, by which commits, and which
  `Task` values name no task" becomes an exported function beside the
  plan reader, and both `dkt sync` and `dkt status` call it. Sync keeps
  its findings and ticking; its behavior does not change. Copying the
  loop instead would let the two commands disagree on what "named" means.
- **Phase and task state are a pure function** of the parsed plan and
  the named commits, so every rule in the `change-status` spec is tested
  without git. Cancelled tasks are the bullets under `## Cancelled
  Tasks`; they are listed after the plan's tasks, with no steps, and left
  out of every count. A task with no steps is never done, so a plan with
  one keeps the change `implementing`; `dkt lint` reports the empty task.
- **Commit time is the committer date.** `gitx.Commit` gains the commit's
  committer time, read in the same `git log` call. Last activity is the
  newest commit of two lists over the same range: the commits whose
  trailers name the change, and the commits that change a file in its
  folder (`Git.Commits` limited to that path, with full history: git's
  default simplification drops a `--no-ff` merge that matches its second
  parent on the path, and that merge is when a change reached the default
  branch). The committer date is when
  the commit landed on the branch it is read from; the author date can
  be weeks older after a rebase and would overstate idleness.
- **The run's start time comes in through `cli.Options`.** `cmd/dkt` sets
  it from the clock; tests set a fixed time. Idle days are whole days,
  rounded down, from the last activity to that time; a commit dated after
  it gives 0. Reading the clock inside a command would make idle days
  untestable; like the environment, it is passed in from `cmd/dkt`.
- **Uncommitted files only for a worktree.** When `Sources[0]` is a
  worktree, `git status` runs in that worktree, limited to the change
  folder, with untracked files listed one by one, and the files are
  counted. A branch has no working files, so the field is null there.
- **No findings for the state itself.** An unsynced task, an empty plan or
  a stale base is data in the result, not a finding: `dkt status` is a
  report, and exiting 1 for an unsynced plan would make every caller treat
  progress as failure. Findings are only `slug-invalid` and
  `change-not-active`, as in `dkt conflicts`.
- **Text output, one block per change.** The first line gives the slug,
  phase, tasks done of total, idle days and uncommitted files; then one
  line per source, `<name>@<7-character commit>` as `dkt conflicts`
  prints them. The JSON result is the full report.

## Risks / Trade-offs

- [A worktree's last commit is old but someone is editing it now] → the
  uncommitted count shows the edits; idle days only count commits.
- [Commit dates are wrong on a machine with a bad clock] → idle days are
  wrong by the same amount; a future date gives 0, never a negative
  number.
- [`plan.md` on the first source is behind its commits because nobody ran
  `dkt sync`] → the task is `started` and marked unsynced, which says
  what to run.
- [Two copies of a change differ] → only the first source is read, as in
  `dkt conflicts`; every source is listed.
