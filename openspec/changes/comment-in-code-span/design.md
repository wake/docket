# Design

## Context

dkt finds HTML comments with `markdown.BlankComments`, after `FenceMask`: outside fenced lines, a comment opener starts a comment wherever it stands, and the first comment closer (the two dashes and angle bracket, or the form with an exclamation mark) ends it; the state carries across lines and over fenced lines. `change.ParsePlan`, `record.Parse`, `specdoc`, `setup` and the lint rules read through it. OpenSpec 2.0.0, the version checked here, finds comments the same way after fences in `maskHtmlComments` of `core/parsers/requirement-blocks.js`, which only decides whether delta text was left unapplied; `maskHtmlComments` of `core/specs-apply.js` blanks comments before fences, for the Purpose of a new main spec, the order `fence-in-comment` covers; its requirement parser does not look at comments. None of them knows code spans. `markdown.CodeSpans` finds the code spans of one line for the placeholder rule: a run of backticks opens a span that the next run of the same length on the line closes. See proposal.md - Why for the effect.

## Goals / Non-Goals

**Goals:**
- Name the comment openers where dkt's reading and a reading that finds each line's code spans first differ, so that a file with no finding is read the same by both. That reading is not Markdown's in full: a span across lines is not part of it (see Risks).

**Non-Goals:**
- Changing `BlankComments`, `CodeSpans` or any reader built on them.
- Following Markdown in anything but code spans; see proposal.md - Non-Goals.

## Decisions

### Two readings walked together

The check walks the lines once, keeping two readings side by side, both skipping the lines `FenceMask` marks:

- **dkt's reading**, as `BlankComments` reads: outside a comment, an opener starts one; inside, the first closer ends it.
- **The span-first reading**: the same, except that outside a comment it also finds code spans, left to right. A run of backticks that a run of the same length closes later on the line opens a code span, which ends after that run; nothing inside it starts a comment. A run with no such match is text. Outside a span, a backslash followed by a backtick or a backslash makes that character text. Inside a comment, backticks are text. A comment that starts a line, after at most three spaces, is an HTML block in Markdown, whose lines hold no code span: on the line where such a comment ends, the rest of the line is read for comments only.

The check reports each opener where dkt's reading starts a comment while the span-first reading is inside a code span.

Each finding is a place the two readings differ: dkt hides the text after the opener, the span-first reading shows it as code. And a file with no finding is read the same way by both: while both are inside a comment they close on the same closer; while both are outside a comment and the span-first reading is outside a span, both start a comment on the same opener; so they first differ where the span-first reading is inside a span and dkt starts a comment, which is a finding. After a finding the readings stay out of step until dkt's comment ends, and an opener inside that comment is not seen; fixing the finding brings it out on the next run.

The HTML block rule came from checking the cases against commonmark.js 0.31.2, the CommonMark reference implementation: on a line that starts with a comment, an opener in backticks after the comment's end is raw HTML, which a browser also reads as a comment, so reporting it would be wrong. Other HTML blocks, indented code and container blocks such as lists and quotes are not followed.

The escapes are limited to backticks and backslashes because they decide where a span starts. A backslash before the opener also makes it text in Markdown, but following that would make the span-first reading differ from dkt's for a reason other than code spans, which the check would then not report.

Alternatives considered:
- Running `CodeSpans` on each line and reporting each opener `BlankComments` takes inside a span. Simpler, but `CodeSpans` does not know comments: a backtick inside a comment would pair with one after it and report a real opener, and a span starting after a comment closes on the same line would be missed when an earlier backtick paired wrongly.
- Comparing the blanked text of both readings. It finds the same first difference, but names a byte, not the opener.

### The line where dkt's comment ends

When dkt's reading closes the comment a reported opener started, the finding takes the closer's line; when the file ends first, the comment runs to the end. The message says which, so the author knows what dkt hides: one rule, one name, two forms of the message.

### Shared pieces

The closer search and the span search already exist inline in `BlankComments`, `FencesInComments` and `CodeSpans`. They move into two small helpers, one finding the end of a comment closer in a string and one finding the end of the span a backtick run opens, in a structural commit before the rule, so the readers and the check find closers and spans the same way. A closer on a fenced line does not end a comment, as in `BlankComments`.

### Checked where the fence rules are checked

The rule runs wherever the fence rules run, through `fenceFindings`, the helper that gathers a file's findings for every lint caller, so no caller changes: each `.md` file of a `docket` change, each record and issue lint reads, and, when no change is named, each main spec in the walk that checks titles.

### Main spec text

Two scenarios of Fence marker inside an HTML comment write the comment opener in a code span; under the new rule the main spec would be reported. The delta modifies that requirement to describe the opener in words, keeping every scenario's meaning, so `dkt lint` passes on this repository once the change is archived.

## Risks / Trade-offs

- [Spans across lines are not found] → An opener in such a span is not reported, and dkt still hides what follows; the rule works within a line, as the placeholder rule does, and finding spans across lines needs paragraph boundaries, which dkt does not read.
- [An opener inside a comment dkt has already started is not seen until that comment is fixed] → Every finding is a real difference, and a file with no finding reads the same under both readings.
