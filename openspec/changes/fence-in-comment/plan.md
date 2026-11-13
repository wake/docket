# Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `dkt lint` reports `fence-in-comment` for a fence marker inside an HTML comment whose code would run past the comment.

**Architecture:** `internal/markdown` gains `FencesInComments`, a second reading of a file's lines in document order that names each such marker; `internal/lint` applies it to every Markdown file of a `docket` change and to each record and issue, next to the placeholder rule. No reader changes how it masks.

**Tech Stack:** Go standard library.

**Spec:** `openspec/changes/fence-in-comment/proposal.md`, `specs/change-lint/spec.md`, `design.md`.

## Global Constraints

- The rule is named exactly `fence-in-comment`; a finding names the repository-relative file and the 1-based line of the marker.
- Fence lines are matched with the expressions `FenceMask` uses (`fenceOpen`, `fenceClose`: the fence rule of OpenSpec 1.14.3); a closer has the opener's character and at least its length.
- A comment opens on `<!--` and closes on the first `-->` or `--!>` after it, as `BlankComments` reads them.
- `FenceMask`, `BlankComments` and every reader built on them keep their behavior.
- Everything in the repository is written in English.

## Review Focus

- A fence opener line that also ends the comment, such as a line of three backticks followed by `-->`: reported, since `FenceMask` opens code there and the comment closes with it open.
- A comment opened mid-line after text, then a fence line on the next line: reported like a comment that opens a line.
- A shorter run, or a run of the other character, inside a comment after an opener: does not close it, so the opener is reported.
- A real fence opener line that holds a comment opener after the marker: opens code, not a comment, so nothing is reported for the lines inside the code.
- A comment that closes and a new one that opens on the same line: the second comment is checked on its own.

## Reuse

| Need | Existing (path:symbol) | Decision | Reason | Evidence |
|---|---|---|---|---|
| Fence lines under OpenSpec's rule | `internal/markdown/markdown.go:fenceOpen` | reuse | The check must match fences exactly as `FenceMask` does | `rg -n "fenceOpen\|fenceClose" internal/markdown` |
| Matching an opener and its closer | `internal/markdown/markdown.go:FenceMask` | extend | Its opener and closer matching moves into two unexported helpers, so both readings share it | `rg -n "fenceClose.FindStringSubmatch" internal/markdown` |
| Finding a fence marker inside a comment | none | new | `BlankComments` skips fenced lines by design, so it never sees a marker inside a comment; no reader reads comments before fences | `rg -n "BlankComments\|FenceMask" internal --glob '!*_test.go'` |
| Applying a rule to every Markdown file of a change | `internal/lint/lint.go:lintChange` | extend | The rule runs where `checkPlaceholders` runs, on the same files | `rg -n "checkPlaceholders\|checkPlaceholderFile" internal/lint` |
| Applying a rule to a record or an issue | `internal/lint/record.go:lintDated` | extend | Records and issues are linted one file at a time there | `rg -n "func lintDated" internal/lint` |
| Reporting a finding | `internal/lint/lint.go:state.add` | reuse | Every rule reports through it | `rg -n "func \(r \*state\) add" internal/lint` |
| Linting a change, a record and an issue in tests | `internal/lint/document_test.go:lintX` | reuse | Also `record_test.go:lintRec` and `issue_test.go:lintIss` | `rg -n "func lint(X\|Rec\|Iss)" internal/lint` |

---

### Task 1: Find fence markers inside comments

**Scenarios:** change-lint#fence-marker-left-in-a-comment, change-lint#code-block-commented-out-whole, change-lint#comment-marker-inside-fenced-code

**Review:** judgment

**Files:**
- Modify: `internal/markdown/markdown.go`
- Test: `internal/markdown/markdown_test.go`

**Interfaces:**
- Produces: `func FencesInComments(lines []string) []int` in package `markdown`: the 0-based index of each reported marker line, in file order; nil when there is none.

- [ ] **Step 1: Move the fence matching of `FenceMask` into helpers (structural commit)**

Add an unexported `type fence struct{ marker byte; length int }`, `func openFence(line string) (fence, bool)` (from `fenceOpen`) and `func (f fence) closedBy(line string) bool` (from `fenceClose`, same character, at least the length), and rewrite `FenceMask` on them with no change in behavior.

- [ ] **Step 2: Run the package tests**

Run: `go test ./internal/markdown/`
Expected: PASS, `TestFenceMask` unchanged.

- [ ] **Step 3: Commit the structural change**

```bash
git add internal/markdown/markdown.go
git commit -m "Share the fence matching of FenceMask"
```

- [ ] **Step 4: Write the failing test `TestFencesInComments`**

A table test calling `FencesInComments(lines)` and comparing with `reflect.DeepEqual` (nil for none), with these rows (`b3` is a line of three backticks, `b4` of four):

