# Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `dkt status` lists records (open by default, merged with `--merged`, `--since`, `--limit`), `dkt sync` ticks a record's tasks, `dkt lint` checks `Record:` trailers, and the four follow-ups from the review of `add-records` are fixed.

**Architecture:** `gitx.Commit` reads the `Record` trailer; `change.MatchTasks` matches `Task` trailers over commits the caller chose, so records and changes share it. `status.LoadRecords` builds the record list from the scan's `Inventory.Records`: open ones read from their first source, merged ones only when asked, with their merge from one first-parent `git log` (`gitx.Git.Additions`) and their tag from the scan's batched tree reads (`scan.RecordTags`). `archive.SyncRecord` ticks a record with the same tick writer as `archive.Sync`. `dkt lint` finds the merge base once and gives lint the branch's commits.

**Tech Stack:** Go 1.26, git.

**Spec:** `openspec/changes/add-record-tracking/proposal.md`, `openspec/changes/add-record-tracking/design.md`, and the delta specs under `openspec/changes/add-record-tracking/specs/` (`records`, `workspace-scan`, `change-status`, `plan-sync`, `change-lint`).

## Global Constraints

- A slug that starts with a date (`change.StartsWithDate`) names a record; a record slug is valid when `record.SlugOf(slug + ".md")` accepts it, else the finding is `slug-invalid`.
- A record is `merged` when one of its sources has kind `default-branch`, else `open`.
- A record's commits: `Commits(<default tip>, <commit>)` filtered to those whose `Record` values hold the slug. For status `<commit>` is the first source's commit; for sync it is `HEAD`.
- JSON, exactly: `result.records[]` with `slug`, `title`, `kind`, `source` (string or null; null also for an empty value), `sources`, `state` (`open` or `merged`), `merge` (null or `{"commit", "time"}`, time in RFC 3339 as `status.Activity.Time`), `tag` (null or the short tag name, such as `v0.9.1`), `tasks`, `progress`; `result.records_total` (int). `dkt sync`: `result.change` and `result.record`, each a string or null, exactly one set.
- Order: open records by slug, then merged records by `merge.time` newest first, ties by slug, null `merge` last by slug.
- Human lines, exactly (a part in brackets is left out when null):
  - open: `<slug>[  "<title>"][  <kind>]  open  <done>/<total> tasks`
  - merged: `<slug>[  "<title>"][  <kind>]  merged <YYYY-MM-DD of merge.time, or "unknown">  <tag, or "untagged">  <done>/<total> tasks`
  - then each source as for changes: `  <name>@<short commit>`
  - after the records, when `--limit` cut: `<n> more merged records not listed; raise --limit`
- Finding rules: `record-missing`, `record-trailer`, `task-unknown`, `plan-unsynced`, `symlink-refused`, `slug-invalid`; record findings carry `change` set to the record slug, as `dkt lint`'s record findings do.
- Usage errors (exit 2): `--since` or `--limit` without `--merged`; any of the three with slugs; `--since` that does not name a commit (`RefExists(rev + "^{commit}")`); `--limit` that is not a positive integer.
- Until `install-schema-per-machine` merges (it merged at `b5db90f`; Task 5 then edits `internal/cli/cmd_archive.go`), files that it changes are not edited here: `internal/cli/cmd_{new,archive,init,inspect}*.go`, `internal/setup/`, `internal/userdirs/`, `internal/workflow/`, `internal/testutil/openspec.go`, `schema.go`, `schema_test.go`, `AGENTS.md`. They may be called (`resolveDefaultBranch`, `cleanRepo`, `testutil.InstallSchema`). `README.md`, `docs/ARCHITECTURE.md` and `docket/order.md` are edited only in Task 10, after that change merges.
- Only `internal/gitx` starts git (test helpers excepted); nothing below `cmd/dkt` reads the process environment; readers take bytes; nothing writes through a link (`safepath.Check`).
- Everything in the repository is in English.
- TDD, one item per round: take one test from a task's list, see it fail, make it pass, then take the next.
- Commits:
  - every commit for a task carries `Change: add-record-tracking` and `Task: <N>` in one final trailer paragraph with `Co-Authored-By`; write the message to a file and use `git commit -F`;
  - structural and behavior changes go in separate commits;
  - author `Claude Code <wake+claude-code@protype.tw>`, passed with `git -c user.name=... -c user.email=...`.

## Review Focus

