# add-status Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `dkt status [<slug>...]`: list every active change, where it lives, its phase, its task progress and its recent activity, as text and as the JSON feed the workbench will read.

**Architecture:** `internal/gitx` gains commit times, path-limited commit lists and a per-file uncommitted list. The trailer reading of `dkt sync` moves into `change.NamedTasks`, which `archive` and the new `internal/status` package both call. `internal/status` computes phase and task states as a pure function and loads each active change from its first source. `internal/cli` adds the read-only command; `cli.Options` gains the run's start time.

**Tech Stack:** Go 1.24 (module floor), standard library; `git` as an external program. No OpenSpec at run time.

**Spec:** `openspec/changes/add-status/proposal.md`, `design.md` and `specs/change-status/spec.md`. Read all three; scenario IDs below refer to the delta spec. The scan is `openspec/specs/workspace-scan/spec.md`; trailers are read as in `openspec/specs/plan-sync/spec.md`; exit codes and the envelope are `openspec/specs/cli-output/spec.md`.

## Global Constraints

- Module path `github.com/wake/docket`; `go 1.24` in go.mod; no new third-party dependency.
- Public repository: code, comments, messages and docs in English; no hostnames, customer names, personal paths or secrets.
- `dkt status` writes nothing, takes no lock, starts no OpenSpec and never fetches.
- Every command accepts `--json` and `--default-branch <name>`, before or after positional arguments.
- Exit codes: 0 nothing to report; 1 a finding; 2 usage error; 3 a source or a file in a change folder could not be read (wins over 1); 4 no git repository or no default branch.
- With `--json`, standard output is exactly one JSON object with `tool`, `version`, `command`, `repo`, `ok`, `findings`, `unreadable`, `result`; every array in it is an array, never null.
- Finding rules, exactly: `change-not-active`, `slug-invalid`.
- Phases, exactly: `design`, `ready`, `implementing`, `done`. Task states, exactly: `pending`, `started`, `done`, `cancelled`.
- JSON field names, exactly as in proposal.md's Interfaces: `changes`, `slug`, `sources` (`kind`, `name`, `commit`), `metadata` (`kind`, `source`, `base`, `created`), `documents` (`proposal`, `design`, `plan`, `specs`), `phase`, `tasks` (`number`, `name`, `state`, `steps`, `ticked`, `commits`, `unsynced`), `progress` (`tasks_done`, `tasks_total`, `steps_ticked`, `steps_total`), `last_activity` (`commit`, `time`, `idle_days`), `uncommitted`. `time` is RFC 3339 (`time.RFC3339`).
- Only `internal/gitx` starts git; nothing reads `os.Getenv`, `os.Environ` or the clock below `cmd/dkt` (tests excepted, and the zero-`Now` default in `RunWith`). Readers take bytes, not paths.
- macOS and Linux only.
- `go test ./...` passes without network access and without an OpenSpec install.
- Every commit is made with `git commit -m "<subject>" --trailer "Change: add-status" --trailer "Task: <N>" --trailer "Co-Authored-By: <the model that wrote it> <noreply@anthropic.com>"`, so the three trailers form the message's last paragraph. Never put a trailer in its own `-m`: git only reads the last paragraph.

## Review Focus

1. A change whose first source is the default branch (merged, waiting for the release step): its commits run from `x-base` to the default tip, and its last activity is the newest commit that names it or touches its folder, such as the merge of its pull request, not the default tip. Test in Task 4.
2. A commit dated after the run's start time (a clock ahead elsewhere) gives 0 idle days, never a negative number. Test in Task 4.
3. `x-base` missing, malformed or not a commit in this repository (a shallow clone): commits are empty, `last_activity` is null, the change is still listed, and nothing is reported unreadable. Test in Task 4.
4. A change folder that is entirely untracked in a worktree: `uncommitted` counts each file, not the folder once (`git status` collapses an untracked folder by default). Tests in Tasks 1 and 4.
5. A `Task: N` trailer naming a cancelled task or no task at all: the task stays `cancelled`, nothing is started by it, and the change is still listed. Test in Task 3.

