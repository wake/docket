---
title: Report file overlap in dkt conflicts
---

Recorded 2026-10-28 with `update-workflow`, which changed only the guidance
Docket ships: plans mark each task `**Review:** judgment` or `**Review:**
checks` and list every file in a `**Files:**` block, and parallel sessions
compare `**Files:**` blocks before starting. Nothing checks this yet. This
changes a spec, so it needs a change.

`dkt conflicts` reports changes that touch the same requirement (`conflict-
detection`). Two changes that touch no common requirement but edit the same
files pass, and today a session has to compare the plans by hand. Report
every path that appears in the `**Files:**` blocks of two or more active
changes, found the same way as the requirement overlaps.

## Open questions

- How to read a path with a note after it (`Modify: AGENTS.md (the "Change
  workflow" section)`) or a line range; probably the text inside the first
  pair of backticks.
- Whether a file overlap fails like a requirement overlap or is reported
  separately as a warning.
