# Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `dkt record` writes a short record of light-review work, `dkt lint` checks it, `dkt sync` ticks its tasks, and titles and record bodies have one length rule in UTF-8 bytes.

**Architecture:** `internal/record` holds what a record is. `dkt record` (`internal/cli/cmd_record.go`) follows `dkt new`'s shape with a new `scan.Records`. `internal/lint/record.go` lints records through the same `fs.FS` as changes. `dkt sync` splits at the top by slug and shares everything after reading the file.

**Tech Stack:** Go 1.24, standard library only; git and the OpenSpec CLI through the existing wrappers.

**Spec:** `openspec/changes/add-dkt-record/proposal.md`, `design.md`, `specs/`.

## Global Constraints

- Title limit: 120 bytes. Body limit: 1,440 bytes. Measured by `len(strings.Join(strings.Fields(text), " "))`, with no rule for any language.
- Record kinds: `fix`, `refactor`, `test`, `prompt`, `checks`.
- Record name: `^[a-z0-9]+(-[a-z0-9]+)*$`, at most 200 characters, not starting with `YYYY-MM-DD`.
- Rule names: `title-too-long`, `record-too-long`, `record-front-matter`, `record-kind`, `record-section`, `record-missing`, `record-trailer`, `placeholder`, `task-heading`, `slug-invalid`, `symlink-refused`.
- `shipped/record/template.md` is settled text: copy it from Task 3 byte for byte.
- No new dependency. Commits carry `Change: add-dkt-record/<N>` in the last paragraph. Structural changes and behavior changes go in separate commits.
- `go vet ./...` and `go test ./...` pass after every task.
- `add-dkt-conflicts` is editing `internal/scan/scan.go`, `internal/gitx/refs.go`, `internal/cli/command.go` and `AGENTS.md`: touch those only where the task says, and keep new code in new files.

## Review Focus

- A record written by `dkt record`, with the comments replaced by text, must pass `dkt lint` with no finding, through the real template. Pinned in Task 4 (`TestRecordRoundTrip`).
- A repository with no commit yet: `dkt record` and `dkt sync` must work, and `dkt lint` must skip the branch rules. Pinned in Tasks 3 and 5.
- `docket` is a regular file: `dkt record` must fail, write nothing and leave the file. Pinned in Task 3.
- A record with CRLF line endings or a BOM: lint must find its sections and `dkt sync` must keep every other byte. Pinned in Tasks 4 and 5.
- Text outside the four sections must not escape the length limit. Pinned in Task 4.

## Reuse

