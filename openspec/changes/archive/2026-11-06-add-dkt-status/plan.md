# Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `dkt status --json` reports in `result.next` the first idea in `docket/order.md` still to do, and in `result.reason` why when there is none.

**Architecture:** `internal/idea` holds what an idea and the order are, picks the next one (pure functions), and finds the work that names each idea through `internal/scan`. `internal/cli/cmd_status.go` reads from the default branch tip, turns problems into findings and prints the result.

**Tech Stack:** Go 1.24, standard library only; git through `internal/gitx`.

**Spec:** `openspec/changes/add-dkt-status/proposal.md`, `design.md`, `specs/ideas/spec.md`.

## Global Constraints

- Rule names: `order-format`, `order-item`, `idea-front-matter`, `issue-front-matter`, `idea-needs`.
- Reasons: `no-order`, `order-invalid`, `waiting`, `none-left`; empty when `next` is set.
- `result`: `next` (`{path, title}` or null), `reason`, `waiting_on` (never null).
- Order line: `- docket/ideas/<name>.md` or `- docket/issues/<slug>.md`. Idea path: `^docket/ideas/[^/\s]+\.md$`. Needs: `needs: [<path>, <path>]` on one line, ideas only.
- Exit codes: 0 with no problem, 1 with a finding, 3 with an unreadable source.
- A command writes nothing here, contacts no remote and does not run OpenSpec.
- Tasks 1 and 2 touch only new files, because `add-dkt-issue` changes `internal/cli/command.go`, `internal/scan/records.go`, `AGENTS.md` and `README.md`. Do not edit those, or anything in `shipped/`, before Task 3.
- No new dependency. Commits carry `Change: add-dkt-status/<N>` in the last paragraph, with no other trailer.
- `go vet ./...` and `go test ./...` pass after every task.

## Review Focus

- Quoted values: `dkt new` and `dkt record` write `source: "docket/ideas/a.md"`, with quotes, and that must count as naming the idea. Pinned in Task 2 by a test using that exact form.
- A local default branch ahead of `origin`: the tip is the remote-tracking commit, so work only on the local branch is not on the tip. Pinned in Task 2 (`TestStatusLocalAheadOfTip`).
- No remote at all: the local `main` is the tip, and work on it is on the tip. Pinned in Task 2 (`TestStatusNoRemote`).
- CRLF line endings in `docket/order.md` or an idea must read the same. Pinned in Task 1.
- A change folder in a worktree without `proposal.md` names nothing and is not unreadable. Pinned in Task 2 (`TestStatusFolderWithoutProposal`).

## Revision 1 (2026-11-06): issues in the order

The user asked, through the lead, for `docket/order.md` to hold local issues as well as ideas, with no `dropped` on an issue, `needs` for ideas only, and no addresses in another tool. Tasks 1 to 3 stay as written; Task 4 changes what they built. Task 5 takes in two findings from a review of Tasks 1 to 3 that ran before this revision.

## Reuse

| Need | Existing (path:symbol) | Decision | Reason | Evidence |
|---|---|---|---|---|
| Front matter | internal/change/frontmatter.go:ReadFrontMatter | reuse | same form: one line per key; ideas add no `specs` | `rg -n "func ReadFrontMatter" internal/` |
| One-line text | internal/change/metadata.go:ValidLine | reuse | for `title` | `rg -n "func ValidLine" internal/` |
| Lines of a file | internal/markdown/markdown.go:Lines | reuse | as the other parsers read files, CRLF included | `rg -n "func Lines" internal/markdown/` |
| The tip | internal/gitx/branch.go:DefaultBranch | reuse | the one definition of the tip | `rg -n "DefaultBranch\(\)" internal/` |
| Reading from a commit | internal/gitx/blobs.go:ReadBlobs | reuse | one git call per commit | `rg -n "ReadBlobs\(" internal/` |
| Commit of a ref use | internal/scan/scan.go:Use.Commit, internal/gitx/refs.go:Refs | extend | a change's use has its commit; a record's does not, and `internal/scan/records.go` is changing in `add-dkt-issue`, so `Refs` fills it in (see design.md, Risks) | `rg -n "Commit:" internal/scan/` |
| Finding changes | internal/scan/scan.go:Changes | reuse | every worktree and branch, archived included | `rg -n "func Changes" internal/scan/` |
| Finding records | internal/scan/records.go:Records | reuse | same sources | `rg -n "func Records" internal/scan/` |
| Reading in a worktree | internal/safepath/safepath.go:Check, internal/safefile/safefile.go:ReadNoFollow | reuse | refuses links as `conflicts.worktreeSpecs` does | `rg -n "safepath.Check\|ReadNoFollow" internal/` |
| Result and findings | internal/cli/envelope.go:Outcome, Finding, Unreadable | reuse | every command's shape | `rg -n "type Outcome" internal/cli/` |
| Where an issue lives | internal/issue/issue.go:Dir | reuse | the one definition of the folder | `rg -n "Dir = " internal/issue/` |
| An idea and the order | none | new | nothing reads them in this version | `rg -n "docket/ideas\|order.md" internal/` |

