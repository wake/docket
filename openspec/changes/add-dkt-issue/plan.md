# Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `dkt issue <name>` and `dkt issue template`, scan and lint issue files, sharing the record code instead of copying it.

**Architecture:** First a structural task makes the record code take the folder, the sections and the command name as parameters, with no behavior change. Then a small `internal/issue` package, `scan.Issues`, `dkt issue`, `dkt issue template` and the issue lint rules use that code with the issue values.

**Tech Stack:** Go 1.24 standard library; `internal/record`, `internal/scan`, `internal/lint`, `internal/cli`, `internal/change`, `internal/testutil`.

**Spec:** `openspec/changes/add-dkt-issue/proposal.md`, `design.md`, `specs/issues/spec.md`, `specs/records/spec.md`, `specs/change-lint/spec.md`, `specs/workspace-scan/spec.md`, `specs/cli-output/spec.md`.

## Global Constraints

- Everything committed is in English, with no host names, personal paths or names.
- `go vet ./...` and `go test ./...` pass at the end of every task. No new dependencies.
- Exact names: folder `docket/issues`; sections `Current`, `Expected`, `Unchanged`, `Evidence`; rules `issue-front-matter`, `issue-section`, `issue-too-long`, and the shared `title-too-long`, `placeholder`, `slug-invalid`, `symlink-refused`, `record-missing`; result field `issues`; shipped files `shipped/issue/template.md` and `shipped/issue/external.md`.
- Record behavior, output and messages do not change, except the `record-missing` message (Task 5).
- Shipped prompt text is copied byte for byte from the approved drafts; do not reword it.
- Each commit carries `Change: add-dkt-issue/<N>` in its last paragraph.

## Review Focus

- `dkt issue template` must never create `docket/issues/<date>-template.md`, with or without flags: Task 4 tests both.
- A record and an issue with the same slug: `dkt lint <slug>` lints both and reports no `record-missing`: Task 5 tests it.
- An issue with a `## Tasks` section is reported, not accepted the way a record accepts it: Task 5 tests it.
- Text left from the issue template is reported as `placeholder` against the issue template, not the record template: Task 5 tests it.
- A scan for records must not count issue files, and a scan for issues must not count record files: Task 2 tests both directions.

## Reuse

| Need | Existing (path:symbol) | Decision | Reason | Evidence |
|---|---|---|---|---|
| Name rules and dated slug | internal/record/record.go:CheckName, Slug | extend | `CheckName` takes the command name for its message | `rg -n "func CheckName\|func Slug" internal/record/` |
| Sections and body size | internal/record/parse.go:Parse, Parsed.BodySize | extend | Add `Parsed.Size(sections)`; `Parse` already reads any level-two sections | `rg -n "func Parse\|BodySize" internal/record/` |
| Scan of a folder across sources | internal/scan/records.go:Records | extend | Becomes `Dated(g, dir)`; `Records` and `Issues` call it | `rg -n "func Records\|record.Dir" internal/scan/` |
| Dated write with suffix, link refusal, cleanup | internal/cli/cmd_record.go:runRecord, cleanupRecord, openRecordFile, writeRecordFile | extend | The write loop moves to a helper both commands call; the test seams stay | `rg -n "func runRecord\|func cleanupRecord\|openRecordFile\|writeRecordFile" internal/cli/` |
| Result object of a create command | internal/cli/cmd_new.go:newResult, usedPlaces, usedLines | reuse | `newResult.Kind` gets `omitempty`; records and changes always set it | `rg -n "type newResult\|func usedPlaces\|func usedLines" internal/cli/` |
| Lint of dated files | internal/lint/record.go:readRecords, recordScope, lintRecord, checkRecordFront, checkRecordSections | extend | Take a descriptor of the folder, sections, kinds and rule prefix | `rg -n "^func " internal/lint/record.go` |
| Placeholders from a template | internal/lint/document.go:addPlaceholders, checkPlaceholderFile | reuse | Same check with the issue template | `rg -n '^func (addPlaceholders\|checkPlaceholderFile)' internal/lint/document.go` |
| Files the branch adds | internal/gitx/blobs.go:Git.AddedSince | reuse | Called with the issues folder | `rg -n '^func \(g Git\) AddedSince' internal/gitx/blobs.go` |
| Command registration and usage | internal/cli/command.go:commands, recordCommand | reuse | `issueCommand` beside `recordCommand` | `rg -n "var commands\|func recordCommand" internal/cli/` |
| Issue folder, sections, path | none | new | `internal/issue`: issues have no kind and no tasks, so they do not belong in `record` | `rg -n "docket/issues" -g '*.go' internal` |

