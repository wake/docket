# Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** File issues under `docket/issues/`, written by `dkt issue`, checked by `dkt lint`, listed by `dkt status` with a state derived from the records and changes that name them, and the `docket` skill sending settled work to an issue.

**Architecture:** File issues are records one step earlier, so they run on the record code: the write path (`record.Create`), the slug rules, the section parser and the word count are shared, not copied. A new package `internal/issue` holds what differs (the folder, the four sections, the front matter with `dropped` and `idea`). The scan reads `docket/issues` wherever it reads `docket/records`. `dkt status` derives each issue's state from the `source` of every record and change, active or archived.

**Tech Stack:** Go 1.26, `go.yaml.in/yaml/v3` (already a dependency), the standard library.

**Spec:** `openspec/changes/add-issues/proposal.md`, `specs/issues/spec.md`, `specs/workspace-scan/spec.md`, `specs/change-lint/spec.md`, `design.md`.

## Global Constraints

- Folder `docket/issues`; file `docket/issues/<YYYY-MM-DD>-<name>.md`; sections exactly `## Current`, `## Expected`, `## Unchanged`, `## Found`, in that order.
- Front matter keys: `title` (`change.ValidTitle`), `source` (non-empty), optional `dropped` (only `true`), optional `idea` (a regular file under `docket/ideas/` in the worktree).
- Each section at most 60 words outside comments, counted by `record.Words`.
- Lint rules: `issue-front-matter`, `issue-section`, `issue-too-long`, `placeholder`, `issue-source`.
- A record or change names an issue when its source, with surrounding whitespace trimmed, equals `docket/issues/<slug>.md`.
- States listed by `dkt status`: `pending`, `in-progress`. `dropped`, `moved` and `handled` are never listed.
- `dkt status --json`: `result.issues`, each `{slug, title, source, state, handlers: [{kind, slug, merged}], sources}`; `title` and `source` null when unreadable; never null as a whole (empty list).
- Exit codes as `cli-output`: 0 ok, 1 findings, 2 usage, 3 unreadable, 4 environment.
- `dkt` never runs an outside issue tool and reads no network.
- No new dependency. CI runs jscpd with `--fail-on-new-clones=0`: share code instead of copying it.
- Commits carry `Change: add-issues` and `Task: <N>` in the last paragraph; structure and behavior in separate commits.
- Public repository: English only in code, tests and docs (README.zh-TW.md excepted).

## Review Focus

- A source written with surrounding spaces, `  docket/issues/2026-11-02-x.md `, still names the issue; a source that only mentions the path in a sentence does not. Tests in Task 4 and Task 5.
- An issue with `dropped: true` and an open record naming it is not listed: what people decided wins. Test in Task 5.
- An archived change made before `docket.yaml`, whose source is in `.openspec.yaml` as `x-source`, still counts as a handler. Test in Task 5.
- An issue that exists only uncommitted in another worktree is listed as `pending` with that worktree as its source. Test in Task 5.
- `dkt issue template --title X`, or `dkt issue template extra`, is a usage error, not an issue named `template`. Test in Task 3.

## Reuse

