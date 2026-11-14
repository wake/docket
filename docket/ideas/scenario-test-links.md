---
title: Keep the link from a spec scenario to the tests that check it
---

During a change, `dkt lint` requires every scenario the deltas add or modify to be claimed by a task (`scenario-unclaimed`), and each task writes its tests first, so every scenario has tests when the change merges. After archiving, the plan is in the archive and the tests do not say which scenario they check, so the link is gone. A later fix, refactor or direct merge can change behavior a main spec defines, and nothing leads from the scenario to the test that should have caught it, or shows a scenario that no test checks any more.

Preferred direction (discussed 2026-11-15): tests name the scenarios they check with the reference the plan already uses, `<capability-path>#<scenario-slug>`, in a comment or in the test's name, and `dkt lint` reports main spec scenarios that no test names. Prose specs cannot run, but this is the closest they come to TDD's red and green: a scenario with no test is red.

Found in the discussion of how Docket relates to spec-driven development, 2026-11-15.

## Open questions

- How dkt finds the tests in a project of any language: a setting naming the test files, or a search of the whole worktree.
- What it reports for a reference to a scenario that was renamed or removed.
- How a project with many main specs adopts the rule: an opt-in, or a pass that adds the references first.
- How to mark scenarios no automated test can check, such as prompt text checked by hand.
