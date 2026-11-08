---
title: A fence marker inside an HTML comment hides the headings after it
---

dkt finds fenced code with `markdown.FenceMask` before it blanks HTML comments with `markdown.BlankComments`, so a fence marker inside a comment opens a fence that never closes. In

````markdown
<!--
```
-->
### Requirement: Real
````

the heading is outside both the comment and any code, yet it is treated as code and not read. Every reader that masks this way is affected: `change.ParsePlan` (tasks in `plan.md` and records), `dkt lint`, and the delta and main spec reader of `dkt snapshot` (`internal/specdoc`).

Found in the pre-merge review of the first pull request of `project-snapshot`, 2026-11-08.

## Open questions

- What OpenSpec itself does with a fence marker inside a comment, since `markdown.FenceMask` follows OpenSpec's fence rule; the fix should keep dkt reading files the way OpenSpec does.
- Whether a single pass that tracks comments and fences together, or blanking comments that lie outside fences before finding fences, is the smaller change.
