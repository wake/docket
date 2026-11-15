# Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** The Reuse table lists only what the change builds (`extend`, `new`) in four columns without Evidence, may be empty, and the earlier five-column table still passes lint.

**Architecture:** `checkReuse` in `internal/lint/plan.go` picks its rules from the table's header: the four-column header, or the earlier five-column one, under which Evidence is ignored and `reuse` is a valid Decision. The schema's plan instruction and the plan template describe the four-column table.

**Tech Stack:** Go; the shipped schema and template, installed by `dkt init`.

**Spec:** `proposal.md`, `specs/change-lint/spec.md` and `design.md` in this change folder.

## Global Constraints

- The four-column header is exactly `| Need | Existing (path:symbol) | Decision | Reason |`; the earlier one is the same followed by `Evidence`. Headers match without regard to case, as today.
- Every finding about the table keeps the rule `reuse-table`. When the header is neither, the message names the four columns.
- No new exported symbols.

## Review Focus

- A five-column table whose last column is not Evidence (for example `Notes`): a header finding, not the earlier rules.
- A table with the four columns plus Evidence in another position, or an extra sixth column: a header finding.
- An earlier five-column table with no rows: accepted, like an empty four-column table.
- A Decision written in backticks (`` `extend` ``) or upper case: treated as today, backticks stripped, no other change.
- The plan instruction and the template must agree on the header, since authors copy one or the other.

## Reuse

| Need | Existing (path:symbol) | Decision | Reason |
|---|---|---|---|
| Rules for the four-column and the earlier table | `internal/lint/plan.go:checkReuse` | extend | The table's checks live here; the header picks which rules apply |

---

### Task 1: Lint takes the four-column table and the earlier one

**Scenarios:** change-lint#reuse-row-with-a-missing-path, change-lint#external-dependency-row, change-lint#empty-reuse-table, change-lint#reuse-decision-in-the-table, change-lint#earlier-table-with-an-empty-evidence, change-lint#reuse-row-in-the-earlier-table-with-a-missing-path

**Review:** checks

**Files:**
- Modify: `internal/lint/plan.go`
- Modify: `internal/lint/plan_test.go`
- Modify: `internal/cli/realopenspec_test.go`

- [x] A four-column table with no rows, and an earlier five-column table with no rows, give no finding.
- [x] In the four-column table, a `reuse` row gives `reuse-table` on that row; an `extend` row with a missing path gives `reuse-table`; `external:` and a `new` row with a Reason give nothing; a `new` row without a Reason gives `reuse-table`.
- [x] In the earlier table, with every Evidence cell empty, each row case of the item above gives the same result as in the four-column table, except that a `reuse` row is accepted there: naming an existing path or `external:` it gives nothing, naming a missing path it gives `reuse-table`.
- [x] Any other header, including five columns that do not end with Evidence, gives one `reuse-table` finding on the header line whose message lists the four columns.
- [x] The tests in `internal/lint/plan_test.go` use the four-column table, with the earlier table kept only in tests about it; the sample plan in `internal/cli/realopenspec_test.go` uses the four-column table.

### Task 2: The plan instruction and the template describe the four-column table

**Scenarios:** none

**Review:** checks

**Files:**
- Modify: `shipped/schema/schema.yaml`
- Modify: `shipped/schema/templates/plan.md`
- Modify: `openspec/schemas/docket/schema.yaml`
- Modify: `openspec/schemas/docket/templates/plan.md`
- Modify: `internal/setup/history.go`

- [x] The plan instruction's Reuse paragraph says the table lists what the change builds, gives the four-column header, says when a row is `extend` or `new` and that the table may be empty, and no longer mentions Evidence or `reuse` rows.
- [x] The template's table header is the four columns.
- [x] The copies under `openspec/schemas/docket/` match `shipped/`, installed by this worktree's own build with `init --force`.
- [x] `internal/setup/history.go` holds the new hashes under v0.27.0, and `go test ./...` passes, `TestEarlierHoldsShippedFiles` included.
- [x] `openspec instructions plan --change reuse-table-slim` prints the four-column template, and `dkt lint reuse-table-slim` from this worktree's build reports nothing.