- A `Record:` value holding path characters (`../x`, `a/b`): lint must report `record-trailer` and read nothing outside `docket/records`. Test: Task 5 (`TestRecordTrailerPathValue`).
- A record merged on the default branch and edited again on an unmerged branch: it is `merged`, read from the branch, and not listed by default. Test: Task 3 (`TestMergedRecordEditedOnBranchHidden`).
- Several records merged in one commit: same `merge.time`, so the order and the `--limit` cut fall back to slug order. Test: Task 4 (`TestMergedSameCommitBySlug`).
- A record that reached the default branch by fast-forward or rebase, with no merge commit: its `merge` is the commit that added it. Test: Task 4 (`TestAdditionsFastForward`).
- A failed `dkt record` in a folder another run has just written to: cleanup removes only empty folders and never another run's file. Test: Task 8 (`TestCreateFailedWriteKeepsOtherRecords`).

## Reuse

| Need | Existing (path:symbol) | Decision | Reason | Evidence |
|---|---|---|---|---|
| Read commits and their trailers | internal/gitx/log.go:Git.Commits | extend | a seventh field for the `Record` trailer | `rg -n "trailers:key" internal/gitx/` |
| Match `Task` trailers to tasks | internal/change/trailers.go:NamedTasks | extend | the matching moves to `MatchTasks`, which takes commits already chosen; `NamedTasks` keeps its signature | `rg -n "func NamedTasks" internal/` |
| A record's commits | none | new | `record.CommitsFor`; `NamedTasks` chooses by `Change` inline and nothing chooses by `Record` | `rg -n "\.Record\b" internal/` |
| Records in every source | internal/scan/scan.go:Inventory.Records | reuse | slugs and ranked sources | `rg -n "Records " internal/scan/scan.go` |
| Where a record is held | internal/scan/scan.go:Inventory.RecordSources | reuse | named records, `record-missing` | `rg -n "func (inv Inventory) RecordSources" internal/scan/` |
| Read files from a source | internal/scan/scan.go:ReadFiles | reuse | one call per source for the picked records | `rg -n "func ReadFiles" internal/scan/` |
| Parse a record | internal/record/parse.go:Parse | reuse | front matter and `Plan` | `rg -n "func Parse" internal/record/` |
| Record slug rules | internal/record/record.go:SlugOf | reuse | | `rg -n "func SlugOf" internal/record/` |
| Tell a record slug from a change slug | internal/change/new.go:StartsWithDate | reuse | as `lint.Run` does | `rg -n "StartsWithDate" internal/` |
| Task states and progress | internal/status/status.go:Summarize | reuse | phase is ignored for records | `rg -n "func Summarize" internal/status/` |
| Load records for status | internal/status/load.go:Load | new | `status.LoadRecords`; `Load` reads change folders, a base and activity, none of which a record has | `rg -n "func Load" internal/status/` |
| Files a first-parent history added | none | new | `gitx.Git.Additions`; `Commits` lists commits, not the files they add, and `AddedFiles` diffs one base | `rg -n "diff-filter" internal/gitx/` |
| Tags by creation date, peeled | internal/gitx/git.go:listRefs | new | `gitx.Git.Tags`; `listRefs` prints name, object and symref only, without peeling or sorting | `rg -n "for-each-ref" internal/gitx/` |
| Which tag holds a record | internal/scan/trees.go:readTrees | reuse | through the new `scan.RecordTags`, one batch per tree level | `rg -n "func readTrees" internal/scan/` |
| Kind of an unreadable tag | internal/scan/scan.go:Kind | extend | `KindTag` | `rg -n "KindDefaultBranch" internal/scan/scan.go` |
| Tick steps | internal/change/tick.go:Tick | reuse | | `rg -n "func Tick" internal/change/` |
| Write without following a link | internal/archive/sync.go:writeOrRefuse | reuse | in `SyncRecord` | `rg -n "func writeOrRefuse" internal/archive/` |
| Refuse a link before reading | internal/safepath/safepath.go:Check | reuse | | `rg -n "func Check" internal/safepath/` |
| Record sync | internal/archive/sync.go:Sync | new | `archive.SyncRecord`; `Sync` loads a change folder and its base through `loadAtHead` | `rg -n "func loadAtHead" internal/archive/` |
| Default branch, exit 4 | internal/cli/cmd_new.go:resolveDefaultBranch | reuse | called from `cmd_sync.go`, not edited | `rg -n "func resolveDefaultBranch" internal/cli/` |
| Merge base and added files | internal/cli/cmd_lint.go:addedFiles | extend | takes the base found once | `rg -n "func addedFiles" internal/cli/` |
| Lint inputs | internal/lint/lint.go:Input | extend | one `AddedErr`, and `Commits` | `rg -n "AddedRecordsErr" internal/` |
| Usage errors | internal/cli:UsageError | reuse | | `rg -n "type UsageError" internal/cli/` |
| A positive integer flag | external: Go `flag.Func` | reuse | tells an absent `--limit` from `--limit 0` | `go doc flag.FlagSet.Func` |
| Word count | internal/record/parse.go:Words | extend | | `rg -n "func Words" internal/record/` |
| Write a new record | internal/record/record.go:Create | extend | cleanup on a failed write | `rg -n "func Create" internal/record/` |
| Test repositories | internal/testutil/repo.go:NewRepo, Repo.Commit, Repo.AddWorktree | reuse | | `rg -n "^func" internal/testutil/repo.go` |
| CLI test fixtures | internal/cli/cmd_new_test.go:cleanRepo, internal/cli/cmd_status_test.go:runStatus, internal/cli/cmd_sync_test.go:runSync | reuse | `cleanRepo` is not edited | `rg -n -e "func cleanRepo" -e "func runStatus" -e "func runSync" internal/cli/` |
| The two skill copies agree | schema_test.go:TestInstalledSkillMatchesShipped | reuse | the check for Task 9 | `rg -n "func TestInstalledSkillMatchesShipped" schema_test.go` |

