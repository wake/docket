# Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ideas derive their state from the records and changes whose source names them; `docket/order.md` loses its checkboxes; `dkt lint` checks the order and the ideas; `dkt status` reports the order, the next piece of work and the ideas outside the order; the `docket` skill keeps ideas and the order out of a change's worktree.

**Architecture:** A source names the path in its first field (`change.SourcePath`), for issues and ideas alike. A new package `internal/idea` reads idea front matter and order lines; lint uses it on the worktree's files, status on the default branch tip's. Status finds handlers in one pass over records and changes, extracted from the issue code, and matches each source against the issue paths and the idea paths together.

**Tech Stack:** Go 1.26, `go.yaml.in/yaml/v3` (already a dependency), the standard library.

**Spec:** `openspec/changes/add-idea-state/proposal.md`, `design.md`, `specs/ideas/spec.md`, `specs/docket-metadata/spec.md`, `specs/issues/spec.md`, `specs/change-lint/spec.md`.

## Global Constraints

- Ideas are the regular files `docket/ideas/*.md` directly in that folder; the order is `docket/order.md`. Subfolders, other files and symbolic links under `docket/ideas/` are not ideas.
- Idea front matter keys: `title` (`change.ValidTitle`), optional `dropped` (only `true`), optional `needs` (a YAML list of strings, each a regular idea file other than the idea itself). Other keys are ignored.
- The order: lines split by `markdown.Lines`, each trimmed with `markdown.Trim`; the first non-blank line is exactly `# Order`; every other non-blank line matches ``^([0-9]+)\. `([^`]+)`$``; the first line is numbered 1 and each next line one more than the line before it.
- Lint rules: `order-format`, `order-idea`, `idea-front-matter`, `idea-source`. Order findings carry `File: docket/order.md` and the 1-based line; idea findings carry the idea's path and line 1; no `Change`.
- The path a source names: `change.SourcePath(s)` = the first of `strings.Fields(s)`, with one trailing `,` removed; `""` for a blank source.
- Idea states: `pending`, `in-progress`, `done`, `dropped`; an order line can also be `missing`.
- `dkt status --json`: `result.order` `[{number, path, title, state, handlers, waiting_on}]`, `result.next` (int or null), `result.ideas` `[{path, title, state, handlers}]`. Handlers are `{kind, slug, merged}` as for issues. Lists are never null; `title` is null when unreadable.
- Status reads ideas and the order from the default branch tip only; lint reads the worktree.
- Text lines, after the issue lines: `order <n> <state> <path>` + ` "<title>"` when known + ` (<handlers>)` when any; `idea <path> <state>` + title + handlers; `next <n> <path>`. Handlers print as the issue line prints them (`record 2026-11-02-fix-a`, `change add-x merged`).
- Exit codes as `cli-output`: 0 ok, 1 findings, 2 usage, 3 unreadable, 4 environment.
- No new dependency. CI runs jscpd with `--fail-on-new-clones=0`: share code instead of copying it.
- Commits carry `Change: add-idea-state` and `Task: <N>` in the last paragraph, with `Co-Authored-By`, no blank line between them; structure and behavior in separate commits.
- Public repository: English only in code, tests and docs (README.zh-TW.md excepted).

## Review Focus

- An order file written with CRLF line endings or trailing spaces still parses, line for line, with the same line numbers. Test in Task 2.
- An order line or a `needs` entry that names a path outside `docket/ideas/`, such as `README.md`, `docket/ideas/../README.md` or `docket/ideas/drafts/a.md`, is no idea: `order-idea` or `idea-front-matter`. Test in Task 2.
- An idea whose `needs` lists the idea itself is `idea-front-matter`. Test in Task 2.
- A source that is blank, or the path followed by a lone comma (`docket/ideas/x.md,`), names `""` and `docket/ideas/x.md` respectively; `see docket/ideas/x.md` names `see`. Test in Task 1.
- A tip with `docket/ideas/drafts/a.md`, `docket/ideas/README` and a symbolic link `docket/ideas/l.md` lists none of them as ideas, and an order line naming one is `missing`. Test in Task 3.

## Reuse

| Need | Existing (path:symbol) | Decision | Reason | Evidence |
|---|---|---|---|---|
| Handlers of a path: records and change copies, read from the tip when it holds them, else the first source | internal/status/issues.go:LoadIssues | extend | the pass moves into `findHandlers`, which matches against any set of paths, so issues and ideas share one read | `rg -n "readFrom|tipHeld" internal/status` |
| Matching a source against an issue path | internal/status/issues.go:LoadIssues | extend | `strings.TrimSpace(source)` becomes `change.SourcePath(source)` | `rg -n "TrimSpace\(source\)" internal/status` |
| The path a source names | internal/change/metadata.go:ReadMetadata | new | nothing reads a path out of a source; `ReadMetadata` returns the source as written. `SourcePath` sits beside it | `rg -n "SourcePath\|Fields\(.*[Ss]ource" internal` |
| A source that names no file under a folder | internal/lint/issue.go:checkIssueSource | extend | becomes `checkSources`, checking `docket/issues/` and `docket/ideas/` with their rules | `rg -n "checkIssueSource" internal/lint` |
| A path is a regular file of the worktree | internal/lint/issue.go:run.isFile | reuse | does not follow links; used for order lines, `needs` and idea sources | `rg -n "func \(r \*run\) isFile" internal/lint` |
| Regular files of a worktree folder | internal/lint/record.go:run.regularNames | reuse | lists `docket/ideas` and `docket` without following links | `rg -n "func \(r \*run\) regularNames" internal/lint` |
| Split front matter from the body | internal/record/parse.go:Split | reuse | ideas have the same `---` framing | `rg -n "func Split" internal/record` |
| Decode one YAML mapping, typed | internal/change/docket.go:DecodeMapping | reuse | a `needs` that is not a list fails to decode | `rg -n "func DecodeMapping" internal/change` |
| Title form | internal/change/docket.go:ValidTitle | reuse | same form as every title | `rg -n "func ValidTitle" internal/change` |
| Lines with CRLF, BOM; JavaScript-like trim | internal/markdown/markdown.go:Lines | reuse | with `markdown.Trim` for each line | `rg -n "func Lines\|func Trim" internal/markdown` |
| Files of the tip | internal/scan/scan.go:ReadFiles | reuse | `ListFiles` keeps regular blobs only, so links and subfolder files are filtered by the keep function and the mode | `rg -n "func ReadFiles\|100644" internal/scan internal/gitx` |
| The default branch tip as a source | internal/scan/scan.go:Scan | extend | its `dbSrc` literal becomes `DefaultBranch.Source()`, which status uses too | `rg -n "Kind: KindDefaultBranch" internal` |
| Handler type and JSON | internal/status/issues.go:Handler | reuse | ideas list the same `{kind, slug, merged}` | `rg -n "type Handler struct" internal/status` |
| States `pending` and `done` | internal/status/status.go:Pending, Done | reuse | same words; `InProgress` from issues.go too | `rg -n "Pending *=\|Done *=\|InProgress" internal/status` |
| The handler list in a text line | internal/cli/cmd_status.go:statusIssueLine | extend | its handler suffix becomes `handlerList`, used by the order and idea lines | `rg -n "func statusIssueLine" internal/cli` |
| Idea front matter with `needs` | internal/issue/issue.go:FrontMatter | new | the issue type has `source` and `idea`, not `needs`, and decoding into it would drop `needs` silently | `rg -n "type FrontMatter struct" internal` |
| Parse the order | internal/markdown/markdown.go:FindTable | new | nothing reads `docket/order.md`; the table helpers parse tables, not numbered lines | `rg -n "order\.md" internal` |

---

### Task 1: The path a source names

**Scenarios:** issues#handler-with-a-note, issues#source-with-a-note-names-no-issue, issues#source-names-no-issue, issues#source-names-an-issue, issues#handler-on-a-branch, issues#handler-merged, issues#merged-change-not-yet-archived, issues#merged-record-edited-elsewhere

**Review:** checks

**Files:**
- Modify: `internal/change/metadata.go`
- Modify: `internal/change/metadata_test.go`
- Modify: `internal/status/issues.go`
- Modify: `internal/status/issues_test.go`
- Modify: `internal/lint/issue.go`
- Modify: `internal/lint/issue_test.go`

**Interfaces:**
- Produces: `change.SourcePath(source string) string`.

- [ ] **Step 1: Write the failing tests.** `TestSourcePath` in `internal/change/metadata_test.go`, a table: `"docket/ideas/x.md"` → `"docket/ideas/x.md"`; `"  docket/ideas/x.md  "` → same; `"docket/ideas/x.md, item 1 of docket/order.md"` → same; `"docket/ideas/x.md (order 5)"` → same; `"docket/ideas/x.md,"` → same; `"see docket/ideas/x.md"` → `"see"`; `""` and `"   "` → `""`; `"a,,"` → `"a,"`. `TestHandlerWithANote` in `internal/status/issues_test.go`: the tip holds `issAF` and a record whose source is `issAF + ", found while fixing y"`; no issue is listed (handled). `TestSourceWithANoteNamesNoIssue` in `internal/lint/issue_test.go`: a record the branch adds with `source: docket/issues/2026-11-02-x.md, from review of #40` and no such file reports `issue-source` naming the whole source; the same source with the file present reports none.
- [ ] **Step 2: Run** `go test ./internal/change/ ./internal/status/ ./internal/lint/ -run 'TestSourcePath|TestHandlerWithANote|TestSourceWithANote'` — FAIL.
- [ ] **Step 3: Implement** `SourcePath` in `internal/change/metadata.go`; in `LoadIssues` match `named[change.SourcePath(source)]`; in `checkIssueSource` test `change.SourcePath(source)` against the folder and the file, and keep naming the trimmed source in the message.
- [ ] **Step 4: Run** `go test ./internal/change/ ./internal/status/ ./internal/lint/ ./internal/cli/` — PASS, `TestHandlerSourceMatching` unchanged.
- [ ] **Step 5: Commit** `feat(change): a source names the path in its first field`.

