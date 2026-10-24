# add-lint Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `dkt lint [<slug>...]`, which reports every broken structural rule of the change folders in the current worktree, with the Markdown, delta, proposal and plan readers that later changes reuse.

**Architecture:** Three readers that take bytes and return values: `internal/markdown` (lines, fenced-code masks, headings, pipe tables, comments, code spans), `internal/delta` (delta and main specs, ported from OpenSpec 1.14.3) and two new files in `internal/change` (proposal.md and plan.md). `internal/lint` runs the rules over an `fs.FS` of the worktree and returns a report; `internal/cli` adds the `lint` command, which computes the files the branch adds under `docs/superpowers/` with two new `gitx` methods and turns the report into the usual envelope.

**Tech Stack:** Go 1.24 (module floor), standard library only; `git` as an external program; OpenSpec 1.14.3 only in the contract test; GitHub Actions.

**Spec:** `openspec/changes/add-lint/proposal.md`, `design.md` and `specs/change-lint/spec.md`. Read all three; scenario IDs below refer to the delta spec. The delta rules are ported from the npm package `@fission-ai/openspec@1.14.3`, files `dist/core/parsers/code-fence.js`, `requirement-blocks.js`, `requirement-text.js`, `markdown-parser.js` and `change-parser.js`; where this plan and that source disagree, the source wins and the contract test decides.

## Global Constraints

- Module path `github.com/wake/docket`; `go 1.24` in go.mod; no new third-party dependency.
- Public repository: code, comments, messages and docs in English; no hostnames, customer names, personal paths or secrets.
- `dkt lint` writes nothing, never contacts a network service and needs no OpenSpec CLI at run time.
- Every command accepts `--json` and `--default-branch <name>`, before or after positional arguments.
- Exit codes: 0 clean; 1 problems found or request refused; 2 usage error; 3 a needed source could not be read (wins over 1); 4 unusable environment.
- With `--json`, standard output is exactly one JSON object with `tool`, `version`, `command`, `repo`, `ok`, `findings`, `unreadable`, `result`; `findings` and `unreadable` are arrays, never null. Each finding carries `rule` and `message`, and `change`, `file` and `line` when they apply.
- Readers take bytes, not paths. Only `internal/lint` reads the worktree, and only through the `fs.FS` it is given; only `internal/gitx` starts git; nothing reads `os.Getenv` or `os.Environ` below `cmd/dkt` (tests excepted).
- Line numbers are 1-based and count lines after a leading BOM is removed and CRLF or lone CR is turned into LF. `file` is relative to the repository root, slash-separated.
- macOS and Linux only.
- `go test ./...` passes without network access and without an OpenSpec install; the contract test runs only when `DOCKET_TEST_OPENSPEC` is set.
- Every commit carries `--trailer "Change: add-lint" --trailer "Task: <N>"` and `--trailer "Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"` (use the model that actually wrote the commit).

## Review Focus

1. This repository's own change folders must lint clean: `**Scenarios:**` references wrapped in backticks, Existing cells such as `` `AGENTS.md`: "Change workflow (superpowers × OpenSpec)" ``, Evidence cells with escaped `\|`, and `external:` rows written with backticks. Unit tests in Tasks 4 and 6; the whole repository in Task 9.
2. CRLF line endings and a BOM in proposal.md and plan.md, not only in deltas: the same findings on the same line numbers. Tests in Tasks 1 and 5.
3. Scenario names in other scripts or with punctuation (`需求 R1#3`, `Join (when full)`): slugs follow the rule, and reference checks and duplicate checks use them. Tests in Tasks 2 and 7.
4. Fenced code blocks in proposal.md and plan.md holding `### Task 9: x`, `**Scenarios:** none`, `## Reuse` or a template comment: no rule reads them. Tests in Tasks 4, 5 and 6.
5. A change folder, or one file in it, that cannot be read (mode 000): listed as unreadable, every other change still linted, exit 3. Test in Task 5 (skipped as root).

## Reuse

