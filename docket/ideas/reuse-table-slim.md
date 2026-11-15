---
title: The Reuse table asks for proof of every lookup, not only of what is built new
---

A plan's `## Reuse` table has a row for every capability the plan needs, and `dkt lint` (`reuse-table`) wants an Evidence cell, the search behind the decision, on every row. In `fence-in-comment` that was eight rows for 222 lines of code, most of them `reuse` rows naming a function the plan calls anyway.

The table exists to stop a change from building what the repository already has. The rows that carry that risk are `new` and `extend`; a `reuse` row says only that existing code is used. The Evidence cell asks the author to show a search was run, which a reviewer cannot check from the cell and has to repeat to trust.

Direction: the table lists only what the change builds or changes (`new`, `extend`), and the Evidence column goes; whether something that exists was missed is the reviewers' question, at plan review and before merging. This changes the `Reuse table` requirement of `change-lint`, the plan instruction in the schema and the plan template.

Found in the maintainer's review of the workflow, 2026-11-15.

## Open questions

- Whether a plan that builds nothing new still needs the section, or lint accepts a plan without it.
- How plans written before the change lint: accept both headers, or only check plans of open changes.