### Task 2: Lint the order and the ideas

**Scenarios:** ideas#idea-file-read, ideas#order-file-read, ideas#order-line-with-a-checkbox, ideas#gap-in-the-order, ideas#order-names-a-missing-idea, ideas#same-idea-twice, ideas#dropped-idea-in-the-order, ideas#linked-order-file, ideas#idea-without-a-title, ideas#need-names-no-idea, ideas#source-names-no-idea, change-lint#order-and-ideas-linted, change-lint#order-not-linted-with-names

**Review:** judgment

**Files:**
- Create: `internal/idea/idea.go`
- Create: `internal/idea/idea_test.go`
- Create: `internal/lint/idea.go`
- Create: `internal/lint/idea_test.go`
- Modify: `internal/lint/lint.go`
- Modify: `internal/lint/issue.go`
- Modify: `internal/lint/record.go`
- Modify: `docket/order.md`

**Interfaces:**
- Consumes: `change.SourcePath`.
- Produces in `internal/idea`: `const Dir = "docket/ideas"`, `const OrderFile = "docket/order.md"`; `type FrontMatter struct{ Title string; Dropped *bool; Needs []string }` with yaml keys `title`, `dropped`, `needs`; `func Parse(b []byte) (FrontMatter, error)` — `record.Split` then `change.DecodeMapping`; an invalid title reads as `""`; the error is the missing front matter or the decode error. `func IsIdeaPath(p string) bool` — `p` is `docket/ideas/<name>.md` with `<name>` free of `/`, and `p` is a clean path. `type Line struct{ Number int; Path string; At int }` (`At` is the 1-based line in the file); `type Problem struct{ At int; Message string }`; `func ParseOrder(b []byte) ([]Line, []Problem)` — lines of the form are returned even after a problem; a problem for the missing heading (at the first non-blank line, or 1), for each non-blank line not of the form, and for each line whose number is not one more than the line before it (the first must be 1).
- Produces in `internal/lint`: `checkSources(r *run, c *loaded, file, source string)` replacing `checkIssueSource`, and `(r *run) lintIdeas()` run when no name is given.

