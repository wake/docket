# add-archive Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `dkt sync`, `dkt archive` and `dkt rebase`: tick plan.md from commit trailers, archive a change only after checking that nothing it replaces moved underneath it, and accept newer specs as a change's base.

**Architecture:** Small readers first: `internal/change` reads `.openspec.yaml`, rewrites its `x-base` line, finds the steps of each task and ticks them byte for byte; `internal/gitx` lists commits with their `Change` and `Task` trailers, uncommitted paths, and files at a revision that may be missing; `internal/delta` keeps each requirement's block text. A new package `internal/archive` holds the three operations (`Sync`, `Rebase`, `Archive`) and the checks they share (loading a change, the base comparison, the reading cross-check, evidence). It reads through `os.Root`, writes through `safepath.Check`, starts git through `gitx` and OpenSpec through an interface that `*openspec.Runner` satisfies, so its tests use a Go fake. `internal/cli` adds the three commands, holding the `openspec/` lock while they run.

**Tech Stack:** Go 1.24 (module floor), standard library plus the existing `go.yaml.in/yaml/v3`; `git` as an external program; OpenSpec 1.14.3 for `archive` at run time and in tests gated by `DOCKET_TEST_OPENSPEC`.

**Spec:** `openspec/changes/add-archive/proposal.md`, `design.md`, `specs/plan-sync/spec.md` and `specs/change-archive/spec.md`. Read all four; scenario IDs below refer to the two delta specs. OpenSpec behavior this plan relies on was checked against `@fission-ai/openspec@1.14.3`: `archive <slug> -y --json` prints `{"archive":{"change","archivedAs","path",...},...}` and continues past incomplete tasks under `--yes`; blocks are compared with `normalizeBlockRaw` (line endings normalized, then trimmed) and names match exactly after `normalizeRequirementName` (`dist/core/specs-apply.js`, `dist/core/parsers/requirement-blocks.js`).

## Global Constraints

- Module path `github.com/wake/docket`; `go 1.24` in go.mod; no new third-party dependency.
- Public repository: code, comments, messages and docs in English; no hostnames, customer names, personal paths or secrets. `evidence.md` is committed: it names the repository by its `origin` URL or `none`, never by a local path.
- None of the three commands commits, pushes or creates a branch. `sync` writes only checkbox characters of plan.md; `rebase` writes only the `x-base` line of `.openspec.yaml`; `archive` writes through `openspec archive` and then `evidence.md`.
- `sync` and `rebase` never start OpenSpec. `archive` checks the OpenSpec version before any other step.
- Every write goes through `safepath.Check` and opens the file with `O_NOFOLLOW`; a link on the path is a `symlink-refused` finding and nothing is written. Before `openspec archive`, the paths OpenSpec will write (the change folder, `openspec/changes/archive`, each touched main spec) get the same check.
- `sync`, `archive` and `rebase` hold the `openspec/` lock (`lockOpenSpec`) for the whole run.
- Every command accepts `--json` and `--default-branch <name>`, before or after positional arguments.
- Exit codes: 0 clean; 1 problems found or request refused; 2 usage error; 3 a needed source could not be read (wins over 1); 4 unusable environment (no default branch for `archive` or `rebase`, no usable OpenSpec for `archive`).
- With `--json`, standard output is exactly one JSON object with `tool`, `version`, `command`, `repo`, `ok`, `findings`, `unreadable`, `result`; every array in it is an array, never null.
- Finding rules, exactly: `change-missing`, `metadata-invalid`, `base-missing`, `no-commits`, `plan-missing`, `task-unknown`, `plan-unsynced`, `plan-unticked`, `no-task-commits`, `uncommitted-changes`, `behind-default-branch`, `change-not-merged`, `base-drift`, `name-taken`, `readings-disagree`, `openspec-failed`, `evidence-failed`, `no-merge-base`, `symlink-refused`, and the rules `dkt lint` reports.
- Only `internal/gitx` starts git; only `internal/openspec` starts OpenSpec; nothing reads `os.Getenv` or `os.Environ` below `cmd/dkt` (tests excepted).
- Line numbers are 1-based and count lines after a leading BOM is removed and CRLF or lone CR is turned into LF. `file` in findings is relative to the repository root, slash-separated.
- macOS and Linux only.
- `go test ./...` passes without network access and without an OpenSpec install; tests that need the real CLI run only when `DOCKET_TEST_OPENSPEC` is set.
- Every commit carries `--trailer "Change: add-archive" --trailer "Task: <N>"` and `--trailer "Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"` (use the model that actually wrote the commit).

## Review Focus

1. This repository's own history: `dkt sync --check` exits 0 for `add-init-and-new` and `add-lint`, and `dkt archive add-init-and-new add-lint` succeeds in a scratch clone with OpenSpec 1.14.3. Their history has merge commits that carry `Change:` without `Task:`, fix commits for earlier tasks, and plans ticked by hand. Run in Task 8.
2. plan.md with CRLF line endings and a BOM: `sync` changes the box characters and nothing else. Tests in Tasks 1 and 3.
3. Trailers git accepts that the plan does not have: `Task: 03`, `Task: 3a`, a Task trailer for a cancelled task, and a lowercase key `task: 3`, which git matches. The first three are `task-unknown`; the last one ticks Task 3. Tests in Tasks 2 and 3.
4. Checkboxes that are not steps: `- [ ]` inside a fence within a task, in Review Focus, and in Cancelled Tasks are never ticked and never stop the plan check. Tests in Tasks 1 and 6.
5. Blocks that differ between `x-base` and now only in line endings or trailing blank lines are not a drift; an edit inside a scenario is. Test in Task 4.

## Reuse