---

### Task 1: Read `Record` trailers and match tasks over chosen commits

**Scenarios:** none

**Review:** checks

**Files:**
- Modify: `internal/change/trailers.go`
- Modify: `internal/change/trailers_test.go`
- Modify: `internal/gitx/log.go`
- Modify: `internal/gitx/log_test.go`
- Modify: `internal/record/record.go`
- Modify: `internal/record/record_test.go`

**Interfaces:**
- Produces:
  - `change.MatchTasks(p Plan, commits []gitx.Commit) (named []NamedTask, unknown []UnknownTrailer, anyTrailer bool)`: `NamedTasks`' matching over every given commit.
  - `change.NamedTasks(p Plan, slug string, commits []gitx.Commit)`, unchanged: `MatchTasks` over the commits whose `Change` holds `slug`.
  - `gitx.Commit.Record []string`: the `Record` trailer values, read like `Change`.
  - `record.CommitsFor(slug string, commits []gitx.Commit) []gitx.Commit`: the commits whose `Record` holds `slug`, in order.

- [ ] **Step 1: Pin `MatchTasks` first.** Write `TestMatchTasksTakesEveryCommit` (`change`): a commit with `Task: 1` and no `Change` trailer names Task 1. Run `go test ./internal/change/`: FAIL (`MatchTasks` undefined).
- [ ] **Step 2: Extract the exported `MatchTasks` (structural).** Run `go test ./internal/change/ ./internal/status/ ./internal/archive/` and `go vet ./...`: PASS, with no existing test changed. Commit the test and the extraction as `refactor(change): match Task trailers over commits already chosen`.
- [ ] **Step 3: Write the failing tests**
  - `TestCommitsRecordTrailer` (`gitx`): a commit with message `x\n\nRecord: 2026-11-01-fix-x\nTask: 2` has `Record` `[2026-11-01-fix-x]` and `Task` `[2]`; a commit with `record: a` has `Record` `[a]`; a commit with neither has `Record` nil.
  - `TestCommitsFor` (`record`): of three commits with `Record` `[a]`, `[b]`, `[a b]`, slug `a` gives the first and third.
- [ ] **Step 4: Run them.** `go test ./internal/gitx/ ./internal/record/`: FAIL (`Record` and `CommitsFor` undefined).
- [ ] **Step 5: Implement** the `Record` field, read with `%(trailers:key=Record,valueonly,unfold)` as a seventh NUL-separated field, and `CommitsFor`.
- [ ] **Step 6: Run** `go test ./...` and `go vet ./...`: PASS.
- [ ] **Step 7: Commit** `feat(gitx, record): read Record trailers`.

### Task 2: `dkt sync` ticks a record's tasks

**Scenarios:** records#record-tasks-ticked, records#record-sync-ignores-merged-commits, records#record-sync-check, records#record-sync-unknown-task, records#record-sync-needs-default-branch, records#record-sync-through-link, plan-sync#record-file-only, plan-sync#nothing-else-changes, plan-sync#task-with-commits-ticked, plan-sync#task-without-commits-left-alone, plan-sync#two-tasks-in-one-commit, plan-sync#commits-counted-from-the-base

**Review:** judgment

**Files:**
- Create: `internal/archive/syncrecord.go`
- Create: `internal/archive/syncrecord_test.go`
- Modify: `internal/cli/cmd_sync.go`
- Modify: `internal/cli/cmd_sync_test.go`

**Interfaces:**
- Consumes: `change.MatchTasks`, `record.CommitsFor` (Task 1).
- Produces:
  - `archive.SyncRecord(e Env, slug string, check bool) (SyncReport, error)`: reads `docket/records/<slug>.md` under `e.Root` after `safepath.Check`, takes `record.CommitsFor(slug, e.Git.Commits(e.Default.Commit, "HEAD"))`, ticks as `Sync` does, and writes with `writeOrRefuse`. Findings: `record-missing` (file `docket/records/<slug>.md`), `task-unknown` (as `computeSync` words it, `File` the record), `plan-unsynced` with `check` (message `Task <N> has commits and <n> unticked steps; run dkt sync <slug>`), `symlink-refused`. `SyncReport.Change` holds the slug.
  - `cli.SyncResult` becomes `{Change *string "change"; Record *string "record"; Check bool "check"; Ticked []archive.TaskTicks "ticked"}`.