| Need | Existing (path:symbol) | Decision | Reason | Evidence |
|---|---|---|---|---|
| Line endings, BOM and fenced-code masks | none | new | No Go code reads Markdown yet; OpenSpec's fence rule is ported from `code-fence.js` | `rg -n "Fence\|BOM\|CRLF" --type go` → none |
| Headings, pipe tables, comments and code spans in proposal.md and plan.md | none | new | Nothing parses Markdown structure; a CommonMark library would disagree with OpenSpec's line-based reading (design.md, Non-Goals) | `rg -n "func .*(Table\|Heading)" --type go` → none |
| Reading deltas and main specs as OpenSpec 1.14 does | none | new | Ported rather than called so that `dkt lint` needs no OpenSpec at run time (proposal, Builds On) | `rg -n "Requirement:" --type go` → none outside tests |
| Scenario slugs from names in any script | `internal/change/new.go:ValidateSlug` | new | `ValidateSlug` checks a change slug (ASCII kebab-case); nothing derives a slug from a name | `rg -n "func .*Slug" --type go -g '!*_test.go'` → `change.ValidateSlug` only |
| Reading a change folder confined to the worktree | external: Go `os.Root` and `Root.FS` | reuse | Walks `openspec/changes/` and `specs/`; `scan.ReadFile` reads one file per call and cannot list a directory | `rg -n "OpenRoot" --type go` → `internal/scan/scan.go:ReadFile` |
| The placeholder strings | `schema.go:Schema` | reuse | The shipped templates are embedded already | `rg -n "func Schema" schema.go` |
| The default branch for the routing check | `internal/scan/defaultbranch.go:ResolveDefaultBranch` | reuse | Same resolution as every other command | `rg -n "func ResolveDefaultBranch" internal/scan` |
| The merge base for the routing check | `internal/gitx/git.go:Git.MergeBase` | reuse | Already returns ok false for unrelated histories | `rg -n "func \(g Git\) MergeBase" internal/gitx` |
| Files the branch adds, committed and uncommitted | `internal/gitx/git.go:Git` | extend | Two methods on the only package that starts git | `rg -n "diff-filter\|status --porcelain" internal/` → none |
| Findings, unreadable sources, exit codes, envelope | `internal/cli/envelope.go:Outcome` | reuse | The shared result model of every command | `rg -n "type Outcome" internal/cli` |
| Registering the command | `internal/cli/run.go:commands` | extend | Add `lintCommand` | `rg -n "var commands" internal/cli` |
| Test repositories | `internal/testutil/repo.go:NewRepo` | reuse | Temporary repositories with a controlled environment | `rg -n "func NewRepo" internal/testutil` |
| Running OpenSpec in the contract test | `internal/openspec/runner.go:New` | reuse | Applies the version gate and the given environment | `rg -n "func New" internal/openspec` |
| A CI job with Node.js and OpenSpec 1.14.3 | `.github/workflows/ci.yml` | reuse | The `openspec` job already runs `go test ./...` with `DOCKET_TEST_OPENSPEC` | `rg -n "DOCKET_TEST_OPENSPEC" .github/workflows/ci.yml` |
| Running the lint rules | none | new | No rule engine exists | `rg -n "package lint" --type go` → none |

## Revision 1 (2026-10-24): final review fixes

The whole-branch review found a false `scenario-duplicate` when a delta renames and modifies one requirement, and comment handling that differed between proposals and deltas and ran over fenced text. `markdown.StripComments` (Task 1) is replaced by `markdown.BlankComments(lines, mask)`, which leaves fenced lines alone, closes a comment on `-->` or `--!>` and keeps line count and offsets; `change` and `delta` both use it. The duplicate check also drops the main spec's requirements that the delta renames. `Git.UncommittedAdded` skips files added and then deleted in the worktree, a Reuse path may end in `/`, and `lint.RoutedDirs` is the one list of routed folders. No spec sentence changes; no task is added.

---

### Task 1: Markdown lines, fences, headings and tables

**Scenarios:** none

**Files:**
- Create: `internal/markdown/markdown.go`, `internal/markdown/table.go`
- Test: `internal/markdown/markdown_test.go`, `internal/markdown/table_test.go`

**Interfaces:**
- Produces (package `markdown`; indexes are 0-based line indexes, a line number is index + 1):
  - `func Lines(b []byte) []string` — removes one leading UTF-8 BOM, turns CRLF and lone CR into LF, splits on LF.
  - `func FenceMask(lines []string) []bool` — true for every line of a fenced code block, both fence lines included. A fence opens on a line of optional whitespace followed by three or more backticks or three or more tildes, and closes on a line holding only optional whitespace, a run of the same character at least as long, and optional whitespace; an unclosed fence runs to the end.
  - `type Heading struct { Level int; Text string; Index int }`
  - `func Headings(lines []string, mask []bool) []Heading` — every unmasked line matching `^(#{1,6})\s+(.+)$`; `Text` is trimmed, with a closing run `[ \t]+#+[ \t]*$` removed.
  - `func Body(lines []string, mask []bool, h Heading) (from, to int)` — the lines after `h` up to, not including, the next unmasked heading of level `h.Level` or less.
  - `type Row struct { Cells []string; Index int }`, `type Table struct { Header []string; Rows []Row; Index int }`
  - `func FindTable(lines []string, mask []bool, from, to int) (Table, bool)` — the first table in `lines[from:to]` outside the mask: a line containing `|`, then a delimiter line matching `^\s*\|?\s*:?-+:?\s*(\|\s*:?-+:?\s*)*\|?\s*$`, then every following line that contains `|` and is not blank. Cells split on `|` not preceded by `\`; one leading and one trailing pipe are optional; cells are trimmed and `\|` becomes `|`; a row is padded with empty cells to the header's length.
  - `func StripComments(s string) string` — removes every `<!--` … `-->`; an unclosed comment runs to the end.
  - `func CodeSpans(line string) [][2]int` — byte ranges `[start, end)` of inline code spans on one line, delimiters included: a run of N backticks opens and the next run of exactly N backticks closes; an unmatched run is literal text.

- [x] **Step 1: Write the failing tests:** `TestLines` (`"﻿a\r\nb\rc\n"` → `["a","b","c",""]`); `TestFenceMask` (backticks, tildes, indented fence, a longer closing fence closes, a shorter or different-character one does not, unclosed runs to the end); `TestHeadings` (`## Why`, `### Task 1: x ###` → text `Task 1: x`, `#NoSpace` is not a heading, a heading inside a fence is skipped); `TestBody`; `TestFindTable` (header, delimiter with alignment colons, rows; `\|` inside a cell; a row with fewer cells; a table inside a fence is not found; a header with no delimiter is not a table); `TestStripComments` (multi-line, unclosed); `TestCodeSpans` (`` a `b` c ``, ``` ``x`y`` ```, unmatched backtick).
- [x] **Step 2: Run** `go test ./internal/markdown/` — expected: FAIL (package does not build).
- [x] **Step 3: Implement** the functions above.
- [x] **Step 4: Run** `go test ./internal/markdown/` — expected: PASS.
- [x] **Step 5: Commit** `feat(markdown): read lines, fences, headings and tables`.