| Need | Existing (path:symbol) | Decision | Reason | Evidence |
|---|---|---|---|---|
| Reading `x-base`, `x-kind` and `x-source`, and moving `x-base` | `internal/change/new.go:AppendMetadata` | extend | `AppendMetadata` only appends; the reader and the line rewrite sit next to it with the same YAML library | `rg -n "x-base" --type go -g '!*_test.go'` → `internal/change/new.go` only |
| YAML parsing | `external: go.yaml.in/yaml/v3` | reuse | Already a direct dependency | `rg -n "yaml" go.mod` |
| The steps of each task section in plan.md | `internal/change/plan.go:ParsePlan` | extend | `ParsePlan` already finds `### Task N` sections; `Task` gains `Steps` | `rg -n "func ParsePlan" internal/change` |
| Ticking checkboxes byte for byte | none | new | `markdown.Lines` normalizes line endings, so a writer must map line numbers back to byte offsets; nothing writes Markdown yet | `rg -n '\[x\]' --type go -g '!*_test.go'` → none |
| Commits with their `Change` and `Task` trailers | `internal/gitx/git.go:Git` | extend | Only `gitx` starts git; nothing reads `git log` yet | `rg -n '"log"\|trailers' internal/gitx` → none |
| Uncommitted paths for the clean-tree check | `internal/gitx/added.go:Git.UncommittedAdded` | extend | Same `git status --porcelain=v1 -z` call, listing every path instead of added ones under given folders; one shared helper runs it | `rg -n '"status"' internal/gitx` → `added.go` only |
| A main spec at a revision where it may not exist | `internal/gitx/git.go:Git.Show` | extend | `Show` fails the same way for a missing path and a broken repository; `ShowIfExists` tells them apart without reading git's messages | `rg -n "func \(g Git\) Show" internal/gitx` |
| Whether HEAD contains the default branch tip | `internal/gitx/git.go:Git.IsAncestor` | reuse | Exact check | `rg -n "func \(g Git\) IsAncestor" internal/gitx` |
| Change folders at the default branch tip | `internal/gitx/git.go:Git.ListDirs` | reuse | Lists `openspec/changes/` at a revision | `rg -n "func \(g Git\) ListDirs" internal/gitx` |
| The new base for `rebase` | `internal/gitx/git.go:Git.MergeBase` | reuse | Returns ok false for unrelated histories | `rg -n "func \(g Git\) MergeBase" internal/gitx` |
| The default branch | `internal/scan/defaultbranch.go:ResolveDefaultBranch` | reuse | Same resolution as every command | `rg -n "func ResolveDefaultBranch" internal/scan` |
| Requirement blocks of main specs | `internal/delta/delta.go:ParseSpec` | extend | Its block loop already knows each block's range; `Requirement` gains `Text` | `rg -n "func \(d doc\) requirements" internal/delta` |
| Reading a change's deltas | `internal/delta/delta.go:Parse` | reuse | The OpenSpec-compatible reader from `add-lint` | `rg -n "func Parse\(" internal/delta` |
| Comparing Docket's reading with `openspec show --json --deltas-only` | `internal/delta/contract_test.go:openspecLines` | extend | The comparison moves into production as `archive.CrossCheck`, and the contract test calls it, so the real CLI checks the production code | `rg -n "deltas-only" --type go` → the contract test only |
| The lint step | `internal/lint/lint.go:Run` | reuse | `dkt lint` itself, one slug at a time | `rg -n "func Run" internal/lint` |
| Files the branch adds, for the lint step | `internal/cli/cmd_lint.go:addedFiles` | reuse | Same input `dkt lint` gives the routing rule | `rg -n "func addedFiles" internal/cli` |
| Running OpenSpec | `internal/openspec/runner.go:Runner` | reuse | Version gate and environment; `archive` takes it through a two-method interface | `rg -n "func \(r \*Runner\)" internal/openspec` |
| Serializing writers under `openspec/` | `internal/cli/cmd_new.go:lockOpenSpec` | reuse | The lock `dkt new` takes | `rg -n "func lockOpenSpec" internal/cli` |
| Mapping OpenSpec availability errors to exit 4 | `internal/cli/cmd_new.go:envOrErr` | reuse | Shared by every command that starts OpenSpec | `rg -n "func envOrErr" internal/cli` |
| Refusing writes through links | `internal/safepath/safepath.go:Check` | reuse | The repository's write rule | `rg -n "func Check" internal/safepath` |
| Findings, unreadable sources, exit codes, envelope | `internal/cli/envelope.go:Outcome` | reuse | The shared result model | `rg -n "type Outcome" internal/cli` |
| Registering the commands | `internal/cli/run.go:commands` | extend | Add `syncCommand`, `rebaseCommand`, `archiveCommand` | `rg -n "var commands" internal/cli` |
| A fake OpenSpec for command tests | `internal/testutil/openspec.go:FakeOpenSpec` | extend | Add `validate`, `show` and `archive` to the script | `rg -n "fakeOpenSpecScript" internal/testutil` |
| Test repositories | `internal/testutil/repo.go:NewRepo` | reuse | Temporary repositories with a controlled environment | `rg -n "func NewRepo" internal/testutil` |
| A proposal that lints clean, for command tests | `internal/cli/cmd_lint_test.go:lintProposal` | reuse | Already passes every proposal rule | `rg -n "const lintProposal" internal/cli` |
| Running sync, archive and rebase | none | new | No package closes changes; `internal/archive` holds all three and the checks they share | `rg -n "^package (archive\|plansync)" --type go` → none |

---

### Task 1: `.openspec.yaml` reader, plan steps and ticking

**Scenarios:** none

**Files:**
- Create: `internal/change/metadata.go`, `internal/change/tick.go`
- Modify: `internal/change/plan.go`
- Test: `internal/change/metadata_test.go`, `internal/change/tick_test.go`, `internal/change/plan_test.go`

