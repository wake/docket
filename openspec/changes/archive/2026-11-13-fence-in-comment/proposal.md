---
title: "Report a fence marker inside an HTML comment that hides what follows"
kind: feature
source: "docket/ideas/fence-in-comment.md"
---

## Why

dkt and OpenSpec find fenced code before they look at HTML comments, so a fence marker inside a comment, such as a commented-out half of a code sample, opens code that runs past the comment's end. Every heading, task step and table row after it is then read as code: `dkt sync` does not tick the hidden steps, `dkt snapshot` does not list the hidden requirements, and `openspec archive` does not apply them, all without a word. Reading comments first would make dkt disagree with OpenSpec, which archives the specs; reporting the marker makes the silent case visible while both keep reading the file the same way.

## What Changes

- `dkt lint` reports `fence-in-comment`, naming the file and line, for a line inside an HTML comment that opens fenced code under the fence rule dkt shares with OpenSpec, when no line before the comment's end closes it. It checks every Markdown file of a `docket` change, each record and issue it lints, and, when no change is named, each main spec whose title it checks.
- A comment that holds a whole code block, its opening and closing fence lines both inside it, is not reported: it hides nothing.

## Non-Goals

- No change to how dkt reads files: fenced code is still found before HTML comments, as OpenSpec does, so the plan, the deltas and the main specs read as they do today.
- No check of `docket/workflow.md`: a heading hidden there is already reported as a missing section.
- No new check in `dkt inspect`, which checks only the main specs' titles.
- No adoption of the newer fence rule of OpenSpec 1.16, under which a backtick fence's info string may not hold a backtick; the check follows the fence rule dkt uses today.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `change-lint`: a new requirement, Fence marker inside an HTML comment, adds the `fence-in-comment` rule.

## Impact

- `internal/markdown` gains a function that finds, in a file's lines, each fence marker inside an HTML comment whose code would run past the comment.
- `internal/lint` applies it to the Markdown files of a `docket` change, to records, to issues and to the main specs, which lint reads once for both their titles and this rule.
- Tests in `internal/markdown/markdown_test.go` and `internal/lint`.
