## Context

`dkt lint` checks the plan of every active change in the worktree; archived changes are not linted. Other projects that upgrade Docket may have active changes whose plan has the five-column table with `reuse` rows and Evidence cells. The idea left two questions open: whether a plan that builds nothing needs the section, and how plans written before the change are linted.

## Goals / Non-Goals

**Goals:**
- Upgrading Docket never makes an active change's plan fail lint because of its Reuse table.

**Non-Goals:**
- Converting earlier tables; docket-doctor does not touch plans.

## Decisions

### The section and the table's header stay required; the table may be empty

A plan that builds nothing keeps the header with no rows, which the shipped template already has, so it costs the author nothing and says the question was asked. A missing section is more likely a skipped step than a deliberate "nothing new".

Alternatives: lint accepts a plan without the section (a forgotten table then passes unseen); the section holds a word such as `none` instead of a table (a second form for lint and authors to know).

### The earlier header is accepted by its columns

The table's header decides the rules: the four columns take the new rules; the five columns ending with Evidence take the same rules, except that Evidence is ignored and `reuse` is also a valid Decision, checked as `extend` is. The header message names the four columns, so a table with neither header is pointed at the current format.

Alternatives: accept only the new header (an upgrade fails every active change at once); accept `reuse` in any table (the four-column table would not say what it lists).

## Risks / Trade-offs

- [An author copies an earlier plan and keeps its five columns and `reuse` rows] → lint passes; plan review sees the rows. Dropping the earlier header later turns this into a finding.
- [Without Evidence, a `new` row's claim that nothing fits is not backed by anything in the plan] → reviewers check it, as the idea intends; the implementer's report of each exported symbol it added comes before the pre-merge review.

## Migration Plan

Nothing to run: active changes keep their tables, and new plans follow the updated instruction and template, which `dkt init` installs when it updates the schema.