## Reuse

| Need | Existing (path:symbol) | Decision | Reason | Evidence |
|---|---|---|---|---|
| Every active change, its sources and archived uses | `internal/scan/scan.go:Scan`, `Inventory.Uses` | reuse | The `workspace-scan` scan and its source order | `rg -n "func Scan\|func \(inv Inventory\) Uses" internal/scan` |
| Default branch and scan with the shared exit 4 | `internal/cli/cmd_new.go:scanDefaultBranch` | reuse | Same resolution and error mapping as `new` and `conflicts` | `rg -n "func scanDefaultBranch" internal/cli` |
| Read a change folder from any source | `internal/scan/scan.go:ReadFiles` | reuse | One call per change with a `keep` filter, as `conflicts.Load` does | `rg -n "func ReadFiles" internal/scan` |
| Which files are delta specs | `internal/delta/capability.go:CapabilityPath` | reuse | The one rule lint, archive and conflicts use | `rg -n "func CapabilityPath" internal/delta` |
| `.openspec.yaml` metadata | `internal/change/metadata.go:ParseMetadata` | reuse | Reads `x-kind`, `x-source`, `x-base`, `created` | `rg -n "func ParseMetadata" internal/change` |
| Tasks, steps and cancelled tasks | `internal/change/plan.go:ParsePlan` | reuse | `Plan.Tasks[].Steps[].Done`, `Plan.Cancelled` | `rg -n "func ParsePlan" internal/change` |
| Commits with trailers, their time, and limited to a folder | `internal/gitx/log.go:Git.Commits` | extend | Add the committer time to `Commit` and an optional path limit; existing callers are unchanged | `rg -n "func \(g Git\) Commits" internal/gitx` |
| Whether `x-base` names a commit | `internal/gitx/git.go:Git.RefExists` | reuse | `RefExists(base + "^{commit}")` after refusing a leading `-`, as `dkt sync` and `dkt archive` check it | `rg -n 'commit\}' internal` → `archive/load.go`, `archive/rebase.go` |
| Which tasks a change's commits name | `internal/archive/sync.go:computeSync` | extend | The loop moves into `change.NamedTasks`; `computeSync` keeps its findings and ticking on top of it | `rg -n "Task trailer\|cm.Task" internal/archive` → `sync.go` only |
| Uncommitted files below a folder, one per file | `internal/gitx/added.go:Git.Uncommitted` | extend | `Uncommitted` collapses untracked folders and takes no path; `UncommittedUnder` uses the same `status` helper with `all` and paths, and `Uncommitted` stays as it is for `dkt archive` | `rg -n "func \(g Git\) status" internal/gitx` |
| Unreadable sources in the envelope | `internal/cli/cmd_conflicts.go:scanUnreadable` | reuse | The shared mapping with `<name>:<path>` | `rg -n "func scanUnreadable" internal/cli` |
| Slug format | `internal/change/new.go:ValidateSlug` | reuse | Same rule as `new` and `conflicts` | `rg -n "func ValidateSlug" internal/change` |
| Findings, unreadable, exit codes, envelope | `internal/cli/envelope.go:Outcome` | reuse | The shared result model | `rg -n "type Outcome" internal/cli` |
| Registering the command | `internal/cli/run.go:commands` | extend | Add `statusCommand` | `rg -n "var commands" internal/cli` |
| The run's start time | none | new | Nothing in Docket reads the time; it joins `cli.Options`, the process boundary | `rg -n "time\." --type go internal cmd` → no use outside tests |
| Phase, task states, activity of a change | none | new | Nothing computes a change's progress; `dkt sync` only ticks | `rg -n "phase\|idle" --type go internal` → none |
| Test repositories and worktrees | `internal/testutil/repo.go:NewRepo` | reuse | `Commit`, `AddWorktree`, `CloneTo`; commit times through `GIT_COMMITTER_DATE` in `Repo.Env` | `rg -n "func \(r \*Repo\)" internal/testutil` |