**Interfaces:**
- Produces (package `change`):
  - `type Metadata struct { Schema, Created, Kind, Source, Base string }` — YAML keys `schema`, `created`, `x-kind`, `x-source`, `x-base`.
  - `func ParseMetadata(b []byte) (Metadata, error)` — an error when `b` is not YAML or not a mapping; a missing key leaves its field empty.
  - `func SetBase(b []byte, base string) ([]byte, error)` — replaces what follows `x-base:` on the one line that starts with `x-base:` with ` ` + `base`, keeping every other byte, the line's own ending, and a missing final newline. An error when there is no such line, more than one, or when `ParseMetadata` of the result does not give `Base == base`.
  - `type Step struct { Line int; Done bool }`, and `Task` gains `Steps []Step` — every unfenced line of the task's section (its heading up to the next unfenced heading of level 3 or less, as `markdown.Body` gives it) matching `^[ \t]*[-*+][ \t]+\[([ xX])\]`; `Done` when the mark is `x` or `X`.
  - `func Tick(b []byte, lines []int) ([]byte, error)` — for each distinct line number, as `markdown.Lines` counts lines, replaces the space of the first `[ ]` on that line with `x`. Every other byte, including a BOM and CRLF or lone CR endings, stays as it was. An error, and no result, when a number is out of range or its line has no `[ ]` after a bullet.

- [x] **Step 1: Write the failing tests:** `TestParseMetadata` (all five keys; `x-source: "a: b"` quoted; a missing `x-base` gives `""`; `schema: [` is an error; a YAML list is an error); `TestSetBase` (only the value changes and the bytes before and after are identical; CRLF file keeps `\r\n` on that line; a quoted `x-base: "abc"` is replaced; no final newline stays without one; no `x-base` line is an error; two `x-base` lines are an error); `TestPlanSteps` (four steps of a task with lines and `Done`; `[X]` is done; a `#### Notes` heading inside the task keeps the steps under it; `  - [ ] nested` is a step; a fenced `- [ ] x` inside the task, a `- [ ]` under `## Review Focus` and a `- [ ]` under `## Cancelled Tasks` belong to no task); `TestTick` (LF plan; a plan with a BOM and CRLF endings differs from the input only in the ticked characters; a lone-CR plan; a line listed twice is ticked once; line 0, a line past the end and a line that holds `[x]` in the input are errors).
- [x] **Step 2: Run** `go test ./internal/change/` — expected: FAIL (undefined: `ParseMetadata`, `SetBase`, `Tick`, `Steps`).
- [x] **Step 3: Implement** the functions and the `Steps` field above.
- [x] **Step 4: Run** `go test ./internal/change/` — expected: PASS.
- [x] **Step 5: Commit** `feat(change): read .openspec.yaml and tick plan steps in place`.

### Task 2: Commits with trailers, uncommitted paths, optional files

**Scenarios:** none

**Files:**
- Create: `internal/gitx/log.go`
- Modify: `internal/gitx/added.go`, `internal/gitx/git.go`
- Test: `internal/gitx/log_test.go`, `internal/gitx/added_test.go`, `internal/gitx/git_test.go`

**Interfaces:**
- Produces (package `gitx`):
  - `type Commit struct { SHA, Subject string; Change, Task []string }`
  - `func (g Git) Commits(base, head string) ([]Commit, error)` — runs `git log -z --reverse --format=%H%x1f%s%x1f%(trailers:key=Change,valueonly,separator=%x1e)%x1f%(trailers:key=Task,valueonly,separator=%x1e) <base>..<head> --`, oldest first. Trailer values are trimmed and empty ones dropped.
  - `func (g Git) Uncommitted() ([]string, error)` — the sorted paths `git status --porcelain=v1 -z --untracked-files=normal --no-renames` lists: staged, unstaged and untracked, not ignored. `UncommittedAdded` and `Uncommitted` share one helper that runs `git status`.
  - `func (g Git) ShowIfExists(rev, path string) ([]byte, bool, error)` — false when `git ls-tree -z --name-only <rev> -- <path>` lists nothing; the blob otherwise; an error for an unknown revision.

- [x] **Step 1: Write the failing tests:** `TestCommits` (three commits after a base: one with `Change: add-x` and `Task: 1`, one with `Change: add-x`, `Task: 1` and `Task: 2`, one without trailers; order oldest first; subjects; the base commit is not listed; a lowercase `task: 3` key is read; `Task: 4` in a paragraph that is not the last is not read); `TestUncommitted` (clean repository → empty; a modified tracked file, a staged new file and an untracked file are listed; an ignored file is not); `TestShowIfExists` (an existing file at a revision; a missing path → false and no error; an unknown revision → error).
- [x] **Step 2: Run** `go test ./internal/gitx/` — expected: FAIL (undefined: `Commits`, `Uncommitted`, `ShowIfExists`).
- [x] **Step 3: Implement** the three methods and the shared status helper.
- [x] **Step 4: Run** `go test ./internal/gitx/` — expected: PASS.
- [x] **Step 5: Commit** `feat(gitx): list commits with trailers, uncommitted paths and optional files`.

### Task 3: `dkt sync`

**Scenarios:** `plan-sync#task-with-commits-ticked`, `plan-sync#task-without-commits-left-alone`, `plan-sync#two-tasks-in-one-commit`, `plan-sync#nothing-else-changes`, `plan-sync#trailer-for-an-unknown-task`, `plan-sync#check-finds-unticked-work`, `plan-sync#check-on-a-synced-plan`

**Files:**
- Create: `internal/archive/archive.go` (package doc, shared types), `internal/archive/load.go`, `internal/archive/sync.go`, `internal/cli/cmd_sync.go`
- Modify: `internal/cli/run.go`
- Test: `internal/archive/sync_test.go`, `internal/cli/cmd_sync_test.go`