- [ ] **Step 1: Write the failing tests** (`internal/cli/cmd_sync_test.go`; each builds on `cleanRepo`, then `git checkout -b fix-x`; the record has `## Tasks` with `### Task 1: A` (one step) and `### Task 2: B` (two steps), all unticked):
  - `TestRecordTasksTicked`: a branch commit `work\n\nRecord: 2026-11-01-fix-x\nTask: 2` → exit 0; Task 2's two steps are `[x]`, Task 1's is not; `result.record` `2026-11-01-fix-x`, `result.change` null; `git diff` changes only those two `[ ]`; `git status --porcelain` lists only the record.
  - `TestRecordSyncIgnoresMergedCommits`: the commit with `Record: <slug>` and `Task: 1` is made on `main` before branching → Task 1 stays unticked.
  - `TestRecordSyncCheck`: as `TestRecordTasksTicked` with `--check` → the record byte-for-byte unchanged, a `plan-unsynced` finding naming Task 2, exit 1.
  - `TestRecordSyncUnknownTask`: commits naming `Task: 9` and `Task: 2` → `task-unknown` naming the first commit's short SHA, Task 2 ticked, exit 1.
  - `TestRecordSyncNeedsDefaultBranch`: `git branch -m main trunk` → exit 4, the record unchanged.
  - `TestRecordSyncThroughLink`: `docket/records` is a link to a folder outside the repository holding the record → exit 1, `symlink-refused` naming the link, the outside file unchanged.
  - `TestRecordSyncMissing`: no record file → `record-missing`, exit 1.
  - `TestRecordSyncBadSlug`: `dkt sync 2026-11-01-` → `slug-invalid`, exit 1.
- [ ] **Step 2: Run them.** `go test ./internal/cli/ -run 'TestRecordSync|TestRecordTasksTicked'`: FAIL.
- [ ] **Step 3: Implement** `SyncRecord` and the routing in `cmd_sync.go`: a dated slug takes `resolveDefaultBranch(c)` (its error exits with 4) and no lock; any other slug runs as today with `result.change` set and `result.record` null.
- [ ] **Step 4: Run** `go test ./internal/archive/ ./internal/cli/`: PASS, the existing sync tests included (they claim the `plan-sync` scenarios that keep their wording).
- [ ] **Step 5: Commit** `feat(sync): tick a record's tasks from its commits`.

### Task 3: `dkt status` lists open records and named ones

**Scenarios:** records#open-records-listed, records#merged-records-hidden, records#record-front-matter-unreadable, records#record-tasks-from-trailers, records#named-record-missing, change-status#one-change-in-focus, change-status#slug-not-active, change-status#record-in-focus

**Review:** judgment

**Files:**
- Create: `internal/status/records.go`
- Create: `internal/status/records_test.go`
- Modify: `internal/cli/cmd_status.go`
- Create: `internal/cli/cmd_status_records_test.go`

**Interfaces:**
- Consumes: `change.MatchTasks`, `record.CommitsFor` (Task 1).
- Produces:
  - `status.Open = "open"`, `status.Merged = "merged"`.
  - `type status.Merge struct { Commit string "commit"; Time string "time" }`.
  - `type status.Record struct { Slug string; Title, Kind, Source *string; Sources []scan.Source; State string; Merge *Merge; Tag *string; Tasks []Task; Progress Progress }`, JSON names as in the Global Constraints.
  - `type status.RecordQuery struct { Slugs []string; Merged bool; Since string; Limit int }`: with `Slugs`, those records whatever their state and the other fields unused; `Limit` 0 means none.
  - `status.LoadRecords(g gitx.Git, inv scan.Inventory, q RecordQuery) (records []Record, total int, unread []scan.Unreadable, err error)`. This task handles the open records and `Slugs`; a named merged record gets state `merged` with `Merge` and `Tag` nil until Task 4.
  - `cli.StatusResult` gains `Records []status.Record "records"` and `RecordsTotal int "records_total"`.
  - `statusRecordLine(r status.Record) string` in `cmd_status.go`.