---

### Task 1: Commit times, path-limited commits, uncommitted files under a folder

**Scenarios:** none

**Files:**
- Modify: `internal/gitx/log.go`, `internal/gitx/added.go`
- Test: `internal/gitx/log_test.go`, `internal/gitx/added_test.go`

**Interfaces:**
- Produces (package `gitx`):
  - `Commit` gains `Time time.Time`: the committer date, read with `%cI` in the same `git log` call.
  - `func (g Git) Commits(base, head string, paths ...string) ([]Commit, error)` — unchanged for existing callers; with paths, only commits that change a file under them (`git log ... <base>..<head> -- <paths>`).
  - `func (g Git) UncommittedUnder(paths ...string) ([]string, error)` — the sorted paths below `paths` with staged, unstaged or untracked changes, untracked files listed one by one (`--untracked-files=all`); ignored files are left out.

- [ ] **Step 1: Write the failing tests:** `TestCommitsTime` (two commits made with `GIT_COMMITTER_DATE` set to `2026-01-02T03:04:05Z` and `2026-01-05T00:00:00+08:00`: `Time` equals each, compared with `time.Time.Equal`); `TestCommitsPaths` (three commits, the second and third touching `a/`: `Commits(base, head, "a")` returns those two, oldest first, with their trailers; `Commits(base, head)` still returns all three); `TestUncommittedUnder` (an untracked folder `x/` with two files, a modified tracked file `x/t.md`, a staged new file `x/s.md`, an untracked file `y/u.md` and an ignored file `x/i.log`: `UncommittedUnder("x")` returns the four `x/` paths sorted; a clean folder gives an empty list).
- [ ] **Step 2: Run** `go test ./internal/gitx/` — expected: FAIL (`Time` and `UncommittedUnder` undefined).
- [ ] **Step 3: Implement** the `Time` field (parse with `time.Parse(time.RFC3339, ...)`; the `-z` stream becomes groups of five fields), the `paths` parameter, and `UncommittedUnder` on the existing `status` helper. Update the doc comments.
- [ ] **Step 4: Run** `go test ./internal/gitx/ ./internal/archive/` — expected: PASS (archive is the existing caller of `Commits`).
- [ ] **Step 5: Commit** `feat(gitx): commit times, commits limited to a folder, uncommitted files under one`.

### Task 2: One reading of Task trailers for sync and status

**Scenarios:** none

**Files:**
- Create: `internal/change/trailers.go`
- Modify: `internal/archive/sync.go`
- Test: `internal/change/trailers_test.go`

**Interfaces:**
- Consumes: `gitx.Commit` (Task 1), `change.Plan`.
- Produces (package `change`):
  - `type NamedTask struct { Task Task; Commits []gitx.Commit }` — a task of the plan and the commits that name it, oldest first.
  - `type UnknownTrailer struct { Commit gitx.Commit; Value string }` — a `Task` value on a commit for the change that is not a task heading of the plan.
  - `func NamedTasks(p Plan, slug string, commits []gitx.Commit) (named []NamedTask, unknown []UnknownTrailer, anyTrailer bool)` — exactly the reading `computeSync` does today: only commits whose `Change` values include `slug`; each `Task` value counted once per commit; a value is a task number only when it matches `^[1-9][0-9]*$` and names a `### Task` heading (the first heading of a duplicated number wins); `named` in plan order with only the tasks some commit names; `unknown` in commit order; `anyTrailer` true when any commit for the change carries a `Task` trailer.
- `computeSync` calls `NamedTasks` and keeps building its `task-unknown` findings (same message), `Tasks`, `Lines` and `AnyTrailer` from the result.