| Need | Existing (path:symbol) | Decision | Reason | Evidence |
|---|---|---|---|---|
| One line of text | internal/change/metadata.go:ValidLine | reuse | source keeps it as is; the title limit is a separate check | `rg -n "ValidLine" internal/` |
| Date test on a slug | internal/change/metadata.go:CheckSlug | extend | export the regexp as `HasDate`, so lint and sync use the same test | `rg -n "datePrefix" internal/` |
| Free suffix | internal/change/metadata.go:Suffixed | reuse | | `rg -n "func Suffixed" internal/` |
| Front matter writer | internal/change/metadata.go:FrontMatter | reuse | same three keys | `rg -n "func FrontMatter" internal/` |
| Front matter reader | internal/change/frontmatter.go:ReadFrontMatter | reuse | | `rg -n "func ReadFrontMatter" internal/` |
| Tasks and steps | internal/change/plan.go:ParsePlan | reuse | given the record with lines blanked | `rg -n "func ParsePlan" internal/` |
| Ticking | internal/change/tick.go:Tick | reuse | | `rg -n "func Tick" internal/` |
| Trailer values | internal/change/metadata.go:TaskRefs | reuse | takes any slug | `rg -n "func TaskRefs" internal/` |
| Trailers by key | internal/gitx/log.go:ChangeTrailers | extend | the format string is all that differs | `rg -n "ChangeTrailers" internal/` |
| Files added on the branch | internal/gitx/blobs.go:AddedSince | reuse | | `rg -n "func .*AddedSince" internal/` |
| Refusing a link | internal/cli/cmd_new.go:refuse | reuse | | `rg -n "func refuse" internal/` |
| Result and used list | internal/cli/cmd_new.go:newResult | extend | move the used list and its lines to helpers | `rg -n "newUse" internal/` |
| Sections | internal/markdown/markdown.go:Headings | reuse | with FenceMask, BlankComments, Body | `rg -n "^func " internal/markdown/markdown.go` |
| Placeholder rule | internal/lint/document.go:checkPlaceholderFile | extend | split reading a template from the walk over schema templates | `rg -n "placeholders" internal/lint/` |
| Task heading rules | internal/lint/plan.go:checkTaskNumbers | extend | take the file name | `rg -n "planFile" internal/lint/plan.go` |
| Reading without a link | internal/safefile/safefile.go:ReadNoFollow | reuse | | `rg -n "ReadNoFollow" internal/` |
| Unreadable worktree reason | internal/scan/scan.go:worktreeDirs | extend | move the "directory missing" check to a helper | `rg -n "worktree prune" internal/` |
| Scan for records | none | new | `Changes` builds change uses from directories and is being edited by `add-dkt-conflicts` | `rg -n "func Changes" internal/scan/` |
| Files in a tree | internal/gitx/refs.go:TreeFiles | reuse | `add-dkt-conflicts` added it; `scan.Records` keeps the files directly in the folder | `rg -n "func .*TreeFiles" internal/gitx/` |
| Text length | none | new | no length helper exists; `quote` only truncates | `rg -n "strings.Fields" internal/` |
| Record parsing | none | new | a record has its own sections | `rg -n "record" internal/change internal/lint` |
| Clock | none | new | no command reads the time | `rg -n "time\." internal/cli cmd/` |

## Revision 1 (2026-11-06): the tree reader of add-dkt-conflicts, and four pull requests

