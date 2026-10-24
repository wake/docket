# add-conflicts Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `dkt conflicts [<slug>]`: list every requirement that more than one active change touches, across all worktrees and branches the scan finds, so parallel work learns of an overlap before archiving would let one change replace the other.

**Architecture:** `internal/gitx` lists the regular files below a folder at a revision; `internal/scan` uses it to read every file below a folder from any scanned source (a worktree through `os.Root`, a branch through git). `internal/delta` gets the one rule that maps `specs/<path>/spec.md` to a capability path, which `lint` and `archive` switch to. A new package `internal/conflicts` loads each active, never-archived change's deltas from its first source and computes the overlaps as a pure function. `internal/cli` adds the read-only command.

**Tech Stack:** Go 1.24 (module floor), standard library; `git` as an external program. No OpenSpec at run time.

**Spec:** `openspec/changes/add-conflicts/proposal.md`, `design.md` and `specs/conflict-detection/spec.md`. Read all three; scenario IDs below refer to the delta spec. The scan it builds on is `openspec/specs/workspace-scan/spec.md`; exit codes and the envelope are `openspec/specs/cli-output/spec.md`.

## Global Constraints

- Module path `github.com/wake/docket`; `go 1.24` in go.mod; no new third-party dependency.
- Public repository: code, comments, messages and docs in English; no hostnames, customer names, personal paths or secrets.
- `dkt conflicts` writes nothing, takes no lock, starts no OpenSpec and never fetches.
- Every command accepts `--json` and `--default-branch <name>`, before or after positional arguments.
- Exit codes: 0 no overlap and nothing else to report; 1 an overlap or another finding; 2 usage error (more than one slug); 3 a source or a delta spec could not be read (wins over 1); 4 no git repository or no default branch.
- With `--json`, standard output is exactly one JSON object with `tool`, `version`, `command`, `repo`, `ok`, `findings`, `unreadable`, `result`; every array in it is an array, never null.
- Finding rules, exactly: `requirement-overlap`, `change-not-active`, `slug-invalid`.
- Operation names, exactly and in this order: `ADDED`, `MODIFIED`, `REMOVED`, `RENAMED-FROM`, `RENAMED-TO`.
- Only `internal/gitx` starts git; nothing reads `os.Getenv` or `os.Environ` below `cmd/dkt` (tests excepted). Readers take bytes, not paths.
- macOS and Linux only.
- `go test ./...` passes without network access and without an OpenSpec install.
- Every commit is made with `git commit --trailer "Change: add-conflicts" --trailer "Task: <N>" --trailer "Co-Authored-By: <the model that wrote it> <noreply@anthropic.com>"`, so the three trailers form the message's last paragraph. Never put a trailer in its own `-m`: git only reads the last paragraph.

## Review Focus

1. One change that renames "A" to "B" and modifies "B" in the same delta: B is touched by that change once, with `MODIFIED` and `RENAMED-TO`; A with `RENAMED-FROM`; the change never overlaps with itself. Test in Task 3.
2. Names that differ only in surrounding spaces overlap; names that differ in case do not, as OpenSpec matches them. Test in Task 3.
3. Nested capability paths: `a/b` and `a` are different capabilities, and the same requirement name in both is no overlap. Tests in Tasks 2 and 3.
4. A worktree whose change folder holds a link (a `spec.md` that links to a file outside the worktree, a `specs/` subfolder that links elsewhere): the link is not followed and not read. Test in Task 1.
5. The same slug in two worktrees (one with uncommitted edits) and on its remote-tracking branch: reported once, read from the scan's first source, never overlapping itself, every source listed. Test in Task 4.

## Reuse