| Need | Existing (path:symbol) | Decision | Reason | Evidence |
|---|---|---|---|---|
| Write a dated file: never-reused slug, exclusive create, no write through links, cleanup after a failed write | internal/record/record.go:Create | reuse | already takes the folder as a parameter | `rg -n "func Create" internal/record` |
| Dated slug of a name, and the slug of a file name | internal/record/record.go:Slug | reuse | the same `YYYY-MM-DD-<name>` rule; `SlugOf` alike | `rg -n "func Slug" internal/record` |
| Name rules before the date is added | internal/record/record.go:ValidateName | extend | its message names `dkt record`; it takes the command's name | `rg -n "ValidateName" internal` |
| Split front matter, blank comments, read level-two sections | internal/record/parse.go:Parse | extend | split out the part before `change.ParseDocket` so issues decode their own front matter | `rg -n "func Parse" internal/record` |
| Word count of a section | internal/record/parse.go:Words | reuse | same limit and counting | `rg -n "func Words" internal/record` |
| Section order, presence, emptiness and length checks | internal/lint/record.go:checkRecordSections | extend | rule names and section list become parameters | `rg -n "checkRecordSections" internal/lint` |
| Template text left in a file | internal/lint/placeholder.go:checkPlaceholderFile | reuse | collect the issue template's comments as the record template's are | `rg -n "collectPlaceholders" internal/lint` |
| Files a branch adds under a folder | internal/cli/cmd_lint.go:addedFiles | reuse | add `docket/issues` to the folders asked for | `rg -n "func addedFiles" internal/cli` |
| Find dated files in every source | internal/scan/scan.go:readDisk | extend | `readDisk`, `readTree`, `readTreeEach` and `readTrees` read a list of dated folders instead of the records folder alone | `rg -n "records \[\]string" internal/scan` |
| Sources of a slug | internal/scan/scan.go:RecordSources | extend | an `IssueSources` twin over the same lookup | `rg -n "func (inv Inventory) RecordSources" internal/scan` |
| Read files of a source | internal/scan/scan.go:ReadFiles | reuse | issues, records and archived metadata are read with it | `rg -n "func ReadFiles" internal/scan` |
| A record's state: open or merged | internal/status/records.go:stateOf | reuse | a handler is merged when the default branch tip holds it | `rg -n "func stateOf" internal/status` |
| A change's source, active or archived | internal/change/metadata.go:ReadMetadata | reuse | reads `docket.yaml` and the older `.openspec.yaml` | `rg -n "func ReadMetadata" internal/change` |
| Active changes with their metadata | internal/status/load.go:Load | reuse | `Change.Metadata.Source` | `rg -n "Metadata.*Source" internal/status` |
| Embed a template | schema.go:RecordTemplate | extend | an `IssueTemplate` beside it | `rg -n "go:embed" schema.go` |
| Register a command | internal/cli/run.go:commands | extend | add `issueCommand` | `rg -n "var commands" internal/cli` |
| Print a shipped template | internal/cli/cmd_workflow.go:workflowCommand | new | `dkt issue template` follows its `template` branch; there is nothing to call | `rg -n "args\[0\] == \"template\"" internal/cli` |
| Issue front matter with `dropped` and `idea` | internal/change/docket.go:ParseDocket | new | `change.Docket` has no `dropped` or `idea`, and decoding into it drops them silently | `rg -n "type Docket struct" internal/change` |

---

### Task 1: Share the record code

**Scenarios:** none

**Review:** checks

**Files:**
- Modify: `internal/record/parse.go`
- Modify: `internal/record/record.go`
- Modify: `internal/cli/cmd_record.go`
- Modify: `internal/lint/record.go`
- Test: `internal/record/parse_test.go`

**Interfaces:**
- Produces: `record.Split(b []byte) (front []byte, sections []Section, plan change.Plan, err error)` — `err` is the missing-front-matter error `Parse` reports today; `Parse` becomes `Split` followed by `change.ParseDocket`.
- Produces: `record.ValidateName(command, name string) error` — the date message says "dkt <command> adds the date".
- Produces in `internal/lint`: `type sectionRules struct{ Prefix string; Sections []string; Optional string }` and `checkSections(r *run, c *loaded, name string, secs []record.Section, rules sectionRules)` reporting `<Prefix>-section` and `<Prefix>-too-long`; `checkRecordSections` calls it with `{"record", record.Sections, record.TasksSection}`.

Structure only: every existing test passes unchanged in behavior.

