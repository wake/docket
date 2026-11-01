## ADDED Requirements

### Requirement: Schema of each change
Each change SHALL give its `schema`. For a change of another schema, instead of the phase and task rules for `plan.md`, its steps SHALL be the checkboxes of its `tasks.md`, ticked and total, with no tasks listed and zero tasks done of zero; its phase SHALL be `design` when it has no `tasks.md` or no checkbox, `ready` when none is ticked, `done` when all are, and `implementing` otherwise.

#### Scenario: Change of another schema shown
- **WHEN** `add-x` has `schema: spec-driven` and a `tasks.md` with three checkboxes, one ticked, and no `plan.md`
- **THEN** `dkt status --json` gives `add-x` the schema `spec-driven`, the phase `implementing`, 1 of 3 steps, an empty task list and 0 of 0 tasks

#### Scenario: No tasks file
- **WHEN** `add-x` has `schema: spec-driven`, a `proposal.md` and no `tasks.md`
- **THEN** its phase is `design` and it has 0 of 0 steps

#### Scenario: Docket change
- **WHEN** `add-y` was made by `dkt new`
- **THEN** `dkt status --json` gives `add-y` the schema `docket`, and its phase and tasks come from its `plan.md`
