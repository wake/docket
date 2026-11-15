# Implementation Plan

**Goal:** `dkt lint` reports `fence-info-backtick` for each line that starts with three or more backticks, holds another backtick after them, and lies outside fenced code as OpenSpec 1.16 reads the file, in the files `fence-in-comment` checks. dkt and OpenSpec 1.14 read every such line as fenced code.

**Architecture:** `internal/markdown` reads the file a second time with OpenSpec 1.16's opener, sharing `FenceMask`'s walk and closing rule, and returns each backtick-info line that reading leaves outside code. `internal/lint` runs it wherever it runs `FencesInComments`: the Markdown files of a `docket` change, records, issues, and the main specs in the walk that checks titles.

**Tech Stack:** Go, standard library only.

**Spec:** `openspec/changes/fence-rule-drift/proposal.md`, `design.md`, `specs/change-lint/spec.md`.

## Global Constraints

- `FenceMask`, `BlankComments`, `FencesInComments` and every reader built on them read files exactly as before.
- The opener is matched with the expression `FenceMask` uses; the 1.16 reading adds only the backtick condition of OpenSpec 1.16's `getFenceMarker`, and shares the walk and closing rule with `FenceMask`.
- No new dependency. Structural and behavior edits go in separate commits.

## Review Focus

- The rest of the line is cut at the first U+2028 or U+2029 before looking for a backtick, as JavaScript's `.` in `(.*)` stops there.
- A line is reported by where OpenSpec 1.16 reads it, not dkt: a backtick-info line both read inside code is not reported, and one 1.16 reads outside code while dkt is inside a fence is.
- A tilde fence whose info holds a backtick is not reported.
- `dkt inspect`, which calls `SpecTitles`, reports no `fence-info-backtick`.

## Reuse

| Need | Existing (path:symbol) | Decision | Reason |
|---|---|---|---|
| Read a file with OpenSpec 1.16's fence rule | internal/markdown/markdown.go:FenceMask | extend | Same walk and closing rule, with the opener as the only difference, so the check cannot drift from the readers |
| Tell a backtick-info line | internal/markdown/markdown.go:openFence | extend | Same opener expression `FenceMask` uses |
| Apply the rule to change files, records and issues | internal/lint/document.go:checkFenceInComments | extend | Already runs on exactly the files the rule checks |
| Apply the rule to the main specs | internal/lint/lint.go:Run | extend | Its spec walk already applies `fence-in-comment` to each main spec |

---

### Task 1: The fence-info-backtick rule

**Scenarios:** change-lint#inline-code-at-the-start-of-a-line, change-lint#backtick-fence-with-a-plain-info-string, change-lint#tilde-fence-whose-info-string-holds-a-backtick, change-lint#line-inside-fenced-code, change-lint#line-both-rules-read-inside-fenced-code, change-lint#line-after-the-two-readings-swap, change-lint#line-in-a-record, change-lint#line-in-a-main-spec

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
- `markdown.FenceInfoBackticks(lines []string) []int`: the 0-based index of each backtick-info line OpenSpec 1.16 reads outside fenced code, in file order.
- Rule `fence-info-backtick`, message exactly: `backticks that open a line and hold another backtick: dkt and OpenSpec 1.14 read the line as fenced code, OpenSpec 1.16 and later as inline code; use single backticks or a tilde fence`

- [ ] `FenceMask` returns the same masks as before, its existing tests passing unchanged
- [ ] `FenceInfoBackticks` returns the line made of three backticks, `npm test` and three backticks; a four-backtick line whose info holds a backtick; and such a line indented with spaces; tested in `internal/markdown`
- [ ] `FenceInfoBackticks` returns nothing for a backtick opener with a plain info string, a tilde opener whose info holds a backtick, a line of six backticks, such a line inside a block fenced by tildes, and a line whose only backtick after the marker follows a U+2028, and one where it follows a U+2029; tested in `internal/markdown`
- [ ] For the lines of scenario Line both rules read inside fenced code it returns the first line only, and for those of Line after the two readings swap the first and the fourth; tested in `internal/markdown`
- [ ] Every file in the tests above with no line returned gets the same mask from `FenceMask` and from the 1.16 reading; tested in `internal/markdown`
- [ ] `dkt lint` reports the rule with the fixed message on the line, in plan.md, the design, the proposal and a nested `.md` file of a `docket` change, and reports none for the plain-info, tilde and inside-fence cases, and on the first line only for the lines of Line both rules read inside fenced code; tested in `internal/lint/document_test.go`
- [ ] `dkt lint` reports the rule in a record and in an issue; tested in `internal/lint/record_test.go` and `internal/lint/issue_test.go`
- [ ] `dkt lint` naming no change reports the rule on the line of `openspec/specs/orders/spec.md`, and `SpecTitles` does not; tested in `internal/lint/document_test.go`
- [ ] A change of another schema is not checked; tested in `internal/lint/document_test.go`