- [ ] **Step 1: Write the failing test** `TestNamedTasks` (a plan with Tasks 1, 2 and 3 and a duplicated `### Task 2` heading; commits: one for another slug with `Task: 1`, one for the slug with `Task: 2` twice and `Task: 9`, one with `Task: 01`, one with `Task: 1`: `named` is Task 1 with one commit and Task 2 (the first heading) with one commit, in that order; `unknown` is `9` and `01` with their commits; `anyTrailer` true; with no commit for the slug: all three results empty or false).
- [ ] **Step 2: Run** `go test ./internal/change/` — expected: FAIL (`NamedTasks` undefined).
- [ ] **Step 3: Implement** `NamedTasks` in `internal/change/trailers.go` and switch `computeSync` to it.
- [ ] **Step 4: Run** `go test ./internal/change/ ./internal/archive/ ./internal/cli/` — expected: PASS; the existing sync and archive tests are the guard that behavior did not change.
- [ ] **Step 5: Commit** `refactor: one reading of Task trailers for sync and status`.

### Task 3: Phase and task states

**Scenarios:** change-status#plan-not-written, change-status#nothing-started, change-status#commit-names-a-task, change-status#every-task-done, change-status#cancelled-task-left-out, change-status#commits-not-yet-synced

**Files:**
- Create: `internal/status/status.go`
- Test: `internal/status/status_test.go`

**Interfaces:**
- Consumes: `change.Plan`, `change.NamedTask` (Task 2).
- Produces (package `status`):
  - Phase constants `Design = "design"`, `Ready = "ready"`, `Implementing = "implementing"`, `Done = "done"`; task state constants `Pending = "pending"`, `Started = "started"`, `TaskDone = "done"`, `Cancelled = "cancelled"`.
  - `type Task struct { Number int \`json:"number"\`; Name string \`json:"name"\`; State string \`json:"state"\`; Steps int \`json:"steps"\`; Ticked int \`json:"ticked"\`; Commits []string \`json:"commits"\`; Unsynced bool \`json:"unsynced"\` }` — `Commits` are full SHAs, never nil.
  - `type Progress struct { TasksDone int \`json:"tasks_done"\`; TasksTotal int \`json:"tasks_total"\`; StepsTicked int \`json:"steps_ticked"\`; StepsTotal int \`json:"steps_total"\` }`
  - `func Summarize(plan *change.Plan, named []change.NamedTask) (phase string, tasks []Task, progress Progress)` — `plan` is nil when the change has no `plan.md`. Tasks in plan order (a duplicated number is listed once, its first heading), then each cancelled task (`Plan.Cancelled`, with its `Name`, no steps, no commits, state `cancelled`) in the order listed. A task is `done` when it has steps and all are ticked; `started` when a step is ticked or `named` holds it; `pending` otherwise. `Unsynced` when it has commits and an unticked step. Counts leave cancelled tasks out. Phase as the `change-status` Phase requirement says; `tasks` is never nil.

- [ ] **Step 1: Write the failing tests:** `TestPlanNotWritten` (nil plan → `design`, no tasks, zero progress); `TestEmptyPlan` (a plan with no task → `design`); `TestNothingStarted` (three tasks, nothing ticked, no commits → `ready`, all `pending`, progress 0/3 tasks, 0/N steps); `TestCommitNamesATask` (no tick, `named` holds Task 1 with commit `c1` → `implementing`, Task 1 `started` with `Commits` `["c1"]` and `Unsynced` true); `TestEveryTaskDone` (all steps ticked → `done`, progress 2/2); `TestCancelledTaskLeftOut` (Tasks 1 and 2 ticked, Task 3 cancelled → `done`, Task 3 last with state `cancelled`, progress 2/2); `TestCommitsNotYetSynced` (Task 2 has a commit and one of two steps ticked → `started`, `Unsynced` true, `Ticked` 1, `Steps` 2); `TestStepsTickedNoCommits` (a ticked step and no commit → `started`, `Unsynced` false); `TestTaskWithoutSteps` (a task with no steps and the others done → phase `implementing`, that task `pending`).
- [ ] **Step 2: Run** `go test ./internal/status/` — expected: FAIL (package missing).
- [ ] **Step 3: Implement** `Summarize` and the types in `internal/status/status.go`.
- [ ] **Step 4: Run** `go test ./internal/status/` — expected: PASS.
- [ ] **Step 5: Commit** `feat(status): phase and task states of a change`.