---

### Task 1: What an idea and the order are, and which is next

**Scenarios:** ideas#idea-read, ideas#order-read, ideas#first-idea-next, ideas#dropped-idea-skipped, ideas#taken-idea-skipped, ideas#need-merged, ideas#need-in-progress, ideas#need-ordered-later, ideas#need-dropped, ideas#nothing-left

**Review:** checks

**Files:**
- Create: `internal/idea/idea.go`
- Create: `internal/idea/idea_test.go`

**Interfaces:**
- Produces, in package `idea`:
  - `const OrderFile = "docket/order.md"`, `const LineForm = "- docket/ideas/<name>.md"`, `const NeedsForm = "needs: [<path>, <path>]"`
  - `type Idea struct { Title string; Dropped bool; Needs []string }`
  - `func Parse(b []byte) (Idea, error)`: the error says what is wrong, and names `NeedsForm` for any front matter error or bad `needs`.
  - `type Line struct { N int; Path string }`: `N` is the 1-based line number in the file.
  - `func ParseOrder(b []byte) (lines []Line, bad []int)`: `bad` holds the 1-based numbers of the lines not of the form.
  - `type Work struct { Any, OnTip bool }`
  - `func Next(lines []Line, ideas map[string]Idea, work map[string]Work) (next string, waitingOn []string)`: `next` is empty when no idea can start; `waitingOn` holds the unmet needs of the first idea that waits, in the order written, only then. Every path in `lines` and in their `Needs` is in `ideas`.

- [x] **Step 1: Write the failing tests** in `internal/idea/idea_test.go`.
  - `TestParse`, a table: title only; title, `dropped: true` and `needs: [docket/ideas/a.md, docket/ideas/b.md]`; `dropped: false`; `needs: []`; CRLF line endings; quoted title. Each with the expected `Idea`.
  - `TestParseRefuses`, a table, each an error: no front matter; no closing `---`; no title; a title of only spaces; `dropped: maybe`; `needs: docket/ideas/a.md` with no brackets; `needs: [a.md]`; `needs: [docket/ideas/x/a.md]`; `needs:` with the paths on the lines under it. The errors for the `needs` cases contain `NeedsForm`.
  - `TestParseOrder`: `# Order`, a blank line and two lines give two `Line`s with their numbers and no `bad`; CRLF gives the same; `1. docket/ideas/a.md`, `- docket/ideas/a.md: note`, `- docket/x.md` and `* docket/ideas/a.md` are each in `bad` with their line numbers; an empty file gives neither.
  - `TestNext`, a table over `lines`, `ideas` and `work`: first idea next; the first dropped; the first with `Work{Any: true}`; a need with `Work{Any: true, OnTip: true}` lets the idea start; a need with `Work{Any: true}` makes it wait, with `waitingOn` holding the need and `next` empty when nothing else can start; `b` needing `a`, ordered `b`, `a`, gives `a`; a dropped need lets the idea start; everything dropped or taken gives `next` empty and `waitingOn` empty; the first waits and the second can start gives the second, with `waitingOn` empty.