- [ ] **Step 1: Write the failing tests** (`internal/cli/cmd_status_records_test.go`, with `runStatus`):
  - `TestOpenRecordsListed`: worktree B holds an uncommitted `docket/records/2026-11-01-fix-x.md`; branch `fix-y` commits `2026-11-01-fix-y.md`; run in the main worktree → `result.records` slugs `[2026-11-01-fix-x 2026-11-01-fix-y]`, each `state` `open` with its title, kind and source; fix-x's `sources[0].kind` `worktree`, fix-y's `local-branch`; the human output has `2026-11-01-fix-x  "<title>"  fix  open  0/0 tasks` after the change lines.
  - `TestMergedRecordsHidden`: `main` holds `2026-10-31-fix-a.md` only → `records` `[]`, `records_total` 0, exit 0.
  - `TestMergedRecordEditedOnBranchHidden`: as above, plus branch `edit` changes that file → still `[]`.
  - `TestRecordFrontMatterUnreadable`: an open record without front matter → listed with `title`, `kind`, `source` null, exit 0.
  - `TestUnreadableRecordFile`: an uncommitted record in worktree B made unreadable with `chmod 0` → the file in `unreadable`, the record listed with `title`, `kind`, `source` null, exit 3.
  - `TestRecordTasksFromTrailers`: an open record on branch `fix-x` with two tasks; a commit there `work\n\nRecord: <slug>\nTask: 1` → Task 1 `started`, `commits` `[<sha>]`, `unsynced` true; Task 2 `pending`; `progress` `{0 2 0 2}` for one step each.
  - `TestNamedRecordMissing`: `dkt status 2026-11-01-fix-q` → `record-missing` with `change` `2026-11-01-fix-q`, exit 1.
  - `TestRecordInFocus`: `statusChange` for `add-x` and `add-y`, an open record → `dkt status add-y 2026-11-01-fix-x --json` gives `changes` `[add-y]`, `records` `[2026-11-01-fix-x]`, exit 0.
  - `TestNamedRecordBadSlug`: `dkt status 2026-11-01-` → `slug-invalid`, exit 1.
  - `TestTwoRecordsOnOneBranch` (`internal/status/records_test.go`): two open records on one branch, each named by its own commit's `Record` and `Task: 1` → each lists only its own commit.
- [ ] **Step 2: Run them.** `go test ./internal/cli/ ./internal/status/ -run 'Record'`: FAIL.
- [ ] **Step 3: Implement** `LoadRecords` (group the picked records by first source, one `scan.ReadFiles` and at most one `Commits` per source) and the routing in `cmd_status.go`: dated slugs go to `RecordQuery.Slugs`, undated ones to the change path as today; `records_total` is `len(records)` here.
- [ ] **Step 4: Run** `go test ./...`: PASS, existing status tests included.
- [ ] **Step 5: Commit** `feat(status): list open records and named ones`.

### Task 4: Merged records, their merge and tag, and the filters

**Scenarios:** records#merged-records-with-merge-and-tag, records#merged-since, records#limit-reports-total, records#filter-flags-need-merged, records#named-record-listed

**Review:** judgment

**Files:**
- Create: `internal/gitx/additions.go`
- Create: `internal/gitx/additions_test.go`
- Create: `internal/scan/tags.go`
- Create: `internal/scan/tags_test.go`
- Modify: `internal/scan/scan.go`
- Modify: `internal/status/records.go`
- Modify: `internal/status/records_test.go`
- Modify: `internal/cli/cmd_status.go`
- Modify: `internal/cli/cmd_status_records_test.go`

**Interfaces:**
- Consumes: `status.LoadRecords`, `status.RecordQuery`, `statusRecordLine` (Task 3).
- Produces:
  - `type gitx.Addition struct { Path, SHA string; Time time.Time }`.
  - `gitx.Git.Additions(tip, since, dir string) ([]Addition, error)`: newest first, one per commit and file, the files under `dir` that each commit on the first-parent history of `tip` (and not reachable from `since` when not empty) added against its first parent, the root commit against nothing: `log --root --first-parent --diff-merges=first-parent --diff-filter=A --name-only` (`--root` because `log.showRoot` may be off).
  - `gitx.Git.Tags() ([]Ref, error)`: the tags whose object peels to a commit, by creation date then name (`for-each-ref --sort=refname --sort=creatordate`); `Name` the full refname, `SHA` the commit. git fails the whole listing when any tag's object is missing, and so does `Tags`.
  - `scan.KindTag Kind = "tag"`.
  - `scan.RecordTags(g gitx.Git, tags []gitx.Ref) (map[string]string, []Unreadable)`: for each record slug some tag holds, the short name of the first tag in the given order whose tree holds `docket/records/<slug>.md`, read with `readTrees`; a tag whose tree or records folder cannot be read is `Unreadable` with `KindTag`.
  - `LoadRecords` fills merged records: with `Merged` or a named merged record; `Since` passed to `Additions`; `Limit` cuts merged records after sorting; `total` counts open and merged records before the cut; `Tags` and `RecordTags` run only when a merged record is listed. When `Tags` fails, one `Unreadable` with `KindTag`, `Name` `refs/tags` and git's reason, and every `tag` null.