### Task 2: Delta and main spec reader

**Scenarios:** `change-lint#requirement-outside-any-delta-section`, `change-lint#header-inside-a-code-block`, `change-lint#unpaired-rename`

**Files:**
- Create: `internal/delta/delta.go`, `internal/delta/slug.go`
- Test: `internal/delta/delta_test.go`, `internal/delta/slug_test.go`

**Interfaces:**
- Consumes: `markdown.Lines`, `markdown.FenceMask` (Task 1).
- Produces (package `delta`; `Line` fields are 1-based):
  - `type Scenario struct { Name string; Line int }`
  - `type Requirement struct { Name string; Line int; Scenarios []Scenario }`
  - `type Rename struct { From, To string; Line int }` — `Line` is the FROM line.
  - `type Problem struct { Line int; Message string }`
  - `type Delta struct { Added, Modified, Removed []Requirement; Renamed []Rename; Problems []Problem }` — each list in document order; `Removed` entries carry no scenarios.
  - `func Parse(b []byte) Delta`
  - `func ParseSpec(b []byte) []Requirement` — a main spec: the requirements in its first unmasked `^##\s+Requirements\s*$` section (case-insensitive), with the same block and scenario rules as `Parse`.
  - `func Slug(name string) string` — `strings.ToLower`, then every run of characters that are neither `unicode.IsLetter` nor `unicode.IsDigit` becomes one `-`, then leading and trailing `-` are trimmed.
- `Parse` ports `parseDeltaSpec` and the `show` path (`ChangeParser.toRequirements`, `MarkdownParser.parseScenarios`):
  - Sections are unmasked lines matching `^(##)\s+(.+)$`, title trimmed; `ADDED Requirements`, `MODIFIED Requirements`, `REMOVED Requirements` and `RENAMED Requirements` compare lower-cased and exactly, and a repeated section applies in full, in order.
  - A requirement header is an unmasked line matching `(?i)^###\s*Requirement:\s*(.+)\s*$`; its name is the capture with `[ \t]+#+[ \t]*$` removed, then trimmed; names keep their case. A block runs to the next requirement header or `## ` line.
  - Scenarios are the headings directly under the requirement heading in its block, as OpenSpec's section tree builds them (level 4 in practice), named by removing a closing `[ \t]+#+[ \t]*$` run and a leading `(?i)Scenario:\s*`, then trimming. A scenario whose body (up to the next unmasked heading of its level or above) is empty after trimming is dropped. A stray `###` heading inside a block ends that requirement's scenarios.
  - REMOVED names come from requirement headers and from bullet lines (`-`, `*` or `+`) holding a requirement header, optionally wrapped in one backtick on each side, in document order (`parseRemovedNames`).
  - RENAMED pairs come per section from `FROM:` and `TO:` lines, each with an optional bullet (`-`, `*` or `+`) and a requirement header optionally wrapped in backticks (`parseRenamedPairs`). A FROM followed by a TO with no FROM in between is a pair; a FROM displaced by another FROM, a TO with no pending FROM, and a FROM still pending at the end of the section are problems.
  - `Problems`, one per occurrence, with messages naming the case: a requirement header outside the four delta sections, above the first `## ` line included; a `## ` section other than the four and `Purpose`; text, other than comments and `###` lines, before the first requirement of an ADDED or MODIFIED section; text, other than `# ` titles and comments, above the first `## ` line; each unpaired FROM or TO; and, on line 1, a file with none of the four sections.

