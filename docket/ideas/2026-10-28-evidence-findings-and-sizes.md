---
title: Record review findings and sizes in evidence.md
---

Recorded 2026-10-28 with `update-workflow`, which changed only the guidance
Docket ships: plans mark each task `**Review:** judgment` or `**Review:**
checks` and list every file in a `**Files:**` block, and parallel sessions
compare `**Files:**` blocks before starting. Nothing checks this yet. This
changes a spec, so it needs a change.

Whether the workflow holds up can only be told from numbers kept per change:
lines of product code and tests in the change's range, the review findings
by severity and by reviewer, and which of them were fixed. `dkt archive`
already writes `evidence.md` from the range (`change-archive`), so the sizes
are mechanical. Findings need an input.

## Open questions

- The input: a `review.md` in the change folder in a fixed format, or a
  trailer on each fix commit naming the finding it fixes.
- Whether `dkt status` should show the same numbers before the change is
  archived.