- [ ] **Step 1: Write `TestSplit`** in `internal/record/parse_test.go`: a file with front matter `title: X` and two sections returns `front == "title: X"`, two sections with their lines, no error; a file without the opening `---` returns the missing-front-matter error.
- [ ] **Step 2: Run** `go test ./internal/record/ -run TestSplit` — FAIL (undefined: Split).
- [ ] **Step 3: Extract `Split`** from `Parse`; `Parse` calls it. Change `ValidateName` to take the command; update `cmd_record.go`. Extract `checkSections` from `checkRecordSections`, with the rule names built from `Prefix` and the heading list from `Sections` plus `Optional`.
- [ ] **Step 4: Run** `go test ./internal/record/ ./internal/lint/ ./internal/cli/ -run 'Record|Split'` — PASS, and the messages of the record findings are unchanged.
- [ ] **Step 5: Commit** `refactor(record): split the parser and the section checks out for reuse`.

### Task 2: Scan finds file issues

**Scenarios:** workspace-scan#issue-on-a-branch, workspace-scan#no-issues-folder

**Review:** checks

**Files:**
- Create: `internal/issue/issue.go`
- Modify: `internal/scan/scan.go`
- Modify: `internal/scan/trees.go`
- Test: `internal/scan/scan_test.go`, `internal/scan/trees_test.go`

**Interfaces:**
- Produces: package `internal/issue` with `const Dir = "docket/issues"` (more in Task 3).
- Produces: `scan.Issue struct{ Slug string; Sources []Source }`, `Inventory.Issues []Issue` (never nil, sorted by slug), `func (inv Inventory) IssueSources(slug string) []Source`.
- An unreadable `docket/issues` is reported as `Unreadable{Source, Path: "docket/issues"}` and leaves the source readable, as the records folder does.

- [ ] **Step 1: Structure commit.** Replace the `records []string, recErr error` pair in `readDisk`, `readTree`, `readTreeEach`, `readTrees` and `found.add` with lists keyed by folder, over a package list `datedDirs` that holds `record.Dir` only. Run `go test ./internal/scan/` — PASS. Commit `refactor(scan): read dated files from a list of folders`.
- [ ] **Step 2: Write the failing tests:** `TestIssueOnABranch` (a branch and worktree B hold `docket/issues/2026-11-02-x.md`; the scan from worktree A reports it once, worktree B first, the branch second), `TestNoIssuesFolder` (no `docket/issues`: no issue, the source readable), `TestIssuesFolderUnreadable` (`docket/issues` is a file at a commit: one `Unreadable` with that path), and the batch-tree path in `trees_test.go` reading `docket/issues` at a commit.
- [ ] **Step 3: Run** `go test ./internal/scan/ -run 'Issue'` — FAIL.
- [ ] **Step 4: Add `issue.Dir` to `datedDirs`**, `Inventory.Issues`, `IssueSources`.
- [ ] **Step 5: Run** `go test ./internal/scan/` — PASS.
- [ ] **Step 6: Commit** `feat(scan): find file issues`.

### Task 3: The issue template and dkt issue

**Scenarios:** issues#template-printed, issues#issue-written, issues#missing-source-writes-nothing, issues#issue-name-with-a-date, issues#same-name-same-day, issues#unreadable-source-blocks-the-issue, issues#symlinked-issues-folder-refused, issues#failed-write-leaves-nothing

**Review:** judgment

**Files:**
- Create: `issue/template.md`
- Modify: `schema.go`
- Modify: `internal/issue/issue.go`
- Create: `internal/issue/issue_test.go`
- Create: `internal/cli/cmd_issue.go`
- Create: `internal/cli/cmd_issue_test.go`
- Modify: `internal/cli/run.go`
- Modify: `internal/record/record_test.go`