- [ ] **Step 1: Write the failing tests for the parser** in `internal/idea/idea_test.go`: `TestParseOrder` — `"# Order\n\n1. `docket/ideas/a.md`\n2. `docket/ideas/b.md`\n"` gives two lines `{1, "docket/ideas/a.md", 3}`, `{2, "docket/ideas/b.md", 4}` and no problem; the same text with `\r\n` and trailing spaces gives the same; ``3. [ ] `docket/ideas/c.md` `` as the third line is a problem at that line; numbers 1, 2, 4 give one problem at the line numbered 4; a first line numbered 2 is a problem; a file without `# Order` is a problem at its first non-blank line; a line `- later` is a problem and the lines around it are still returned. `TestParse` — `title: X` and `needs: [docket/ideas/y.md]` decode; `needs: docket/ideas/y.md` (a string) is an error; `dropped: yes` is an error; a title with a newline reads as `""`; no front matter is an error. `TestIsIdeaPath` — `docket/ideas/a.md` true; `README.md`, `docket/ideas/../README.md`, `docket/ideas/drafts/a.md`, `docket/ideas/a.txt`, `./docket/ideas/a.md` false.
- [ ] **Step 2: Run** `go test ./internal/idea/` — FAIL (undefined).
- [ ] **Step 3: Implement** `internal/idea/idea.go`.
- [ ] **Step 4: Run** `go test ./internal/idea/` — PASS.
- [ ] **Step 5: Write the failing lint tests** in `internal/lint/idea_test.go`, each with the worktree's files and `Input{}` with no slugs unless said: `TestIdeaFileRead` (two ideas, one needing the other, and an order naming both: no finding); `TestOrderLineWithACheckbox`, `TestGapInTheOrder`, `TestOrderWithoutHeading` (`order-format` at the line, `File` `docket/order.md`); `TestOrderNamesAMissingIdea`, `TestOrderNamesNoIdeaPath` (lines naming `README.md`, `docket/ideas/../README.md`, `docket/ideas/drafts/a.md` with those files present: `order-idea` for each), `TestSameIdeaTwice` (lines 2 and 5: one `order-idea` at line 5), `TestDroppedIdeaInTheOrder`; `TestLinkedOrderFile` (`docket/order.md` a symbolic link to a file outside the repository: one `order-format` naming `docket/order.md`, and the target's content produces no other finding); `TestIdeaWithoutATitle`, `TestNeedNamesNoIdea` (a missing idea, and `README.md` and `docket/ideas/drafts/a.md` with those files present), `TestNeedNamesItself`, `TestIdeaDroppedNotTrue`, `TestIdeaWithoutFrontMatter`, `TestIdeaFrontMatterNotAMapping` (a YAML list), `TestNeedsNotAList` (`needs: docket/ideas/a.md`) (each `idea-front-matter`, `File` the idea, line 1); `TestIdeaLinkSkipped` (a symbolic link under `docket/ideas/` gets no finding); `TestSourceNamesNoIdea` (change `add-x` with `source: docket/ideas/2026-11-02-none.md, item 4`, `Slugs: ["add-x"]`: `idea-source`; a record the branch adds with such a source: `idea-source`); `TestOrderNotLintedWithNames` (`Slugs: ["add-x"]` and a gap in the order: no `order-format`); `TestNoOrderNoIdeas` (neither exists: no finding, no unreadable).
- [ ] **Step 6: Run** `go test ./internal/lint/ -run 'Order|Idea|Need|SourceNamesNoIdea|NoOrder'` — FAIL.
- [ ] **Step 7: Implement** `internal/lint/idea.go`: `lintIdeas` reads `docket/ideas` with `regularNames`, parses each regular `*.md` (front matter checks as the Interfaces say, `needs` entries checked with `idea.IsIdeaPath` and `run.isFile`, and not equal to the idea's own path), then the order: `order-format` when `docket` lists `order.md` as not regular, else `ParseOrder` problems as `order-format`, and for each line `order-idea` when `!idea.IsIdeaPath(path) || !r.isFile(path, idea.Dir+"/")`, when an earlier line named the same path, or when the idea parsed with `dropped: true`. Unreadable files go to `r.unreadable`. Call it from `Run` beside `lintIssues`. Rename `checkIssueSource` to `checkSources` and add the `docket/ideas/` check with `idea-source` and the message `source %q names no idea file`.
- [ ] **Step 8: Rewrite `docket/order.md`** without checkboxes: ``1. `docket/ideas/2026-10-29-document-titles.md` `` through line 14, same order.
- [ ] **Step 9: Run** `go test ./internal/idea/ ./internal/lint/ ./internal/cli/` — PASS; `go build -o "$S/dkt-idea" ./cmd/dkt && "$S/dkt-idea" lint` in the worktree (`$S` the session scratchpad) — `dkt lint: ok`.
- [ ] **Step 10: Commit** `feat(lint): check the order and the ideas`.

### Task 3: Status reports the order and the ideas

**Scenarios:** ideas#idea-pending, ideas#handler-on-a-branch, ideas#every-handler-merged, ideas#one-handler-of-two-merged, ideas#dropped-idea, ideas#idea-on-a-branch-only, ideas#order-listed, ideas#next-skips-done-dropped-and-missing, ideas#waiting-on-a-need, ideas#no-order-file, ideas#unparsed-order-line, ideas#ideas-outside-the-order, ideas#order-not-listed-with-slugs, ideas#order-in-the-text-output, docket-metadata#path-alone, docket-metadata#path-with-a-note

**Review:** judgment

**Files:**
- Create: `internal/status/handlers.go`
- Create: `internal/status/ideas.go`
- Create: `internal/status/ideas_test.go`
- Modify: `internal/status/issues.go`
- Modify: `internal/status/issues_test.go`
- Modify: `internal/scan/scan.go`
- Modify: `internal/scan/defaultbranch.go`
- Modify: `internal/cli/cmd_status.go`
- Create: `internal/cli/cmd_status_ideas_test.go`

**Interfaces:**
- Consumes: `idea.Dir`, `idea.OrderFile`, `idea.Parse`, `idea.IsIdeaPath`, `idea.ParseOrder`, `change.SourcePath`.
- Produces: `func (db DefaultBranch) Source() Source` in `internal/scan`, used by `Scan` for its tip source.
- Produces in `internal/status`: `func findHandlers(g gitx.Git, inv scan.Inventory, paths map[string]bool) (map[string][]Handler, []scan.Unreadable, error)` — the record and change pass of `LoadIssues`, keyed by the path each source names, each list sorted by kind then slug, one handler per kind and slug; it reads nothing when `paths` is empty. `type Backlog struct{ Issues []Issue; Order []OrderLine; Next *int; Ideas []Idea }`; `func LoadBacklog(g gitx.Git, inv scan.Inventory) (Backlog, []scan.Unreadable, error)` replacing `LoadIssues`; `type OrderLine struct{ Number int; Path string; Title *string; State string; Handlers []Handler; WaitingOn []string }` with json keys `number`, `path`, `title`, `state`, `handlers`, `waiting_on`; `type Idea struct{ Path string; Title *string; State string; Handlers []Handler }` with json keys `path`, `title`, `state`, `handlers`; `const Dropped = "dropped"`, `const Missing = "missing"`.
- Produces in `internal/cli`: `StatusResult` gains `Order []status.OrderLine` (`order`), `Next *int` (`next`), `Ideas []status.Idea` (`ideas`); `func handlerList(hs []status.Handler) string` (`""` or ` (record a, change b merged)`).

- [ ] **Step 1: Structure commit.** Move the record and change pass of `LoadIssues` into `findHandlers` in `internal/status/handlers.go`; rename `LoadIssues` to `LoadBacklog`, returning `Backlog{Issues: ...}` with `Order`, `Ideas` empty and `Next` nil; add `DefaultBranch.Source()` and use it in `Scan`; extract `handlerList` from `statusIssueLine`. Update `cmd_status.go` and the `loadIssues` test helper. Run `go test ./internal/status/ ./internal/scan/ ./internal/cli/` — PASS. Commit `refactor(status): find handlers for any set of paths`.
- [ ] **Step 2: Write the failing tests** in `internal/status/ideas_test.go`, with a helper `loadBacklog(t, r)` as `loadIssues` is: `TestIdeaPending` (tip holds an idea not in an order: `Ideas` lists it `pending`); `TestIdeaHandlerOnABranch` (an unmerged branch holds `add-x` with the idea's path as source: `in-progress`, handler `{change, add-x, false}`); `TestEveryHandlerMerged` (tip holds a record naming the path alone and an archived change naming it followed by `, second part`: `done`, so not in `Ideas`; with the idea in the order, the line is `done` with both handlers merged — this also covers `docket-metadata#path-alone` and `#path-with-a-note`, with a third source `docket/ideas/x.md (order 5)` on another archived change); `TestOneHandlerOfTwoMerged` (tip record plus an uncommitted change in worktree B: `in-progress`); `TestDroppedIdea` (`dropped: true` with a merged record: `dropped`, not in `Ideas`); `TestIdeaOnABranchOnly` (a branch adds the idea, the tip does not: neither `Ideas` nor any order line has it); `TestOrderListed` (line 1 done, line 2 in progress: states, handlers, `Next` 2); `TestNextSkipsDoneDroppedAndMissing` (lines 1 done, 2 dropped, 3 no such idea, 4 pending: line 3 `missing` with nil title, `Next` 4); `TestWaitingOnANeed` (line 1's idea needs `b`, `b` pending and not in the order: `WaitingOn` `["docket/ideas/2026-11-01-b.md"]`; once `b` is done, `WaitingOn` is empty); `TestNoOrderFile` (no `docket/order.md`: `Order` empty, `Next` nil, no unreadable); `TestUnparsedOrderLine` (`1.` line and `- later`: one line); `TestIdeasOutsideTheOrder` (ideas a in the order, b pending, c done, d dropped: `Ideas` is b only, sorted by path); `TestNotIdeas` (tip holds `docket/ideas/drafts/a.md`, `docket/ideas/README`, a symbolic link `docket/ideas/l.md`, and an order naming `docket/ideas/l.md`: no idea listed, the line `missing`); `TestIssuesAndIdeasTogether` (a record names an issue and a change names an idea: one `LoadBacklog` gives the issue its record and the idea its change); `TestIdeaFrontMatterUnreadable` (the tip's idea has no front matter and is named by line 1: the line is listed with a nil title, state `pending`, empty `WaitingOn`, and nothing is unreadable).
- [ ] **Step 3: Write the failing CLI tests** in `internal/cli/cmd_status_ideas_test.go`: `TestStatusOrderListed` (JSON `result.order`, `result.next` 2, `result.ideas` `[]`, and the text holds `order 2 in-progress docket/ideas/2026-11-02-b.md "B" (change add-x)\n` and `next 2 docket/ideas/2026-11-02-b.md\n` and no `order 1 `); `TestStatusIdeaInProgressOutsideOrder` (text `idea docket/ideas/2026-11-02-c.md in-progress "C" (record 2026-11-02-fix-c)\n`; a pending idea outside the order has no text line but is in `result.ideas`); `TestStatusOrderNotListedWithSlugs` (`dkt status add-x --json`: `order` and `ideas` are `[]`, `next` is null); `TestStatusNoOrder` (`order` `[]`, `next` null, exit 0).
- [ ] **Step 4: Run** `go test ./internal/status/ ./internal/cli/ -run 'Idea|Order|Handler|Next|Waiting|NotIdeas|OneRead'` — FAIL.
- [ ] **Step 5: Implement** `internal/status/ideas.go` and the rest of `LoadBacklog`: read `docket/order.md` and the ideas from `inv.DefaultBranch.Source()` with one `scan.ReadFiles(g, tip, "docket", keep)` where `keep` takes `order.md` and the names `idea.IsIdeaPath("docket/"+rel)` accepts; call `findHandlers` once with the issue paths and the idea paths; state per the spec (`dropped`; else `pending` with no handler, `in-progress` when a handler is not merged, `done` otherwise); order lines from `ParseOrder`, each with its idea's title, state (`missing` when the tip has no such idea), handlers and `WaitingOn` (each `needs` path whose idea is not `done`, a path with no idea included); `Next` the first line `pending` or `in-progress`; `Ideas` the ideas no line names that are `pending` or `in-progress`, sorted by path. An idea whose front matter does not parse is still an idea: nil title, not dropped, no `needs`. Lists are `[]`, never nil. In `cmd_status.go`, without slugs, call `LoadBacklog`, fill the result, and append the text lines after the issue lines: one per order line not `done`, one per `in-progress` idea in `Ideas`, then the next line.
- [ ] **Step 6: Run** `go test ./internal/status/ ./internal/scan/ ./internal/cli/` — PASS; `go build -o "$S/dkt-idea" ./cmd/dkt && "$S/dkt-idea" status --json` in the worktree — lines 1 to 9 `done`, line 10 `in-progress` with `change add-idea-state`, `next` 10.
- [ ] **Step 7: Commit** `feat(status): report the order, the next work and the ideas`.

### Task 4: The skill, AGENTS.md and the docs

**Scenarios:** ideas#idea-found-during-a-change, ideas#change-takes-an-idea, ideas#idea-with-a-further-part, ideas#agent-picks-the-next-work

**Review:** checks

**Files:**
- Modify: `.claude/skills/docket/SKILL.md`
- Modify: `.agents/skills/docket/SKILL.md`
- Modify: `AGENTS.md`
- Modify: `README.md`
- Modify: `README.zh-TW.md`
- Modify: `docs/ARCHITECTURE.md`

**Interfaces:**
- Consumes: the commands and fields of Tasks 2 and 3.

- [ ] **Step 1: The `docket` skill**, both copies identical. Under the triage, after "An idea that is not ready goes to `docket/ideas/`, one file per idea.", add the rules of the four requirements: a change's or record's worktree leaves `docket/order.md` and `docket/ideas/` unchanged, and an idea found during the work or an edit to the order is a direct merge of its own; work that takes an idea passes the idea's path alone as `--source`, as for an issue; a part an idea's work leaves for later becomes a new idea whose `needs` names the first; the next piece of work is the idea of the line `result.next` of `dkt status --json` names. Add the idea case to the two `--source` sentences under Full review and Light review that now name only the issue.
- [ ] **Step 2: AGENTS.md:** replace "the first unticked line is the next piece of work" with the next piece of work being the line `dkt status` reports as next. Leave the Docket block unchanged.
- [ ] **Step 3: README.md and README.zh-TW.md:** a paragraph beside the issue paragraph: ideas and `docket/order.md` (the line format, no checkboxes), their states, what `dkt lint` checks, `result.order`, `result.next`, `result.ideas`, and that only the default branch's copies count. **docs/ARCHITECTURE.md:** rows for `internal/idea` (`Parse`, `IsIdeaPath`, `ParseOrder`), `change.SourcePath`, `status.LoadBacklog` and `status.findHandlers` replacing `LoadIssues`, `scan.DefaultBranch.Source`.
- [ ] **Step 4: Verify:** `diff .claude/skills/docket/SKILL.md .agents/skills/docket/SKILL.md` — no output; `go test ./...` — PASS (the setup tests read the shipped skill); `dkt lint` — ok; each of the four scenarios has a sentence in the skill that a reader can point to.
- [ ] **Step 5: Commit** `docs(skill): ideas and the order have one writer`.