Asked by the lead on rebasing onto `add-dkt-conflicts` (PR #86), and by the maintainer's instructions, which split a pull request past about 800 lines.

- `add-dkt-conflicts` added `gitx.Git.TreeFiles`, recursive and with a link flag. Task 2 uses it through `scan.treeRecords`, which keeps the files directly in the folder; it does not create `internal/gitx/tree.go`.
- The requirement Repository required takes its text from the main spec after PR #86.
- This change merges as four pull requests, in task order, each on the default branch once the one before has merged: Tasks 1 and 2, then Task 3, then Task 4, then Tasks 5 and 6 and the archive.

---

### Task 1: Text length and change titles

**Scenarios:** change-metadata#blank-title, change-metadata#source-on-two-lines, change-metadata#title-over-the-limit, change-metadata#chinese-title-at-the-limit, change-metadata#long-source, change-metadata#whitespace-collapsed, change-lint#proposal-without-front-matter, change-lint#unknown-kind, change-lint#change-title-too-long

**Review:** checks

**Files:**
- Modify: `internal/change/metadata.go`, `internal/cli/cmd_new.go`, `internal/lint/document.go`
- Test: `internal/change/metadata_test.go`, `internal/cli/cmd_new_test.go`, `internal/lint/document_test.go`

**Interfaces:**
- Produces: `change.TitleLimit = 120`; `change.Size(text string) int`; `change.TitleTooLong(title string) (msg string, tooLong bool)`, whose message reads `title is 121 bytes; the limit is 120`; `change.HasDate(s string) bool`, used by `CheckSlug`.

- [x] **Step 1: Write the failing tests**
  - `TestSize`: `Size("  a   b \n")` is 3; `Size(strings.Repeat("字", 40))` is 120; `Size("")` is 0.
  - `TestTitleTooLong`: 120 bytes gives `false`; 121 gives `true` and a message holding `121` and `120`; 40 Chinese characters give `false`; 120 bytes once three spaces count as one give `false`.
  - `TestHasDate`: `2026-10-30-x` true, `2026-10-30` true, `add-x` false.
  - `TestNewTitleTooLong`: `dkt new add-x --kind feature --source x --title <121 bytes>` exits 2, the message holds `121` and `120`, no folder exists, and the fake OpenSpec was not run.
  - `TestNewTitleAtLimit`: a `--title` of 40 Chinese characters creates the change. `TestNewLongSource`: a `--source` of 500 bytes creates the change.
  - `TestLintTitleTooLong`: a proposal title of 121 bytes reports `title-too-long` on the title's line, with `121` and `120`, and no `title-missing`.
- [x] **Step 2: Run them and see them fail**

Run: `go test ./internal/change ./internal/cli ./internal/lint`. Expected: compile errors or failures on the new names.

- [x] **Step 3: Implement**

`Size` is `len(strings.Join(strings.Fields(text), " "))`, with the comment `shortcut: Cyrillic, Arabic, Indic and Thai text reach the limit up to three times sooner than English; weigh Chinese, Japanese and Korean characters as 3 and others as 1 if that is reported.` In `runNew`, a new case after the title's `ValidLine` case returns `UsageError{Msg: "new needs --title: " + msg}`. In `checkFront`, the title's `else` adds `title-too-long` at `f.Line`. Do not change `ValidLine`.

- [x] **Step 4: Run the tests and see them pass**

Run: `go vet ./... && go test ./...`. Expected: all pass.

- [x] **Step 5: Commit** with `Change: add-dkt-record/1`.

### Task 2: The scan finds record files

**Scenarios:** workspace-scan#record-only-in-another-worktree, workspace-scan#record-only-on-a-branch, workspace-scan#project-without-records, workspace-scan#linked-records-folder-in-another-worktree

**Review:** checks

**Files:**
- Create: `internal/gitx/tree.go`, `internal/gitx/tree_test.go`, `internal/scan/records.go`, `internal/scan/records_test.go`
- Modify: `internal/scan/scan.go` (only the first lines of `worktreeDirs`)

**Interfaces:**
- Produces: `func (g Git) TreeFiles(commit, dir string) ([]string, error)`: the repository paths of the blobs directly under `dir` in `commit`'s tree, links included, in git's order; a tree with no such `dir` gives none. `func Records(g gitx.Git) (Inventory, error)`: `Inventory.Uses` has one `Use{Slug, Path, Source}` per record file and source, `Archived` false; sources and the unreadable reasons as in `Changes`.

- [x] **Step 1: Write the failing tests**
  - `TestTreeFiles`: a commit with `d/a.md`, `d/sub/b.md` and a link `d/l.md` gives `d/a.md` and `d/l.md`; `TreeFiles(commit, "none")` gives none and no error.
  - `TestRecordsOtherWorktree`, `TestRecordsOnBranch`: as the scenarios; the use's `Slug` is `2026-11-06-fix-x` and its `Path` is `docket/records/2026-11-06-fix-x.md`.
  - `TestRecordsNoFolder`: no `docket/` gives no use and no unreadable. `TestRecordsDocketIsFile`: a file at `docket` gives the same.
  - `TestRecordsLinkedFolder`: worktree B's linked `docket/records` is unreadable with a reason naming it, and no use.
  - `TestRecordsMissingWorktree`: a deleted worktree is unreadable, with `git worktree prune` in the reason.
  - `TestRecordsOnlyMarkdownFiles`: in a worktree, `docket/records/notes.txt` and a directory `docket/records/x.md/` give no use.
- [x] **Step 2: Run them and see them fail**

Run: `go test ./internal/gitx ./internal/scan`. Expected: undefined `TreeFiles` and `Records`.

- [x] **Step 3: Move the "directory missing" check out of `worktreeDirs`** into `worktreeMissing(root string) string` in `scan.go`, and commit it alone with `go test ./internal/scan` green.
- [x] **Step 4: Implement `TreeFiles` and `Records`**

`TreeFiles` runs `ls-tree -z --full-tree <commit> -- <dir>/` and keeps type `blob`. `Records` loops over `g.Worktrees()` and `g.Refs()` as `Changes` does, caching by commit. It treats `ENOTDIR` and a missing folder as no records and keeps names ending in `.md`, with the suffix removed for the slug.

- [x] **Step 5: Run the tests and see them pass**

Run: `go vet ./... && go test ./...`. Expected: all pass.

- [x] **Step 6: Commit** with `Change: add-dkt-record/2`.

### Task 3: `dkt record`

**Scenarios:** records#record-written, records#unknown-record-kind, records#missing-record-source, records#record-title-over-the-limit, records#name-with-a-date, records#name-that-is-not-kebab-case, records#same-name-same-day, records#unreadable-source-blocks-the-record, records#concurrent-records, records#linked-records-folder-refused, records#failed-write-leaves-nothing, cli-output#init-outside-a-repository, cli-output#new-outside-a-repository, cli-output#record-outside-a-repository, cli-output#sync-outside-a-repository, cli-output#lint-outside-a-repository, cli-output#conflicts-outside-a-repository

**Review:** judgment

**Files:**
- Create: `shipped/record/template.md`, `internal/record/record.go`, `internal/record/record_test.go`, `internal/cli/cmd_record.go`, `internal/cli/cmd_record_test.go`
- Modify: `shipped_test.go`, `internal/cli/command.go`, `internal/cli/run.go`, `internal/cli/cmd_new.go`

**Interfaces:**
- Consumes: `change.HasDate`, `change.TitleTooLong`, `scan.Records`.
- Produces: `record.Dir = "docket/records"`; `record.Kinds`; `record.BodyLimit = 1440`; `record.CheckName(name string) error`; `record.Slug(now time.Time, name string) string`, with the date as `2006-01-02` in `now`'s location; `record.Path(slug string) string`. `Options.Now func() time.Time` and `Context.Now`, nil meaning `time.Now`. In `cmd_new.go`: `usedPlaces(uses []scan.Use, slug string) []newUse` and `usedLines(slug string, used []newUse) []string`.

- [ ] **Step 1: Move the used list and its human lines out of `runNew`** into `usedPlaces` and `usedLines`, and commit it alone with `go test ./internal/cli` green.
- [ ] **Step 2: Write the failing tests**, all with `Options.Now` fixed at 2026-10-30:
  - `TestRecordWritten`: the scenario's command; the file reads `---\ntitle: "Lint timeout"\nkind: fix\nsource: "review of #40"\n---\n\n` followed by the template byte for byte; `result` has `slug`, `requested`, `path`, `title`, `kind`, `source`, and `used` as `[]`.
  - `TestRecordQuotedTitle`: `--title 'Fix: "x"'` is read back by `change.ReadFrontMatter` as `Fix: "x"`.
  - `TestRecordFlags`, table-driven: unknown kind exits 2 and lists the five; missing kind, source or title, and a two-line title, exit 2; a 121-byte title exits 2 with `121` and `120`. Each leaves no `docket/`.
  - `TestRecordName`, table-driven: `2026-10-30-fix-x` gives `slug-invalid` saying `dkt record` adds the date; `Fix_X`, `-x`, `x--y` and 201 characters give `slug-invalid`; each exits 1 and leaves no `docket/`. 200 characters are accepted, as long as the file name fits.
  - `TestRecordSuffix`: the same-name scenario, with the use on a branch; `result.used` is `[{"path":"docket/records/2026-10-30-fix-x.md","sources":["refs/heads/other"]}]`.
  - `TestRecordSuffixInWorktree`: a second run in one worktree writes `-2`.
  - `TestRecordUnreadableSource`: a deleted worktree: exit 3, `unreadable` lists it, nothing written.
  - `TestRecordConcurrent`: two `Run` calls in goroutines, as the scenario.
  - `TestRecordLinkedFolder`: `docket/records` and, separately, `docket` as links: `symlink-refused` naming the link, exit 1, nothing written on either side. `TestRecordLinkedFile`: the record's own path as a dangling link: `symlink-refused`, exit 1.
  - `TestRecordDocketIsFile`: a regular file at `docket`: non-zero exit, and the file is unchanged.
  - `TestRecordWriteFails`: replacing `writeRecordFile` to fail: no record file, no `docket/records`, no `docket`, and the error is reported. `TestRecordWriteFailsKeepsFolder`: when `docket/records` already existed, it stays.
  - `TestRecordNoCommits`: in a repository with no commit, `dkt record` exits 0 and writes the file.
  - `TestRecordOutsideRepo`: `no-repository`, exit 4, nothing written.
  - `TestShippedFileSet` gains `shipped/record/template.md`.
- [ ] **Step 3: Run them and see them fail**

Run: `go test . ./internal/record ./internal/cli`. Expected: undefined names, unknown command `record`.

- [ ] **Step 4: Create `shipped/record/template.md`** with exactly this text:

```markdown
<!-- Keep the title to one line and the rest short; `dkt lint` checks the
length. The details belong in the code and the pull request. Replace every
comment with your text. -->

## What

<!-- What changed, in a few sentences: the files or the behavior touched.
fix: the behavior restored. refactor: what moved or was restructured.
test: which behavior the new tests cover. prompt: which text changed, and
which copies. checks: which checks run differently. -->

## Why

<!-- Why it was needed. fix: the symptom and its cause. refactor: what the
old shape made hard. test: what was not covered. prompt: what agents did
wrong or could not do. checks: what the old settings missed. -->

## Verification

<!-- How it was checked and the result, by the light review in the docket
skill's default steps, or the user's own way: the regression test seen failing, then passing; the
mutation that made a test fail; the mechanical check of the prompt; the
pull request's own check run. Say what was not verified, too. -->

## Review

<!-- Who reviewed the diff (a model, or a person), what the review found,
and how each finding was handled. -->
```

- [ ] **Step 5: Implement `internal/record/record.go`, `runRecord` and the command**

`recordCommand` follows `newCommand`, with the usage `record <name> --kind <fix|refactor|test|prompt|checks> --source <text> --title <text> [--json]`, and joins the `commands` list. Order in `runRecord`: flags (`UsageError`), `record.CheckName` (`slug-invalid`), `safepath.Check` on the requested path (`refuse`), `scan.Records` (unreadable), then the candidates: skip one any use names; `safepath.Check`; make each missing folder with `os.Mkdir`, remembering those this run made; open with `O_WRONLY|O_CREATE|O_EXCL|syscall.O_NOFOLLOW`, going to the next suffix on `fs.ErrExist`. Write through the package variable `writeRecordFile`. On failure remove the file and the folders this run made, joining an error that names anything left. `cmd/dkt/main.go` needs no change: a nil `Now` means `time.Now()`.

- [ ] **Step 6: Run the tests and see them pass**

Run: `go vet ./... && go test ./...`. Expected: all pass.

- [ ] **Step 7: Commit** with `Change: add-dkt-record/3`.

### Task 4: `dkt lint` checks records

**Scenarios:** records#record-the-branch-adds, records#merged-record-left-alone, records#records-without-a-default-branch, records#named-record-linted, records#named-record-missing, records#record-kind-not-known, records#record-without-front-matter, records#record-title-too-long, records#record-section-empty, records#record-sections-out-of-order, records#other-heading-in-a-record, records#text-outside-the-sections, records#template-text-left-in-a-record, records#record-over-the-limit, records#chinese-text-at-the-limit, records#comments-and-tasks-not-counted, records#record-tasks-checked, records#record-without-tasks, records#task-heading-outside-tasks, change-lint#lint-every-active-change, change-lint#named-change-missing, change-lint#named-change-skips-main-specs, change-lint#named-change-skips-records, change-lint#clean-change, change-lint#records-in-the-result

**Review:** judgment

**Files:**
- Create: `internal/record/parse.go`, `internal/record/parse_test.go`, `internal/lint/record.go`, `internal/lint/record_test.go`
- Modify: `internal/lint/lint.go`, `internal/lint/plan.go`, `internal/lint/document.go`, `internal/cli/cmd_lint.go`
- Test: `internal/cli/cmd_lint_test.go`

**Interfaces:**
- Consumes: `change.HasDate`, `change.Size`, `change.TitleTooLong`, `record.Dir`, `record.Kinds`, `record.BodyLimit`.
- Produces: `record.Parse(b []byte) record.Parsed`, with `Parsed{Front change.Front; HasFront bool; FrontErr error; Sections []record.Section; Outside []int; Tasks change.Plan}`; `Section{Name string; Line int; Text string}`, the text with comments blanked; `Outside` the 1-based lines with text outside any section; `Tasks` parsed from the lines after `## Tasks`, the others blanked so that line numbers hold. `Parsed.BodySize() int`, the sum of `change.Size` over the four named sections. `lint.Input` gains `Records []string` (paths the branch adds; nil when skipped) and `RecordTemplate []byte`. `lint.Report` gains `Records []string`, sorted. `checkTaskHeadings(r *state, c *loaded, file string, p change.Plan)`, moved out of `checkPlan`. `lintResult.Records []string`, never null.

- [ ] **Step 1: Make the plan's task heading rules and the placeholder reader take a file and a template**, with no change in behavior, and commit it alone with `go test ./internal/lint` green.
- [ ] **Step 2: Write the failing tests**
  - `TestParse`: sections, lines and text; a `## x` inside a fence is not a section; a `# comment` line in the front matter is not a heading; `Outside` holds a paragraph before `## What` and text under a `# Title`; a comment before `## What` is not outside.
  - `TestParseTasks`: `### Task 1: Move` under `## What` is not in `Tasks`; one under `## Tasks` is, with its true line number.
  - `TestParseCRLFAndBOM`: a record with a BOM and CRLF endings parses to the same sections.
  - `TestBodySize`: a comment and `## Tasks` do not count; 480 Chinese characters split across the four sections give 1440.
  - `TestLintRecord...` (lint package, one per rule, through `Input.Records`): `record-kind` at the kind's line for `kind: chore` and for no kind; `record-front-matter` for no front matter, an unreadable one, no title, a blank source; `title-too-long` with `121` and `120` and no `record-front-matter`; `record-section` naming `Review` when it is empty, for a missing section, for `## Why` before `## What`, for `## Notes`, and on the line of a paragraph before `## What`; `placeholder` on the line of a comment left from the template; `record-too-long` with `1441` and `1440`, none at 1440 and none for the scenario with comments and tasks; `task-heading` naming both lines for two `### Task 1`; no `task-heading` without `## Tasks` and for the heading under `## What`. Every finding has `File` set and `Change` empty.
  - `TestLintRecordScope`: `Input.Records` entries that are not regular `.md` files directly under `docket/records` are not linted (`docket/records/sub/x.md`, `docket/records/x.txt`, a link); `Report.Records` is sorted.
  - `TestLintNamedRecord`, `TestLintNamedRecordMissing` (`record-missing` naming it, `Change` empty), `TestLintOnlyRecordsNamedSkipsChanges`: no change and no main spec title is linted.
  - CLI, in `cmd_lint_test.go`: `TestLintRecordTheBranchAdds` (uncommitted, unknown kind, exit 1); `TestLintMergedRecordLeftAlone`; `TestLintRecordsNoDefaultBranch` (`result.records` is `[]`, no finding); `TestLintNamedChangeSkipsRecords`; `TestLintResultRecords` (`["2026-11-06-fix-x"]`) and `result.records` as `[]` for the clean change.
  - `TestRecordRoundTrip`: `dkt record`, then `dkt lint` reports `placeholder` and `record-section`; with every comment replaced by a sentence, `dkt lint` reports nothing and exits 0.
  - Update the expected human lines in `cmd_lint_test.go` to `Linted 1 changes, 0 records and 0 main spec titles.` and the skip notes.
- [ ] **Step 3: Run them and see them fail**

Run: `go test ./internal/record ./internal/lint ./internal/cli`. Expected: undefined names and failing assertions.

- [ ] **Step 4: Implement**

In `lint.Run`, split `in.Slugs` by `change.HasDate`; with names and none for changes, lint no change. `lintRecord(r, slug, b)` builds `&loaded{Dir: record.Dir, Files: {slug + ".md": b}}`, so `r.add`, `checkPlaceholderFile` and `checkTaskHeadings` work with an empty `Change`. The list of records comes from one `fs.ReadDir(in.Root, record.Dir)`, keeping every regular `.md` file on `state`; `Input.Records` and the names only decide which of them are linted. A missing folder is empty, and any other error goes to `r.unreadable`, with no `record-missing` then. In `runLint`, with no names and a merge base, call `g.AddedSince(base, record.Dir)` separately from the routing call. Read `shipped/record/template.md` from `docket.Shipped`. The skip notes become `No default branch found; the routing rule and record checks were skipped.` and `No merge base with <name>; the routing rule and record checks were skipped.`

- [ ] **Step 5: Run the tests and see them pass**

Run: `go vet ./... && go test ./...`. Expected: all pass.

- [ ] **Step 6: Commit** with `Change: add-dkt-record/4`.

### Task 5: `dkt sync` on records, and the trailer check

**Scenarios:** records#record-tasks-ticked, records#bare-record-trailer-names-no-task, records#record-sync-check, records#record-sync-missing, records#record-sync-through-a-link, records#trailer-names-no-record, records#trailer-names-a-record, records#trailer-with-a-bad-task-list, records#trailer-names-an-earlier-record, records#merged-trailer-ignored

**Review:** judgment

**Files:**
- Modify: `internal/gitx/log.go`, `internal/cli/cmd_sync.go`, `internal/cli/cmd_inspect.go`, `internal/cli/cmd_lint.go`, `internal/lint/lint.go`, `internal/lint/record.go`
- Test: `internal/gitx/log_test.go`, `internal/cli/cmd_sync_test.go`, `internal/cli/cmd_lint_test.go`, `internal/lint/record_test.go`

**Interfaces:**
- Consumes: `record.Parse`, `record.Path`, `change.HasDate`, `change.TaskRefs`, `change.Tick`.
- Produces: `gitx.Trailers{SHA string; Values []string}`; `func (g Git) RecordTrailers(rev string) ([]Trailers, error)`; `lint.RecordTrailer{Commit, Value string}` and `lint.Input.RecordTrailers []lint.RecordTrailer`.

- [ ] **Step 1: Rename `Trailers.Change` to `Values`, and make the log reader take the key**, with `ChangeTrailers` keeping its signature. Move `runSync`'s body after reading the file into a function taking the file, the parsed tasks, the trailer key, the commits and the finding's `Change`, with no change in behavior. Commit this alone with `go test ./...` green.
- [ ] **Step 2: Write the failing tests**
  - `TestRecordTrailers`: a commit with `Record: a/1` in its last paragraph is read, and `Change: b` on it is not.
  - `TestSyncRecordTicks`: the scenario; a diff of the file shows only the two boxes; `result.slug` is the slug; findings have `Change` empty.
  - `TestSyncRecordCRLF`: a record with a BOM and CRLF keeps every other byte.
  - `TestSyncRecordBareTrailer`: nothing ticked, no finding, exit 0.
  - `TestSyncRecordCheck`: the file is unchanged, `task-unticked` names Task 2, exit 1.
  - `TestSyncRecordUnknownTask`: `Record: <slug>/9` reports `trailer-unknown-task`; `TestSyncRecordMalformed`: `Record: <slug>/x` reports `trailer-malformed`; both messages say `Record:`.
  - `TestSyncRecordIgnoresChangeTrailer`: `Change: <slug>/1` ticks nothing.
  - `TestSyncRecordHeadingOutsideTasks`: a commit for Task 1, with a `### Task 1` under `## What` and none under `## Tasks`, reports `trailer-unknown-task` and leaves the boxes under `## What`.
  - `TestSyncRecordMissing`, `TestSyncRecordLinked`: as the scenarios, the second with and without `--check`.
  - `TestSyncRecordNoCommits`: in a repository with no commit, an untracked record: exit 0, nothing ticked.
  - `TestLintRecordTrailer` (lint package): a `RecordTrailer` naming no file reports `record-trailer` with the commit's short SHA and the value; `2026-11-01-fix-x` and `2026-11-01-fix-x/2` report none when the file exists, also when the file is not in `Input.Records`; `2026-11-01-fix-x/x`, `2026-11-01-fix-x/` and `2026-11-01-fix-x/1/2` report it although the file exists; a value holding `..` reports it; a link at that path reports it.
  - CLI: `TestLintTrailerNamesNoRecord`, `TestLintTrailerNamesEarlierRecord` and `TestLintMergedTrailerIgnored`, as the scenarios; `TestLintNamedSkipsTrailers`: with a name, no `record-trailer`.
- [ ] **Step 3: Run them and see them fail**

Run: `go test ./internal/gitx ./internal/lint ./internal/cli`. Expected: undefined names, `change-missing` for a date slug.

- [ ] **Step 4: Implement**

`runSync` branches on `change.HasDate(slug)`: `safepath.Check`, `safefile.ReadNoFollow`, `record-missing` on `fs.ErrNotExist`, `record.Parse(src).Tasks`, the key `Record`, and `Change` empty in findings. Messages say the key in place of `Change`, and `<file> has no Task N` from the file's base name, so a change's messages stay as they are. In `runLint`, with no names and a merge base, pass `g.RecordTrailers(base + "..HEAD")`, one `RecordTrailer` per value. In lint, the check uses the whole listing from Task 4: the slug is the value before the first `/`, and a value that `change.TaskRefs(value, slug)` does not accept is reported too.

- [ ] **Step 5: Run the tests and see them pass**

Run: `go vet ./... && go test ./...`. Expected: all pass.

- [ ] **Step 6: Commit** with `Change: add-dkt-record/5`.

### Task 6: Skill text and this repository's instructions

**Scenarios:** none

**Review:** checks

**Files:**
- Modify: `shipped/skill/SKILL.md`, `.claude/skills/docket/SKILL.md`, `.agents/skills/docket/SKILL.md`, `AGENTS.md`

- [ ] **Step 1: Edit `shipped/skill/SKILL.md`**

Replace `in a few sentences each,` with `keeping them short,`. Replace `and fill its sections.` with `and fill its sections, then run `dkt lint` and fix what it reports.` Change nothing else: this text was settled by the maintainer.

- [ ] **Step 2: Update the installed copies**

Run: `go run ./cmd/dkt init --force`. The `dkt` on PATH is built from `main` and embeds the old text, so it must be this tree's. Expected: `git status --short` lists only the two copies besides the shipped file, and `diff shipped/skill/SKILL.md .claude/skills/docket/SKILL.md` and the same for `.agents` print nothing.

- [ ] **Step 3: Edit `AGENTS.md`**

Remove `dkt record` from the list of commands that do not exist yet, and remove the bullet that begins `Light review opens no record`.

- [ ] **Step 4: Check it mechanically**

Run: `rg -n 'few sentences each' shipped .claude/skills .agents/skills` (expected: nothing), `go test ./...` (expected: all pass) and `go run ./cmd/dkt lint add-dkt-record` (expected: no finding).

- [ ] **Step 5: Commit** with `Change: add-dkt-record/6`.