- [x] **Step 2: Run** `go test ./internal/idea/`; expect FAIL: the package does not exist.

- [x] **Step 3: Write `internal/idea/idea.go`.** `Parse` uses `change.ReadFrontMatter`; a `false` from it is an error, as is a `specs` key. `ParseOrder` uses `markdown.Lines`; a line starting with `#` and a blank line are skipped.

- [x] **Step 4: Run** `go test ./internal/idea/`; expect PASS. Mutate and see a test fail for each, then restore: treat a dropped need as unmet; treat `Any` as `OnTip` for a need; skip bad order lines silently.

- [x] **Step 5: Run** `go vet ./...` and `go test ./...`; expect PASS.

- [x] **Step 6: Commit** the two files with the trailer `Change: add-dkt-status/1`.

### Task 2: dkt status reads from the tip and finds the work

**Scenarios:** ideas#order-edited-in-a-worktree-only, ideas#idea-only-on-a-branch, ideas#no-openspec, ideas#change-in-another-worktree, ideas#record-on-another-branch, ideas#archived-change, ideas#source-with-other-text, ideas#work-with-no-front-matter, ideas#no-order-file, ideas#empty-order-file, ideas#no-default-branch, ideas#line-not-of-the-form, ideas#order-names-a-missing-idea, ideas#later-line-broken, ideas#idea-without-a-title, ideas#needs-as-a-block, ideas#need-names-a-missing-idea, ideas#same-file-names-another-idea-on-the-tip, ideas#broken-proposal-elsewhere, ideas#linked-proposal, ideas#broken-record-elsewhere, ideas#problem-and-a-broken-proposal, ideas#text-output, ideas#argument-given

**Review:** judgment

**Files:**
- Create: `internal/idea/work.go`
- Create: `internal/cli/cmd_status.go`
- Create: `internal/cli/cmd_status_test.go`

**Interfaces:**
- Consumes: `idea.Parse`, `idea.ParseOrder`, `idea.Next`, `idea.Work`, `idea.OrderFile`, `idea.LineForm` from Task 1.
- Produces:
  - in package `idea`: `func FindWork(g gitx.Git, tip string) (map[string]Work, []scan.Unreadable, error)`, keyed by the `source` value of each change's `proposal.md` and each record, for files that hold one. `Any` is true when any copy has that value; `OnTip` only when a copy read from commit `tip` does. Take a use's commit from `scan.Use.Commit`, and when that is empty and its `Source` is a ref, from `Refs`. A file's front matter that cannot be read goes in the `[]scan.Unreadable`, as `<source>:<path>`; so does a worktree file that is a symbolic link. A missing file names nothing.
  - in package `cli`: `func statusCommand() Command`, with `Name: "status"`, `Args: 0` and `Usage: "status [--json]"`. Task 3 adds it to `commands`; until then the tests run it with `RunWith(opts, []func() Command{statusCommand})`.

- [x] **Step 1: Write the failing tests** in `internal/cli/cmd_status_test.go`, through a helper that runs the command with `--json` and decodes the envelope, using `testutil.NewRepo`. Each scenario in this task's line has one test, with the values in the spec. In addition:
  - `TestStatusQuotedSource`: a change's front matter holds `source: "docket/ideas/a.md"`, quotes included; the idea is taken.
  - `TestStatusLocalAheadOfTip`: `refs/remotes/origin/main` and a local `main` one commit ahead that adds a record naming the need; an idea that needs it waits.
  - `TestStatusNoRemote`: no remote; `main` holds the order, its ideas and a record naming the need; the idea can start.
  - `TestStatusLinkedRecord`: another worktree's record is a symbolic link; `unreadable` lists it and dkt exits with 3.
  - `TestStatusBrokenProposalOnBranch`: an unmerged branch's `proposal.md` has front matter that cannot be read; `unreadable` lists the branch and file and dkt exits with 3.
  - `TestStatusFolderWithoutProposal`: another worktree holds `openspec/changes/add-x/` with no `proposal.md`; `unreadable` is empty and dkt exits with 0.
  - `TestStatusWritesNothing`: `git status --short` is the same before and after.
  - `TestStatusFromAnotherWorktree`: running from a worktree on a stale branch gives the same `next` as from the main worktree.

