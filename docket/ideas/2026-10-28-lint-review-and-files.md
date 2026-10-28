---
title: Lint the Review line and the Files block of each task
---

Recorded 2026-10-28 with `update-workflow`, which changed only the guidance
Docket ships: plans mark each task `**Review:** judgment` or `**Review:**
checks` and list every file in a `**Files:**` block, and parallel sessions
compare `**Files:**` blocks before starting. Nothing checks this yet. This
changes a spec, so it needs a change.

`dkt lint` already requires a `**Scenarios:**` line in every task (`change-
lint`). Extend it: every task that is not cancelled has one `**Review:**`
line whose value is `judgment` or `checks`, and a `**Files:**` block with at
least one path.

## Open questions

- Whether a plan written before the rule fails or only warns.
- Whether `judgment` should be required for some tasks, for example one
  whose `**Files:**` block touches a public interface.
