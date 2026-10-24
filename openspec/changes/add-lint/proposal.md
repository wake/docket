# Proposal

## Why

The rules that make one task list and a clean handoff work are only advice
to an agent today: the sections a proposal must have, task headings that
superpowers can execute, every scenario claimed by a task, a reason for every
cancelled task, designs written into the change folder instead of somewhere
else. When an agent skips one, nothing fails. `dkt lint` turns each of these
rules into an error a person, CI job or agent can see.

## What Changes

- Readers for a change folder: proposal.md, the delta specs and plan.md.
  The delta reader follows OpenSpec 1.14's own parsing rules.
- `dkt lint [<slug>...]` with checks for proposal sections, the Change Log,
  leftover template text, task headings, the reuse table, scenario
  references, duplicate scenario names, cancelled and compensating tasks,
  plans and designs written outside the change folder, and delta structure
  OpenSpec would not apply.
- A contract test comparing Docket's reading of deltas with
  `openspec show --json --deltas-only` on a shared set of examples.

## Non-Goals

- Does not judge whether content is right, complete or well written.
- Does not run `openspec validate`; `dkt archive` does that.
- Does not check tests against scenarios; that is `dkt check`, a later
  milestone.
- Does not fix what it reports.
- Does not lint changes in other worktrees; each worktree lints its own.

## Builds On

- From `add-init-and-new`: the result envelope and exit codes
  (`cli-output`), the default branch and merge base (`workspace-scan`), and
  the embedded schema templates (`project-setup`), which double as the list
  of placeholders.
- OpenSpec 1.14.3's parsing rules, ported rather than called
  (`src/core/parsers/requirement-blocks.ts`, `code-fence.ts`), with
  `openspec show --json` as the reference in the contract test.

## Capabilities

### New Capabilities

- `change-lint`: the structural rules `dkt lint` enforces on a change folder,
  and how it reports them.

### Modified Capabilities

None.

## Impact

- New readers that `add-conflicts` and `add-archive` reuse.
- The contract test runs in the existing CI job that has Node.js and
  OpenSpec 1.14.3; a new CI job runs `dkt lint` on this repository.
- `dkt lint` needs no OpenSpec CLI at run time.

## Implementation Contract

### Observable Behavior

- `dkt lint` prints one finding per broken rule, each with its rule, file,
  line and message, and exits 1 if there are any, 0 if there are none.
- A change with no plan.md yet is linted on its proposal and deltas only.

### Interfaces

- `dkt lint [<slug>...] [--json]`
- Rule identifiers in findings: `proposal-section`, `change-log`,
  `placeholder`, `task-heading`, `reuse-table`, `scenario-ref`,
  `scenario-unclaimed`, `scenario-duplicate`, `cancelled-task`,
  `compensates`, `routing`, `unparseable`, `change-missing`.
- Scenario reference format: `<capability-path>#<scenario-slug>`.

### Failure Modes

- A named change that does not exist in the worktree: a finding, exit 1.
- A delta Docket cannot classify: an `unparseable` finding, never a skip.
- A file that cannot be read: reported as unreadable, exit 3.

### Acceptance Criteria

- Scope: `change-lint#lint-every-active-change`,
  `change-lint#named-change-missing`, `change-lint#no-openspec-installed`.
- Proposal: `change-lint#non-goals-missing`, `change-lint#empty-failure-modes`,
  `change-lint#builds-on-left-empty`,
  `change-lint#change-log-row-without-source`, `change-lint#header-only-change-log`,
  `change-lint#template-comment-left`.
- Plan: `change-lint#plan-not-written-yet`, `change-lint#task-heading-at-the-wrong-level`,
  `change-lint#duplicate-task-number`, `change-lint#reuse-row-with-a-missing-path`,
  `change-lint#new-row-without-a-reason`, `change-lint#external-dependency-row`,
  `change-lint#empty-reuse-table`,
  `change-lint#row-without-evidence`, `change-lint#cancelled-task-without-a-reason`,
  `change-lint#compensating-a-cancelled-task`.
- Scenarios: `change-lint#scenario-nobody-claims`, `change-lint#reference-to-an-unknown-scenario`,
  `change-lint#scaffolding-task`, `change-lint#two-scenarios-named-success`.
- Routing: `change-lint#design-written-to-the-superpowers-folder`,
  `change-lint#no-default-branch-for-routing`.
- Structure: `change-lint#requirement-outside-any-delta-section`,
  `change-lint#header-inside-a-code-block`, `change-lint#unpaired-rename`,
  `change-lint#examples-agree-with-openspec`.

## Change Log

| Date | Source | Change | Impact |
|---|---|---|---|
| 2026-10-24 | self: writing the add-init-and-new plan | Reuse rows may name an outside dependency as `external: <name>`; lint skips the path check for them | One requirement sentence and one scenario; no task exists yet |
| 2026-10-24 | self: writing the add-lint plan | The `.openspec.yaml` reader moves to `add-archive`, its first user, since no lint rule reads it; a missing named change is reported with the rule `change-missing`; the contract test runs in the existing `openspec` CI job, gated by `DOCKET_TEST_OPENSPEC` like the other real-CLI tests, and a new job runs `dkt lint` | No spec sentence changes |
| 2026-10-24 | review: whole-branch review of add-lint | A delta that renames and modifies one requirement no longer reports its scenarios as duplicates; comments are read the same way in proposals and deltas and never across fenced code | No spec sentence changes; plan Revision 1 |