- [ ] **Step 1: Write the failing tests**
  - `TestAdditionsFirstParent` (`gitx`): a branch adds `docket/records/a.md` and merges with `--no-ff` → one `Addition` for it whose `SHA` is the merge commit; with `since` the merge commit → none.
  - `TestAdditionsFastForward`: the branch merges by fast-forward → the `SHA` is the branch commit that added it.
  - `TestAdditionsNewestFirst`: `a.md` added, deleted, added again → two additions, the later first.
  - `TestAdditionsRootCommit`: the root commit adds `a.md`, with `log.showRoot=false` in the repository config → one `Addition` whose `SHA` is the root commit.
  - `TestTagsByCreationDate`: tags made with `GIT_COMMITTER_DATE` set: `v2` (lightweight, 2026-01-02), `v1` (annotated, 2026-01-01), `b` and `a` (both 2026-01-03), and `tree` pointing at a tree → `[v1 v2 a b]`, each `SHA` the peeled commit.
  - `TestTagsMissingObject`: a lightweight tag whose commit object is removed → an error.
  - `TestRecordTags` (`scan`): `v1` before the record exists, `v2` after → `{slug: "v2"}`; a tag on a commit made only for it (with a file no other commit has), whose root tree object is removed → one `Unreadable` with kind `tag` naming that tag.
  - `TestMergedRecordsWithMergeAndTag` (cli): branch adds `2026-10-30-fix-a`, merges `--no-ff` into `main`, then tag `v0.9.1` → `--merged --json` lists it `merged` with `merge.commit` the merge commit, `merge.time` its committer time in RFC 3339, `tag` `v0.9.1`; the human line ends `merged <date>  v0.9.1  0/0 tasks`.
  - `TestMergedSince`: `fix-a` merged, tag `v0.9.1`, then `fix-b` merged → `--merged --since v0.9.1` lists only `fix-b`; `records_total` 1.
  - `TestLimitReportsTotal`: five records merged one per commit with distinct `GIT_COMMITTER_DATE` → `--merged --limit 2` lists the two newest, `records_total` 5, the human output ends `3 more merged records not listed; raise --limit`.
  - `TestRecordsTotalOpenAndMerged`: one open record on a branch and three merged ones → `--merged --limit 1` lists the open record and the newest merged one, `records_total` 4.
  - `TestMergedSameCommitBySlug`: `fix-b` and `fix-a` added in one commit → listed `fix-a`, `fix-b`; `--limit 1` keeps `fix-a`.
  - `TestFilterFlagsNeedMerged`: exit 2 for `--limit 2`; `--since main`; `--merged 2026-10-30-fix-a`; `--merged --limit 0`; `--merged --limit x`; `--merged --since nosuch`.
  - `TestNamedRecordListed`: `dkt status 2026-10-30-fix-a --json` → `merged`, its `merge` and `tag`, exit 0.
  - `TestUnreadableTag`: as in `TestRecordTags`, a tag whose root tree object is removed, `--merged` → that tag in `unreadable`, exit 3; without `--merged`, exit 0.
  - `TestTagListingFails`: a lightweight tag whose commit object is removed, `--merged` → one `unreadable` entry `refs/tags`, the merged records listed with `tag` null, exit 3.
- [ ] **Step 2: Run them.** `go test ./internal/gitx/ ./internal/scan/ ./internal/status/ ./internal/cli/ -run 'Addition|Tag|Merged|Limit|Filter|NamedRecordListed'`: FAIL.
- [ ] **Step 3: Implement** `Additions`, `Tags`, `RecordTags`, the merged part of `LoadRecords`, and the flags on `status`: `--merged` (bool), `--since` (string), `--limit` (through `flag.Func`, so an absent flag differs from `0`).
- [ ] **Step 4: Run** `go test ./...` and `go vet ./...`: PASS.
- [ ] **Step 5: Commit** `feat(status): merged records with their merge, tag and filters`.

### Task 5: `dkt lint` reads the branch once and checks `Record` trailers

**Scenarios:** change-lint#design-written-to-the-superpowers-folder, change-lint#no-default-branch-for-routing, change-lint#one-unreadable-entry, records#trailer-names-no-record, records#trailer-names-a-record

**Review:** judgment

**Files:**
- Modify: `internal/cli/cmd_lint.go`
- Modify: `internal/cli/cmd_lint_test.go`
- Modify: `internal/cli/cmd_archive.go` (its one `addedFiles` call)
- Modify: `internal/lint/lint.go`
- Modify: `internal/lint/routing.go`
- Modify: `internal/lint/routing_test.go`
- Modify: `internal/lint/record.go`
- Modify: `internal/lint/record_test.go`