````go
{"marker left", []string{"<!--", b3, "-->", "### Task 2: Real"}, []int{1}},
{"tildes", []string{"<!--", "~~~", "-->"}, []int{1}},
{"indented marker", []string{"<!--", "  " + b3, "-->"}, []int{1}},
{"whole block", []string{"<!--", b3, "x", b3, "-->"}, nil},
{"comment marker in code", []string{b3, "<!--", b3, "### H"}, nil},
{"opener holds comment marker", []string{b3 + " <!--", "x", b3, "y"}, nil},
{"marker closes comment", []string{"<!--", b3 + " -->", "b"}, []int{1}},
{"comment opened mid-line", []string{"text <!-- note", b3, "-->"}, []int{1}},
{"comment closed before fence", []string{"<!-- a -->", b3, "x", b3}, nil},
{"unclosed comment", []string{"<!--", b3, "x"}, nil},
{"shorter run does not close", []string{"<!--", b4, b3, "-->"}, []int{1}},
{"other character does not close", []string{"<!--", b3, "~~~", "-->"}, []int{1}},
{"longer run closes", []string{"<!--", b3, b4, "-->"}, nil},
{"closer with text does not close", []string{"<!--", b3, b3 + " x", "-->"}, []int{1}},
// In document order line 3 opens code, so lines 4 to 6 are code, not a comment.
{"code after a reported comment", []string{"<!--", b3, "-->", b3, "<!--", "~~~", "-->"}, []int{1}},
{"bang close", []string{"<!--", b3, "--!>"}, []int{1}},
{"two comments", []string{"<!--", b3, "-->", "<!--", "~~~", "-->"}, []int{1, 4}},
{"reopened on one line", []string{"<!-- a --> <!--", b3, "-->"}, []int{1}},
{"no comment", []string{"a", b3, "x", b3}, nil},
````

- [ ] **Step 5: Run it to see it fail**

Run: `go test ./internal/markdown/ -run TestFencesInComments`
Expected: FAIL, `FencesInComments` undefined.

- [ ] **Step 6: Implement `FencesInComments(lines []string) []int`**

One pass in document order, as design.md says. Outside code and comments, a line `openFence` accepts opens real code and is not scanned for comments; inside real code only its `closedBy` line matters. Otherwise the line is scanned for `<!--`, `-->` and `--!>` as `BlankComments` scans. Inside a comment, a line is first matched as a fence line: with no code open in the comment, `openFence` opens it there (remember the index); with one open, `closedBy` closes it. When the comment closes with code still open in it, append the remembered index and forget that code. A comment still open at the end reports nothing. Doc comment: what it returns and why (`FenceMask` reads such a marker as code that runs past the comment).

- [ ] **Step 7: Run the package tests**

Run: `go test ./internal/markdown/`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/markdown/markdown.go internal/markdown/markdown_test.go
git commit -m "Find fence markers inside HTML comments"
```

### Task 2: Report fence-in-comment in lint

**Scenarios:** change-lint#fence-marker-left-in-a-comment, change-lint#code-block-commented-out-whole, change-lint#comment-marker-inside-fenced-code, change-lint#fence-marker-in-a-record

**Review:** checks

**Files:**
- Modify: `internal/lint/document.go`
- Modify: `internal/lint/lint.go`
- Modify: `internal/lint/record.go`
- Test: `internal/lint/document_test.go`
- Test: `internal/lint/record_test.go`
- Test: `internal/lint/issue_test.go`

**Interfaces:**
- Consumes: `markdown.FencesInComments(lines []string) []int` from Task 1.
- Produces: the finding rule `fence-in-comment`.

- [ ] **Step 1: Write the failing tests**

In `document_test.go`, `TestFenceInComment`, with `lintX` and `goodProposal`, one subtest per row (each lints `proposal.md` = `goodProposal` plus the file shown, or the proposal shown):

````go
// marker left in plan: reported on line 2 of openspec/changes/add-x/plan.md, once
"plan.md": "<!--\n```\n-->\n### Task 2: Real\n"
// whole block in design: no fence-in-comment
"design.md": "<!--\n```\nx\n```\n-->\n"
// comment marker in fenced code in proposal: no fence-in-comment
"proposal.md": goodProposal + "\n```\n<!--\n```\n"
// nested file: reported on line 2 of openspec/changes/add-x/deep/more.md
"deep/more.md": "<!--\n~~~\n-->\n"
````

Check with `has(rep, "fence-in-comment", <file>, <line>, "")` and `count(rep, "fence-in-comment")`.

In `record_test.go`, `TestLintRecordFenceInComment`: a record from `recFront` and the four sections, with the three lines of the nested-file case after the text of `## Why`: `has(rep, "fence-in-comment", recPath, <line of ~~~>, "")`. In `issue_test.go`, `TestLintIssueFenceInComment`: the same in an issue's first section, through `lintIss`.

- [ ] **Step 2: Run them to see them fail**

Run: `go test ./internal/lint/ -run 'FenceInComment'`
Expected: FAIL, no `fence-in-comment` finding.

- [ ] **Step 3: Implement `checkFenceInComments(r *state, c *loaded, name string)` in `document.go`**

For each index from `markdown.FencesInComments(markdown.Lines(c.Files[name]))`, `r.add(c, "fence-in-comment", name, i+1, ...)` with the message `fence marker inside an HTML comment: dkt and OpenSpec read the lines after the comment as code; close the code inside the comment or remove the marker`. Call it in `lintChange` after `checkPlaceholders` for each `.md` file of the change, sorted, and in `lintDated` after the placeholder check for the file.

- [ ] **Step 4: Run the lint tests and the full suite**

Run: `go test ./internal/lint/ && gofmt -l . && go vet ./... && go test ./...`
Expected: PASS, `gofmt -l` prints nothing.

- [ ] **Step 5: Commit**

```bash
git add internal/lint/
git commit -m "Report fence markers inside HTML comments in dkt lint"
```