- [x] **Step 2: Run** `go test -run TestStatus ./internal/cli/`; expect FAIL: `statusCommand` does not exist.

- [x] **Step 3: Write `internal/idea/work.go`** using `scan.Changes` and `scan.Records` and the Reuse rows. Reads from a ref are grouped by commit into one `ReadBlobs` call each. Reads from a worktree check `safepath.Check`, then `safefile.ReadNoFollow`. `scan.Unreadable` entries from both scans are carried over.

- [x] **Step 4: Write `internal/cli/cmd_status.go`.** In order: take the tip, with `no-order` when there is none; read `idea.OrderFile`, with `no-order` when it is missing; `ParseOrder`, each `bad` line an `order-format` finding on `idea.OrderFile` with its line and `idea.LineForm` in the message; `no-order` when there are no lines and nothing bad; read each line's idea and each need's idea from the tip, in one `ReadBlobs` per round: `order-idea` on the line, `idea-front-matter` on the idea's file, `idea-needs` on the file of the idea that needs it; with any finding, `order-invalid` and no `FindWork`; otherwise `FindWork` and `Next`, then `waiting` or `none-left` when `next` is empty. `waiting_on` is never null. Human lines: `Next: <path> (<title>)`, or one line for each reason.

- [x] **Step 5: Run** `go test -run TestStatus ./internal/cli/ ./internal/idea/`; expect PASS. Mutate and see a test fail for each, then restore: read the order from the worktree instead of the tip; match `source` by prefix; set `OnTip` for every ref; ignore a worktree symbolic link's refusal.

- [x] **Step 6: Run** `go vet ./...` and `go test ./...`; expect PASS.

- [x] **Step 7: Commit** the three files with the trailer `Change: add-dkt-status/2`.

### Task 3: Make the command available

Start this task only after `add-dkt-issue` has merged, on a branch rebased on it.

**Scenarios:** cli-output#status-outside-a-repository, cli-output#init-outside-a-repository, cli-output#new-outside-a-repository, cli-output#record-outside-a-repository, cli-output#issue-outside-a-repository, cli-output#sync-outside-a-repository, cli-output#lint-outside-a-repository, cli-output#conflicts-outside-a-repository

**Review:** checks

**Files:**
- Create: `openspec/changes/add-dkt-status/specs/cli-output/spec.md`
- Modify: `internal/cli/command.go`
- Modify: `internal/cli/cmd_status_test.go`
- Modify: `internal/cli/output_test.go` or the test that holds the other `Repository required` scenarios
- Modify: `openspec/changes/add-dkt-status/plan.md`
- Modify: `AGENTS.md`
- Modify: `README.md`

- [x] **Step 1: Write the delta** `specs/cli-output/spec.md`: under `## MODIFIED Requirements`, the whole of `### Requirement: Repository required` as it stands in `openspec/specs/cli-output/spec.md`, with `dkt status` added to the list and a new scenario `Status outside a repository`: **WHEN** a user runs `dkt status --json` outside any git repository, **THEN** the findings hold `no-repository` and dkt exits with 4. Change this task's `**Scenarios:**` line to `cli-output#status-outside-a-repository`, which can only be named once the delta exists. Run `openspec validate add-dkt-status --strict` and `dkt lint add-dkt-status`.

- [x] **Step 2: Write the failing test** for that scenario beside the others for that requirement, and switch the helper in `cmd_status_test.go` to `Run`. Run them; expect FAIL: unknown command.

- [x] **Step 3: Add** `statusCommand` to `commands` in `internal/cli/command.go`.

- [x] **Step 4: Run** `go test ./internal/cli/`; expect PASS. `dkt --help`, from a build of this branch, lists `status`.