- [x] **Step 1: Write the failing tests:** `TestParseAdded` (two requirements with scenarios; names and lines); `TestScenarioRules` (`#### Edge case` is a scenario, `#### Scenario: Changed ####` → `Changed`, an empty `#### Scenario: Empty` is dropped); `TestRequirementHeaderForms` (`###Requirement: X`, `### requirement: X`, `### Requirement: First ###` → `First`, `### Requirement: C#` keeps `C#`); `TestHeaderInsideACodeBlock` (fenced `### Requirement:` and `#### Scenario:` lines inside a requirement are neither read nor reported); `TestRemovedForms` (header, `-`, `*` and `+` bullets with and without backticks); `TestRenamedPairs` (pairs with `-`, `*`, `+` and no bullet); `TestUnpairedRename` (FROM with no TO → a problem on the FROM line; TO with no FROM; FROM, FROM, TO → the first FROM is a problem); `TestRequirementOutsideAnyDeltaSection` (`### Requirement: Cap` under `## Notes` → problems on the requirement line and on the `## Notes` line); `TestRepeatedSection`; `TestCaseOfSectionTitles` (`## added requirements` applies, `## ADDED  Requirements` with two spaces does not and is a problem); `TestNoDeltaSection` (problem on line 1); `TestCRLFAndBOM` (same result and line numbers as the LF file); `TestParseSpec` (a main spec with `## Purpose` and `## Requirements`); `TestSlug` (`Join when full` → `join-when-full`, `Join (when full)` → `join-when-full`, `需求 R1#3` → `需求-r1-3`, `  --A--  ` → `a`).
- [x] **Step 2: Run** `go test ./internal/delta/` — expected: FAIL.
- [x] **Step 3: Implement** `Parse`, `ParseSpec` and `Slug`.
- [x] **Step 4: Run** `go test ./internal/delta/` — expected: PASS.
- [x] **Step 5: Commit** `feat(delta): read deltas and main specs as OpenSpec 1.14 does`.

### Task 3: Contract test against OpenSpec

**Scenarios:** `change-lint#examples-agree-with-openspec`

**Files:**
- Create: `internal/delta/testdata/deltas/README.md`, one directory per example under `internal/delta/testdata/deltas/`
- Test: `internal/delta/contract_test.go`

**Interfaces:**
- Consumes: `delta.Parse` (Task 2); `openspec.New`, `Runner.Run` (from `add-init-and-new`); `testutil.Env`.
- Each example directory is a change: a `proposal.md` with only `## Why` and `## What Changes`, and `specs/<capability-path>/spec.md` files. Examples: `basic` (two ADDED requirements); `every-operation` (ADDED, MODIFIED, REMOVED in header and `*` bullet form, RENAMED with `-`, `*` and `+`, with the sections out of the usual order — RENAMED first, REMOVED before MODIFIED — and MODIFIED twice; OpenSpec 1.14.3 reports each file grouped by operation in the order ADDED, MODIFIED, REMOVED, RENAMED, which this example pins); `fenced` (fenced requirement and scenario headers inside requirement text); `odd-forms` (`###Requirement:`, `### requirement:`, closing `###`, `#### Edge case`, an empty scenario, a repeated ADDED section, lower-case section titles); `nested` (`specs/area/cap/spec.md`). The README says what each example covers and credits OpenSpec (MIT) for any case adapted from its tests.
- `TestDeltasAgreeWithOpenSpec` skips unless `DOCKET_TEST_OPENSPEC` is set. For each example, and for CRLF and BOM copies of `basic` that the test writes itself (git would normalize committed line endings), it copies the change to `<tmp>/openspec/changes/<name>/`, runs `show <name> --type change --json --deltas-only` through `openspec.New(<tmp>, append(testutil.Env(t), "DOCKET_OPENSPEC="+value))`, and compares, per capability, the sequence of (operation, requirement name, scenario names) for ADDED, MODIFIED and REMOVED and (from, to) for RENAMED with Docket's reading, which lists each file's Added, Modified, Removed and Renamed in that order. The JSON fields are `deltas[].spec`, `.operation`, `.requirement.name`, `.requirement.scenarios[].name` and `.rename.from` / `.rename.to`.

