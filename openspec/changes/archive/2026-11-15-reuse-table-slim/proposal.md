---
title: "The Reuse table lists only what the change builds, without an Evidence column"
kind: change-request
source: "docket/ideas/reuse-table-slim.md"
---

## Why

A plan's Reuse table has a row for every capability the plan needs, and lint wants an Evidence cell, the search behind the decision, on every row. The rows that guard against building what the repository already has are `new` and `extend`; a `reuse` row only says that existing code is used, and an Evidence cell shows a search was run without letting a reviewer check it. In `fence-in-comment` that made eight rows for 222 lines of code. Whether something that exists was missed is a question for the reviewers, at plan review and before merging.

## What Changes

- The Reuse table has four columns, Need, Existing (path:symbol), Decision and Reason, and lists only what the change builds: Decision is `extend` (Existing names the code it builds on) or `new` (Reason says why nothing that exists fits). `dkt lint` reports a `reuse` row in this table.
- The table may have no rows, for a plan that builds nothing; the `## Reuse` section and the table's header stay required.
- A table with the earlier five columns, ending with Evidence, is still accepted, so a change opened before the upgrade keeps passing lint: its Evidence is not checked, and its `reuse` rows are checked as `extend` rows are.
- The plan instruction in the schema and the plan template describe the new table.

## Non-Goals

- No change to how Existing is checked: a path that must exist in the repository, or `external:`.
- No change to `docket/workflow.md`: plan review and the pre-merge review already cover whether something that exists was missed, and "Task execution" already asks the implementer to report each exported symbol it added.
- No date for dropping the earlier five-column table; that is a later decision.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `change-lint`: the Reuse table requirement takes the four-column table that lists only `extend` and `new` rows and may be empty; a new requirement accepts the earlier five-column table.

## Impact

- `internal/lint/plan.go` and its tests.
- `shipped/schema/schema.yaml` (the plan instruction) and `shipped/schema/templates/plan.md`, with their installed copies under `openspec/schemas/docket/` and their new hashes in `internal/setup/history.go`.
- `internal/cli/realopenspec_test.go`, whose sample plan uses the earlier table.
