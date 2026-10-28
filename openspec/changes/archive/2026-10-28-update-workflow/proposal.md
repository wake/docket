# Proposal

## Why

The workflow Docket ships puts a review gate on every task and leaves pull request size to outside rules. On `add-status` (545 lines of product code) that meant 22 subagent dispatches, 13 cross-model review runs and 4 stacked pull requests. Only three gates found real problems: the review of the plan before coding, the mechanical checks on each task (a failing test replayed, the changed lines mutated), and one adversarial review of the whole change before merging. Established methodology points the same way: cheap checks run on every step, expensive review is batched, and units are cut by concern, not by line count. The shipped guidance should say so before the next change runs under it.

## What Changes

- The "Change workflow" section of `AGENTS.md`, which `dkt init` prints for other projects, gets three triage paths (open a change; fix or restructure with one review before merging; merge docs and bumps after the automated checks), review at the two ends of a change, per-task review only for tasks that need judgment, one pull request per change, the cases that justify a split, and rules for sessions that work in parallel.
- The `plan` instruction of the `docket` schema: a task is a slice that can be verified on its own, tightly coupled tasks are merged, every task has a `**Review:**` line (`judgment` or `checks`), and its `**Files:**` block lists every file it touches.
- The plan template gets the `**Review:**` line.
- The `apply` instruction: only `**Review:** judgment` tasks get their own reviewer, every task gets the mechanical checks, and the whole change is reviewed once before merging.
- The workflow steps in `README.md` and `README.zh-TW.md` follow suit.
- A follow-up note in `docs/ideas/` for the checks this change does not add.

## Non-Goals

- No lint rule for the `**Review:**` line or the `**Files:**` block; they are guidance in this change (see the follow-up note).
- No file-overlap check in `dkt conflicts` and no review metrics in `evidence.md`.
- No names of models, review products or local tools in the shipped text.
- No change to how `dkt init` installs or prints anything; only the text it prints changes.

## Builds On

- `schema.go:AgentsSection`, which cuts the "Change workflow" section out of `AGENTS.md` for `dkt init` to print; its heading stays the same.
- `openspec/schemas/docket/schema.yaml` (the `plan` and `apply` instructions) and `openspec/schemas/docket/templates/plan.md`, embedded by `schema.go:Schema`.
- capability `change-lint`: the `**Scenarios:**` line rule and the leftover-template rule, which treats an angle-bracket placeholder left from a template as a finding. The new `**Review:**` placeholder relies on that rule to be filled in.
- capability `project-setup`: `dkt init` prints the recommended routing section; its requirements do not describe the text.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None: no requirement in `openspec/specs/` describes the wording of the shipped guidance, so this change sets `skip_specs: true`.

## Impact

`AGENTS.md`, `openspec/schemas/docket/schema.yaml`, `openspec/schemas/docket/templates/plan.md`, `schema_test.go`, both READMEs and a new file under `docs/ideas/`. Projects that already ran `dkt init` see the template as differing from the shipped one until they run `dkt init --force`.

## Implementation Contract

### Observable Behavior

`dkt init` prints the new "Change workflow" section and installs the new plan template and schema instructions. `openspec instructions plan` and `openspec instructions apply` return the new instructions. A plan written from the new template that still holds `<checks|judgment>` gets a `placeholder` finding from `dkt lint`.

### Interfaces

- The section heading `## Change workflow (superpowers × OpenSpec)` is unchanged.
- New plan line per task: `**Review:** judgment` or `**Review:** checks`, after the `**Scenarios:**` line.
- No command, flag or output field changes.

### Failure Modes

- A project with the old template installed keeps it, and `dkt init` reports the file as differing, as it does today; `--force` replaces it.
- `dkt lint` and `dkt sync` must keep reading a plan that has the new `**Review:**` line; this change's own plan uses it.

### Acceptance Criteria

- `go test ./...` passes with tests that the plan template carries the `**Review:**` line and that the shipped section names the three triage paths and the review before merging (no delta scenario: `skip_specs`).
- `dkt lint update-workflow` and `dkt sync update-workflow --check` pass on this change's own plan, which uses the new line (no delta scenario: `skip_specs`).
- `openspec validate update-workflow --strict` passes (no delta scenario: `skip_specs`).

## Change Log

| Date | Source | Change | Impact |
|---|---|---|---|