**Interfaces:**
- Consumes: `record.Create`, `record.Slug`, `record.ValidateName("issue", name)`, `record.Split`, `scan.Inventory.IssueSources`.
- Produces: `docket.IssueTemplate() string` (embeds `issue/template.md`).
- Produces in `internal/issue`: `var Sections = []string{"Current", "Expected", "Unchanged", "Found"}`; `type FrontMatter struct{ Title string; Source string; Dropped *bool; Idea string }` with yaml keys `title`, `source`, `dropped`, `idea`; `func ParseFrontMatter(b []byte) (FrontMatter, error)` (fails when not a mapping or a key does not decode to its type; an invalid title reads as ""); `type File struct{ FrontMatter FrontMatter; FrontMatterErr error; Sections []record.Section }`; `func Parse(b []byte) File`; `func Render(title, source, template string) ([]byte, error)`; `type Result struct{ Slug, Path, Title, Source, Requested string; Used []string }` with json keys `slug`, `path`, `title`, `source`, `requested`, `used` (`used` never null).
- Command: `dkt issue <name> --title <text> --source <text>` and `dkt issue template`; result of the latter `{"text": ...}`.

`issue/template.md`, exactly:

```markdown
<!-- Each section: at most 60 words, outside comments; link to the
details. Replace every comment with your text. -->

## Current

<!-- What happens now, and where: the command, file or screen, and how
to see it. -->

## Expected

<!-- What should happen instead, and where that is settled: the spec
sentence, the docs, or the behavior that shipped before. If nothing
settles it, this is an idea, not an issue. -->

## Unchanged

<!-- What the fix must keep as it is: the behavior next to the problem
that a careless fix could break. -->

## Found

<!-- How it was found: a review, a trial or someone's report, with the
evidence (a log line, a commit, a link). -->
```

- [ ] **Step 1: Write the failing tests** in `internal/issue/issue_test.go`: `TestParseFrontMatter` (`dropped: true` reads as true; `dropped: yes` is an error; `idea` read; a list is an error); `TestRender` (front matter keys in the order title, source, then the template). In `internal/cli/cmd_issue_test.go`, one test per scenario: `TestIssueTemplatePrinted` (`--json` `result.text` holds the four headings in order), `TestIssueWritten` (the slug, the file's front matter and sections, exit 0), `TestIssueMissingSource` (exit 2, nothing under `docket/issues`), `TestIssueNameWithDate` (exit 1, message says `dkt issue` adds the date), `TestIssueSameNameSameDay` (another worktree's branch holds the slug: `-2` written), `TestIssueUnreadableSource` (exit 3, nothing written), `TestIssueSymlinkedFolder` (exit 1, the link named, nothing written on either side), `TestIssueTemplateArgs` (`dkt issue template extra` and `dkt issue template --title X` exit 2). In `internal/record/record_test.go`, `TestCreateFailedWriteRemovesIssuesFolder`: `Create` into a missing `docket/issues` with a failing write leaves neither the file nor the folder.
- [ ] **Step 2: Run** `go test ./internal/issue/ ./internal/cli/ ./internal/record/ -run 'Issue|FrontMatter|Render'` — FAIL.
- [ ] **Step 3: Implement** `internal/issue`, the template and its embed, and `cmd_issue.go`, following `cmd_record.go`: usage errors before any read; `template` handled before the name rules, refusing any flag or other argument; `safepath.Check` on the issue path; the scan's unreadable sources stop it with exit 3; `record.Create` with `inv.IssueSources(cand) != nil` as `used`; a `*record.LinkError` becomes the symlink refusal naming `docket/issues/<name>`. Register `issueCommand` after `recordCommand`.
- [ ] **Step 4: Run** `go test ./internal/issue/ ./internal/cli/ ./internal/record/` — PASS.
- [ ] **Step 5: Commit** `feat(issue): dkt issue writes a file issue from the shipped template`.

### Task 4: Lint file issues and the sources that name them

**Scenarios:** issues#issue-front-matter-invalid, issues#issue-section-empty, issues#issue-template-text-left, issues#issue-section-over-the-limit, issues#source-names-no-issue, issues#source-names-an-issue, change-lint#lint-issues-the-branch-adds, change-lint#issues-without-a-default-branch