| Need | Existing (path:symbol) | Decision | Reason | Evidence |
|---|---|---|---|---|
| Every active change and every archived use, from all sources | `internal/scan/scan.go:Scan` | reuse | The `workspace-scan` scan, with its source order | `rg -n "func Scan" internal/scan` |
| The default branch, with exit 4 when there is none | `internal/cli/cmd_new.go:resolveDefaultBranch` | reuse | Same resolution and error mapping as every command | `rg -n "func resolveDefaultBranch" internal/cli` |
| Regular files below a folder at a revision | `internal/gitx/git.go:Git.ListDirs` | extend | `ListDirs` lists directories one level deep; `ListFiles` sits next to it with the same `ls-tree` call, recursive and filtered by mode | `rg -n "ls-tree" internal/gitx` → `ListDirs`, `ShowIfExists` only |
| Every file below a folder in a scanned source | `internal/scan/scan.go:ReadFile` | extend | `ReadFile` reads one file from a worktree through `os.Root` or from a branch through `git show`; `ReadFiles` does the same for a folder, and `Unreadable` gains a `Path` for a single unreadable file | `rg -n "func ReadFile" internal/` → `scan.ReadFile` only |
| Capability path of a delta spec | `internal/lint/structure.go:deltas` | extend | `lint` and `archive` (`internal/archive/load.go:loadDeltas`) each carry the rule; it moves into `delta.CapabilityPath` and both call it, with no change in behavior | `rg -n 'TrimSuffix\(.*"/spec.md"' --type go` → `lint/structure.go`, `archive/load.go` |
| Reading a delta spec | `internal/delta/delta.go:Parse` | reuse | The OpenSpec-compatible reader; names come trimmed | `rg -n "func Parse\(" internal/delta` |
| Slug format check | `internal/change/new.go:ValidateSlug` | reuse | Same rule as `dkt new` | `rg -n "func ValidateSlug" internal/change` |
| Scan unreadable sources in the envelope | `internal/cli/cmd_new.go:newCommand` | extend | The mapping from `scan.Unreadable` (with the `worktree:` prefix) is inline in `newCommand`; it becomes `scanUnreadable`, used by `new` and `conflicts` | `rg -n "worktree:" internal/cli` → `cmd_new.go` only |
| Findings, unreadable sources, exit codes, envelope | `internal/cli/envelope.go:Outcome` | reuse | The shared result model; an overlap as a finding gives exit 1 without a special case | `rg -n "type Outcome" internal/cli` |
| Registering the command | `internal/cli/run.go:commands` | extend | Add `conflictsCommand` | `rg -n "var commands" internal/cli` |
| Test repositories and worktrees | `internal/testutil/repo.go:NewRepo` | reuse | Temporary repositories, `AddWorktree`, `CloneTo` | `rg -n "func \(r \*Repo\)" internal/testutil` |
| Overlaps between changes | none | new | Nothing compares changes with each other; `archive`'s base check compares one change with the main specs | `rg -n "overlap" --type go` → none |

---

### Task 1: Read every file below a folder from any source

**Scenarios:** none

**Files:**
- Modify: `internal/gitx/git.go`, `internal/scan/scan.go`
- Test: `internal/gitx/git_test.go`, `internal/scan/readfiles_test.go`

**Interfaces:**
- Produces (package `gitx`): `func (g Git) ListFiles(rev, dir string) ([]string, error)` — runs `git ls-tree -r -z <rev> -- <dir>/` and returns, sorted, the repository-relative paths of entries of type `blob` with mode `100644` or `100755`; links (`120000`) and submodules are left out. A missing `dir` gives an empty list and no error; an unknown revision is an error.
- Produces (package `scan`):
  - `Unreadable` gains `Path string`: empty when the whole source is unreadable, otherwise the repository-relative path of the one file or folder that could not be read.
  - `func ReadFiles(g gitx.Git, src Source, dir string, keep func(rel string) bool) (map[string][]byte, []Unreadable, error)` — every regular file below `dir` (repository-relative, slash-separated) for which `keep` of its path relative to `dir` is true, keyed by that relative path. Files `keep` rejects are neither read nor reported. For a worktree, walks `dir` through `os.OpenRoot(src.Name)` and its `FS()`, reading files including uncommitted and untracked ones; entries that are not regular files (links included) are skipped and links are never followed. For any other source, lists with `Git.ListFiles(src.Commit, dir)` and reads each kept file with `Git.Show`. A missing `dir` gives an empty map and nothing unreadable. A kept file that cannot be read, or a folder that cannot be listed, is in the second result with `Source: src`, its `Path` and the reason, and the rest is still read. The error is only for a `dir` that `filepath.IsLocal` rejects.

