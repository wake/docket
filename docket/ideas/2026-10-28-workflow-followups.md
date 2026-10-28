# Idea: checks behind the review-at-both-ends workflow

Status: idea, not designed. Recorded 2026-10-28 with `update-workflow`.

## Background

`update-workflow` changed only the guidance Docket ships. Plans now mark each
task `**Review:** judgment` or `**Review:** checks` and list every file in a
`**Files:**` block; review sits at the two ends of a change; parallel sessions
compare `**Files:**` blocks before starting. Nothing checks any of this yet.
Each item below changes a spec, so each needs a change of its own.

## 1. Lint the `**Review:**` line and the `**Files:**` block

`dkt lint` already requires a `**Scenarios:**` line in every task
(`change-lint`). Extend it: every task that is not cancelled has one
`**Review:**` line whose value is `judgment` or `checks`, and a `**Files:**`
block with at least one path.

Open questions:

- Whether a plan written before the rule fails or only warns.
- Whether `judgment` should be required for some tasks, for example one whose
  `**Files:**` block touches a public interface.

## 2. File overlap in `dkt conflicts`

`dkt conflicts` reports changes that touch the same requirement
(`conflict-detection`). Two changes that touch no common requirement but edit
the same files pass, and today a session has to compare the plans by hand.
Report every path that appears in the `**Files:**` blocks of two or more
active changes, found the same way as the requirement overlaps.

Open questions:

- How to read a path with a note after it (`Modify: AGENTS.md (the "Change
  workflow" section)`) or a line range; probably the text inside the first
  pair of backticks.
- Whether a file overlap fails like a requirement overlap or is reported
  separately as a warning.

## 3. Review findings and sizes in `evidence.md`

Whether the new workflow holds up can only be told from numbers kept per
change: lines of product code and tests in the change's range, the review
findings by severity and by reviewer, and which of them were fixed.
`dkt archive` already writes `evidence.md` from the range
(`change-archive`), so the sizes are mechanical. Findings need an input.

Open questions:

- The input: a `review.md` in the change folder in a fixed format, or a
  trailer on each fix commit naming the finding it fixes.
- Whether `dkt status` should show the same numbers before the change is
  archived.
