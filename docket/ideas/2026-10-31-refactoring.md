---
title: Refactoring in Docket
---

Recorded 2026-10-31, while `add-records` was being implemented. The
maintainer asked how Docket supports refactoring, and whether it belongs to
Docket or to superpowers. Whether Docket should do more is to be researched
when the idea is taken up; nothing below is decided.

## Where it stands

- How to restructure code is the agent's craft. superpowers has no
  refactoring skill of its own; its test-driven development skill has the
  refactor step of red, green, refactor.
- Docket covers the process:
  - Triage sends a restructuring that changes no behavior to light review.
    One that changes an interface others depend on changes behavior and
    goes to full review.
  - The workflow template's light review verifies it: the tests that cover
    the code pass before and after, and mutating the changed lines makes
    one fail. When none fails, the tests come first, as a test-only edit.
  - The workflow template's full review keeps structural and behavior
    changes in separate commits, and splits a pull request that mixes a
    structural refactor with behavior, the refactor first. The light review
    path says nothing about either.
  - `add-records` gives the work a record of kind `refactor`, optional tasks
    for a large one, and the `Record:` and `Task:` trailers.
  - Rules that keep new boundaries in place, such as an arch-go
    configuration, are check settings: kind `checks`.

## Leads

- Deterministic tools change code without an agent editing it by hand:
  `gopls` rename, extract and inline; `gopatch`; Comby; OpenRewrite for
  Java; `rsc.io/rf`. The prompt could prefer one where it fits, and have the
  command written in the record's Verification.
- `dkt` cannot tell a structural commit from a behavior commit without
  judging content, so any rule on them stays in prompt text and review.
- A large restructuring planned from the facts in
  `2026-10-31-codebase-decomposition.md` (which files change together) is
  where the two ideas meet.

## Related

- `2026-10-31-codebase-decomposition.md`
- `2026-10-29-records.md`

## Open questions

- Whether a refactor with tasks needs more than a record, such as an order
  of tasks that keeps the tests green after each one.
- Whether the `docket` skill or the workflow template should mention
  refactoring tools, and how, without naming one language's tools.
- Whether the light review path should also keep structural and behavior
  commits apart.
- How a refactor large enough to move capabilities between domains is
  triaged.