- [x] **Step 5: Edit `AGENTS.md`**: take `dkt status` out of the line of commands that do not exist yet, and remove the stopgap "The user decides the next piece of work." Keep what the line says about any command still missing; if none is left, remove the line and its stopgaps. **Edit `README.md`**: add `dkt status` to the commands table: "Names the next idea to work on from `docket/order.md`".

- [x] **Step 6: Run** `go vet ./...` and `go test ./...`; expect PASS.

- [x] **Step 7: Commit** the files with the trailer `Change: add-dkt-status/3`.

### Task 4: Issues in the order

**Scenarios:** ideas#issue-in-the-order, ideas#issue-next, ideas#issue-taken, ideas#issue-with-dropped-or-needs, ideas#need-names-an-issue, ideas#external-issue-address, ideas#order-names-a-missing-issue, ideas#issue-without-a-title

**Review:** checks

**Compensates:** Task 1

**Files:**
- Modify: `internal/idea/idea.go`
- Modify: `internal/idea/idea_test.go`
- Modify: `internal/cli/cmd_status.go`
- Modify: `internal/cli/cmd_status_test.go`

**Interfaces:**
- Produces, in package `idea`: `func ParseItem(path string, b []byte) (Idea, error)`: under `issue.Dir` it reads only the title, into an `Idea` with no `Dropped` and no `Needs`, whatever else the front matter holds; otherwise it is `Parse`. `LineForm` becomes `- docket/ideas/<name>.md or - docket/issues/<slug>.md`. `needPath` stays ideas only.

- [x] **Step 1: Write the failing tests.** In `idea_test.go`: `TestParseOrder` takes `- docket/issues/2026-11-01-fix-x.md` as a line, and `- https://example.com/issues/7` as bad; `TestParseRefuses` refuses `needs: [docket/issues/x.md]`; `TestParseItem` reads an issue with `title`, `source`, `dropped: true` and `needs: [docket/ideas/a.md]` as only its title, refuses an issue with no front matter or no title, and treats a path under `docket/ideas/` as `Parse` does. In `cmd_status_test.go`, one test per scenario in this task's line, with the spec's values, and rename the rule in the existing tests to `order-item`.

- [x] **Step 2: Run** `go test ./internal/idea/` and `go test -run TestStatus ./internal/cli/`; expect FAIL.

- [x] **Step 3: Change the code.** `orderLine` takes either folder. `runStatus` uses `ParseItem`, reports `issue-front-matter` for a path under `issue.Dir` and `idea-front-matter` otherwise, and `order-item` for a missing item, with a message that says item rather than idea. The text for `none-left` says "every item" and the one for `waiting` "the items left".

- [x] **Step 4: Run** the same tests; expect PASS. Mutate and see a test fail for each, then restore: let `orderLine` take any path under `docket/`; read `Dropped` for an issue; let a need name an issue.

- [x] **Step 5: Run** `go vet ./...` and `go test ./internal/idea/ ./internal/cli/`; expect PASS.

- [x] **Step 6: Commit** the four files with the trailer `Change: add-dkt-status/4`.

### Task 5: Findings from the earlier review

**Scenarios:** none

**Review:** checks

**Files:**
- Modify: `internal/idea/work.go`
- Modify: `internal/idea/idea.go`
- Modify: `internal/idea/idea_test.go`
- Modify: `README.md`

- [x] **Step 1: In `internal/idea/work.go`**, read each path once per commit: several refs at one commit list the same files. Every source still gets its result, so that a file that cannot be read is listed for each ref holding it and `OnTip` still depends only on the commit. No new test: the existing `TestStatus` tests cover the behavior, so see them pass and say which ones reach this path.

- [x] **Step 2: In `internal/idea/idea.go`**, remove the refusal of a `specs` key, which the spec does not name, and its test row. Run `go test ./internal/idea/`; expect PASS.

- [x] **Step 3: In `README.md`**, make the `dkt status` row say: Names the next idea or issue to work on from `docket/order.md`.

- [x] **Step 4: Run** `go vet ./...` and `go test ./...`; expect PASS.

- [x] **Step 5: Commit** the four files with the trailer `Change: add-dkt-status/5`.