**Interfaces:**
- Consumes: `gitx.Commit.Record` (Task 1).
- Produces:
  - `lint.Input`: `AddedRecordsErr` is removed; `AddedErr` covers the routed files, the records and the commits; new `Commits []gitx.Commit`, the commits from the merge base to HEAD.
  - The one unreadable entry has source `added files`.
  - Rule `record-trailer`, only when no name is given: for each commit and each distinct `Record` value `v` such that `record.SlugOf(v + ".md")` fails or `docket/records/<v>.md` is not a regular file in `Input.Repo`; message `commit <short> has trailer Record: <v>, which names no record under docket/records`.
  - `branchBase(c *Context) (string, error)` in `cmd_lint.go`: the merge base, `""` when HEAD has no commit; the errors `addedFiles` gives today. `addedFiles(c *Context, base string, dirs ...string) ([]string, error)`.

- [ ] **Step 1: Write the failing tests**
  - `TestRecordTrailerNamesNoRecord` (`lint`): `Commits` with `Record` `[fix-x]`, the repository holding `docket/records/2026-11-01-fix-x.md` → one `record-trailer` naming the short SHA and `fix-x`.
  - `TestRecordTrailerNamesARecord`: `Record` `[2026-11-01-fix-x]` → none.
  - `TestRecordTrailerPathValue`: values `../x`, `a/b` and `2026-11-01-x/../../y` → one finding each, even with `y.md` at the repository root and `docket/x.md` present.
  - `TestRecordTrailerSkippedWithNames`: the same commits and a named change → no `record-trailer`.
  - Change `TestRoutingUnreadable` (`routing_test.go`) and `TestLintRecordsUnknown` (`record_test.go`) to the single source `added files`, with the one `AddedErr`.
  - `TestLintOneUnreadableEntry` (cli): `git branch -m main trunk` → `sources` of `unreadable` exactly `[added files]`, exit 3. Change `TestNoDefaultBranchForRouting` from `[routing records]` to `[added files]`.
  - `TestLintRecordTrailer` (cli): a branch commit carrying `Record: fix-x` beside the record `2026-11-01-fix-x` → `record-trailer`, exit 1.
- [ ] **Step 2: Run them.** `go test ./internal/lint/ ./internal/cli/ -run 'RecordTrailer|Unreadable|DefaultBranch'`: FAIL.
- [ ] **Step 3: Implement** `branchBase`, the reshaped `addedFiles`, `Commits` from `c.Git.Commits(base, "HEAD")` when `base` is not empty, the single unreadable entry, and `record-trailer` in `internal/lint/record.go`.
- [ ] **Step 4: Run** `go test ./...`: PASS.
- [ ] **Step 5: Commit** `feat(lint): one look at the branch, and Record trailers that name no record`.

### Task 6: Count the prolonged sound mark as half a word

**Scenarios:** records#prolonged-sound-mark-counted, records#section-over-the-limit, records#chinese-text-counted

**Review:** checks

**Files:**
- Modify: `internal/record/parse.go`
- Modify: `internal/record/parse_test.go`

**Interfaces:**
- Produces: `record.Words` counts U+30FC `ー` and U+FF70 `ｰ` as half a word that ends a run.

- [ ] **Step 1: Write the failing test** `TestWordsProlongedSoundMark`: `Words("コーヒー")` 2; `Words("ｺｰﾋｰ")` 2; `Words("ー")` 0.5.
- [ ] **Step 2: Run it.** `go test ./internal/record/ -run Words`: FAIL (3 for `コーヒー`).
- [ ] **Step 3: Implement** in `Words`; update its comment.
- [ ] **Step 4: Run** `go test ./internal/record/ ./internal/lint/`: PASS, the existing word-limit tests included.
- [ ] **Step 5: Commit** `fix(record): count the prolonged sound mark as half a word`.

### Task 7: The scan lists a worktree once

**Scenarios:** workspace-scan#worktree-listed-once, workspace-scan#worktree-on-an-unmounted-volume, workspace-scan#record-on-a-branch

**Review:** checks

**Files:**
- Modify: `internal/scan/scan.go`
- Modify: `internal/scan/scan_test.go`

**Interfaces:**
- Produces: in `Scan`, a worktree's ancestry is settled before `noteRead` files its records error, so a worktree unreadable as a whole has one entry.

- [ ] **Step 1: Write the failing tests**
  - `TestWorktreeListedOnce`: the broken ancestry of `TestWorktreeWithFailedAncestryCheckIsUnreadable` plus that worktree's `docket/records` made unreadable as in `TestScanRecordsUnreadable` → exactly one `Unreadable` for the worktree, with `Path` `""`.
  - `TestRecordOnABranch`: branch `fix-x` commits `docket/records/2026-11-01-fix-x.md`, worktree B is on `fix-x` → `Records` holds the slug once with `Sources` `[worktree B, refs/heads/fix-x]`. (It may pass at once: it pins the `Records found` requirement.)