### Task 4: Load every active change from its first source

**Scenarios:** change-status#changes-in-worktrees-and-branches, change-status#archived-change-not-listed, change-status#commit-names-a-task, change-status#idle-change, change-status#uncommitted-edits, change-status#no-commit-yet

**Files:**
- Create: `internal/status/load.go`
- Test: `internal/status/load_test.go`

**Interfaces:**
- Consumes: `scan.Inventory`, `scan.ReadFiles`, `delta.CapabilityPath`, `change.ParseMetadata`, `change.ParsePlan`, `change.NamedTasks`, `Git.Commits`, `Git.UncommittedUnder` (Tasks 1–3).
- Produces (package `status`):
  - `type Metadata struct { Kind \`json:"kind"\`; Source \`json:"source"\`; Base \`json:"base"\`; Created \`json:"created"\` }` (strings).
  - `type Documents struct { Proposal, Design, Plan bool; Specs []string }` with JSON names `proposal`, `design`, `plan`, `specs`; `Specs` sorted capability paths, never nil.
  - `type Activity struct { Commit string \`json:"commit"\`; Time string \`json:"time"\`; IdleDays int \`json:"idle_days"\` }` — `Time` formatted with `time.RFC3339`.
  - `type Change struct { Slug string \`json:"slug"\`; Sources []scan.Source \`json:"sources"\`; Metadata *Metadata \`json:"metadata"\`; Documents Documents \`json:"documents"\`; Phase string \`json:"phase"\`; Tasks []Task \`json:"tasks"\`; Progress Progress \`json:"progress"\`; LastActivity *Activity \`json:"last_activity"\`; Uncommitted *int \`json:"uncommitted"\` }`
  - `func Load(g gitx.Git, inv scan.Inventory, now time.Time) ([]Change, []scan.Unreadable, error)` — every change of `inv.Changes` whose slug has no archived use (`inv.Uses`), in slug order. Reads `openspec/changes/<slug>` from `Sources[0]` with one `scan.ReadFiles` call keeping `.openspec.yaml`, `proposal.md`, `design.md`, `plan.md` and the delta specs. `Metadata` is nil when `.openspec.yaml` is missing or does not parse. Commits are `g.Commits(base, Sources[0].Commit)` when `base` is a commit — not empty, not starting with `-`, and `g.RefExists(base + "^{commit}")`, as `internal/archive/load.go` checks it — otherwise none. Last activity is the newest by `Time` of the commits whose `Change` values include the slug and of `g.Commits(base, Sources[0].Commit, "openspec/changes/<slug>")`, nil when there is none. Idle days: whole days from its `Time` to `now`, rounded down, and 0 when `now` is earlier. `Uncommitted`, only when `Sources[0]` is a worktree: the count of `gitx.Git{Dir: <worktree path>, Env: g.Env}.UncommittedUnder("openspec/changes/<slug>")`; an error there makes that worktree unreadable for this change (`Path` the folder) and leaves `Uncommitted` nil. The error result is only for failures not tied to one source.