---

## Revision 1 (2026-11-06): README lists dkt issue

The lead, in this session: the README rewrite (PR #98) lists only the commands on main, so this change adds `dkt issue` to it. Task 6 also edits `README.md`.

### Task 1: Make the record code take its folder, sections and command as parameters

**Scenarios:** none

**Review:** judgment

**Files:**
- Modify: `internal/record/record.go`
- Modify: `internal/record/parse.go`
- Modify: `internal/scan/records.go`
- Modify: `internal/cli/cmd_record.go`
- Modify: `internal/cli/cmd_new.go`
- Modify: `internal/lint/record.go`
- Modify: `internal/lint/lint.go`
- Modify: the existing tests that call a changed signature, only to pass the record values

**Interfaces:**
- Produces:
  - `record.CheckName(name, command string) error`; its date message reads `name must not start with a date (YYYY-MM-DD): <command> adds the date`, so records pass `"dkt record"` and keep today's message.
  - `func (p Parsed) Size(sections []string) int`, the sum of `change.Size` over the named sections; `BodySize()` returns `p.Size(BodySections)`.
  - `scan.Dated(g gitx.Git, dir string) (Inventory, error)`, today's `Records` with `dir` for `record.Dir`; `scan.Records(g)` returns `scan.Dated(g, record.Dir)`.
  - In `internal/cli`, a helper that runs the write loop of `runRecord` for a folder: it takes the folder, the requested slug, the bytes and the inventory, and returns the slug and path written and the uses, or the outcome or error `runRecord` returns today. `cleanupRecord` takes the folder it checks for links. `newResult.Kind` is tagged `json:"kind,omitempty"`.
  - In `internal/lint`, a descriptor for a kind of dated file (folder, body sections, extra sections allowed after them, kinds or none, whether `### Task` headings under `## Tasks` are checked, rule prefix, template name, limit), with the record descriptor reproducing today's rules, messages and order of findings; the folder listing and the unreadable flag are kept per folder.

- [x] **Step 1: Run** `go test ./internal/record/ ./internal/scan/ ./internal/lint/ ./internal/cli/` and keep the result: this is the behavior to preserve.
- [x] **Step 2: Refactor** to the interfaces above, one package at a time, running that package's tests after each. Do not change any test assertion; only change a test's call when a signature changed.
- [x] **Step 3: Check** that no record output changed: `git diff` touches no expected message, rule name or JSON field in a test. Only records call the new parameters in this task, so a parameter that is ignored cannot fail a record test; the issue tests of Tasks 2, 3 and 5 are written to fail in that case (another worktree, a linked folder, an unreadable source, the issue sections and template). The lead's mutation check here covers the record values passed in: changing one of them must make a record test fail.
- [x] **Step 4: Run** `go vet ./...` and `go test ./...`; expect PASS. Commit as a structural change.

### Task 2: The issue package, the issue template and the scan for issues

**Scenarios:** workspace-scan#issue-only-on-a-branch, workspace-scan#records-are-not-issues

**Review:** checks

**Files:**
- Create: `internal/issue/issue.go`
- Create: `internal/issue/issue_test.go`
- Create: `shipped/issue/template.md`
- Modify: `internal/scan/records.go`
- Modify: `internal/scan/records_test.go`
- Modify: `shipped_test.go`

**Interfaces:**
- Consumes: `scan.Dated`, `record.BodyLimit` from Task 1.
- Produces: package `issue` with `const Dir = "docket/issues"`, `var Sections = []string{"Current", "Expected", "Unchanged", "Evidence"}`, `const BodyLimit = record.BodyLimit`, `func Path(slug string) string` (`Dir + "/" + slug + ".md"`); `scan.Issues(g gitx.Git) (Inventory, error)` returning `scan.Dated(g, issue.Dir)`.

- [x] **Step 1: Write the failing tests.** `internal/scan/records_test.go`: `TestIssuesOnBranch` (a remote-tracking branch holds `docket/issues/2026-11-06-lint-timeout.md`: `scan.Issues` reports that slug with the ref as source) `TestIssuesInWorktree` (another worktree holds an uncommitted `docket/issues/2026-11-06-lint-timeout.md`: `scan.Issues` reports it with that worktree as source), `TestIssuesLinkedFolder` (another worktree's `docket/issues` is a symbolic link: `scan.Issues` lists that worktree as unreadable and reports no use from the link's target) and `TestIssuesAndRecordsApart` (a worktree and a branch with only a record file: `scan.Issues` reports nothing from them; with only an issue file: `scan.Records` reports nothing from them). `internal/issue/issue_test.go`: `TestTemplateSections`, the level-two headings of `shipped/issue/template.md` (read from `docket.Shipped`) are `issue.Sections` in order. `shipped_test.go`: add `shipped/issue/template.md` to `TestShippedFileSet`.
- [x] **Step 2: Run** `go test ./internal/scan/ ./internal/issue/ .`; expect FAIL (no `scan.Issues`, no package, no file).
- [x] **Step 3: Implement** the package and `scan.Issues`, and copy `shipped/issue/template.md` byte for byte from the approved draft `docs/pages/prompts/tpl-issue.en.md` in the main worktree (the lead gives its absolute path).
- [x] **Step 4: Run** `go vet ./...` and `go test ./...`; expect PASS. Commit.

### Task 3: dkt issue writes an issue

**Scenarios:** issues#issue-written, issues#missing-issue-source, issues#issue-title-over-the-limit, issues#issue-name-with-a-date, issues#same-issue-name-same-day, issues#concurrent-issues, issues#linked-issues-folder-refused, issues#failed-issue-write-leaves-nothing, cli-output#issue-outside-a-repository, cli-output#init-outside-a-repository, cli-output#new-outside-a-repository, cli-output#record-outside-a-repository, cli-output#sync-outside-a-repository, cli-output#lint-outside-a-repository, cli-output#conflicts-outside-a-repository

**Review:** judgment

**Files:**
- Create: `internal/cli/cmd_issue.go`
- Create: `internal/cli/cmd_issue_test.go`
- Modify: `internal/cli/command.go`
- Modify: `internal/cli/command_test.go`

**Interfaces:**
- Consumes: the write helper, `record.CheckName`, `record.Slug`, `newResult` from Task 1; `issue.Dir`, `issue.Path`, `scan.Issues` from Task 2.
- Produces: `issueCommand()` in `commands` after `recordCommand`, usage `issue <name> --source <text> --title <text> [--json] | issue template [--json]`, `Args: 1`; `runIssue(c *Context, name, source, title string) (Outcome, error)`. The result is `newResult` without `kind`. The usage errors read `issue needs --source: one line of text that is not blank`, `issue needs --title: ...` and `issue needs a name`, as the record ones do.

- [x] **Step 1: Write the failing tests** in `cmd_issue_test.go`, following `cmd_record_test.go` (its date injection, branch setup, link and failed-write seams): one test per scenario above for `dkt issue`, checking the file's front matter (`title`, `source`, no `kind`) and that the rest of it is `shipped/issue/template.md`, and `TestIssueUnreadableSource` (a listed worktree cannot be read: nothing written, `unreadable` lists it, exit 3). Add the outside-a-repository scenario to the existing table in `command_test.go`, and `issue` with its flags to `TestCommandHelp`.
- [x] **Step 2: Run** `go test ./internal/cli/ -run 'Issue|CommandHelp|Repository'`; expect FAIL.
- [x] **Step 3: Implement** `runIssue` in the order `runRecord` checks: flags, title length, `record.CheckName(name, "dkt issue")`, slug, template, link check on the requested path, `scan.Issues`, unreadable sources, then the write helper.
- [x] **Step 4: Run** `go vet ./...` and `go test ./...`; expect PASS. Commit.

### Task 4: dkt issue template prints the template for another tool

**Scenarios:** issues#template-for-another-tool, issues#template-with-a-flag

**Review:** checks

**Files:**
- Create: `shipped/issue/external.md`
- Modify: `internal/cli/cmd_issue.go`
- Modify: `internal/cli/cmd_issue_test.go`
- Modify: `internal/issue/issue_test.go`
- Modify: `shipped_test.go`

**Interfaces:**
- Consumes: `issueCommand`, `runIssue` from Task 3; `issue.Sections` from Task 2.
- Produces: `dkt issue template [--json]`: the text output is the file's bytes; `result` is `{"template": "<bytes>"}`; `--source` or `--title` with `template` is a `UsageError` reading `issue template takes no other argument or flag`. A second argument is already the usage error `unexpected argument` from `Args: 1`, which the spec accepts.

- [x] **Step 1: Write the failing tests.** `cmd_issue_test.go`: `TestIssueTemplate` (`--json`: `result.template` equals `shipped/issue/external.md`, no `docket/` created, exit 0; text: stdout equals the file), `TestIssueTemplateWithFlag` (`--title X --source x`, and separately a second argument: exit 2 each time, nothing written, no `docket/issues/<date>-template.md`). `internal/issue/issue_test.go`: `TestExternalSections`, the level-two headings of `shipped/issue/external.md` are `issue.Sections` in order. `shipped_test.go`: add the file to `TestShippedFileSet`.
- [x] **Step 2: Run** them; expect FAIL.
- [x] **Step 3: Copy** `shipped/issue/external.md` byte for byte from the approved draft `docs/pages/prompts/tpl-issue-external.en.md`, and handle `template` in `issueCommand` before `runIssue`.
- [x] **Step 4: Run** `go vet ./...` and `go test ./...`; expect PASS. Commit.

### Task 5: dkt lint checks issues

**Scenarios:** issues#issue-the-branch-adds, issues#merged-issue-left-alone, issues#issue-without-front-matter, issues#issue-title-too-long, issues#issue-section-empty, issues#record-section-in-an-issue, issues#issue-over-the-limit, records#named-record-linted, records#named-record-missing, records#named-issue-linted, change-lint#lint-every-active-change, change-lint#named-change-missing, change-lint#named-change-skips-main-specs, change-lint#named-change-skips-records, change-lint#clean-change, change-lint#records-in-the-result, change-lint#issues-in-the-result

**Review:** judgment

**Files:**
- Modify: `internal/lint/lint.go`
- Modify: `internal/lint/record.go`
- Create: `internal/lint/issue_test.go`
- Modify: `internal/lint/record_test.go`
- Modify: `internal/cli/cmd_lint.go`
- Modify: `internal/cli/cmd_lint_test.go`

**Interfaces:**
- Consumes: the lint descriptor from Task 1; `issue.Dir`, `issue.Sections`, `issue.BodyLimit` from Task 2.
- Produces: `lint.Input.Issues []string` (paths the branch adds under `docket/issues`, nil when there is no default branch or merge base) and `lint.Input.IssueTemplate []byte`; `lint.Report.Issues []string`; in `dkt lint`, `result.issues`, never null, and the text line `Linted %d changes, %d records, %d issues and %d main spec titles.`; the issue descriptor (no kinds, no extra sections, prefix `issue`, limit `issue.BodyLimit`, template `shipped/issue/template.md`). A dated name is looked up in both folders; `record-missing` only when neither holds a regular file, with the message `no record or issue named %q: neither %s nor %s is a regular file`.

- [x] **Step 1: Write the failing tests.** `internal/lint/issue_test.go`: one test per issue scenario above, plus `TestIssueTasksSection` (`## Tasks` holding a malformed `### Task x` heading in an issue: `issue-section` naming `Tasks`, and no `task-heading` or other task finding), `TestIssuePlaceholder` (a comment-free line left from the issue template: `placeholder`), and `TestSameSlugRecordAndIssue` (both files exist with that slug: both linted, no `record-missing`). Update `record_test.go` only where the named-record message changed. `cmd_lint_test.go`: `result.issues` in the clean-change case, the issues-in-the-result scenario, the new text line.
- [x] **Step 2: Run** `go test ./internal/lint/ ./internal/cli/ -run 'Issue|Lint|Record'`; expect FAIL.
- [x] **Step 3: Implement** with the descriptor; `cmd_lint.go` fills `Issues` with `g.AddedSince(base, issue.Dir)` where it fills `Records`, and reads the issue template.
- [x] **Step 4: Run** `go vet ./...` and `go test ./...`; expect PASS. Commit.

### Task 6: AGENTS.md and README

**Scenarios:** none

**Review:** checks

**Files:**
- Modify: `AGENTS.md`
- Modify: `README.md`

- [x] **Step 1: Edit** the bullet on what does not exist yet: remove `dkt issue` from it and remove its sub-bullet "To record an issue, tell the user instead; it is recorded once `dkt issue` exists." Keep everything else. In `README.md`, add `dkt issue` and `dkt issue template` to the list of commands, in the list's own style, after rebasing on main once the README rewrite has merged.
- [x] **Step 2: Run** `go run ./cmd/dkt issue template` and `go run ./cmd/dkt lint` at the worktree root; both exit 0. Commit.
