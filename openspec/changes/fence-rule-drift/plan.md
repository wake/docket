# Implementation Plan

**Goal:** `dkt lint` reports `fence-info-backtick` for each line that dkt reads as the start of fenced code while OpenSpec 1.16 and later read it as inline code, in the files `fence-in-comment` checks.

**Architecture:** `internal/markdown` gains one pure function that walks the lines as `FenceMask` does and returns the openers whose marker is a backtick and whose rest of line holds a backtick. `internal/lint` runs it wherever it runs `FencesInComments`: the Markdown files of a `docket` change, records, issues, and the main specs in the walk that checks titles.

**Tech Stack:** Go, standard library only.

**Spec:** `openspec/changes/fence-rule-drift/proposal.md`, `design.md`, `specs/change-lint/spec.md`.

## Global Constraints

- `FenceMask`, `BlankComments`, `FencesInComments` and every reader built on them read files exactly as before.
- The opener is matched with the expression `FenceMask` uses; the rule adds only the backtick condition of OpenSpec 1.16's `getFenceMarker`.
- No new dependency. Structural and behavior edits go in separate commits.

## Review Focus

- The rest of the line is cut at the first U+2028 or U+2029 before looking for a backtick, as JavaScript's `.` in `(.*)` stops there.
- Only lines `FenceMask` reads as openers are reported: a line inside fenced code, or a tilde fence, is not.
- `dkt inspect`, which calls `SpecTitles`, reports no `fence-info-backtick`.

## Reuse

| Need | Existing (path:symbol) | Decision | Reason |
|---|---|---|---|
| Find the openers OpenSpec 1.16 rejects | internal/markdown/markdown.go:openFence | extend | Same opener expression and walk as `FenceMask`, so the check cannot drift from the readers |
| Apply the rule to change files, records and issues | internal/lint/document.go:checkFenceInComments | extend | Already runs on exactly the files the rule checks |
| Apply the rule to the main specs | internal/lint/lint.go:Run | extend | Its spec walk already applies `fence-in-comment` to each main spec |

---

### Task 1: The fence-info-backtick rule

**Scenarios:** change-lint#inline-code-at-the-start-of-a-line, change-lint#backtick-fence-with-a-plain-info-string, change-lint#tilde-fence-whose-info-string-holds-a-backtick, change-lint#line-inside-fenced-code, change-lint#line-in-a-record, change-lint#line-in-a-main-spec

**Review:** judgment

**Files:**
- Modify: `internal/markdown/markdown.go`
- Modify: `internal/markdown/markdown_test.go`
- Modify: `internal/lint/document.go`
- Modify: `internal/lint/lint.go`
- Modify: `internal/lint/record.go`
- Modify: `internal/lint/document_test.go`
- Modify: `internal/lint/record_test.go`
- Modify: `internal/lint/issue_test.go`

**Interfaces:**
- `markdown.FenceInfoBackticks(lines []string) []int`: the 0-based index of each such opener, in file order.
- Rule `fence-info-backtick`, message exactly: `backtick fence whose info string holds a backtick: dkt and OpenSpec 1.14 read the lines after it as code, OpenSpec 1.16 and later read the line as inline code; use single backticks or a tilde fence`

- [ ] `FenceInfoBackticks` returns the line made of three backticks, `npm test` and three backticks; a four-backtick opener whose info holds a backtick; and an opener indented with spaces; tested in `internal/markdown`
- [ ] `FenceInfoBackticks` returns nothing for a backtick opener with a plain info string, a tilde opener whose info holds a backtick, a line of six backticks, such a line inside a block fenced by tildes, and a line whose only backtick after the marker follows a U+2028; tested in `internal/markdown`
- [ ] After a reported opener, lines up to the fence that closes it are not reported, and a second such opener after that fence is; tested in `internal/markdown`
- [ ] `dkt lint` reports the rule with the fixed message on the line, in plan.md, the design, the proposal and a nested `.md` file of a `docket` change, and reports none for the plain-info, tilde and inside-fence cases; tested in `internal/lint/document_test.go`
- [ ] `dkt lint` reports the rule in a record and in an issue; tested in `internal/lint/record_test.go` and `internal/lint/issue_test.go`
- [ ] `dkt lint` naming no change reports the rule on the line of `openspec/specs/orders/spec.md`, and `SpecTitles` does not; tested in `internal/lint/document_test.go`
- [ ] A change of another schema is not checked; tested in `internal/lint/document_test.go`