**Interfaces:**
- Consumes: `change.ParseMetadata`, `change.Tick`, `Task.Steps` (Task 1); `Git.Commits` (Task 2); `change.ParsePlan`.
- Produces (package `archive`; its doc says it closes changes: `Sync` ticks plan.md, `Archive` checks and archives, `Rebase` moves a change's base):
  - `type Finding struct { Rule, Change, File string; Line int; Message string }` and `type Unreadable struct { Source, Reason string }`, with the meanings `lint` gives them.
  - `type Env struct { Git gitx.Git; Root string; Default scan.DefaultBranch }` — `Root` is the repository root and `Git` runs there; `Default` is used by `Archive` and `Rebase` only.
  - `type TaskTicks struct { Task, Steps int }`, JSON names `task` and `steps`.
  - `type SyncReport struct { Change string; Ticked []TaskTicks; Findings []Finding; Unreadable []Unreadable }`
  - `func Sync(e Env, slug string, check bool) (SyncReport, error)` — the error is for failures not tied to the change (git itself failing).
  - Unexported, used by Tasks 4 and 6: a loader that reads `openspec/changes/<slug>/` through `os.Root`: `.openspec.yaml`, plan.md when asked, and every `specs/<capability-path>/spec.md` as `delta.Delta` by capability path. It takes two switches from its caller: whether to read plan.md, and whether `x-base` must name a commit (`Sync` and `Archive` say yes, `Rebase` no). Findings: no folder → `change-missing`; `.openspec.yaml` missing, not YAML or without `x-base` → `metadata-invalid`; `x-base` that is not a commit (`Git.RefExists(base + "^{commit}")`) when it must be → `base-missing`; plan.md missing when asked → `plan-missing`. A file that exists but cannot be read goes to `Unreadable`.
  - Unexported, used by Task 6: the sync computation, so the plan check reports exactly what `sync --check` reports. It keeps the commits whose `Change` values include the slug; a `Task` value that is not `^[1-9][0-9]*$` or not the number of a `### Task` heading is a `task-unknown` finding on plan.md naming the value and the short SHA (a cancelled task included); every other value selects that task, and the task's unticked steps are the lines to tick. It also returns, per task, the commits that name it, for `evidence.md`, and whether any commit for the change carries a `Task` trailer at all, known or not, for the plan check.
- `Sync`: HEAD unborn → `no-commits`; otherwise load, compute, and unless `check` write `change.Tick` of plan.md through `safepath.Check` and `O_NOFOLLOW` when there is something to tick. With `check`, each task that would be ticked is a `plan-unsynced` finding on its heading line, saying how many steps and to run `dkt sync <slug>`, and nothing is written. `Ticked` lists the tasks ticked (or that would be), with how many steps, in plan order. `task-unknown` findings never stop the other tasks from being ticked.
- `dkt sync <slug> [--check]`: exactly one slug, else a usage error. It takes `lockOpenSpec`, calls `archive.Sync` with `Env{Git: c.Git, Root: c.Repo.Root}`, never builds an OpenSpec runner, and returns `Result` `{"change":"add-x","check":false,"ticked":[{"task":3,"steps":4}]}` (`ticked` is `[]` when empty) and one human line per task, `ticked 4 steps of Task 3`. One helper in `cmd_sync.go` turns `archive.Finding` and `archive.Unreadable` into the outcome's lists; `rebase` and `archive` use it too.

- [x] **Step 1: Write the failing tests** in `internal/archive/sync_test.go`, on repositories built with `testutil.NewRepo` whose commit messages end in a trailer paragraph: `TestTaskWithCommitsTicked` (two commits with `Change: add-x` and `Task: 3`, Task 3 has four unticked steps → all four ticked, `Ticked` is `[{3 4}]`); `TestTaskWithoutCommitsLeftAlone` (Task 4 unchanged byte for byte); `TestTwoTasksInOneCommit`; `TestNothingElseChanges` (plan.md with a BOM and CRLF endings differs only in the ticked characters, and `git status --porcelain` lists plan.md only); `TestTrailerForAnUnknownTask` (`Task: 9` → one `task-unknown` finding naming 9 and the commit, Task 1 still ticked; also `Task: 03`, `Task: 3a` and a Task trailer for a task under Cancelled Tasks are `task-unknown`); `TestCheckFindsUntickedWork` (plan.md unchanged, one `plan-unsynced` finding for Task 2); `TestCheckOnASyncedPlan` (after `Sync`, `Sync` with check has no findings); `TestSyncOtherChangeIgnored` (a commit with `Change: add-y` and `Task: 1` ticks nothing in `add-x`); `TestSyncLoadFindings` (missing folder, `.openspec.yaml` without `x-base`, `x-base` not a commit, no plan.md: one finding each with the rules above). In `internal/cli/cmd_sync_test.go`: `TestSyncCommand` (exit 0 and the JSON result above; `DOCKET_OPENSPEC` pointing at a path that does not exist changes nothing); `TestSyncCheckExitsOne`; `TestSyncNeedsOneSlug` (no slug and two slugs → exit 2); `TestSyncSymlinkedPlanRefused` (plan.md is a link → `symlink-refused`, the link target unchanged).
- [x] **Step 2: Run** `go test ./internal/archive/ ./internal/cli/` — expected: FAIL.
- [x] **Step 3: Implement** the package files and the command, and add `syncCommand` to `commands`.
- [x] **Step 4: Run** `go test ./internal/archive/ ./internal/cli/` — expected: PASS.
- [x] **Step 5: Commit** `feat: dkt sync ticks plan.md from commit trailers`.

### Task 4: Base comparison and `dkt rebase`

**Scenarios:** `change-archive#requirement-changed-since-base`, `change-archive#other-requirement-in-the-same-capability`, `change-archive#added-name-now-taken`, `change-archive#rebase-after-merging-the-default-branch`, `change-archive#nothing-to-move`

**Files:**
- Create: `internal/archive/base.go`, `internal/archive/rebase.go`, `internal/cli/cmd_rebase.go`
- Modify: `internal/delta/delta.go`, `internal/cli/run.go`
- Test: `internal/delta/delta_test.go`, `internal/archive/base_test.go`, `internal/archive/rebase_test.go`, `internal/cli/cmd_rebase_test.go`

**Interfaces:**
- Consumes: the loader (Task 3); `change.SetBase` (Task 1); `Git.ShowIfExists` (Task 2); `Git.MergeBase`.
- Produces:
  - `delta.Requirement` gains `Text string`: the block from its header line up to the next requirement header or `## ` line, lines joined with LF and trimmed with `markdown.Trim`, as OpenSpec's `normalizeBlockRaw` does. `Parse` and `ParseSpec` both set it.
  - `type Drift struct { Capability, Requirement, Base, Now string }`, JSON names `capability`, `requirement`, `base` and `now` — `Base` and `Now` are block texts, `""` when the requirement is absent on that side.
  - Unexported, used by Task 6: the base check over a loaded change. For each capability of its deltas it reads `openspec/specs/<capability>/spec.md` at `x-base` (`ShowIfExists`; missing → no blocks) and in the worktree (missing → no blocks), and matches requirements by exact name. A MODIFIED or REMOVED name, or a RENAMED FROM name, whose `Text` differs between the two sides is a `Drift` and a `base-drift` finding on the worktree main spec, at the requirement's line when it exists there, saying to update the delta and then run `dkt rebase <slug>`. An ADDED name present now and absent at `x-base` is a `Drift` and a `name-taken` finding. It also returns how many requirements it checked, for `evidence.md`.
  - `type RebaseReport struct { Change, From, To string; Moved bool; Changed []Drift; Findings []Finding; Unreadable []Unreadable }`
  - `func Rebase(e Env, slug string) (RebaseReport, error)` — loads the change without its plan and without requiring `x-base` to be a commit; the new base is `MergeBase(HEAD, e.Default.Commit)`, and no merge base is a `no-merge-base` finding. When it equals `x-base`, nothing is written and `Moved` is false. Otherwise it lists in `Changed` every requirement the change touches (ADDED, MODIFIED and REMOVED names, both names of a RENAMED pair) whose block differs between the old and the new base, both read with `ShowIfExists`, then writes `change.SetBase` of `.openspec.yaml` through `safepath.Check` and `O_NOFOLLOW`. An old `x-base` that is not a commit counts as having no blocks, so the base still moves; that case gives no `base-missing` finding.
- `dkt rebase <slug>`: exactly one slug; the default branch resolved as `dkt new` resolves it (`scan.ErrNoDefaultBranch` → `EnvError`); `lockOpenSpec`; no OpenSpec runner. `Result` `{"change","from","to","moved","changed":[{"capability","requirement","base","now"}]}`; human lines `x-base: <from7> -> <to7>` or `x-base is already <to7>`, then `changed since the old base: <capability> / <requirement>` per drift.

- [x] **Step 1: Write the failing tests:** in `internal/delta/delta_test.go`, `TestRequirementText` (the block of a main-spec requirement and of an ADDED one, with the header line, CRLF input giving LF text, trailing blank lines trimmed). In `internal/archive/base_test.go`, repositories where a commit after `x-base` edits `openspec/specs/booking/spec.md`: `TestRequirementChangedSinceBase` (`add-y` modifies "Cancel booking" and the main spec's "Cancel booking" changed after `x-base` → one `Drift` with both texts and a `base-drift` finding); `TestOtherRequirementInTheSameCapability` (only "Refund points" changed → no findings); `TestAddedNameNowTaken` (`add-y` adds "Waitlist cap", which appeared after `x-base` → `name-taken`); `TestBlocksDifferingOnlyInWhitespace` (CRLF endings and one more trailing blank line → no drift; a changed `THEN` line inside a scenario → a drift); `TestRemovedAndRenamedChecked` (a REMOVED name and a RENAMED FROM name whose blocks changed → two drifts); `TestCapabilityMissingAtBase` (a delta for a capability with no main spec at `x-base` and none now → no findings). In `internal/archive/rebase_test.go`: `TestRebaseAfterMergingTheDefaultBranch` (branch for `add-y` made at `c1`, `x-base` `c1`; main then changes "Cancel booking"; main merged into the branch → `x-base` is the new merge base, `Changed` names "Cancel booking", `.openspec.yaml` differs from before only on the `x-base` line); `TestNothingToMove` (`Moved` false, file unchanged, no findings); `TestRebaseOldBaseMissing` (an `x-base` that names no commit → moved, every touched requirement present at the new base listed); `TestRebaseSymlinkedMetadataRefused` (`.openspec.yaml` is a link → `symlink-refused`, the link target unchanged). In `internal/cli/cmd_rebase_test.go`: `TestRebaseCommand` (JSON result and exit 0 with `DOCKET_OPENSPEC` pointing at a path that does not exist); `TestRebaseNoDefaultBranch` (exit 4).
- [x] **Step 2: Run** `go test ./internal/delta/ ./internal/archive/ ./internal/cli/` — expected: FAIL.
- [x] **Step 3: Implement** `Text`, the base check, `Rebase` and the command, and add `rebaseCommand` to `commands`.
- [x] **Step 4: Run** `go test ./internal/delta/ ./internal/archive/ ./internal/cli/` — expected: PASS.
- [x] **Step 5: Commit** `feat: dkt rebase moves x-base and lists what changed underneath`.

### Task 5: Reading cross-check

**Scenarios:** `change-archive#readings-disagree`

**Files:**
- Create: `internal/archive/crosscheck.go`
- Modify: `internal/delta/contract_test.go`
- Test: `internal/archive/crosscheck_test.go`

**Interfaces:**
- Produces:
  - `type Readings struct { Docket, OpenSpec []string }`, JSON names `docket` and `openspec`.
  - `func CrossCheck(deltas map[string]delta.Delta, show string) (Readings, bool, error)` — `deltas` is keyed by capability path; `show` is the standard output of `openspec show <slug> --type change --json --deltas-only`. Both sides become lines `<capability>: <OPERATION>|<name>|<scenario names as %q of a []string>`, or `<capability>: RENAMED|<from>|<to>`, capabilities sorted. On Docket's side each capability lists ADDED, MODIFIED, REMOVED (no scenarios) and RENAMED entries in that order and in document order within each; on OpenSpec's side entries keep the order of the output's `deltas` array within each `spec`. The bool is true when the two line lists are equal; the error is for output that is not the expected JSON. Neither list is ever nil.
- `TestDeltasAgreeWithOpenSpec` in `internal/delta/contract_test.go` keeps its examples and its `DOCKET_TEST_OPENSPEC` gate, reads each copied example into a `map[string]delta.Delta`, and fails with both readings when `archive.CrossCheck` does not agree, or when OpenSpec's reading is empty. Its own JSON types and line helpers go.

- [x] **Step 1: Write the failing tests:** `TestCrossCheckAgrees` (a hand-written `show` output for one capability with ADDED, MODIFIED, REMOVED and RENAMED entries and the matching deltas → true and the expected lines); `TestReadingsDisagree` (OpenSpec's output lacks a requirement Docket reads → false, and the requirement is in `Docket` but not in `OpenSpec`); `TestCrossCheckScenarioNames` (one scenario name differs → false); `TestCrossCheckBadJSON` (an error).
- [x] **Step 2: Run** `go test ./internal/archive/` — expected: FAIL.
- [x] **Step 3: Implement** `CrossCheck` and rewrite the contract test to use it.
- [x] **Step 4: Run** `go test ./internal/archive/ ./internal/delta/`, then with `DOCKET_TEST_OPENSPEC` set to OpenSpec 1.14.3, `go test ./internal/delta/ -run TestDeltasAgreeWithOpenSpec` — expected: PASS both times, the second without a skip.
- [x] **Step 5: Commit** `feat(archive): compare Docket's reading of a change with OpenSpec's`.

### Task 6: Archive steps and evidence

**Scenarios:** `change-archive#uncommitted-changes`, `change-archive#branch-behind-the-default-branch`, `change-archive#change-not-merged-yet`, `change-archive#second-of-two-fails`, `change-archive#unticked-task`, `change-archive#squashed-pull-request`, `change-archive#requirement-changed-since-base`, `change-archive#two-changes-in-one-run-touch-one-requirement`, `change-archive#other-requirement-in-the-same-capability`, `change-archive#added-name-now-taken`, `change-archive#readings-disagree`, `change-archive#evidence-written`, `change-archive#nothing-committed`

**Files:**
- Create: `internal/archive/run.go`, `internal/archive/evidence.go`
- Test: `internal/archive/run_test.go`, `internal/archive/evidence_test.go`

**Interfaces:**
- Consumes: the loader and the sync computation (Task 3), the base check (Task 4), `CrossCheck` (Task 5), `Git.Uncommitted` (Task 2), `Git.IsAncestor`, `Git.ListDirs`, `Git.Head`, `lint.Report`.
- Produces:
  - `type Runner interface { CheckVersion() (string, error); Run(args ...string) (string, error) }` — `*openspec.Runner` satisfies it.
  - `type Input struct { Env; OpenSpec Runner; Lint func(slug string) lint.Report; Version, Origin string; Slugs []string }` — `Version` is Docket's version and `Origin` the `origin` URL, `""` when there is none.
  - `type Archived struct { Change, Folder, Evidence string }`, JSON names `change`, `folder` and `evidence` — repository-relative paths; `Evidence` is `""` when writing it failed.
  - `type Stop struct { Change, Step string }`, JSON names `change` and `step` — `Change` is `""` for `preconditions`.
  - `type ArchiveReport struct { Archived []Archived; Stopped *Stop; Drifts []Drift; Readings *Readings; Findings []Finding; Unreadable []Unreadable }`
  - `func Archive(in Input) (ArchiveReport, error)` — the error is for OpenSpec being unavailable or unsupported (passed through so the command can exit 4) and for failures not tied to a change.
- Order: `in.OpenSpec.CheckVersion()` first. Then the `preconditions`, all checked before anything changes: HEAD unborn → `no-commits`; `Uncommitted` lists anything → `uncommitted-changes`, naming up to five paths and saying to commit or stash them; `IsAncestor(e.Default.Commit, HEAD)` false → `behind-default-branch`, saying to merge `<name>` first; for each slug, absent from `ListDirs(e.Default.Commit, "openspec/changes")` → `change-not-merged`. Then for each slug in order, the steps `plan`, `lint`, `base`, `validate`, `cross-check`, `archive`, `evidence`, stopping at the first step with a finding or an unreadable source, with `Stopped` naming the change and step:
  - `plan`: load with the plan, run the sync computation; its `task-unknown` findings, a `plan-unsynced` finding per task it would tick, a `plan-unticked` finding on the heading of every other task with an unticked step, naming the task's number and name, and one `no-task-commits` finding on plan.md when no commit since `x-base` carries `Change: <slug>` with a `Task:` trailer, asking whether the pull request was squashed.
  - `lint`: `in.Lint(slug)`; its findings and unreadable sources are reported as they are.
  - `base`: the base check; its drifts go to `Drifts`.
  - `validate`: `Run("validate", slug, "--strict")`; an `*openspec.FailedError` is an `openspec-failed` finding with OpenSpec's standard output and error output.
  - `cross-check`: `Run("show", slug, "--type", "change", "--json", "--deltas-only")`, then `CrossCheck` on the loaded deltas; disagreement or an error is `readings-disagree`, with `Readings` set.
  - `archive`: first `safepath.Check` on `openspec/changes/<slug>`, `openspec/changes/archive` and `openspec/specs/<capability>/spec.md` for every capability of the change's deltas; a link is `symlink-refused` and OpenSpec is not called. Then `Run("archive", slug, "-y", "--json")`; a failure is `openspec-failed`. The result's `archive.archivedAs` must match `^\d{4}-\d{2}-\d{2}-<slug>$` and name a directory under `openspec/changes/archive/`, else `evidence-failed`. The change counts as archived from here on.
  - `evidence`: write `openspec/changes/archive/<archivedAs>/evidence.md`; a failure is `evidence-failed` (or `symlink-refused`), and the change stays in `Archived` with `Evidence` `""`.
- `evidence.md`, exactly, with `|` in a cell written `\|` and line breaks in a cell written as spaces; tasks in plan order, commits oldest first, a task with no commit as one row `| N | none | |`:

```
# Evidence: <slug>

| Field | Value |
|---|---|
| Date | <YYYY-MM-DD from archivedAs> |
| Docket | <Version> |
| OpenSpec | <CheckVersion result> |
| Repository | <Origin, or none> |
| HEAD | <full SHA> |
| x-base | <full SHA> |
| x-kind | <x-kind, or none> |
| x-source | <x-source, or none> |

## Task commits

| Task | Commit | Subject |
|---|---|---|
| 1 | <full SHA> | <subject> |

## Checks

- `openspec validate <slug> --strict`: <first non-empty line of its standard output, trimmed>
- Base check: <n> requirements checked, none changed since x-base
```

- [x] **Step 1: Write the failing tests** in `internal/archive/run_test.go`, with a Go fake `Runner` that records its calls, answers `validate` and `show` from fields, and for `archive` moves the change folder to `openspec/changes/archive/2026-11-02-<slug>`, runs an optional per-slug hook, and prints the JSON OpenSpec prints; `Lint` is a stub returning a report per slug. `TestUncommittedChanges` (an untracked file → `uncommitted-changes`, `Stopped` `{"" preconditions}`, no call but `--version`, the tree unchanged); `TestBranchBehindTheDefaultBranch`; `TestChangeNotMergedYet` (the change exists only in a commit after the default tip); `TestSecondOfTwoFails` (the lint stub reports a finding for `add-y` → `add-x` archived, `add-y` untouched, the finding reported, `Stopped` `{add-y lint}`); `TestUntickedTask` (Task 5 with an unticked step and no commit → `plan-unticked` naming Task 5, `Stopped` at `plan`; a `- [ ]` under Review Focus does not stop it); `TestSquashedPullRequest` (every step ticked by hand, the only commits since `x-base` lack `Task:` trailers → `no-task-commits`, `Stopped` at `plan`); `TestArchiveStopsOnDrift` (the setup of `TestRequirementChangedSinceBase` → `Stopped` at `base`, `Drifts` set, no `validate` call); `TestArchivePassesOtherRequirement` (the setup of `TestOtherRequirementInTheSameCapability` → archived); `TestArchiveStopsOnTakenName` (the setup of `TestAddedNameNowTaken` → `name-taken`, `Stopped` at `base`); `TestArchiveSymlinkedSpecRefused` (`openspec/specs/booking/spec.md` is a link → `symlink-refused`, `Stopped` at `archive`, no `archive` call); `TestTwoChangesInOneRunTouchOneRequirement` (the hook for `add-x` rewrites "Cancel booking" in `openspec/specs/booking/spec.md` → `add-x` archived, `add-y` stopped at `base`); `TestArchiveReadingsDisagree` (`show` lacks a requirement → `readings-disagree`, `Readings` set, no `archive` call); `TestOpenSpecValidateFails` (a `FailedError` → `openspec-failed` with both outputs); `TestUnexpectedArchivedAs` (`archivedAs` of another slug → `evidence-failed`); `TestNothingCommitted` (after a successful run HEAD is unchanged and `git status --porcelain` lists the archive's changes). In `internal/archive/evidence_test.go`: `TestEvidenceWritten` (the exact text above for a change with two tasks, one without commits, an `x-source` holding `|`, and `Origin` `""` → `none`).
- [x] **Step 2: Run** `go test ./internal/archive/` — expected: FAIL.
- [x] **Step 3: Implement** `Archive` and the evidence writer.
- [x] **Step 4: Run** `go test ./internal/archive/` — expected: PASS.
- [x] **Step 5: Commit** `feat(archive): check, archive and record evidence for each change`.

### Task 7: `dkt archive`

**Scenarios:** `change-archive#uncommitted-changes`, `change-archive#evidence-written`, `change-archive#nothing-committed`, `change-archive#two-changes-in-one-run-touch-one-requirement`

**Files:**
- Create: `internal/cli/cmd_archive.go`
- Modify: `internal/cli/run.go`, `internal/testutil/openspec.go`
- Test: `internal/cli/cmd_archive_test.go`

**Interfaces:**
- Consumes: `archive.Archive` (Task 6), `addedFiles`, `lockOpenSpec`, `envOrErr`, `lint.Run`, `docket.Schema()`.
- `dkt archive <slug>...`: at least one slug, else a usage error. First `openspec.New(c.Repo.Root, c.Env)` and its `CheckVersion()`, both through `envOrErr`, so an unusable OpenSpec is reported before anything else; then the default branch resolved as `dkt new` resolves it; `lockOpenSpec`; the root opened once with `os.OpenRoot`, and `addedFiles(c)` computed once, before the first change; `Lint` runs `lint.Run` with `Slugs: []string{slug}` on that root; `Version` is `cli.Version`; `Origin` comes from `c.Repo.Origin`. An error from `archive.Archive` goes through `envOrErr`. `Result` `{"archived":[{"change","folder","evidence"}],"stopped":null|{"change","step"},"drifts":[...],"readings":null|{"docket":[...],"openspec":[...]}}`, every array `[]` when empty. Human lines: `archived <change> as <folder>` per change; `stopped at <step> for <change>` (`stopped at preconditions` when `change` is empty); per drift `<capability> / <requirement> at x-base:` and its lines indented by two spaces (or `  (absent)`), then `now:` and the same; for readings `docket:` and `openspec:` with their lines indented.
- `FakeOpenSpec` gains `validate` (prints `Change '<slug>' is valid`), `show` (prints the file named by `FAKE_OPENSPEC_SHOW`), and `archive` (moves `openspec/changes/<slug>` to `openspec/changes/archive/2026-01-02-<slug>` and prints `{"archive":{"change":"<slug>","archivedAs":"2026-01-02-<slug>","path":"<dir>/openspec/changes/archive/2026-01-02-<slug>"}}`), each honoring `FAKE_OPENSPEC_FAIL`.

- [x] **Step 1: Write the failing tests:** a fixture helper that builds a repository on `main` whose change `add-x` lints clean (`lintProposal`, a plan with one task whose steps are ticked and whose commit carries `Change: add-x` and `Task: 1`, one ADDED requirement with a scenario, `openspec/config.yaml` with `schema: docket`, `testutil.InstallSchema`), with an `origin` remote URL set, and asserts `dkt lint add-x` exits 0 before the test goes on. `TestArchiveCommandRefusesUncommitted` (exit 1, `uncommitted-changes`, `stopped` `{"change":"","step":"preconditions"}`); `TestArchiveCommandArchives` (fake OpenSpec with a matching `FAKE_OPENSPEC_SHOW` → exit 0, `archived` lists the folder and `evidence.md`, which names the origin URL, the commit for Task 1 and the fake's version; HEAD unchanged); `TestArchiveNeedsOpenSpec` (`DOCKET_OPENSPEC` pointing at a missing path → exit 4); `TestArchiveNeedsASlug` (exit 2); `TestArchiveWithOpenSpec`, skipped unless `DOCKET_TEST_OPENSPEC` is set, with the real CLI: `add-x` and `add-y` share an `x-base` and both modify `booking` / "Cancel booking"; `dkt archive add-x add-y` exits 1, `add-x` is archived with specs merged and `evidence.md` written, `add-y` is untouched and stopped at `base`, and HEAD is unchanged.
- [x] **Step 2: Run** `go test ./internal/cli/` — expected: FAIL.
- [x] **Step 3: Implement** the command and the fake's three commands, and add `archiveCommand` to `commands`.
- [x] **Step 4: Run** `go test ./internal/cli/`, then with `DOCKET_TEST_OPENSPEC` set, `go test ./internal/cli/ -run TestArchiveWithOpenSpec` — expected: PASS both times, the second without a skip.
- [x] **Step 5: Commit** `feat: dkt archive checks a change before OpenSpec merges it`.

### Task 8: Documentation and this repository's own changes

**Scenarios:** none

**Files:**
- Modify: `docs/ARCHITECTURE.md`, `AGENTS.md`, `docs/ideas/2026-10-24-init-and-new-followups.md`, `openspec/changes/add-archive/plan.md`

- [x] **Step 1: Update** `docs/ARCHITECTURE.md`: a codemap row for `internal/archive` (closes changes: sync, archive, rebase, the base check, the reading cross-check, evidence); the `internal/change` row also names plan steps and ticking; capability index rows for `change.ParseMetadata`, `change.SetBase`, `change.Tick`, `Git.Commits`, `Git.Uncommitted`, `Git.ShowIfExists`, `archive.Sync`, `archive.Rebase`, `archive.Archive` and `archive.CrossCheck`.
- [x] **Step 2: Update** `AGENTS.md` Status: `dkt init`, `dkt new`, `dkt lint`, `dkt sync`, `dkt archive` and `dkt rebase` exist; `dkt conflicts` comes with `add-conflicts`; drop the sentences about ticking plans by hand and leaving merged changes unarchived. In the follow-ups idea file, remove the item about the routing section naming commands that do not exist yet.
- [x] **Step 3: Run** on this repository, from the branch: `go run ./cmd/dkt sync add-init-and-new --check` and `go run ./cmd/dkt sync add-lint --check` — expected: exit 0 both. Then in a scratch clone of the branch (not this worktree), with `DOCKET_OPENSPEC` set to OpenSpec 1.14.3: `go run ./cmd/dkt archive add-init-and-new add-lint --default-branch main` — expected: exit 0, both archived, six main specs written (the five of `add-init-and-new` and `change-lint`), two `evidence.md` files. Record the outputs in the task report; commit nothing from the clone.
- [x] **Step 4: Run** `go vet ./...`, `go test ./...`, `go run ./cmd/dkt lint`, then `go run ./cmd/dkt sync add-archive` and `go run ./cmd/dkt sync add-archive --check` — expected: all pass, and the last exits 0.
- [x] **Step 5: Commit** `docs: map the archive package and say which commands exist`, including plan.md as ticked by `dkt sync`.