- [x] **Step 1: Write the examples and the test.**
- [x] **Step 2: Run** `DOCKET_TEST_OPENSPEC="npx -y @fission-ai/openspec@1.14.3" go test -run TestDeltasAgreeWithOpenSpec -v ./internal/delta/` — expected: PASS; any mismatch is fixed in `delta.Parse` (Task 2's tests gain the case), never by editing the example to avoid it.
- [x] **Step 3: Run** `go test ./internal/delta/` without the variable — expected: PASS with the contract test skipped.
- [x] **Step 4: Commit** `test(delta): compare the delta reader with OpenSpec 1.14.3`.

### Task 4: Proposal and plan readers

**Scenarios:** none

**Files:**
- Create: `internal/change/proposal.go`, `internal/change/plan.go`
- Test: `internal/change/proposal_test.go`, `internal/change/plan_test.go`

**Interfaces:**
- Consumes: package `markdown` (Task 1).
- Produces (package `change`; `Line` fields are 1-based, 0 when absent):
  - `type Section struct { Title string; Line int; Body string }` — `Body` is the section's text up to the next heading of its level or above, with comments removed and surrounding whitespace trimmed.
  - `type Proposal struct { Sections []Section; Contract []Section; ChangeLog *markdown.Table }` — `Sections` are the level-2 sections in order; `Contract` the level-3 sections inside `## Implementation Contract`; `ChangeLog` the first table under `## Change Log`, nil when there is none.
  - `func ParseProposal(b []byte) Proposal`
  - `func (p Proposal) Section(title string) (Section, bool)` — compares titles without regard to case.
  - `type Task struct { Number int; Name string; Line int; Scenarios []string; ScenariosLine int; Compensates []Ref }` — `Scenarios` holds the items of the task's `**Scenarios:**` line (split on `,`, trimmed, surrounding backticks removed; empty for `none`); `ScenariosLine` is 0 when the line is missing.
  - `type Ref struct { Number int; Line int; OK bool }` — one `**Compensates:**` line; `OK` is false when the line is not exactly `**Compensates:** Task N`.
  - `type BadHeading struct { Text string; Level int; Line int }` — a heading whose text starts with `Task` and a number but is not `### Task N: <name>`.
  - `type Cancelled struct { Number int; Name, Reason string; Line int; OK bool }` — one bullet under `## Cancelled Tasks`; `OK` is false when it is not `- Task N: <name> - <reason>` with a non-empty reason.
  - `type Plan struct { Tasks []Task; BadHeadings []BadHeading; Cancelled []Cancelled; ReuseLine int; Reuse *markdown.Table }` — `ReuseLine` is the `## Reuse` heading's line, 0 when there is none; `Reuse` its first table.
  - `func ParsePlan(b []byte) Plan`
- Task headings match `^Task ([1-9][0-9]*): (\S.*)$` at level 3; any heading whose text matches `^Task\s+\d` and does not is a `BadHeading`. A task's lines run to the next heading of level 3 or above; nothing inside a fence counts anywhere.

- [x] **Step 1: Write the failing tests:** `TestParseProposalSections` (titles, bodies with the template comment stripped to empty, Contract subsections, Change Log table rows); `TestProposalCRLF` (Review Focus 2); `TestFencedHeadingsIgnored` (Review Focus 4: a fenced `## Non-Goals` is not a section, a fenced `### Task 9: x` is not a task); `TestParsePlanTasks` (numbers, names, lines, `**Scenarios:**` items with and without backticks, `none`, a missing line); `TestBadHeadings` (`## Task 3: Parser`, `### Task 0: x`, `### Task 2 Parser`); `TestCancelledBullets` (`- Task 5: Front desk screen - moved to add-x` ok, `- Task 5: Front desk screen` not ok); `TestCompensates`; `TestReuseTable` (Review Focus 1, an inline fixture in this repository's style: escaped `\|` in Evidence becomes `|`, backticked Existing cells such as `` `AGENTS.md`: "Change workflow" `` are kept as written).
- [x] **Step 2: Run** `go test ./internal/change/` — expected: FAIL.
- [x] **Step 3: Implement** `ParseProposal` and `ParsePlan`.
- [x] **Step 4: Run** `go test ./internal/change/` — expected: PASS.
- [x] **Step 5: Commit** `feat(change): read proposal.md and plan.md`.

### Task 5: Lint engine, proposal and placeholder rules

**Scenarios:** `change-lint#lint-every-active-change`, `change-lint#named-change-missing`, `change-lint#non-goals-missing`, `change-lint#empty-failure-modes`, `change-lint#builds-on-left-empty`, `change-lint#change-log-row-without-source`, `change-lint#header-only-change-log`, `change-lint#template-comment-left`, `change-lint#plan-not-written-yet`

**Files:**
- Create: `internal/lint/lint.go`, `internal/lint/proposal.go`, `internal/lint/placeholder.go`
- Test: `internal/lint/lint_test.go`, `internal/lint/proposal_test.go`, `internal/lint/placeholder_test.go`

**Interfaces:**
- Consumes: `change.ParseProposal` (Task 4), package `markdown` (Task 1), `docket.Schema`.
- Produces (package `lint`):
  - `type Finding struct { Rule, Change, File string; Line int; Message string }`
  - `type Unreadable struct { Source, Reason string }`
  - `type Report struct { Changes []string; Findings []Finding; Unreadable []Unreadable }` — `Changes` are the linted slugs, sorted; findings sorted by change, file, line and rule.
  - `type Input struct { Repo fs.FS; Templates fs.FS; Slugs []string; Added []string; AddedErr error }` — `Repo` is the worktree root; `Templates` is `docket.Schema()`; `Added` and `AddedErr` feed the routing rule (Task 8).
  - `func Run(in Input) Report`
- Scope: with no slugs, every directory directly under `openspec/changes/` except `archive`, sorted; a missing `openspec/changes/` lints nothing. A named slug that is not a valid `fs` path element or not such a directory gives a finding `change-missing` (no `change` field) and is not linted. A change's files are `proposal.md`, `plan.md` when present, `design.md` when present, every `specs/**/spec.md` (capability path = the directory under `specs/`), and, for the placeholder rule, every regular file in the folder. A read error other than "does not exist" lists the file as unreadable (source = its path) and skips the rules on it; an unreadable change directory is listed and skipped.
- `proposal-section`: proposal.md missing (no line); each missing level-2 section among Why, What Changes, Non-Goals, Builds On, Capabilities, Impact, Implementation Contract, Change Log (no line); each missing level-3 subsection of Implementation Contract among Observable Behavior, Interfaces, Failure Modes, Acceptance Criteria (on the Implementation Contract line); and each of Non-Goals, Builds On and the four subsections whose `Body` is empty (on its heading line). The message names the section.
- `change-log`: no table under Change Log (on its heading); each row whose first cell does not match `^\d{4}-\d{2}-\d{2}$` or whose second cell is empty (on that row's line).
- `placeholder`: the placeholder strings are every HTML comment (as written, possibly across lines) and every match of `<[A-Za-z][^<>\n]*>` in `templates/*.md` of `Templates`. In each file of the change, with fenced lines and code spans blanked out (same offsets, newlines kept), each occurrence of a placeholder is a finding on the line where it starts; the message quotes the placeholder and names the template it came from.
- Plan rules (Tasks 6 and 7) run only when `plan.md` exists.

- [x] **Step 1: Write the failing tests** (an `fstest.MapFS` per case; a helper writes a valid proposal that tests then break): `TestLintEveryActiveChange` (`add-x`, `add-y` and `archive/2026-01-01-add-z` → `Changes` is `[add-x add-y]`); `TestNamedChangeMissing` (`add-z` → one `change-missing` finding naming `add-z`; `../x` and `a/b` the same); `TestNonGoalsMissing`; `TestEmptyFailureModes` (only the template comment → finding on its heading line); `TestBuildsOnLeftEmpty`; `TestChangeLogRowWithoutSource` (finding on the row's line); `TestHeaderOnlyChangeLog` (no `change-log` finding); `TestTemplateCommentLeft` (the Why comment from `docket.Schema()` → a `placeholder` finding on its line; the same comment inside a fence or a code span → none); `TestPlaceholderInPlan` (`### Task 1: <name>` left in plan.md → finding); `TestPlanNotWrittenYet` (no plan.md → no plan-rule or `scenario-unclaimed` findings); `TestValidProposalIsClean` (a complete proposal in this repository's style, with Change Log rows → no findings); `TestCRLFProposal` (Review Focus 2); `TestUnreadableFile` (Review Focus 5: a mode-000 `plan.md` in a temporary directory passed as `os.DirFS` → unreadable entry, the other change still linted; skip as root); `TestUnreadableChangeDirectory` (a mode-000 change folder → unreadable entry naming it, the other change still linted; skip as root); `TestPlaceholderInOtherFile` (a template comment left in a non-Markdown file of the change → finding).
- [x] **Step 2: Run** `go test ./internal/lint/` — expected: FAIL.
- [x] **Step 3: Implement** `Run`, the scope rules and the three rules.
- [x] **Step 4: Run** `go test ./internal/lint/` — expected: PASS.
- [x] **Step 5: Commit** `feat(lint): lint proposals and leftover template text`.

### Task 6: Plan rules

**Scenarios:** `change-lint#task-heading-at-the-wrong-level`, `change-lint#duplicate-task-number`, `change-lint#reuse-row-with-a-missing-path`, `change-lint#new-row-without-a-reason`, `change-lint#external-dependency-row`, `change-lint#empty-reuse-table`, `change-lint#row-without-evidence`, `change-lint#cancelled-task-without-a-reason`, `change-lint#compensating-a-cancelled-task`

**Files:**
- Create: `internal/lint/plan.go`
- Test: `internal/lint/plan_test.go`

**Interfaces:**
- Consumes: `change.ParsePlan` (Task 4); `Run` and `Finding` (Task 5).
- `task-heading`: each `BadHeading` (on its line); each task number used more than once across `Tasks` and `Cancelled` (one finding on the later line, the message naming every line).
- `reuse-table`: no `## Reuse` (no line); no table under it (on the heading); a header other than Need, Existing (path:symbol), Decision, Reason, Evidence, compared trimmed and without regard to case (on the table line); no rows (on the table line); per row, on its line: a Decision other than `reuse`, `extend` or `new` (backticks removed, trimmed); for `reuse` and `extend`, unless Existing with backticks removed starts with `external:`, the text before its first `:`, trimmed, is empty, not `fs.ValidPath`, or does not exist in `Repo`; for `new`, an empty Reason; for every row, an empty Evidence.
- `cancelled-task`: each `Cancelled` with `OK` false.
- `compensates`: each `Ref` with `OK` false, or whose number is not a task in `Tasks`, or is in `Cancelled` (the message says which).

- [x] **Step 1: Write the failing tests:** `TestTaskHeadingAtTheWrongLevel` (`## Task 3: Parser`); `TestDuplicateTaskNumber` (two `### Task 2:` headings, and a task numbered like a cancelled bullet); `TestReuseRowWithAMissingPath` (`reuse` naming `internal/gitx/run.go:Run` absent from `Repo`); `TestNewRowWithoutAReason`; `TestExternalDependencyRow` (`external: git CLI`, and `` external: `syscall.Flock` `` → no finding); `TestEmptyReuseTable`; `TestRowWithoutEvidence`; `TestCancelledTaskWithoutAReason` (`- Task 5: Front desk screen`); `TestCompensatingACancelledTask`; `TestCompensatingAMissingTask`; `TestFencedPlanContent` (Review Focus 4: a fenced `### Task 9: x` is not a task, a fenced `## Reuse` with a table does not stand in for a missing one, and a fenced `**Scenarios:** none` does not give a task its Scenarios line); `TestCRLFPlan` (Review Focus 2: a plan with CRLF line endings and a BOM gives the same findings on the same lines as its LF copy); `TestRepositoryStylePlan` (Review Focus 1: a plan with backticked Existing paths, an `AGENTS.md`: "…" cell, escaped `\|` in Evidence and backticked `external:` rows, all paths present in the `fstest.MapFS` → no plan-rule findings). The repository's real plans are linted in Task 9 and by the `lint` CI job; tests do not read them, since archiving moves them.
- [x] **Step 2: Run** `go test ./internal/lint/` — expected: FAIL.
- [x] **Step 3: Implement** the four rules.
- [x] **Step 4: Run** `go test ./internal/lint/` — expected: PASS.
- [x] **Step 5: Commit** `feat(lint): check task headings, the Reuse table and cancelled tasks`.

### Task 7: Scenario and delta structure rules

**Scenarios:** `change-lint#scenario-nobody-claims`, `change-lint#reference-to-an-unknown-scenario`, `change-lint#scaffolding-task`, `change-lint#two-scenarios-named-success`, `change-lint#requirement-outside-any-delta-section`, `change-lint#header-inside-a-code-block`, `change-lint#unpaired-rename`

**Files:**
- Create: `internal/lint/scenarios.go`, `internal/lint/structure.go`
- Test: `internal/lint/scenarios_test.go`, `internal/lint/structure_test.go`

**Interfaces:**
- Consumes: `delta.Parse`, `delta.ParseSpec`, `delta.Slug` (Task 2); `change.ParsePlan` (Task 4); `Run` (Task 5).
- Known scenarios of a capability: the scenarios of the change's ADDED and MODIFIED requirements for it, plus those of `openspec/specs/<capability-path>/spec.md` when that file exists (read with `ParseSpec`).
- `scenario-ref` (plan only): a task with no `**Scenarios:**` line (on its heading line); an item that is not `none` and not `<capability-path>#<slug>` with a non-empty path and slug (on the Scenarios line); a reference whose capability or slug is not known (on the Scenarios line, naming the reference).
- `scenario-unclaimed` (plan only): each scenario of the change's ADDED and MODIFIED requirements that no task references, on the scenario's line in its delta file, naming `<capability-path>#<slug>`.
- `scenario-duplicate`: per capability, take the main spec's requirements, drop those whose name the delta modifies or removes, add the delta's ADDED and MODIFIED requirements; two scenarios in that set with the same slug → one finding on the later one, naming both places.
- `unparseable`: each `Problem` of each delta file, on its line, with its message.

- [x] **Step 1: Write the failing tests:** `TestScenarioNobodyClaims` (delta adds "Join when full" to `booking`, no task lists `booking#join-when-full` → finding naming it, on the scenario's line); `TestReferenceToAnUnknownScenario` (`booking#join-when-ful`); `TestScaffoldingTask` (`**Scenarios:** none` → nothing); `TestMissingScenariosLine`; `TestReferenceToMainSpec` (a reference to a scenario only in `openspec/specs/booking/spec.md` is known); `TestTwoScenariosNamedSuccess` (two delta requirements in `booking` each with "Success"; also a delta scenario that collides with a main-spec one, and a MODIFIED requirement that replaces the main one without colliding); `TestNonASCIISlugs` (Review Focus 3: `需求 R1#3` referenced as `booking#需求-r1-3`, and `Join (when full)` referenced as `booking#join-when-full`, are known); `TestPunctuatedDuplicates` (Review Focus 3: `Join (when full)` and `Join when full` in one capability, and `需求 R1#3` and `需求 r1 3`, are each reported as `scenario-duplicate`); `TestRequirementOutsideAnyDeltaSection`, `TestHeaderInsideACodeBlock`, `TestUnpairedRename` (each as an `unparseable` finding, or none for the fenced header, at the lint level).
- [x] **Step 2: Run** `go test ./internal/lint/` — expected: FAIL.
- [x] **Step 3: Implement** the four rules.
- [x] **Step 4: Run** `go test ./internal/lint/` — expected: PASS.
- [x] **Step 5: Commit** `feat(lint): check scenario claims, slugs and delta structure`.

### Task 8: Routing and the `dkt lint` command

**Scenarios:** `change-lint#design-written-to-the-superpowers-folder`, `change-lint#no-default-branch-for-routing`, `change-lint#no-openspec-installed`, `change-lint#lint-every-active-change`

**Files:**
- Create: `internal/gitx/added.go`, `internal/lint/routing.go`, `internal/cli/cmd_lint.go`
- Modify: `internal/cli/run.go` (add `lintCommand` to `commands`)
- Test: `internal/gitx/added_test.go`, `internal/lint/routing_test.go`, `internal/cli/cmd_lint_test.go`

**Interfaces:**
- Consumes: `lint.Run`, `lint.Input` (Task 5); `scan.ResolveDefaultBranch`, `scan.ErrNoDefaultBranch`, `Git.MergeBase`, `cli` types (from `add-init-and-new`); `docket.Schema`.
- Produces (package `gitx`):
  - `func (g Git) AddedFiles(base string, paths ...string) ([]string, error)` — `git diff --name-only --diff-filter=A --no-renames -z <base> HEAD -- <paths>`.
  - `func (g Git) UncommittedAdded(paths ...string) ([]string, error)` — `git status --porcelain=v1 -z --untracked-files=all --no-renames -- <paths>`, the paths whose status has `A` or `?` in either column.
- `routing` (package `lint`): each path in `Added` under `docs/superpowers/specs/` or `docs/superpowers/plans/` is a finding with that `file` and no `change`. A non-nil `AddedErr` adds `Unreadable{Source: "routing", Reason: AddedErr.Error()}` and every other rule still runs.
- The `lint` command (positional slugs, no flags of its own): opens the repository root with `os.OpenRoot` and passes `Root.FS()`; computes `Added` for `docs/superpowers/specs` and `docs/superpowers/plans` as the union, sorted and deduplicated, of `AddedFiles(<merge base of HEAD and the default tip>)` when HEAD has a commit and `UncommittedAdded`; when the default branch cannot be resolved, or HEAD and the default tip share no history, `AddedErr` says so and suggests `--default-branch`. It maps each `lint.Finding` to `cli.Finding` and each `lint.Unreadable` to `cli.Unreadable`, and sets `Result` to `type LintResult struct { Changes []string }` (JSON `changes`). It never creates an OpenSpec runner.

- [x] **Step 1: Write the failing tests:** `gitx`: `TestAddedFiles` (a file added on the branch after the merge base is listed, one modified is not, one renamed into the path is listed); `TestUncommittedAdded` (untracked and staged-new files under the path are listed, files elsewhere are not, a path with a space is read correctly). `lint`: `TestRoutingFinding`, `TestRoutingUnreadable`. `cli` (through `RunWith` with `--json`, in `testutil` repositories): `TestDesignWrittenToTheSuperpowersFolder` (an untracked `docs/superpowers/specs/2026-10-25-x-design.md` → a `routing` finding naming it, exit 1); `TestNoDefaultBranchForRouting` (no `main`, `master` or origin, a change with a `proposal-section` problem → that finding, `unreadable` listing `routing`, exit 3); `TestNoOpenSpecInstalled` (`PATH` without `openspec` and no `DOCKET_OPENSPEC` → a clean change exits 0); `TestUnreadableFileExitsThree` (one change with a mode-000 `plan.md` and another with a `proposal-section` problem → both reported, exit 3; skip as root); `TestLintEveryActiveChangeJSON` (`result.changes` is `["add-x","add-y"]`); `TestLintNamedChanges` (`dkt lint add-x` lints only `add-x`); `TestLintCleanExit` (text output ends with `dkt lint: ok`).
- [x] **Step 2: Run** `go test ./internal/gitx/ ./internal/lint/ ./internal/cli/` — expected: FAIL.
- [x] **Step 3: Implement** the two `gitx` methods, the routing rule and the command.
- [x] **Step 4: Run** `go test ./...` — expected: PASS.
- [x] **Step 5: Commit** `feat: dkt lint checks change folders and where designs were written`.

### Task 9: Lint this repository, architecture map and CI

**Scenarios:** none

**Files:**
- Modify: `docs/ARCHITECTURE.md`, `AGENTS.md`, `.github/workflows/ci.yml`, and any change folder under `openspec/changes/` that `dkt lint` reports on
- Test: the `lint` CI job

- [x] **Step 1: Run** `go run ./cmd/dkt lint --default-branch main` in this worktree. Fix each finding in the change folders it names, or, when the finding shows a defect in Docket, fix the defect in the task that owns the rule and add the case to its tests. Expected afterwards: `dkt lint: ok`.
- [x] **Step 2: Update `docs/ARCHITECTURE.md`:** codemap rows for `internal/markdown`, `internal/delta` and `internal/lint`; capability index rows "Read Markdown lines, headings and tables" → `markdown.Lines`, `markdown.Headings`, `markdown.FindTable`; "Read a delta or a main spec" → `delta.Parse`, `delta.ParseSpec`; "Make a scenario slug" → `delta.Slug`; "Read proposal.md or plan.md" → `change.ParseProposal`, `change.ParsePlan`; "Lint change folders" → `lint.Run`; "Files a branch adds" → `Git.AddedFiles`, `Git.UncommittedAdded`.
- [x] **Step 3: Update the Status paragraph of `AGENTS.md`:** `dkt lint` exists; drop "check the lint rules by reading". Leave the Change workflow section unchanged.
- [x] **Step 4: Add a `lint` job to `.github/workflows/ci.yml`:** `ubuntu-latest`; `actions/checkout@v7` with `fetch-depth: 0`; `actions/setup-go@v7` with `go-version: stable`; `go run ./cmd/dkt lint --default-branch main`.
- [x] **Step 5: Run** `gofmt -l . && go vet ./... && go test ./...` and `go run ./cmd/dkt lint --default-branch main` — expected: no output from gofmt, PASS, `dkt lint: ok`.
- [x] **Step 6: Commit** `docs: map the lint packages and run dkt lint in CI`.