**Review:** judgment

**Files:**
- Create: `internal/lint/issue.go`
- Create: `internal/lint/issue_test.go`
- Modify: `internal/lint/lint.go`
- Modify: `internal/lint/placeholder.go`
- Modify: `internal/cli/cmd_lint.go`
- Modify: `internal/cli/cmd_lint_test.go`

**Interfaces:**
- Consumes: `issue.Parse`, `issue.Sections`, `issue.Dir`, `checkSections`, `record.SlugOf`.
- Produces: `lint.Input.AddedIssues []string` and `lint.Input.IssueTemplate string`; `lint.Report.Issues []string` (sorted, never nil in JSON); `dkt lint --json` `result.issues`.
- `issue-source`: for each linted change (its metadata source) and each record the branch adds (its front matter source), a trimmed source starting with `docket/issues/` that is not a regular file in the worktree is reported, naming the source.

- [ ] **Step 1: Write the failing tests** in `internal/lint/issue_test.go`, one per scenario: front matter with an `idea` naming no file, `dropped: false`, no `source`; `## Unchanged` empty; the template's `## Found` comment left (`placeholder`); `## Current` with 61 English words (`issue-too-long` naming `Current` and 61); a record source naming no issue, and one naming an issue that exists; a source with surrounding spaces naming an existing issue gives no finding; a change whose source names no issue. In `cmd_lint_test.go`: the branch adds `2026-11-02-b` while the default branch holds `2026-10-01-a` (only b linted, `result.issues` is `["2026-11-02-b"]`); no default branch with an uncommitted issue (exactly one unreadable entry, no issue finding).
- [ ] **Step 2: Run** `go test ./internal/lint/ ./internal/cli/ -run 'Issue'` — FAIL.
- [ ] **Step 3: Implement** `lintIssues` beside `lintRecords`, run only when no name is given; front matter checks; `checkSections` with `{"issue", issue.Sections, ""}`; `checkPlaceholderFile`; `collectPlaceholders` for `issue/template.md`; `checkIssueSources`; `cmd_lint.go` asks `addedFiles` for `issue.Dir` too and fills `AddedIssues` and `IssueTemplate`.
- [ ] **Step 4: Run** `go test ./internal/lint/ ./internal/cli/` — PASS.
- [ ] **Step 5: Commit** `feat(lint): check file issues and the sources that name them`.

### Task 5: dkt status lists file issues

**Scenarios:** issues#handler-on-a-branch, issues#handler-merged, issues#pending-issue-listed, issues#issue-in-progress, issues#handled-issue-hidden, issues#dropped-and-moved-hidden, issues#issues-not-listed-with-slugs

**Review:** judgment

**Files:**
- Create: `internal/status/issues.go`
- Create: `internal/status/issues_test.go`
- Modify: `internal/cli/cmd_status.go`
- Create: `internal/cli/cmd_status_issues_test.go`

**Interfaces:**
- Consumes: `scan.Inventory.Issues`, `.Records`, `.Archived`, `scan.ReadFiles`, `issue.Parse`, `record.Parse`, `change.ReadMetadata`, `status.Change.Metadata`.
- Produces: `status.Pending = "pending"`, `status.InProgress = "in-progress"`; `type Handler struct{ Kind, Slug string; Merged bool }` (json `kind`, `slug`, `merged`); `type Issue struct{ Slug string; Title, Source *string; State string; Handlers []Handler; Sources []scan.Source }` (json `slug`, `title`, `source`, `state`, `handlers`, `sources`; `handlers` never null); `func LoadIssues(g gitx.Git, inv scan.Inventory, changes []Change) ([]Issue, []scan.Unreadable, error)`.

