# Implementation Plan

**Goal:** `dkt lint` reports `comment-in-code-span` for each comment opener that dkt reads as the start of an HTML comment while a reading that finds code spans first reads it inside a code span, naming the line where dkt's comment ends, in the files `fence-in-comment` checks.

**Architecture:** `internal/markdown` gains one pure function that walks a file's lines once with two readings side by side, dkt's (`BlankComments` over `FenceMask`) and the span-first one, and returns each opener where dkt starts a comment inside a span, with the line where that comment ends. The closer search and the span search, now inline in `BlankComments`, `FencesInComments` and `CodeSpans`, move into two helpers first. `internal/lint` runs the function through `fenceFindings`, the helper that gathers a file's findings for every lint caller, so no caller changes.

**Tech Stack:** Go, standard library only.

**Spec:** `openspec/changes/comment-in-code-span/proposal.md`, `design.md`, `specs/change-lint/spec.md`.

## Global Constraints

- `BlankComments`, `CodeSpans`, `FencesInComments` and every reader built on them return exactly what they return today.
- The span-first reading differs from dkt's only by code spans; its only escapes are a backslash before a backtick or a backslash.
- Structural and behavior edits go in separate commits. No new dependency.
- The change's own documents never write the comment opener inside a code span.

## Review Focus

- A backtick inside a comment does not start a span, and a span starting after a comment closes on the same line is found.
- The line where dkt's comment ends: a later line, the same line, or the end of the file.
- After a finding, the next opener dkt takes once its comment has closed is checked again, on the same line while the span-first reading is still inside the span, and on a later line.
- A closer on a fenced line does not end dkt's comment.
- Runs of different lengths: only a run of the same length closes a span.

## Reuse

| Need | Existing (path:symbol) | Decision | Reason |
|---|---|---|---|
| Find where a comment closer ends | internal/markdown/markdown.go:BlankComments | extend | The closer search moves into a helper that `BlankComments`, `FencesInComments` and the check share |
| Find where the span a backtick run opens ends | internal/markdown/markdown.go:CodeSpans | extend | The span search moves into a helper that `CodeSpans` and the check share |
| Read comments with code spans found first | internal/markdown/markdown.go:BlankComments | new | No reader tracks spans and comments together; the check walks dkt's reading next to the span-first one |
| Apply the rule to every checked file | internal/lint/document.go:fenceFindings | extend | Already gathers the per-file rules for change files, records, issues and main specs |

## Revision 1 (2026-11-15): a comment that starts a line

While implementing Task 1, the implementer checked the markdown cases against commonmark.js 0.31.2. A line that starts with a comment is an HTML block, whose lines hold no code span, so an opener in backticks after that comment's end is raw HTML, read as a comment by a browser too; the span-first reading took it as a span and would have reported it. The span-first reading now follows that block, the delta adds the scenario Comment starting a line, and Task 1's acceptance items name the cases. No completed task changes. Found by the implementer, not requested through any channel.

---

### Task 1: The comment-in-code-span rule

**Scenarios:** change-lint#opener-in-a-code-span-hides-the-rest-of-the-file, change-lint#comment-closed-on-a-later-line, change-lint#comment-closed-on-the-same-line, change-lint#comment-holding-a-code-span, change-lint#comment-starting-a-line, change-lint#backticks-inside-comments, change-lint#escaped-backtick, change-lint#opener-in-fenced-code, change-lint#opener-in-a-record, change-lint#opener-in-an-issue, change-lint#opener-in-a-main-spec, change-lint#fence-marker-left-in-a-comment, change-lint#code-block-commented-out-whole, change-lint#comment-marker-inside-fenced-code, change-lint#fence-marker-in-a-record, change-lint#fence-marker-in-a-main-spec

**Review:** judgment

**Files:**
- Modify: `internal/markdown/markdown.go`
- Modify: `internal/markdown/markdown_test.go`
- Modify: `internal/lint/document.go`
- Modify: `internal/lint/document_test.go`
- Modify: `internal/lint/record_test.go`
- Modify: `internal/lint/issue_test.go`

**Interfaces:**
- `markdown.SpanComment{Line, End int}`: the 0-based index of the opener's line, and of the line where dkt's comment ends, or -1 when it runs to the end of the file.
- `markdown.CommentsInCodeSpans(lines []string) []SpanComment`: in file order.
- Rule `comment-in-code-span`, message exactly `comment opener inside a code span: dkt reads the text after it as an HTML comment up to line <N>, Markdown as code; write the opener in fenced code or in words`, with the 1-based line, or, when the comment runs to the end, `comment opener inside a code span: dkt reads the rest of the file after it as an HTML comment, Markdown as code; write the opener in fenced code or in words`

- [ ] `BlankComments`, `CodeSpans` and `FencesInComments` pass their existing tests unchanged after the helpers are extracted, and so do the `fence-in-comment` tests in `internal/lint`, which cover the scenarios of Fence marker inside an HTML comment whose wording the delta changes
- [ ] `CommentsInCodeSpans` reports, with the right end: an opener in a span with no closer after it (-1); one closed on a later line; one closed in a later span of the same line; one in a span that starts after a comment, opened mid-line, closes on the same line; one in a span on the line after a comment that started a line has ended; one in a span after an escaped backslash; one in a span opened by two backticks with a single backtick before the opener; a comment whose only closer is on a fenced line (-1) and one closed on a line after such a fence; two openers in one span with a closer between them, both ending on that line; and a second opener in a span on a line after dkt's comment from the first has closed; tested in `internal/markdown`
- [ ] `CommentsInCodeSpans` reports nothing for: a comment holding a code span; two comments on one line each holding one backtick; a backslash and backtick, a comment, and a backtick; an opener in a span on a fenced line; an opener after a single unmatched backtick; an opener after an unmatched run of two backticks inside a one-backtick span that closed before it; an opener before a span; an opener in backticks after a comment that starts the line, after up to three spaces, and ends on it, and the same on the line where a comment that started an earlier line ends; and an opener on the line after a line ending inside an open span, which this reading does not carry across lines; tested in `internal/markdown`
- [ ] `dkt lint` reports the rule with the end-of-file message in plan.md, with the later-line message naming the closer's line in the proposal, and naming the same line in the design, and reports none for the comment-holding-a-span, comment-starting-a-line, backticks-inside-comments, escaped-backtick and fenced cases; tested in `internal/lint/document_test.go`
- [ ] `dkt lint` reports the rule in a record and in an issue; tested in `internal/lint/record_test.go` and `internal/lint/issue_test.go`
- [ ] `dkt lint` naming no change reports the rule on the line of `openspec/specs/orders/spec.md`, and `SpecTitles` does not; a change of another schema is not checked; tested in `internal/lint/document_test.go`
- [ ] Before archiving, `dkt lint` naming no change on this branch reports `comment-in-code-span` only on the two scenario lines of `openspec/specs/change-lint/spec.md` that the delta rewrites, and nothing in this change's own files
