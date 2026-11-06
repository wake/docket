# Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `dkt conflicts [<slug>]` reports the requirements that two or more active changes, in any worktree or branch, both touch.

**Architecture:** A new `internal/conflicts` package reads every copy of every active change from the workspace scan (worktree files from disk, branch files from git), parses each distinct copy with OpenSpec in a scratch directory that has a fixed proposal, and finds requirements touched by two or more slugs. `internal/cli/cmd_conflicts.go` wires it to the scan, OpenSpec and the output envelope.

**Tech Stack:** Go 1.24 standard library; git; the OpenSpec CLI (`openspec show --json --deltas-only`).

**Spec:** `openspec/changes/add-dkt-conflicts/proposal.md`, `specs/conflict-detection/spec.md`, `specs/cli-output/spec.md`, `design.md`.

## Global Constraints

- Requires OpenSpec 1.14.0 or later, through `openSpecRunner`; OpenSpec runs with `OPENSPEC_TELEMETRY=0` (the Runner adds it).
- Never read through a symbolic link: a link on the path to a copy's `specs/` folder, or anywhere under it, makes that copy unreadable. Files outside `specs/` (proposal, design, plan) are never read.
- No network: only existing refs are read.
- Output follows `cli-output`: one JSON object with `--json`; exit 0 nothing to report, 1 findings, 3 unreadable, 4 environment.
- Operation names and order: `ADDED`, `MODIFIED`, `REMOVED`, `RENAMED-FROM`, `RENAMED-TO`.
- Name fold: trim, lower-case, every run of whitespace as one space.
- Public repository: no host paths or names in code, tests or docs.

## Review Focus

- A change whose `proposal.md` holds only `dkt new`'s front matter: it must still be compared (the scratch proposal, not the change's).
- The same change in a worktree with uncommitted edits and on its branch: both copies parsed, sources reported per copy.
- A worktree removed while the scan runs, or a spec file replaced by a link: unreadable, never empty.
- OpenSpec failing on one copy: other overlaps still reported, exit 3.
- Many branches holding the same change: parsed once.

## Reuse

| Need | Existing (path:symbol) | Decision | Reason | Evidence |
|---|---|---|---|---|
| Find change folders in worktrees and branches | `internal/scan/scan.go:Changes` | extend | add the commit a branch use was read from, which reading its files needs | `rg -n "func Changes" internal/scan/` |
| List files under a folder of a commit | `internal/gitx/refs.go:TreeDirs` | new | `TreeDirs` lists directories only; files need `ls-tree -r` with modes to tell links apart | `rg -n "ls-tree" internal/gitx/` |
| Read many files of a commit | `internal/gitx/blobs.go:ReadBlobs` | reuse | one `cat-file --batch` call per commit | `rg -n "func (g Git) ReadBlobs" internal/gitx/` |
| Refuse links on a worktree path | `internal/safepath/safepath.go:Check`, `internal/safefile/safefile.go:ReadNoFollow` | reuse | the scan and `dkt init` already use them | `rg -n "safepath.Check\|ReadNoFollow" internal/` |
| Parse a change's deltas | `internal/openspec/json.go:Runner.ShowChange`, `Delta`, `Rename` | reuse | lint's parser of `openspec show --json --deltas-only` | `rg -n "func (r \*Runner) ShowChange" internal/openspec/` |
| Locate and version-check OpenSpec | `internal/cli/runner.go:openSpecRunner` | reuse | gives `openspec-missing` / `openspec-unsupported` | `rg -n "func openSpecRunner" internal/cli/` |
| Command, outcome, findings, exit codes | `internal/cli/command.go:Command`, `internal/cli/envelope.go:ExitCode` | reuse | every command uses them | `rg -n "var commands" internal/cli/` |
| Requirement-name fold | none | new | lint compares exact names; OpenSpec's `foldRequirementName` has no Go counterpart | `rg -n -i "fold\|ToLower" internal/lint/ internal/change/` |
| Bounded parallel calls | none | new | no command runs OpenSpec in parallel yet | `rg -n "WaitGroup\|errgroup" internal --glob '!**/*_test.go'` |
| Read a worktree folder without escaping it | `internal/cli/cmd_lint.go:runLint` (`os.OpenRoot`) | reuse | `os.Root` keeps reads inside the folder even if a path is swapped for a link after the check | `rg -n "os.OpenRoot" internal/` |
| Test repositories and fake OpenSpec | `internal/testutil/repo.go:NewRepo`, `internal/testutil/openspec.go:FakeOpenSpec`, `OpenSpecAnswer` | reuse | the CLI tests of `lint` use them | `rg -n "func OpenSpecAnswer" internal/testutil/` |