- [ ] **Step 1: Write the failing tests:** `TestChangesInWorktreesAndBranches` (worktree A holds `add-x`; `add-y` committed in worktree B and pushed to the origin it was cloned from, then fetched; `Load` returns `add-x`, `add-y` in that order and `add-y` with both sources, the worktree first); `TestArchivedChangeNotListed` (as `conflicts`' test: `add-x` archived on main, an unmerged branch `old` still holds it → not listed); `TestCommitNamesATaskLoaded` (`add-x` with a three-task plan and a commit carrying `Change: add-x` and `Task: 1` → `implementing`, Task 1 `started` with that commit's SHA); `TestIdleChange` (the newest commit for `add-x` made with `GIT_COMMITTER_DATE` 10.5 days before `now` → `LastActivity.Commit` is it, `IdleDays` 10, `Time` in RFC 3339); `TestFutureCommit` (a commit dated one day after `now` → `IdleDays` 0); `TestActivityFromFolderOnly` (a commit that changes `openspec/changes/add-x/design.md` and carries no trailer is the newest → it is the last activity); `TestMergedChangeActivity` (`add-x` merged into main with `--no-ff` and then an unrelated commit on main; `Sources[0]` is the default branch → last activity is the merge commit, not the tip); `TestUncommittedEdits` (worktree with committed `add-x` and uncommitted edits to `plan.md` and `design.md` → `Uncommitted` 2; read from a branch → nil); `TestNoCommitYet` (`add-x` untracked in a worktree, three files → `LastActivity` nil, `Uncommitted` 3); `TestBaseNotACommit` (`x-base` set to 40 zeros → commits empty, `LastActivity` nil, listed, nothing unreadable); `TestMetadataMissing` (no `.openspec.yaml` → `Metadata` nil, `LastActivity` nil, documents still read).
- [ ] **Step 2: Run** `go test ./internal/status/` — expected: FAIL (`Load` undefined).
- [ ] **Step 3: Implement** `Load` in `internal/status/load.go`.
- [ ] **Step 4: Run** `go test ./internal/status/` — expected: PASS.
- [ ] **Step 5: Commit** `feat(status): load every active change from its first source`.

### Task 5: The dkt status command

**Scenarios:** change-status#changes-in-worktrees-and-branches, change-status#no-active-changes, change-status#commits-not-yet-synced, change-status#one-change-in-focus, change-status#slug-not-active, change-status#unreadable-worktree, change-status#writes-nothing

**Files:**
- Create: `internal/cli/cmd_status.go`
- Modify: `internal/cli/envelope.go`, `internal/cli/run.go`, `cmd/dkt/main.go`
- Test: `internal/cli/cmd_status_test.go`