- [ ] **Step 2: Run them.** `go test ./internal/scan/ -run 'ListedOnce|RecordOnABranch'`: `TestWorktreeListedOnce` FAILS with two entries.
- [ ] **Step 3: Implement** the reordering in `Scan`.
- [ ] **Step 4: Run** `go test ./internal/scan/`: PASS, `TestWorktreeOnUnmountedVolume` included.
- [ ] **Step 5: Commit** `fix(scan): list a worktree unreadable as a whole once`.

### Task 8: A failed record write leaves nothing

**Scenarios:** records#failed-write-leaves-nothing

**Review:** judgment

**Files:**
- Modify: `internal/record/record.go`
- Modify: `internal/record/record_test.go`

**Interfaces:**
- Produces:
  - `record.Create` notes which folders on the way to `dir` it made; when the write fails it removes the file, then each folder it made, deepest first, with `os.Remove` (never `RemoveAll`); the error joins the write error and, for each path it could not remove, `<path> was left: <reason>`.
  - Package variables `writeRecord func(f *os.File, b []byte) error` and `removePath func(string) error`, for tests to inject failures, as `archive.writeTemp`.

- [ ] **Step 1: Write the failing tests**
  - `TestCreateFailedWriteLeavesNothing`: a root without `docket`; `writeRecord` fails → `Create` returns that error; neither the record, `docket/records` nor `docket` exists.
  - `TestCreateFailedWriteKeepsExistingFolders`: `docket` exists, `docket/records` does not → `docket` remains, `docket/records` does not.
  - `TestCreateFailedWriteKeepsOtherRecords`: `writeRecord` writes another record into the folder before failing → that record and the folder remain, and the error names `docket/records` as left.
  - `TestCreateFailedCleanupNamesLeftover`: `removePath` fails for the record → the error contains the record's path and `was left`.
- [ ] **Step 2: Run them.** `go test ./internal/record/ -run CreateFailed`: FAIL.
- [ ] **Step 3: Implement** in `Create`.
- [ ] **Step 4: Run** `go test ./internal/record/ ./internal/cli/`: PASS, `dkt record` tests included.
- [ ] **Step 5: Commit** `fix(record): leave no folder after a failed write`.

### Task 9: The skill syncs a record's tasks

**Scenarios:** records#light-review-opens-a-record, records#light-review-syncs-record-tasks

**Review:** checks

**Files:**
- Modify: `.claude/skills/docket/SKILL.md`
- Modify: `.agents/skills/docket/SKILL.md`

**Interfaces:**
- Consumes: `dkt sync <record-slug>` (Task 2).

- [ ] **Step 1: Edit step 4 of `## Light review`** in both copies to: `4. When the record lists tasks, run \`dkt sync <slug>\` and commit the record. Run \`dkt lint\` and fix what it reports. Then open the pull request and merge it with a merge commit.`
- [ ] **Step 2: Check.** `diff .claude/skills/docket/SKILL.md .agents/skills/docket/SKILL.md` prints nothing; `go test . ./internal/setup/` PASS.
- [ ] **Step 3: Commit** `docs(skill): sync a record's tasks before lint`.

### Task 10: The docs, after `install-schema-per-machine` merges

**Scenarios:** none

**Review:** checks

**Files:**
- Modify: `README.md`
- Modify: `docs/ARCHITECTURE.md`
- Modify: `docket/ideas/2026-10-29-records.md`
- Modify: `docket/order.md`

**Interfaces:**
- Consumes: every symbol above.

- [ ] **Step 1: Wait and rebase.** Start only after `install-schema-per-machine` is on `main`; rebase this branch on it and run `go test ./...` and `go vet ./...`: PASS.
- [ ] **Step 2: Edit**
  - `README.md`: the Bookkeeping row says `sync` changes nothing but checkboxes in `plan.md` or a record; the command list shows `dkt status [<slug>|<record>...] [--merged [--since <rev>] [--limit <n>]] [--json]` and `dkt sync <slug>|<record> [--check]`; the light-review paragraph says `dkt status` lists open records and `dkt sync` ticks their tasks.
  - `docs/ARCHITECTURE.md`: `internal/status` also summarizes records; capability rows for `status.LoadRecords`, `change.MatchTasks`, `record.CommitsFor`, `gitx.Git.Additions`, `gitx.Git.Tags`, `scan.RecordTags`, `archive.SyncRecord`.
  - `docket/ideas/2026-10-29-records.md`: the Status section says the second change shipped, and the follow-ups are done.
  - `docket/order.md`: tick item 7.
- [ ] **Step 3: Check.** `dkt lint` passes; every symbol named in `docs/ARCHITECTURE.md` exists (`rg -n "func (LoadRecords|MatchTasks|CommitsFor|RecordTags|SyncRecord)|func \(g Git\) (Additions|Tags)" internal/`).
- [ ] **Step 4: Commit** `docs: records in status and sync`.