- [ ] **Step 1: Write the failing tests:** `TestListFiles` (files at two depths under `dir` are listed, sorted; a file beside `dir` and a committed link under it are not; a missing `dir` → empty, nil; an unknown revision → error). `TestReadFilesWorktree` (a committed file, a modified tracked file and an untracked file are read with their disk contents; a `spec.md` that links to a file outside the worktree and a subfolder that links elsewhere are not read; a kept file with mode 000 — skip as root — is listed unreadable with its `Path` while the others are still returned; a file with mode 000 that `keep` rejects is not listed; a missing `dir` → empty, no unreadable; `dir` `../x` → error). `TestReadFilesBranch` (on a local branch, the files at its commit are read, not the worktree's uncommitted edits; nested paths are keys relative to `dir`; a missing `dir` → empty).
- [ ] **Step 2: Run** `go test ./internal/gitx/ ./internal/scan/` — expected: FAIL (undefined: `ListFiles`, `ReadFiles`, `Path`).
- [ ] **Step 3: Implement** `ListFiles`, the `Path` field and `ReadFiles` as above.
- [ ] **Step 4: Run** `go test ./internal/gitx/ ./internal/scan/` — expected: PASS.
- [ ] **Step 5: Commit** `feat(scan): read every file below a folder from any scanned source`.

### Task 2: One capability-path rule

**Scenarios:** none

**Files:**
- Create: `internal/delta/capability.go`
- Modify: `internal/lint/structure.go`, `internal/archive/load.go`
- Test: `internal/delta/capability_test.go`

**Interfaces:**
- Produces (package `delta`): `func CapabilityPath(rel string) (string, bool)` — `rel` is relative to the change folder and slash-separated; true when it starts with `specs/` and ends with `/spec.md`, and then the part between them (`specs/booking/spec.md` → `booking`, `specs/a/b/spec.md` → `a/b`, `specs/spec.md` → `spec.md`, the value both copies give today). False for anything else (`proposal.md`, `specs/booking/notes.md`, `specs/booking/spec.md.bak`, `other/booking/spec.md`).
- `lint`'s `deltas` and `archive`'s `loadDeltas` call it instead of their own `TrimPrefix`/`TrimSuffix`; their behavior does not change.

- [ ] **Step 1: Write the failing test** `TestCapabilityPath` with the cases above.
- [ ] **Step 2: Run** `go test ./internal/delta/ -run TestCapabilityPath` — expected: FAIL (undefined: `CapabilityPath`).
- [ ] **Step 3: Implement** `CapabilityPath` and switch `lint` and `archive` to it.
- [ ] **Step 4: Run** `go test ./internal/delta/ ./internal/lint/ ./internal/archive/` — expected: PASS, with no change to an existing test.
- [ ] **Step 5: Commit** `refactor: one rule for the capability path of a delta spec`.

### Task 3: Overlaps between changes

**Scenarios:** `conflict-detection#both-modify-one-requirement`, `conflict-detection#rename-against-modify`, `conflict-detection#same-capability-different-requirements`, `conflict-detection#both-add-the-same-name`, `conflict-detection#no-overlaps`

**Files:**
- Create: `internal/conflicts/conflicts.go`
- Test: `internal/conflicts/conflicts_test.go`

**Interfaces:**
- Produces (package `conflicts`; its doc says it finds requirements that more than one active change touches):
  - `const ( Added = "ADDED"; Modified = "MODIFIED"; Removed = "REMOVED"; RenamedFrom = "RENAMED-FROM"; RenamedTo = "RENAMED-TO" )`
  - `type Change struct { Slug string; Sources []scan.Source; Deltas map[string]delta.Delta }` — `Deltas` keyed by capability path.
  - `type Entry struct { Slug string \`json:"slug"\`; Operations []string \`json:"operations"\`; Sources []scan.Source \`json:"sources"\` }`
  - `type Overlap struct { Capability string \`json:"capability"\`; Requirement string \`json:"requirement"\`; Changes []Entry \`json:"changes"\` }`
  - `func Find(changes []Change) []Overlap` — a change touches (capability, name) when its delta for that capability adds, modifies or removes a requirement with that name, or renames from or to it. Every (capability, name) touched by two or more distinct slugs is an overlap. Overlaps are sorted by capability, then requirement; entries by slug; each entry's operations in the constant order above, without repeats; sources as given. A slug that appears twice in `changes` counts once. Never nil.
  - `func Involving(overlaps []Overlap, slug string) []Overlap` — the overlaps with an entry for `slug`, in order. Never nil.

- [ ] **Step 1: Write the failing tests**, each building `[]Change` from delta text through `delta.Parse`: `TestBothModifyOneRequirement` (`add-x` and `add-y` MODIFIED `booking` / "Cancel booking" → one overlap, entries `add-x` [MODIFIED] and `add-y` [MODIFIED]); `TestRenameAgainstModify` (`add-x` renames "Cancel booking" to "Cancel a booking", `add-y` modifies "Cancel booking" → one overlap on "Cancel booking": `add-x` [RENAMED-FROM], `add-y` [MODIFIED]); `TestSameCapabilityDifferentRequirements` (MODIFIED "Cancel booking" against ADDED "Refund points" → none); `TestBothAddTheSameName` (both ADDED "Waitlist cap" → one overlap); `TestRemoveAgainstModify` (`add-x` REMOVED "Cancel booking", `add-y` MODIFIED it → one overlap, `add-x` [REMOVED]); `TestOperationOrder` (one delta that adds, modifies and removes "X", renames "X" to "Z" and renames "W" to "X", against another change modifying "X" → the first change's operations on "X" are exactly [ADDED, MODIFIED, REMOVED, RENAMED-FROM, RENAMED-TO]); `TestNoOverlaps` (three changes on different requirements → empty, not nil); `TestRenameThenModifyInOneChange` (one change renames "A" to "B" and modifies "B", another modifies "B" → one overlap on "B" with the first change's operations [MODIFIED, RENAMED-TO]; no overlap on "A"); `TestNamesMatchAsOpenSpecDoes` ("Cancel booking  " against "Cancel booking" overlaps; "cancel booking" against "Cancel booking" does not); `TestNestedCapabilities` (same name in `a` and `a/b` → none); `TestThreeChangesSorted` (three changes on one requirement, given out of order → entries sorted by slug; two overlaps sorted by capability then name); `TestInvolving` (keeps only the overlaps naming the slug; an unknown slug → empty, not nil).
- [ ] **Step 2: Run** `go test ./internal/conflicts/` — expected: FAIL (package or `Find` undefined).
- [ ] **Step 3: Implement** the package as above.
- [ ] **Step 4: Run** `go test ./internal/conflicts/` — expected: PASS.
- [ ] **Step 5: Commit** `feat(conflicts): find requirements that more than one change touches`.

### Task 4: Load every active change's deltas

**Scenarios:** `conflict-detection#overlap-with-another-worktree`, `conflict-detection#archived-change-left-on-a-branch`, `conflict-detection#sources-listed`

**Files:**
- Modify: `internal/conflicts/conflicts.go`
- Create: `internal/conflicts/load.go`
- Test: `internal/conflicts/load_test.go`

**Interfaces:**
- Consumes: `scan.ReadFiles`, `scan.Unreadable.Path` (Task 1); `delta.CapabilityPath` (Task 2); `Change` (Task 3); `scan.Scan`.
- Produces: `func Load(g gitx.Git, inv scan.Inventory) ([]Change, []scan.Unreadable, error)` — one `Change` per entry of `inv.Changes`, in that order, except slugs that have any entry in `inv.Archived`. Each reads `scan.ReadFiles(g, c.Sources[0], "openspec/changes/<slug>/specs", keep)`, where `keep(rel)` is `delta.CapabilityPath("specs/" + rel)`'s second result, and parses every file it returns with `delta.Parse`; `Sources` is the scan's list as given. A change without a `specs/` folder is returned with an empty `Deltas` map. Unreadable files from `ReadFiles` are returned in the second result, and the change is still returned with the deltas that could be read. The error is for anything else `ReadFiles` returns.

- [ ] **Step 1: Write the failing tests**, on `testutil` repositories scanned with `scan.Scan`: `TestOverlapWithAnotherWorktree` (worktree A holds `add-x` committed on its branch; worktree B, added with `AddWorktree`, holds `add-y` written but not committed; both modify `booking` / "Cancel booking"; scanning from A and running `Find(Load(...))` → one overlap whose `add-y` entry's first source is worktree B); `TestArchivedChangeLeftOnABranch` (`add-x` archived on the default branch tip as `archive/2026-01-01-add-x`, an unmerged branch still holding `openspec/changes/add-x/` with a MODIFIED block, and `add-y` modifying the same requirement → `add-x` is not loaded and there is no overlap); `TestSourcesListed` (`add-y` in worktree B and on `refs/remotes/origin/add-y` from a clone → its `Sources` holds both, each with its commit); `TestSameSlugInManySources` (the same slug in two worktrees, one with an uncommitted edit, and on its remote-tracking branch → loaded once, from the scan's first source, no overlap with itself); `TestChangeWithoutSpecs` (a change with only `.openspec.yaml` → loaded with empty `Deltas`); `TestUnreadableDeltaSpec` (a `spec.md` with mode 000 in a worktree — skip as root → listed unreadable with its `Path`; the change's other deltas still loaded); `TestUnreadableOtherFileIgnored` (a `specs/booking/notes.md` with mode 000 — skip as root → nothing unreadable, the deltas loaded).
- [ ] **Step 2: Run** `go test ./internal/conflicts/` — expected: FAIL (undefined: `Load`).
- [ ] **Step 3: Implement** `Load`.
- [ ] **Step 4: Run** `go test ./internal/conflicts/` — expected: PASS.
- [ ] **Step 5: Commit** `feat(conflicts): load every active change's deltas from its first source`.

### Task 5: `dkt conflicts`

**Scenarios:** `conflict-detection#unrelated-overlap-hidden`, `conflict-detection#slug-not-active`, `conflict-detection#no-overlaps`, `conflict-detection#unreadable-worktree`, `conflict-detection#both-modify-one-requirement`, `conflict-detection#sources-listed`

**Files:**
- Create: `internal/cli/cmd_conflicts.go`
- Modify: `internal/cli/run.go`, `internal/cli/cmd_new.go`
- Test: `internal/cli/cmd_conflicts_test.go`

**Interfaces:**
- Consumes: `conflicts.Load`, `conflicts.Find`, `conflicts.Involving` (Tasks 3 and 4); `scan.Scan`; `resolveDefaultBranch`; `change.ValidateSlug`.
- Produces (package `cli`):
  - `type ConflictsResult struct { Changes []string \`json:"changes"\`; Overlaps []conflicts.Overlap \`json:"overlaps"\` }` — `Changes` is the sorted slugs that were compared; both never nil.
  - `func scanUnreadable(us []scan.Unreadable) []Unreadable` — the mapping `newCommand` does inline today (`worktree:` + path for a worktree, the refname otherwise); with a non-empty `Path` the source reads `<name>:<path>`. `newCommand` calls it with `inv.Unreadable`.
  - `conflictsCommand()`, usage `conflicts [<slug>]`, added to `commands`. More than one slug → `UsageError{Msg: "conflicts takes at most one slug"}`. An invalid slug → `slug-invalid`. A slug not among the loaded changes → `change-not-active` naming the slug, with "archived as <folder>" when it has an archived use. Each overlap → one `requirement-overlap` finding, message `<capability> / "<requirement>": <slug> (<operations joined by ", ">), ...`. Text mode adds, before the findings, for each overlap a line `<capability> / "<requirement>"` and one line per change `  <slug> <operations>: <source names, each with its commit's first 7 characters>`. Unreadable sources are `scanUnreadable(inv.Unreadable)` followed by `scanUnreadable` of those from `Load`.

- [ ] **Step 1: Write the failing tests** (run the built command the way `cmd_new_test.go` does): `TestConflictsBothModify` (two worktrees, overlap → exit 1, `requirement-overlap`, `result.overlaps[0]` has both slugs with `["MODIFIED"]` and their sources with full commits); `TestConflictsNoOverlaps` (two changes on different requirements → exit 0, `ok` true, `result.overlaps` `[]`, `result.changes` both slugs, text ends `dkt conflicts: ok`); `TestConflictsUnrelatedOverlapHidden` (`add-x` and `add-y` overlap, `add-z` touches nothing shared, `dkt conflicts add-z` → exit 0, no overlap); `TestConflictsFocusShowsOwnOverlap` (`dkt conflicts add-x` → exit 1 with the overlap); `TestConflictsSlugNotActive` (`add-q` → exit 1, `change-not-active`; an archived slug → the message names its archive folder); `TestConflictsSlugInvalid` (`Bad_Slug` → exit 1, `slug-invalid`); `TestConflictsTwoSlugs` (exit 2, `usage`); `TestConflictsUnreadableWorktree` (a listed worktree removed from disk and two other changes overlapping → exit 3, the overlap in `result.overlaps`, the worktree in `unreadable`); `TestConflictsSourcesListedInText` (text output names both sources of `add-y` with short commits); `TestConflictsWritesNothing` (the repository snapshot before and after is identical).
- [ ] **Step 2: Run** `go test ./internal/cli/` — expected: FAIL.
- [ ] **Step 3: Implement** `scanUnreadable` (switching `newCommand` to it), the command, and add `conflictsCommand` to `commands`.
- [ ] **Step 4: Run** `go test ./internal/cli/` — expected: PASS, the `dkt new` tests unchanged.
- [ ] **Step 5: Commit** `feat: dkt conflicts lists requirements touched by more than one change`.

### Task 6: Documentation and this repository

**Scenarios:** none

**Files:**
- Modify: `docs/ARCHITECTURE.md`, `AGENTS.md`, `README.md`, `README.zh-TW.md`, `openspec/changes/add-conflicts/plan.md`

- [ ] **Step 1: Update** `docs/ARCHITECTURE.md`: a codemap row for `internal/conflicts` (finds requirements that more than one active change touches); capability index rows for `Git.ListFiles`, `scan.ReadFiles`, `delta.CapabilityPath`, `conflicts.Load`, `conflicts.Find` and `conflicts.Involving`.
- [ ] **Step 2: Update** `AGENTS.md` Status (milestone 1 is complete: `init`, `new`, `lint`, `conflicts`, `sync`, `archive`, `rebase` exist) and the README status line in both languages (the first milestone is complete; the other commands are a draft).
- [ ] **Step 3: Run** on this repository: `go run ./cmd/dkt conflicts` and `go run ./cmd/dkt conflicts add-conflicts --json` — expected: exit 0, no overlap, `result.changes` holds `add-conflicts`. Record the output in the task report.
- [ ] **Step 4: Run** `go vet ./...`, `go test ./...`, `go run ./cmd/dkt lint`, then `go run ./cmd/dkt sync add-conflicts` and `go run ./cmd/dkt sync add-conflicts --check` — expected: all pass, and the last exits 0.
- [ ] **Step 5: Commit** `docs: map the conflicts package and close the first milestone`, including plan.md as ticked by `dkt sync`.