**Interfaces:**
- Consumes: `status.Load` (Task 4), `scanDefaultBranch`, `scanUnreadable`, `change.ValidateSlug`.
- Produces (package `cli`):
  - `Options` gains `Now time.Time` (the run's start time) and `Context` gains `Now time.Time`; `RunWith` copies it, using `time.Now()` when it is zero. `cmd/dkt` sets `Now: time.Now()`.
  - `type StatusResult struct { Changes []status.Change \`json:"changes"\` }`, never nil.
  - `statusCommand()`, usage `status [<slug>...]`, added to `commands` after `conflictsCommand`. Each invalid slug → `slug-invalid`; each valid slug that is not active → `change-not-active` naming it, with "archived as <folder>" when it has an archived use (the message `dkt conflicts` builds); a slug given twice is listed once. With slugs, only the named active changes are loaded (filter `inv.Changes` before `status.Load`), and `result.changes` holds them even when another slug produced a finding. Unreadable: `scanUnreadable(inv.Unreadable)` then `scanUnreadable` of `Load`'s.
  - Text mode, one block per change: `<slug>  <phase>  <tasks_done>/<tasks_total> tasks  <idle>` where `<idle>` is `idle <n>d` or `no commits`, followed by `  <k> uncommitted` when `uncommitted` is not null; then one line per source, `  <name>@<first 7 characters of the commit>` (the `shortCommit` of `dkt conflicts`). The shared output adds findings and `dkt status: ok`.

- [ ] **Step 1: Write the failing tests** (run with `Run(Options{..., Now: <fixed>})` as `cmd_conflicts_test.go` runs `conflicts`): `TestStatusLists` (worktree A `add-x`, worktree B `add-y` → exit 0, `ok` true, `result.changes` slugs `add-x`, `add-y`, each with `sources`, `phase`, `tasks`, `progress`; raw JSON has no `null` for `tasks`, `sources`, `specs` or `commits`); `TestStatusNoActiveChanges` (exit 0, `result.changes` `[]`); `TestStatusText` (text output has the line `add-x  ready  0/2 tasks  no commits  3 uncommitted` for an untracked two-task change of three files, then `  <worktree path>@<7 chars>`, and ends `dkt status: ok`); `TestStatusUnsyncedWritesNothing` (a commit with `Task: 1` and unticked steps → Task 1 `unsynced` true, and a snapshot of every file and of `git for-each-ref` before and after is identical; exit codes asserted); `TestStatusFocus` (`dkt status add-y` → only `add-y`, exit 0; `dkt status add-y add-y` → `add-y` once); `TestStatusSlugNotActive` (`add-q` → exit 1, `change-not-active`; an archived slug → message names its archive folder; `dkt status add-q add-x` → exit 1 and `add-x` still in `result.changes`); `TestStatusSlugInvalid` (`Bad_Slug` → exit 1, `slug-invalid`); `TestStatusUnreadableWorktree` (a listed worktree removed from disk and another worktree holding `add-x` → exit 3, `add-x` listed, the worktree in `unreadable`).
- [ ] **Step 2: Run** `go test ./internal/cli/ -run Status` — expected: FAIL (unknown command `status`).
- [ ] **Step 3: Implement** `Options.Now`, `Context.Now`, `statusCommand`, its registration, and `Now: time.Now()` in `cmd/dkt/main.go`.
- [ ] **Step 4: Run** `go test ./internal/cli/ ./cmd/...` — expected: PASS.
- [ ] **Step 5: Commit** `feat: dkt status lists every active change with its phase, tasks and activity`.

### Task 6: Docs, dogfood and sync

**Scenarios:** none

**Files:**
- Modify: `docs/ARCHITECTURE.md`, `AGENTS.md`, `README.md`, `README.zh-TW.md`, `docs/ideas/2026-10-24-web-workbench.md`
- Create: `docs/ideas/2026-10-24-status-followups.md`
- Modify: `openspec/changes/add-status/plan.md` (ticked by `dkt sync`)

- [ ] **Step 1: Update the docs.** `docs/ARCHITECTURE.md`: a codemap row for `internal/status` and index rows for `status.Load`, `status.Summarize`, `change.NamedTasks`, `Git.UncommittedUnder`, commit times and path limits of `Git.Commits`; the invariant on the environment also names the clock. `AGENTS.md` Status and both READMEs' status lines: `dkt status` exists. README's command list: `dkt status [<slug>...] [--json]` with the fields it gives now, both languages. `docs/ideas/2026-10-24-status-followups.md` (same style as `docs/ideas/2026-10-24-conflicts-followups.md`): owner, pull request and acceptance round are left out of `dkt status` (why each), and a base-drift field could reuse the base check of `dkt archive`. In the Order of `docs/ideas/2026-10-24-web-workbench.md`, item 4 is released in the next version (leave the version number to the release step).
- [ ] **Step 2: Dogfood.** `go run ./cmd/dkt status` and `go run ./cmd/dkt status --json` in this worktree: `add-status` is listed with phase `implementing` and Tasks 1–5 started; record the text output in the report.
- [ ] **Step 3: Sync and check.** `go run ./cmd/dkt sync add-status`, then `go run ./cmd/dkt sync add-status --check` and `go run ./cmd/dkt lint add-status` — expected: exit 0.
- [ ] **Step 4: Run** `gofmt -l .` (empty), `go vet ./...` and `go test ./...` — expected: PASS.
- [ ] **Step 5: Commit** `docs: map the status package and record its follow-ups`.
