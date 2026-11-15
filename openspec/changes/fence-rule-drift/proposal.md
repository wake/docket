---
title: "Report a backtick fence line that OpenSpec 1.16 and later read as inline code"
kind: feature
source: "docket/ideas/fence-rule-drift.md"
---

# Proposal

## Why

Since OpenSpec 1.16.0, a line that starts with three or more backticks and holds another backtick after them, such as ```` ```npm test``` ````, is inline code, not the start of fenced code, as CommonMark says. OpenSpec 1.14 and dkt read it as an opener and read every line after it as code until a closing fence, so under OpenSpec 1.16 or later the specs OpenSpec archives and the requirements, steps and headings dkt reads differ without a word. dkt supports OpenSpec 1.14.0 and later, so it cannot read such a line the way every supported version does; reporting it lets the author rewrite it so that every version reads the file alike.

## What Changes

- `dkt lint` reports `fence-info-backtick`, naming the file and line, for a line that dkt reads as the start of fenced code when the line is a backtick fence whose text after the backticks holds a backtick. It checks the files `fence-in-comment` checks: every Markdown file of a `docket` change, each record and issue it lints, and, when no change is named, each main spec whose title it checks.
- A tilde fence whose text holds a backtick is not reported: every version reads it as an opener.

## Non-Goals

- No change to how dkt reads files: `markdown.FenceMask` keeps the fence rule of OpenSpec 1.14.3, so a file with no finding reads the same as today.
- No adoption of the newer rule and no raise of the minimum OpenSpec version; both were weighed in the idea and set aside.
- No check of `docket/workflow.md`, as for `fence-in-comment`.
- No new check in `dkt inspect`.
- No handling of the readers in OpenSpec that use a simpler fence rule (`extractFirstPurposeLine` in `core/references.js`, `extractPurposeSection` in `core/specs-apply.js`); see design.md.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `change-lint`: a new requirement, Backtick in a backtick fence's info string, adds the `fence-info-backtick` rule.

## Impact

- `internal/markdown` gains a function that finds, in a file's lines, each line `FenceMask` reads as an opener that OpenSpec 1.16 and later do not.
- `internal/lint` applies it where it applies `FencesInComments`: the Markdown files of a `docket` change, records, issues and the main specs.
- Tests in `internal/markdown` and `internal/lint`.
