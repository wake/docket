# Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `dkt snapshot --json` reports a whole project in one read-only run: the default branch, every change and record with its locations, tasks, stages, documents, deltas, merge and release, every idea and issue with its state and position, the order, the main specs and the domain tree.

**Architecture:** Three layers. `internal/specdoc` reads delta and main specs from their headings (pure). `internal/gitx` gains three batched reads: the commit history with parents and trailers, the tags by creation date, and the dates ideas were added. `internal/snapshot` builds the result: `graph.go` works out merges, releases and reachable commits from the history in memory; `source.go` reads the files of each location once per commit; `works.go`, `backlog.go` and `tip.go` turn them into the result's parts; `snapshot.go` runs the reads and puts the parts together. `internal/cli/cmd_snapshot.go` is the command.

**Tech Stack:** Go 1.24, standard library only.

**Spec:** `openspec/changes/project-snapshot/proposal.md`, `design.md` (its Output section is the field-level contract), `specs/project-snapshot/spec.md`, `specs/cli-output/spec.md`.

## Global Constraints

- Field names, types and value sets exactly as `design.md` Output gives them. Arrays never null. Times RFC 3339 with the commit's own offset (`%cI`); dates `YYYY-MM-DD` (the first 10 characters of a `%cI` time, or of a slug).
- Stage names: a change `proposal`, `design`, `plan`, `dev`, `merged`, `archived`; a record `record`, `dev`, `merged`. Task states `done`, `pending`. Idea states `queued`, `pool`, `dropped`, `taken`. Issue states `open`, `in-progress`, `done`. Location kinds `worktree`, `branch`, `remote-branch`. Document kinds `proposal`, `design`, `plan`, `spec-delta`, `record`, `idea`, `issue`, `other`.
- Read-only: nothing in the repository is written (the project index registration of every command is not this change's). No network. OpenSpec is never run.
- Findings reuse the rule names of `dkt status` and `dkt domains`: `order-format`, `order-item`, `idea-front-matter`, `idea-needs`, `issue-front-matter`, `domain-format`, `domain-duplicate`, `domain-unknown`. Exit 3 when anything is unreadable, else 1 with a finding, else 0.
- git is started a number of times bounded by the refs and worktrees, never once per work: the three scans of `workspace-scan` as they are; `for-each-ref` once for tags; `log` twice (history, idea dates); for the files of works, `ls-tree` and `cat-file --batch` once per distinct commit read.
- No new dependency. Commits carry `Change: project-snapshot/<N>` in the last paragraph, with no other trailer. `gofmt -l .` prints nothing, `go vet ./...` and `go test ./...` pass after every task.
- Two pull requests, one after the other (`docket/workflow.md`, Splitting pull requests: product code and tests run far past 1000 lines): the first holds Tasks 1 to 4, which nothing calls yet; the second Tasks 5 to 9 and the archive. The second is opened only once the first is merged.

## Review Focus

- A repository with a hundred branches and worktrees: each distinct commit is read once, so locations that share a commit cost nothing more. Pinned in Task 4 (`TestReadSharesCommits`) and measured in Task 9.
- A worktree on an unmounted disk, a linked `openspec/changes`, a ref whose tree cannot be read: listed in `unreadable`, everything else reported. Pinned in Task 4 (`TestReadUnreadable`) and Task 8 (`TestSnapshotUnmountedWorktree`).
- A shallow clone, where history stops early: merges and dates are what the history holds, never an error. Pinned in Task 2 (`TestHistoryShallow`).
- A delta written with CRLF line endings, a BOM, headings in code blocks or comments: read as OpenSpec reads them. Pinned in Task 1 (`TestParseDeltaEdges`).
- A slug that is a prefix of another (`add-x` and `add-x-2`): trailers of one never count for the other. Pinned in Task 3 (`TestNamesPrefixSlug`).

## Revision 1 (2026-11-08): rulings from the task reviews of Tasks 3 and 4

The task reviewers, in fresh contexts, found two costs the plan had not pinned, and the controller ruled on them; no requirement changed.

- Task 3: `Merge` and `Release` walked the history once per work, which breaks the one-second target on long histories. They now mark each commit once per tip and per tag list, cached on the `Graph` (9d2d3ee, bc0d8a5, `TestMergeLongHistory`, `TestMergeOlderAttribution`). A malformed task list names the work with no task (`TestNamesMalformed`), so a change's `dev` uses `len(tasks) > 0`.
- Task 4: `ls-tree` ran once per wanted path; `gitx.Git.TreeFiles` became variadic, `TreeFiles(commit string, dirs ...string)`, so one run per commit lists every path (0d5b523). This touches `internal/gitx/refs.go`, outside the task's Files block; its two callers pass one dir and are unchanged. A symbolic link inside one wanted path now costs that path only, `<source>:<path>`, while a missing worktree, or a link at or above `openspec/changes` or `docket/records`, still costs the whole source (`TestReadLinkPerPath`).

## Reuse

| Need | Existing (path:symbol) | Decision | Reason | Evidence |
|---|---|---|---|---|
| Changes, records and issues in every location | internal/scan/scan.go:Changes; internal/scan/records.go:Records, Issues | reuse | `workspace-scan` is the scope the spec names | `rg -n "func (Changes\|Records\|Issues)" internal/scan/` |
| Worktrees with HEAD and branch | internal/gitx/worktree.go:Git.Worktrees | reuse | gives path, head, branch, detached | `rg -n "func \(g Git\) Worktrees" internal/gitx/` |
| Refs and their commits | internal/gitx/refs.go:Git.Refs | reuse | heads and remotes without symbolic refs | `rg -n "func \(g Git\) Refs" internal/gitx/` |
| Default branch and tip | internal/gitx/branch.go:Git.DefaultBranch | reuse | `default-branch` rules; the ref is derived as that spec says | `rg -n "func \(g Git\) DefaultBranch" internal/gitx/` |
| Files of a ref | internal/gitx/refs.go:Git.TreeFiles; internal/gitx/blobs.go:Git.ReadBlobs | reuse | `ls-tree -r` and `cat-file --batch` | `rg -n "func \(g Git\) (TreeFiles\|ReadBlobs)" internal/gitx/` |
| Commit history with parents and trailers | internal/gitx/log.go:Git.trailers | new | it reads one key, values only, without parents, time or subject; `History` reads all of them in one run | `rg -n "func \(g Git\) trailers" internal/gitx/` |
| Tags by creation date | none | new | nothing lists tags | `rg -n "refs/tags" internal/` |
| Dates files were added | internal/gitx/blobs.go:Git.AddedSince | new | it lists paths new since a base, without dates | `rg -n "func \(g Git\) AddedSince" internal/gitx/` |
| Trailer value naming a task | internal/change/metadata.go:TaskRefs | reuse | the parser `dkt sync` uses, with the prefix rule | `rg -n "func TaskRefs" internal/change/` |
| Plan tasks and steps | internal/change/plan.go:ParsePlan | reuse | as `plan-sync` counts them | `rg -n "func ParsePlan" internal/change/` |
| Record front matter and tasks | internal/record/parse.go:Parse | reuse | `Tasks` in the plan's format | `rg -n "func Parse" internal/record/` |
| Proposal front matter | internal/change/frontmatter.go:ReadFrontMatter | reuse | the one reader of `title`, `kind`, `source` | `rg -n "func ReadFrontMatter" internal/change/` |
| Change schema | internal/change/metadata.go:SchemaOf | reuse | reads `.openspec.yaml` as `change-metadata` defines | `rg -n "func SchemaOf" internal/change/` |
| Main spec title | internal/change/frontmatter.go:SpecTitle | reuse | `change-metadata` rule | `rg -n "func SpecTitle" internal/change/` |
| Ideas, order, issues front matter | internal/idea/idea.go:Parse, ParseItem, ParseOrder | reuse | the readers `dkt status` uses | `rg -n "func (Parse\|ParseItem\|ParseOrder)" internal/idea/` |
| Order and idea findings | internal/cli/cmd_status.go:runStatus | new | the checks are inline in the command; Task 7 moves them into `idea.Check`, which both call | `rg -n "order-item\|idea-needs" internal/` |
| Domain tree | internal/domain/parse.go:Parse | reuse | `domain.Load` reads an `fs.FS` of the worktree, the snapshot reads the tip's blob | `rg -n "func Parse\|func Load" internal/domain/` |
| Headings outside code and comments | internal/markdown/markdown.go:Lines, FenceMask, BlankComments, Headings | reuse | as `ParsePlan` finds headings: level, closing hashes and whitespace handled | `rg -n 'func Headings\|markdown\.Headings' internal/markdown internal/change` |
| Delta and main spec headings | none | new | `dkt lint` and `dkt conflicts` read deltas through OpenSpec, which the spec forbids here | `rg -n "ADDED Requirements" internal/ --glob '!*_test.go'` |
| Reading worktree files without following links | internal/safefile/safefile.go:ReadNoFollow; internal/safepath:Check | reuse | as the scans refuse links | `rg -n "func (ReadNoFollow\|Check)" internal/safefile/ internal/safepath/` |
| Result envelope and findings | internal/cli/envelope.go:Outcome, Finding, Unreadable | reuse | every command's shape | `rg -n "type Outcome" internal/cli/` |
| Registering the command | internal/cli/command.go:commands | extend | one more entry | `rg -n "var commands" internal/cli/` |
| Test repositories | internal/testutil/repo.go:NewRepo, Env | reuse | real git in temp dirs | `rg -n "func NewRepo" internal/testutil/` |

---

### Task 1: Read delta and main specs

**Scenarios:** project-snapshot#every-operation-read, project-snapshot#heading-in-a-code-block

**Review:** checks

**Files:**
- Create: `internal/specdoc/specdoc.go`
- Create: `internal/specdoc/specdoc_test.go`

**Interfaces:**
- Produces, in package `specdoc`:
  - `type Requirement struct { Name string \`json:"name"\`; Scenarios []string \`json:"scenarios"\` }`
  - `type Removed struct { Name string \`json:"name"\`; Reason string \`json:"reason"\` }`
  - `type Renamed struct { From string \`json:"from"\`; To string \`json:"to"\` }`
  - `type Delta struct { Added, Modified []Requirement; Removed []Removed; Renamed []Renamed }`: JSON names `added`, `modified`, `removed`, `renamed`; no slice nil.
  - `func ParseDelta(b []byte) Delta`
  - `func Requirements(b []byte) []string`: the names of `### Requirement:` headings, never nil.

- [x] **Step 1: Write the failing tests**

- `TestParseDelta`: a delta with `## ADDED Requirements` holding `### Requirement: Join when full` with `#### Scenario: Full slot` and `#### Scenario: Slot frees up`; `## MODIFIED Requirements` with `### Requirement: Cancel` and one scenario; `## REMOVED Requirements` with `### Requirement: Walk-in mark` and a line `**Reason**: Walk-ins moved to paper.`; `## RENAMED Requirements` with `- FROM: \`### Requirement: Party limit\`` and `- TO: \`### Requirement: Group size limit\``. The result equals those four groups exactly, the reason being `Walk-ins moved to paper.`.
- `TestParseDeltaEdges`: CRLF line endings and a BOM give the same result; `### Requirement: Example` inside a fenced block and inside `<!-- -->` is not reported; `### Requirement: Join when full ###` (closing hashes) gives `Join when full`; a `#### Scenario:` before any requirement is ignored; a requirement under no operation heading is ignored; names are trimmed.
- `TestRequirements`: a main spec with two requirements, one more in a code block: the two names in file order; an empty file gives an empty, non-nil slice.

- [x] **Step 2: Run them and see them fail**

Run: `go test ./internal/specdoc/`
Expected: FAIL, the package does not build.

- [x] **Step 3: Implement `internal/specdoc/specdoc.go`**

Lines through `markdown.Lines`, masked with `markdown.FenceMask` and `markdown.BlankComments`, headings found with `markdown.Headings`, all as `change.ParsePlan` does; a heading's text is matched against `ADDED Requirements` and the others, `Requirement:` and `Scenario:`. A removed requirement's reason is the text after `**Reason**:` on the first such line under it. A rename pairs each `FROM:` with the next `TO:`, taking the name after `### Requirement:` inside the backticks.

- [x] **Step 4: Run them and see them pass**

Run: `gofmt -l . && go vet ./... && go test ./internal/specdoc/`
Expected: PASS.

- [x] **Step 5: Commit**

Commit "Read delta and main specs without OpenSpec" with the trailer `Change: project-snapshot/1`.

---

### Task 2: Batched history, tags and added dates

**Scenarios:** none

**Review:** checks

**Files:**
- Create: `internal/gitx/history.go`
- Create: `internal/gitx/history_test.go`

**Interfaces:**
- Produces, in package `gitx`:
  - `type Trailer struct{ Key, Value string }`
  - `type Commit struct { SHA string; Parents []string; Time, Subject string; Trailers []Trailer }`: `Time` as `%cI`; `Trailers` holds the `Change` and `Record` trailers of the last paragraph, as git reads trailers (`%(trailers:key=Change,key=Record,unfold)`).
  - `func (g Git) History(revs ...string) ([]Commit, error)`: every commit reachable from `revs`, each once, in one `git log -z`; no revs gives nil.
  - `type Tag struct{ Name, Commit, Time string }`: `Commit` peeled to the commit; `Time` the tag's creation time (`%(creatordate:iso-strict)`).
  - `func (g Git) Tags() ([]Tag, error)`: `refs/tags`, oldest first by creation date, in one `for-each-ref`; a tag that does not point at a commit is left out.
  - `func (g Git) AddedDates(rev, dir string) (map[string]string, error)`: for each path under `dir` that a commit reachable from `rev` added, the date (`YYYY-MM-DD`, in that commit's own offset) of the oldest such commit; renames count as adds (`--no-renames`); one `git log`.

- [x] **Step 1: Write the failing tests**

With `testutil.NewRepo` and commits made with fixed `GIT_COMMITTER_DATE`:
- `TestHistory`: a merge commit of a branch carrying `Change: add-x/1` and a commit carrying `Record: 2026-11-06-fix-x` in a paragraph that is not the last: `History("main")` holds each commit once, the merge with two parents, the subject, `%cI` times with their offsets, the `Change` trailer, and not the misplaced `Record`.
- `TestHistoryShallow`: in a `git clone --depth 1` of that repository, `History("HEAD")` returns the one commit with no error.
- `TestTags`: a lightweight tag on an older commit and an annotated tag on a newer one, created in the other order: oldest-created first; the annotated tag's `Commit` is the commit, not the tag object; a tag on a blob is left out.
- `TestAddedDates`: `docket/ideas/a.md` added on 2026-11-04 and edited on 2026-11-06, `docket/ideas/c.md` renamed to `docket/ideas/b.md` on 2026-11-07: `a.md` gives `2026-11-04`, `b.md` gives `2026-11-07`; a path outside `dir` is absent.

- [x] **Step 2: Run them and see them fail**

Run: `go test ./internal/gitx/ -run 'TestHistory|TestTags|TestAddedDates'`
Expected: FAIL, the functions are not defined.

- [x] **Step 3: Implement `internal/gitx/history.go`**

`History` separates revs from paths with `--`; fields are NUL-separated as `trailers` does.

- [x] **Step 4: Run them and see them pass**

Run: `gofmt -l . && go vet ./... && go test ./internal/gitx/`
Expected: PASS.

- [x] **Step 5: Commit**

Commit "Read the history, the tags and the dates files were added in one git run each" with the trailer `Change: project-snapshot/2`.

---

### Task 3: Merges, releases and commits that name a work

**Scenarios:** project-snapshot#merge-commit-found, project-snapshot#rebased-work, project-snapshot#first-release, project-snapshot#newer-commit-on-a-branch

**Review:** judgment

**Files:**
- Create: `internal/snapshot/graph.go`
- Create: `internal/snapshot/graph_test.go`

**Interfaces:**
- Consumes: `gitx.Commit`, `gitx.Tag` (Task 2), `change.TaskRefs`.
- Produces, in package `snapshot`:
  - `type Graph struct{ ... }`, `func NewGraph(commits []gitx.Commit) *Graph`
  - `func (g *Graph) Reachable(from ...string) map[string]bool`: commits reachable from `from`, which the graph may not hold (then empty for that one).
  - `type Work struct{ Key, Slug string }`: `Key` is `Change` or `Record`.
  - `func Names(c gitx.Commit, w Work) (named bool, tasks []int)`: whether a trailer of `c` names `w`, with or without tasks, and the task numbers, by `change.TaskRefs`.
  - `type Ref struct{ Commit, Time, Subject string }`
  - `func (g *Graph) Merge(tip string, w Work) *Ref`: the newest commit on the first-parent line of `tip` that brought in a commit naming `w` (a merge commit, or the commit itself on the line); nil when none.
  - `func (g *Graph) Release(tags []gitx.Tag, commit string) *gitx.Tag`: the first tag, in the given order, whose commit reaches `commit`; nil when none.
  - `func (g *Graph) Last(from []string, w Work) *Ref`: among commits reachable from `from` that name `w`, the newest by `Time`, ties broken by SHA; nil when none.
  - `func (g *Graph) TaskCommits(from string, w Work) map[int][]string`: for each task number, the commits reachable from `from` that name it, oldest first.

- [x] **Step 1: Write the failing tests**

On graphs built from literal `gitx.Commit` values:
- `TestMerge`: a first-parent line `m3 → m2 → m1`, `m2` merging a side branch whose two commits carry `Change: add-x/1` and `Change: add-x`: `Merge("m3", {Change, add-x})` is `m2` with its time and subject.
- `TestMergeRebased`: a line where `c2` itself carries `Record: 2026-11-06-fix-x`: `Merge` gives `c2`.
- `TestMergeNewest`: two merges both bring in commits naming the work: the newer one.
- `TestMergeNone`: no commit names it: nil.
- `TestNamesPrefixSlug`: `Change: add-x-2/1` does not name `add-x`; `Change: add-x/1,3` names it with tasks 1 and 3; `Change: add-x` names it with no task.
- `TestRelease`: tags `v0.1.0` (on a commit before the merge), `v0.2.0`, `v0.3.0` (both after): `v0.2.0`.
- `TestLast`: one naming commit reachable from the tip, a newer one only from a branch: `Last([tip, branch])` is the newer.
- `TestTaskCommits`: commits reachable only from another ref are not counted; oldest first.

- [x] **Step 2: Run them and see them fail**

Run: `go test ./internal/snapshot/`
Expected: FAIL, the package does not build.

- [x] **Step 3: Implement `internal/snapshot/graph.go`**

`Merge` walks the first-parent line once; for a merge commit, the commits it brought in are those reachable from its other parents and not from its first parent. `Release` keeps each tag's reachable set only while it is used.

- [x] **Step 4: Run them and see them pass**

Run: `gofmt -l . && go vet ./... && go test ./internal/snapshot/`
Expected: PASS.

- [x] **Step 5: Commit**

Commit "Find the merge, the release and the commits that name a work" with the trailer `Change: project-snapshot/3`.

---

### Task 4: Read each location's files once

**Scenarios:** project-snapshot#unmounted-worktree

**Review:** judgment

**Files:**
- Create: `internal/snapshot/source.go`
- Create: `internal/snapshot/source_test.go`

**Interfaces:**
- Consumes: `scan.Use`, `scan.Unreadable`, `gitx.Git.TreeFiles`, `gitx.Git.ReadBlobs`, `safefile.ReadNoFollow`.
- Produces, in package `snapshot`:
  - `type Files map[string][]byte`: paths relative to the repository root.
  - `type Wanted struct { Source, Commit string; Paths []string }`: a folder or a file to read in a source; `Commit` empty for a worktree, whose `Source` is its absolute path.
  - `func Read(g gitx.Git, want []Wanted) (map[string]Files, []scan.Unreadable, error)`: keyed by `Source`. For a ref, one `ls-tree` and one `ReadBlobs` per distinct commit, whatever the number of sources sharing it, every file under each path. For a worktree, the files on disk, refusing a symbolic link anywhere below the root, which is reported as unreadable naming the link. A source that cannot be read is listed and has no entry; the error is for git failing as a whole.

- [x] **Step 1: Write the failing tests**

- `TestReadRefAndWorktree`: a change folder committed on a branch and edited, uncommitted, in a linked worktree: the branch's entry holds the committed text, the worktree's the edited one, both with every file under the folder.
- `TestReadSharesCommits`: three refs on one commit: the result has all three sources, and git is run for that commit once (count the runs through a fake `git` on the PATH that logs and calls the real one).
- `TestReadUnreadable`: a worktree whose directory was removed, a worktree whose `openspec/changes` is a link, and a `Wanted` whose `Commit` is a blob (`git hash-object -w`): all three in the unreadable list with reasons, the rest read, and no error.
- `TestReadFile`: a `Wanted` naming a single file (a record) reads that file only.

- [x] **Step 2: Run them and see them fail**

Run: `go test ./internal/snapshot/ -run TestRead`
Expected: FAIL.

- [x] **Step 3: Implement `internal/snapshot/source.go`**

- [x] **Step 4: Run them and see them pass**

Run: `gofmt -l . && go vet ./... && go test ./internal/snapshot/`
Expected: PASS.

- [x] **Step 5: Commit, and open the first pull request**

Commit "Read the files of every location once per commit" with the trailer `Change: project-snapshot/4`. Run `dkt sync project-snapshot` and `dkt lint project-snapshot`, commit the plan, and open the first pull request (Tasks 1 to 4). The rest waits for its merge.

---

### Task 5: Works, locations and the copy read

**Scenarios:** project-snapshot#one-change-in-three-locations, project-snapshot#archived-change, project-snapshot#detached-worktree, project-snapshot#default-ref-marked, project-snapshot#worktree-read-first, project-snapshot#archived-copy-wins, project-snapshot#record-metadata, project-snapshot#unreadable-front-matter, project-snapshot#other-schema, project-snapshot#snapshot-of-an-empty-project

**Review:** judgment

**Files:**
- Create: `internal/snapshot/result.go` (the JSON types of `design.md` Output)
- Create: `internal/snapshot/works.go`
- Create: `internal/snapshot/works_test.go`

**Interfaces:**
- Consumes: `scan.Use`, `Files` (Task 4), `*Graph` (Task 3).
- Produces, in package `snapshot`:
  - The result types: `Result`, `Branch`, `WorkEntry`, `Location`, `Task`, `Progress`, `Document`, `DeltaEntry`, `IdeaEntry`, `IssueEntry`, `OrderItem`, `Spec`, `Domains`, `Node`, `WorkRef`, `Handler`, with the JSON names of `design.md` Output.
  - `type Place struct { Kind, Branch, Commit, Time string; Default bool }`: what a location adds to a use; `Branch` and `Commit` empty for null.
  - `type WorksInput struct { Changes, Records []scan.Use; Places map[string]Place; Files map[string]Files; Graph *Graph; Tip string; Tags []gitx.Tag; Dates map[string]string }`: `Places` and `Files` keyed by source; `Dates` holds the dates of ideas and issues by path, for `proposed`.
  - `func Works(in WorksInput) ([]WorkEntry, []scan.Unreadable)`: every field of a work except `position` and the source document's entry, which Task 7 fills, sorted as `design.md` says; a front matter that cannot be read is returned as unreadable, `<source>:<path>`, with the reason.

- [ ] **Step 1: Write the failing tests**

With `WorksInput` built from literal uses, places and files:
- `TestWorksOneEntry`: `add-x` in a worktree, a branch and a remote branch: one entry, three locations in that order with their kinds, branches, commits and times.
- `TestWorksCopyRead`: the worktree's title differs from the branch's: `read_from` the worktree and its title; then with an archived copy on the default ref: `read_from` that location and `archived` its folder.
- `TestWorksDetachedAndDefault`: a detached worktree has `branch` null; only the default ref's location has `default` true.
- `TestWorksRecord`: a record `2026-11-06-fix-x` with title, kind and source: `type` `record`, `schema` `docket`, `created` `2026-11-06`, `deltas` empty.
- `TestWorksUnreadableFrontMatter`: an unclosed front matter: `title`, `kind`, `source` null, and the unreadable list names `<source>:openspec/changes/add-x/proposal.md`.
- `TestResultJSON`: marshal a `Result` built from zero values and one from a full work (including one of another schema): every array field encodes as `[]`, never `null`; `tasks` of the other-schema work and every nullable scalar or object field encode as `null`; the key names equal those of `design.md` Output (compare the decoded key sets against a literal list per object).
- `TestWorksOtherSchema`: `.openspec.yaml` with `schema: spec-driven`: `schema` `spec-driven`, `tasks` null.
- `TestWorksSorted`: by `created`, then `type`, then `slug`, a null `created` last.

- [ ] **Step 2: Run them and see them fail**

Run: `go test ./internal/snapshot/ -run TestWorks`
Expected: FAIL.

- [ ] **Step 3: Implement the types and `Works`**

- [ ] **Step 4: Run them and see them pass**

Run: `gofmt -l . && go vet ./... && go test ./internal/snapshot/`
Expected: PASS.

- [ ] **Step 5: Commit**

Commit "Report each work once, with its locations and the copy read" with the trailer `Change: project-snapshot/5`.

---

### Task 6: Tasks, stages, documents and deltas of a work

**Scenarios:** project-snapshot#task-commits-from-a-worktree, project-snapshot#task-with-a-commit-and-no-ticks, project-snapshot#task-ticked-in-part, project-snapshot#change-without-design, project-snapshot#bare-trailer-is-not-dev, project-snapshot#done-but-not-merged, project-snapshot#stages-of-an-archived-change, project-snapshot#record-with-a-bare-trailer, project-snapshot#merged-record-without-trailers, project-snapshot#merged-without-trailers

**Review:** judgment

**Files:**
- Modify: `internal/snapshot/works.go`
- Modify: `internal/snapshot/works_test.go`

**Interfaces:**
- Consumes: `specdoc.ParseDelta` (Task 1), `Graph.TaskCommits`, `Graph.Merge`, `Graph.Release`, `Graph.Last` (Task 3).
- Produces: the fields `tasks`, `progress`, `stages`, `phase`, `merged`, `merge`, `release`, `last_commit`, `documents` (the work's own files) and `deltas` of `WorkEntry`, filled by `Works`. `merged` is true when a location with `Default` holds the work.

- [ ] **Step 1: Write the failing tests**

- `TestTasksFromReadFrom`: a commit naming Task 1, reachable only from the worktree `read_from`: Task 1 lists it, `state` `done`; a task with one of three steps ticked and no commit: `pending`; `progress` `{1, 2}`.
- `TestStagesChange`: a proposal and a plan with tasks, no delta, no design: `["proposal", "plan"]`, `phase` `plan`; add a commit `Change: add-x`: still no `dev`; add `Change: add-x/1`: `dev`; on the default ref and archived: ends `merged`, `archived`.
- `TestStagesDoneNotMerged`: every task has a commit, not on the default ref: `phase` `dev`, `progress` full.
- `TestStagesRecord`: a record without tasks on a branch with `Record: <slug>`: `["record", "dev"]`; a record on the default ref with no trailer anywhere: `["record", "merged"]`.
- `TestMergeOfWork`: a merged change whose commits carry trailers: `merge` and `release` set; a merged change without trailers: `merged` true, `merge` null.
- `TestDocuments`: a folder with `proposal.md`, `design.md`, `plan.md`, `specs/booking/spec.md` and `notes.md`: kinds `proposal`, `design`, `plan`, `spec-delta`, `other`, each with `source` `read_from`.
- `TestDeltas`: two capabilities, sorted, each with its path and the four groups from `specdoc`.

- [ ] **Step 2: Run them and see them fail**

Run: `go test ./internal/snapshot/ -run 'TestTasks|TestStages|TestMergeOfWork|TestDocuments|TestDeltas'`
Expected: FAIL.

- [ ] **Step 3: Implement them in `Works`**

- [ ] **Step 4: Run them and see them pass**

Run: `gofmt -l . && go vet ./... && go test ./internal/snapshot/`
Expected: PASS.

- [ ] **Step 5: Commit**

Commit "Report the tasks, stages, documents and deltas of a work" with the trailer `Change: project-snapshot/6`.

---

### Task 7: Ideas, the order and issues

**Scenarios:** project-snapshot#positions-skip-other-lines, project-snapshot#idea-in-the-pool, project-snapshot#taken-idea, project-snapshot#queued-idea-waiting, project-snapshot#idea-added-and-edited-later, project-snapshot#idea-moved, project-snapshot#issue-with-a-handler, project-snapshot#open-issue, project-snapshot#one-handler-merged-of-two, project-snapshot#every-handler-merged, project-snapshot#proposed-date-from-an-idea, project-snapshot#change-from-an-idea, project-snapshot#idea-without-a-title, project-snapshot#need-taken-but-not-met, project-snapshot#issue-read-from-its-first-location

**Review:** judgment

**Files:**
- Create: `internal/idea/check.go` (the order and idea checks moved out of `runStatus`)
- Modify: `internal/cli/cmd_status.go` (calls `idea.Check`)
- Create: `internal/snapshot/backlog.go`
- Create: `internal/snapshot/backlog_test.go`

**Interfaces:**
- Produces, in package `idea`: `type Problem struct{ Rule, File string; Line int; Message string }` and `func Check(order []byte, read func(paths []string) (map[string][]byte, error)) (lines []Line, items map[string]Idea, problems []Problem, err error)`: the checks `runStatus` makes today (`order-format`, `order-item`, `idea-front-matter`, `issue-front-matter`, `idea-needs`), with the same messages. A structural move: `dkt status` output must not change.
- Produces, in package `snapshot`: `type BacklogInput struct { Order []byte; Ideas map[string][]byte; Issues []scan.Use; IssueFiles map[string]Files; Dates map[string]string; Works []WorkEntry; Places map[string]Place }` and `func Backlog(in BacklogInput) (ideas []IdeaEntry, issues []IssueEntry, order []OrderItem, problems []idea.Problem)`, which also sets each work's `position` and adds its source idea or issue to its `documents`.

- [ ] **Step 1: Move the checks (structural commit)**

Move the checks into `idea.Check` and call it from `runStatus`; `go test ./internal/cli/ -run TestStatus` and `go test ./internal/idea/` pass unchanged. Commit "Move the order checks of dkt status into the idea package" with the trailer `Change: project-snapshot/7`.

- [ ] **Step 2: Write the failing tests**

- `TestOrderPositions`: an order with a heading, a blank line, `- docket/ideas/a.md`, a comment and `- docket/issues/2026-11-01-fix-x.md`: positions 1 and 2.
- `TestIdeaStates`: one idea dropped, one named by a change on a branch (`taken`, `taken_by` that change), one in the order needing an idea nothing names (`queued`, `waiting_on` that path), one in neither (`pool`, `position` null).
- `TestNeedTakenNotMet`: `b.md` needs `a.md`, and only an unmerged branch holds a change naming `a.md`: `a.md` is `taken`, `b.md` keeps `waiting_on` `["docket/ideas/a.md"]`; once the default ref's own copy names it, `waiting_on` is empty. A need is met only as `ideas` says (dropped, or named by the tip's own copy), never by `state == taken`.
- `TestIdeaDates`: `date` from `Dates`, null when absent.
- `TestIssueStates`: no handler `open`; a merged record and an unmerged change `in-progress`; every handler merged `done`; `date` from the slug; `locations` as for a work.
- `TestIssueAsSource`: a record whose `source` is `docket/issues/2026-11-01-fix-x.md`: its `proposed` is `2026-11-01`, its `documents` start with `{kind: issue, path, source: <the issue's first location>}`.
- `TestIssueFirstLocation`: a worktree's copy of an issue with an edited title and the default ref's with the old: the issue's `title` is the worktree's.
- `TestProposed`: a work whose `source` is an idea dated 2026-11-04 gets `proposed` `2026-11-04`; a work with free-text `source` gets its `created`.
- `TestSourceDocument`: a change whose `source` is an idea on the tip gets a document `{kind: idea, path, source: <default ref>}` first in `documents`; a source naming an idea that does not exist adds nothing.
- `TestBacklogProblems`: an idea in the order without `title` gives `idea-front-matter`; the other ideas are still listed.

- [ ] **Step 3: Run them and see them fail**

Run: `go test ./internal/snapshot/ -run 'TestOrder|TestIdea|TestIssue|TestProposed|TestBacklog'`
Expected: FAIL.

- [ ] **Step 4: Implement `Backlog`**

- [ ] **Step 5: Run them and see them pass**

Run: `gofmt -l . && go vet ./... && go test ./internal/snapshot/ ./internal/idea/ && go test ./internal/cli/ -run TestStatus`
Expected: PASS.

- [ ] **Step 6: Commit**

Commit "Report ideas, the order and issues with their states" with the trailer `Change: project-snapshot/7`.

---

### Task 8: The tip, and putting it together

**Scenarios:** project-snapshot#snapshot-of-an-empty-project, project-snapshot#snapshot-offline, project-snapshot#no-openspec-needed, project-snapshot#tip-from-the-remote-tracking-ref, project-snapshot#no-default-branch, project-snapshot#nested-capability, project-snapshot#tree-with-a-new-capability, project-snapshot#tree-that-cannot-be-used, project-snapshot#stale-capability-in-the-tree, project-snapshot#unmounted-worktree, project-snapshot#ref-whose-tree-cannot-be-read

**Review:** judgment

**Files:**
- Create: `internal/snapshot/tip.go`
- Create: `internal/snapshot/snapshot.go`
- Create: `internal/snapshot/snapshot_test.go`

**Interfaces:**
- Produces, in package `snapshot`:
  - `func Specs(files Files) []Spec`, `func DomainTree(tree []byte, found bool, capabilities []string) (Domains, []domain.Problem)`: from the tip's files.
  - `func Collect(g gitx.Git) (Result, []idea.Problem, []scan.Unreadable, error)`: runs `scan.Changes`, `scan.Records`, `scan.Issues`, `Worktrees`, `Refs`, `DefaultBranch`, `History` (over every location's commit and the tip), `Tags`, `AddedDates(tip, "docket/ideas")`, one `Read`, then `Works`, `Backlog`, `Specs` and `DomainTree`. Domain problems are returned as `idea.Problem` with the domain rule names.

- [ ] **Step 1: Write the failing tests**

On real repositories from `testutil.NewRepo`:
- `TestSnapshotEmpty`: one commit with a README: every array empty, `domains.found` false.
- `TestSnapshotTip`: `refs/remotes/origin/main` and `refs/heads/main` on different commits: `default_branch.ref` is the remote one, with its commit, time and subject.
- `TestSnapshotNoDefaultBranch`: only a branch `trunk` and no `origin/HEAD`: `default_branch` null, `ideas`, `order`, `specs` and `domains.nodes` empty, works still reported.
- `TestSnapshotSpecs`: `openspec/specs/payments/refund/spec.md` with two requirements: listed with its title and both names.
- `TestSnapshotDomains`: a tree listing `booking` under `Users` and an active change adding `refund`: the node holds `["booking"]`, `unclassified` holds `refund`.
- `TestSnapshotDomainProblems`: a tree with a paragraph line: problem `domain-format`, `nodes` empty, `unclassified` holds every capability, `works` still filled; a tree listing `gone`, held by nothing: problem `domain-unknown`, and the node still lists `gone`.
- `TestSnapshotRefNotACommit`: `git update-ref refs/remotes/origin/bad <blob>` and a branch holding `add-x`: `refs/remotes/origin/bad` is unreadable, `add-x` is reported, and `Collect` returns no error.
- `TestSnapshotNoOpenSpec`: `DOCKET_OPENSPEC` pointing at nothing and an active change with a delta: its `deltas` are reported.
- `TestSnapshotUnmountedWorktree`: a worktree whose directory was removed, another holding `add-x`: the first in unreadable, `add-x` reported.
- `TestSnapshotOffline`: the `origin` remote's URL is a path that does not exist, and `refs/remotes/origin/main` holds a change: `Collect` reports it, with no error.
- `TestSnapshotWritesNothing`: the repository's `git status --porcelain` and its files' modification times are unchanged after `Collect`.

- [ ] **Step 2: Run them and see them fail**

Run: `go test ./internal/snapshot/ -run TestSnapshot`
Expected: FAIL.

- [ ] **Step 3: Implement `tip.go` and `snapshot.go`**

- [ ] **Step 4: Run them and see them pass**

Run: `gofmt -l . && go vet ./... && go test ./internal/snapshot/`
Expected: PASS.

- [ ] **Step 5: Commit**

Commit "Put the snapshot together from one round of reads" with the trailer `Change: project-snapshot/8`.

---

### Task 9: `dkt snapshot`

**Scenarios:** project-snapshot#text-output, cli-output#snapshot-outside-a-repository, cli-output#init-outside-a-repository, cli-output#new-outside-a-repository, cli-output#record-outside-a-repository, cli-output#issue-outside-a-repository, cli-output#sync-outside-a-repository, cli-output#lint-outside-a-repository, cli-output#conflicts-outside-a-repository, cli-output#status-outside-a-repository, cli-output#domains-outside-a-repository

**Review:** judgment

**Files:**
- Create: `internal/cli/cmd_snapshot.go`
- Create: `internal/cli/cmd_snapshot_test.go`
- Modify: `internal/cli/command.go` (the command list)
- Modify: `README.md` (the command table)

**Interfaces:**
- Consumes: `snapshot.Collect` (Task 8).
- Produces: `func snapshotCommand() Command`: `Name: "snapshot"`, `Args: 0`, `Usage: "snapshot [--json]"`.

- [ ] **Step 1: Write the failing tests**

- `TestSnapshotCommand`: in a repository with one change: `snapshot --json` exits 0, `command` is `snapshot`, `result.works[0].slug` is the change.
- `TestSnapshotCommandFindings`: an idea in the order without `title`: exit 1, the finding `idea-front-matter`, `result` still present.
- `TestSnapshotCommandUnreadable`: a removed worktree: exit 3, `unreadable` lists it.
- `TestSnapshotOutsideRepository`: exit 4, `no-repository`.
- `TestSnapshotText`: one change `add-x` titled `Add X` whose `phase` is `plan`: the output is `change add-x  plan  Add X` then `0 ideas, 0 issues, 0 specs`.
- The other `cli-output` scenarios of Repository required are pinned by existing tests; `go test ./internal/cli/` shows they still pass.

- [ ] **Step 2: Run them and see them fail**

Run: `go test ./internal/cli/ -run TestSnapshot`
Expected: FAIL.

- [ ] **Step 3: Implement the command and the README row**

The README row, after `dkt domains`: `| \`dkt snapshot\` | Reports the whole project in one read: works, ideas, issues, the order, specs and the domain tree |`.

- [ ] **Step 4: Measure**

Run `time dkt snapshot --json > /dev/null` three times in this repository with the build of this branch, and count git runs with `GIT_TRACE=1 dkt snapshot --json 2>&1 >/dev/null | grep -c 'trace: exec\|built-in: git'`. Record both in the pull request; the target is about one second.

- [ ] **Step 5: Run the whole suite and commit**

Run: `gofmt -l . && go vet ./... && go test ./...`
Expected: PASS.

Commit "Add dkt snapshot" with the trailer `Change: project-snapshot/9`.