---

### Task 1: Copies of active changes

**Scenarios:** conflict-detection#archived-change-left-on-a-branch, conflict-detection#linked-delta-spec-in-another-worktree

**Review:** judgment

**Files:**
- Modify: `internal/scan/scan.go`, `internal/scan/scan_test.go`
- Modify: `internal/gitx/refs.go`, `internal/gitx/refs_test.go`
- Create: `internal/conflicts/copies.go`, `internal/conflicts/copies_test.go`

**Interfaces:**
- Produces: `scan.Use.Commit string` with tag `json:"-"`: the commit of a ref use, empty for a worktree use.
- Produces: `gitx.TreeFile{Path string; Link bool}` and `func (g Git) TreeFiles(commit, dir string) ([]TreeFile, error)`: every blob under `dir` (recursive, full-tree paths, git's order); `Link` is true for mode `120000`; submodules are left out; a missing `dir` gives an empty slice.
- Produces: `conflicts.Copy{Slug, Folder string; Sources []string; Files map[string][]byte}`; `Folder` is the change folder (`openspec/changes/<slug>`), `Files` maps paths under `specs/` (such as `booking/spec.md`) to contents, only files named `spec.md`.
- Produces: `func Copies(g gitx.Git, inv scan.Inventory) ([]Copy, []scan.Unreadable, error)`: one Copy per distinct content of each active slug, in scan order, with every source holding that content; slugs with any archived use are left out. An unreadable copy gives a `scan.Unreadable{Source: "<source>:<folder>", Reason}` and no Copy.

- [ ] **Step 1: Write the failing tests**
  - `TestChangesRecordsCommit` (scan): a branch use's `Commit` equals `git rev-parse <branch>`; a worktree use's is "".
  - `TestTreeFiles` (gitx): a commit with `d/a/spec.md`, `d/b/x.md` and a link `d/c/spec.md` gives three entries, the link with `Link: true`; `TreeFiles(commit, "missing")` gives an empty slice and no error.
  - `TestCopiesMergesIdentical`: `add-x` on `main` and `refs/heads/feat` with the same `specs/booking/spec.md` gives one Copy with both sources; editing the file in a worktree adds a second Copy for that worktree.
  - `TestCopiesSkipsArchived`: `add-x` archived on `main` and active on branch `old` gives no Copy for `add-x`.
  - `TestCopiesLinkedSpec`: a worktree whose `openspec/changes/add-y/specs/booking/spec.md` is a link gives an Unreadable whose Source is `<worktree>:openspec/changes/add-y` and whose Reason names the link, and no Copy from that worktree; the same for a link committed on a branch.
  - `TestCopiesNoSpecs`: an active change without `specs/` gives a Copy with no files.
  - `TestCopiesFolderGone`: an Inventory taken from a scan, then the worktree's `openspec/changes/add-y/` removed before `Copies` runs → an Unreadable for that copy (Reason says the folder is gone since the scan), not an empty Copy.
  - `TestCopiesLinkedSpecsDir`: a worktree whose `openspec/changes/add-y/specs` (the folder itself) is a link → Unreadable naming it.
- [ ] **Step 2: Run them and see them fail** — `go test ./internal/scan/ ./internal/gitx/ ./internal/conflicts/`: compile errors for the missing names.
- [ ] **Step 3: Implement** `Use.Commit` (set from the ref in `Changes`), `TreeFiles` (`ls-tree -r -z --full-tree <commit> -- <dir>/`), and `Copies`: a worktree copy first checks that the change folder still exists (`os.Lstat`; gone → Unreadable), then `safepath.Check(root, folder+"/specs")` (a link → Unreadable; missing → no files), then opens `specs/` with `os.OpenRoot` and walks `root.FS()`, where any entry of type link makes the copy Unreadable and `spec.md` files are read through the Root; any read error is the copy's Unreadable. Ref copies use `TreeFiles` (a `Link` entry → Unreadable) and one `ReadBlobs` per commit. Copies are keyed by their sorted (path, content) pairs.
- [ ] **Step 4: Run the tests and see them pass**, then `go vet ./...`.
- [ ] **Step 5: Commit** with the trailer `Change: add-dkt-conflicts/1`.

### Task 2: Parse copies with OpenSpec

**Scenarios:** conflict-detection#change-still-being-proposed, conflict-detection#copy-openspec-cannot-parse

**Review:** judgment

**Files:**
- Create: `internal/conflicts/parse.go`, `internal/conflicts/parse_test.go`

**Interfaces:**
- Consumes: `Copy` from Task 1; `openspec.Delta`, `openspec.Rename`.
- Produces: `conflicts.Touch{Capability, Requirement, Operation string}`; `conflicts.Parsed{Copy Copy; Touches []Touch}`.
- Produces: `type Show func(dir, slug string) ([]openspec.Delta, error)` and `func Parse(copies []Copy, show Show) ([]Parsed, []scan.Unreadable, error)`: results in the order of `copies`; a copy whose `show` fails gives one Unreadable per source, `Source: "<source>:<folder>"`, `Reason`: the error text.
- Produces: `const Proposal`: the scratch `proposal.md`, with a `## Why` and a `## What Changes` section.

- [ ] **Step 1: Write the failing tests** with a fake `Show` that reads the scratch directory:
  - `TestParseScratchCopy`: the fake sees `<dir>/openspec/changes/<slug>/proposal.md` equal to `Proposal` and `specs/booking/spec.md` equal to the copy's file, and no other file; the change's own `proposal.md` is never written.
  - `TestParseTouches`: deltas ADDED "A", MODIFIED "B", REMOVED "C" and RENAMED from "D" to "E" in `booking` give the touches `ADDED A`, `MODIFIED B`, `REMOVED C`, `RENAMED-FROM D`, `RENAMED-TO E`.
  - `TestParseFailure`: the fake fails for one copy with two sources; Parse returns the other copies, two Unreadables with the error text, and a nil error.
  - `TestParseRunsFour`: with eight copies and a fake that counts concurrent calls, the peak is at most 4 and above 1.
  - `TestParseRemovesScratch`: every scratch directory is gone after Parse returns.
- [ ] **Step 2: Run them and see them fail** — `go test ./internal/conflicts/`.
- [ ] **Step 3: Implement** `Parse`: each copy in its own `os.MkdirTemp` directory, removed after its call; four workers; RENAMED maps `Rename.From` and `Rename.To`; the other operations map each of `Requirements`.
- [ ] **Step 4: Run the tests and see them pass**, then `go vet ./...`.
- [ ] **Step 5: Commit** with the trailer `Change: add-dkt-conflicts/2`.

### Task 3: Find overlaps

**Scenarios:** conflict-detection#both-modify-one-requirement, conflict-detection#rename-against-modify, conflict-detection#same-capability-different-requirements, conflict-detection#both-add-the-same-name, conflict-detection#names-differ-in-case, conflict-detection#copies-that-differ, conflict-detection#own-copies-not-compared, conflict-detection#sources-listed

**Review:** checks

**Files:**
- Create: `internal/conflicts/find.go`, `internal/conflicts/find_test.go`

**Interfaces:**
- Consumes: `Parsed`, `Touch` from Task 2.
- Produces: `conflicts.Entry{Slug string; Operations, Sources []string}` and `conflicts.Overlap{Capability, Requirement string; Changes []Entry}`, with JSON tags `capability`, `requirement`, `changes`, `slug`, `operations`, `sources`.
- Produces: `func Find(parsed []Parsed) []Overlap` (never nil; sorted by capability, folded requirement, then slug; operations in the global order; sources in first-seen order, without repeats) and `func Involving(overlaps []Overlap, slug string) []Overlap` (never nil).

- [ ] **Step 1: Write the failing table test** `TestFind`, one case per scenario above, with `Parsed` values built in the test:
  - two slugs MODIFIED `booking`/"Cancel booking" → one overlap, entries `add-x [MODIFIED]`, `add-y [MODIFIED]`;
  - `add-x` RENAMED-FROM "Cancel booking" (and RENAMED-TO "Cancel a booking"), `add-y` MODIFIED "Cancel booking" → one overlap on "Cancel booking";
  - MODIFIED "Cancel booking" against ADDED "Refund points" → no overlap;
  - both ADDED "Waitlist cap" → one overlap;
  - "Cancel booking" against "Cancel  Booking" → one overlap whose `Requirement` is "Cancel booking";
  - `add-x` copy 1 (branch) MODIFIED "Cancel booking", copy 2 (worktree) MODIFIED "Refund points", `add-y` MODIFIED "Refund points" → one overlap; `add-x`'s Sources is only the worktree;
  - one slug, two copies both MODIFIED "Cancel booking" → no overlap;
  - `add-y` one copy with sources worktree B and `refs/remotes/origin/add-y` → its entry lists both.
  - `TestFindOrder`: input given as `add-z`, `add-y`, `add-x` (reverse), touching `booking` and `accounts`, where `add-y` spells "cancel BOOKING" and `add-x` "Cancel booking" → overlaps sorted `accounts` before `booking`, entries sorted by slug, and the requirement spelled "Cancel booking" (from `add-x`, the first slug), not the first one in the input.
  - `TestFindMergesOperations`: `add-x` copy 1 MODIFIED "R" and RENAMED-TO "R", copy 2 REMOVED "R" and MODIFIED "R" → its entry's Operations is `[MODIFIED REMOVED RENAMED-TO]`, each once, in the global order.
  - `TestInvolving`: keeps the overlaps naming the slug.
- [ ] **Step 2: Run it and see it fail** — `go test ./internal/conflicts/`.
- [ ] **Step 3: Implement** `Find` and `Involving`, keyed by capability and folded name.
- [ ] **Step 4: Run the tests and see them pass**, then `go vet ./...`.
- [ ] **Step 5: Commit** with the trailer `Change: add-dkt-conflicts/3`.

### Task 4: The dkt conflicts command

**Scenarios:** conflict-detection#overlap-with-another-worktree, conflict-detection#one-active-change, conflict-detection#unrelated-overlap-hidden, conflict-detection#slug-not-active, conflict-detection#slug-possibly-in-an-unreadable-worktree, conflict-detection#overlap-found, conflict-detection#no-overlaps, conflict-detection#unreadable-worktree, conflict-detection#change-still-being-proposed, cli-output#conflicts-outside-a-repository, cli-output#init-outside-a-repository, cli-output#new-outside-a-repository, cli-output#sync-outside-a-repository, cli-output#lint-outside-a-repository

**Review:** judgment

**Files:**
- Create: `internal/cli/cmd_conflicts.go`, `internal/cli/cmd_conflicts_test.go`
- Modify: `internal/cli/command.go` (the `commands` list)
- Modify: `internal/cli/realopenspec_test.go`
- Modify: `AGENTS.md` (drop `dkt conflicts` from the commands that do not exist yet)

**Interfaces:**
- Consumes: `scan.Changes`, `conflicts.Copies`, `conflicts.Parse`, `conflicts.Find`, `conflicts.Involving`, `openSpecRunner`.
- Produces: `dkt conflicts [<slug>] [--json]` with `result` `{"changes": [...], "overlaps": [...]}` (both never null; `changes` is every active slug, archived-anywhere ones excluded, sorted and without repeats, whether or not a slug is given), findings `requirement-overlap` (one per overlap, message `<capability> / "<requirement>": add-x (MODIFIED), add-y (RENAMED-FROM)`) and `change-missing` (with `change` set).

- [ ] **Step 1: Write the failing tests** in `cmd_conflicts_test.go`, using `testutil.NewRepo`, worktrees and `FakeOpenSpec` with `OpenSpecAnswer` per slug:
  - `TestConflictsOtherWorktree`: `add-y` uncommitted in worktree B, both modify one requirement; run in A → exit 1, the overlap's `add-y` entry lists B's path.
  - `TestConflictsOneChange`: OpenSpec not on PATH (`DOCKET_OPENSPEC` pointing to a missing file) and, in turn: only `add-x`; `add-x` with two different copies (worktree and branch); `add-x` active plus `add-q` archived; no active change and one removed worktree → no `openspec-missing`, `result.overlaps` is `[]`, and the exit code is 0 (3 for the removed worktree).
  - `TestConflictsChangesList`: `add-y` in a worktree and on a branch, `add-x` in the current worktree, `add-q` archived on `main` and active on branch `old` → `result.changes` is exactly `["add-x", "add-y"]`, also with `dkt conflicts add-x`.
  - `TestConflictsTwoSlugs`: `dkt conflicts add-x add-y` → usage error, exit 2.
  - `TestConflictsFocus`: `add-x`/`add-y` overlap, `add-z` does not; `dkt conflicts add-z` → exit 0, no overlaps; `dkt conflicts add-x` → exit 1, one overlap.
  - `TestConflictsSlugArchived`: `add-q` archived as `openspec/changes/archive/2026-11-01-add-q` → `change-missing` naming that folder, exit 1.
  - `TestConflictsSlugUnreadable`: `add-q` nowhere, a worktree's directory removed → no `change-missing`, the worktree in `unreadable`, exit 3.
  - `TestConflictsNone`: two changes touching different requirements → exit 0, `findings` `[]`, text output `Compared 2 changes; no requirement is touched by two of them.`
  - `TestConflictsUnreadableWorktree`: an overlap and a removed worktree → the overlap reported, exit 3.
  - `TestConflictsOpenSpecMissing`: two changes, no OpenSpec → `openspec-missing`, exit 4.
  - `TestConflictsOutsideRepository`: outside a repository → `no-repository`, exit 4, the fake log empty. The other commands' scenarios in the same requirement are already tested (`TestOutsideRepository` in `command_test.go` for init, `TestNewOutsideRepository`, `TestSyncOutsideRepository`, `TestLintOutsideRepository`); they must still pass.
  - `TestRealOpenSpecConflicts` (skipped unless `DOCKET_TEST_OPENSPEC=1`, like the other real tests): `add-x`'s proposal has only front matter; `add-x` renames "Cancel booking" and `add-y` modifies "Cancel  Booking" → one overlap with `RENAMED-FROM` and `MODIFIED`.
- [ ] **Step 2: Run them and see them fail** — `go test ./internal/cli/ -run 'TestConflicts'`.
- [ ] **Step 3: Implement** `cmd_conflicts.go`: `Args: -1` (the dispatcher treats a positive `Args` as an exact count), and a `UsageError` for more than one positional; scan, `Copies`; with two or more distinct active slugs (counted from the active, not-archived uses, not from copies), `openSpecRunner` and a `Show` that runs `ShowChange` with a copy of the Runner whose `Dir` is the scratch directory; `Find`; with a slug, `Involving`, or `change-missing` when the slug is not active, unless a whole source is unreadable and the slug is archived nowhere. Human lines per the design. Add the command to `commands`, and update AGENTS.md.
- [ ] **Step 4: Run the tests and see them pass**, then `go vet ./...`, `go test ./...`, and `DOCKET_TEST_OPENSPEC=1 go test ./internal/cli/ -run TestRealOpenSpecConflicts`.
- [ ] **Step 5: Commit** with the trailer `Change: add-dkt-conflicts/4`.