State, in this order: `dropped: true` hides the issue; an `idea` hides it; otherwise a handler that the default branch tip does not hold makes it `in-progress`; else a handler the tip holds makes it handled and hidden; else `pending`. Records come from every `inv.Records` entry, read from its first source; changes from `changes` (active) and from each `inv.Archived` folder's `docket.yaml` or `.openspec.yaml`, read from its first source. Read files once per source with `scan.ReadFiles`, and read no record or change at all when no issue is left to decide.

- [ ] **Step 1: Write the failing tests** in `internal/status/issues_test.go`: one per scenario of Issue state (`TestHandlerOnABranch`, `TestHandlerMerged`), and the Review Focus cases (a source with spaces matches; a sentence mentioning the path does not; `dropped: true` with an open record hidden; an archived change with only `.openspec.yaml` `x-source` counts as a merged handler; an issue only uncommitted in worktree B is `pending` with worktree B as its source). In `cmd_status_issues_test.go`, one per listing scenario: pending listed with no handler; in progress with an unmerged handler from worktree B; handled hidden; dropped and moved hidden; with the slug `add-x`, `result.issues` is `[]`; the human output has one line per listed issue after the records.
- [ ] **Step 2: Run** `go test ./internal/status/ ./internal/cli/ -run 'Issue|Handler'` — FAIL.
- [ ] **Step 3: Implement** `LoadIssues` and wire it into `cmd_status.go` when no slug is given; its unreadable entries join the command's. Human line: `issue <slug> <state> <title>`, then ` (<kind> <slug>[ merged], ...)` when it has handlers.
- [ ] **Step 4: Run** `go test ./internal/status/ ./internal/cli/` — PASS.
- [ ] **Step 5: Commit** `feat(status): list file issues with the work that names them`.

### Task 6: The docket skill, the README and the architecture notes

**Scenarios:** issues#bug-found-out-of-scope, issues#issue-in-the-project-tool, issues#record-names-its-issue

**Review:** checks

**Files:**
- Modify: `.agents/skills/docket/SKILL.md`
- Modify: `.claude/skills/docket/SKILL.md`
- Modify: `AGENTS.md`
- Modify: `README.md`
- Modify: `README.zh-TW.md`
- Modify: `docs/ARCHITECTURE.md`

Prompt text, so no test looks for strings; the check is mechanical.

- [ ] **Step 1: The skill, both copies identical.** In `## Triage`, after the idea sentence, add: "Work whose expected behavior is already settled, by a spec sentence, the docs or behavior that shipped, and that is not done now is an issue, not an idea: a bug found outside the work at hand, an optimization, a missing test, a request from someone else. When the project's `AGENTS.md` has a `## Issues` section, write the issue in the tool it names, with the sections `dkt issue template` prints; show it to the user and file it only once they agree. Otherwise run `dkt issue <name> --title "<one line>" --source "<where it was found>"` and fill its sections. When you cannot point to where the expected behavior is settled, it is an idea." In `## Full review` step 1 and `## Light review` step 1 add: "When the work handles an issue, `--source` is the issue alone: the path of a file issue (`docket/issues/<slug>.md`) or its address in the project's tool."
- [ ] **Step 2: Docs.** README (both languages): `dkt issue` in the command list and a paragraph on issues (the template, `## Issues` in `AGENTS.md`, file issues, the derived state). `AGENTS.md`: `dkt issue` in the Status line's command list and `issue/template.md` in the prompt paths. `docs/ARCHITECTURE.md`: the `internal/issue` package and the `issues` capability.
- [ ] **Step 3: Check.** `diff .agents/skills/docket/SKILL.md .claude/skills/docket/SKILL.md` prints nothing; `go run ./cmd/dkt issue template` prints the four sections; `go test ./internal/cli/ -run 'Init|Inspect'` passes (the shipped skill embeds and installs); `go run ./cmd/dkt lint` ok.
- [ ] **Step 4: Commit** `docs(skill): send settled work to an issue` and `docs: issues in the README and the architecture notes`.
