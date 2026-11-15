---
title: "Report a comment opener inside a code span"
kind: feature
source: "docket/ideas/comment-in-code-span.md"
---

# Proposal

## Why

dkt starts an HTML comment at every comment opener outside fenced code, code spans included, and a comment that is never closed runs to the end of the file. Markdown, and so GitHub, reads an opener inside a code span as code, so a document that shows an opener in backticks loses, for dkt, every heading, task step and requirement after it, up to the next comment closer or the end of the file, without a finding. This hid the Reuse table and Task headings of a plan while `fence-in-comment` was written. Reporting the opener lets the author rewrite it while dkt keeps reading files as it does today.

## What Changes

- `dkt lint` reports `comment-in-code-span`, naming the file and line, for each comment opener that dkt reads as the start of an HTML comment while Markdown reads it inside a code span on that line. The message names the line where dkt's comment ends, or says it runs to the end of the file. It checks the files `fence-in-comment` checks: every Markdown file of a `docket` change, each record and issue it lints, and, when no change is named, each main spec whose title it checks.
- A file with no such finding is read by dkt the same way as by a reading that finds the code spans of each line before its comments. A code span that runs across lines is not part of that reading; see Non-Goals.
- Two scenarios of the requirement Fence marker inside an HTML comment wrote the comment opener in a code span; they now describe it in words, so the main spec passes the new rule. Their meaning is unchanged.

## Non-Goals

- No change to how dkt reads files: `markdown.BlankComments` keeps finding comments as OpenSpec does, so a file with no finding reads the same as today.
- No report of an opener outside a code span that is never closed and does not start a line. Markdown reads it as text, but that follows from Markdown's rules for where a comment may start and end, not from code spans; it is a separate idea if wanted.
- No code span that runs across lines: spans are found within a line, as `markdown.CodeSpans` and the placeholder rule find them. An opener in such a span is not reported, though Markdown reads it as code and dkt hides what follows.
- No other difference between dkt's comments and Markdown's, such as an empty comment written as an opener directly followed by the closing angle bracket, a closer written with an exclamation mark, or a backslash before the opener.
- No check of `docket/workflow.md`, and no new check in `dkt inspect`, as for `fence-in-comment`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `change-lint`: a new requirement, Comment opener inside a code span, adds the `comment-in-code-span` rule; the requirement Fence marker inside an HTML comment describes the comment opener of two scenarios in words.

## Impact

- `internal/markdown` gains a function that reads a file's comments as `BlankComments` does and, next to it, with code spans found first, and returns each opener where the two differ, with the line where dkt's comment ends.
- `internal/lint` applies it where it applies the fence rules: the Markdown files of a `docket` change, records, issues and the main specs.
- `openspec/specs/change-lint/spec.md` on archive.
- Tests in `internal/markdown` and `internal/lint`.
