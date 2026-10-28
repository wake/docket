# Shipped Workflow Update Implementation Plan

> **For agentic workers:** Execute task by task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** The workflow guidance Docket ships (the AGENTS.md section `dkt init` prints, the schema's `plan` and `apply` instructions, the plan template) and the READMEs describe review at the two ends of a change, per-task review only for tasks that need judgment, and one pull request per change.

**Architecture:** Text only. `AGENTS.md`, `schema.yaml` and the plan template are embedded by `schema.go`; tests in `schema_test.go` pin the new wording that other projects depend on.

**Tech Stack:** Go 1.24 tests; Markdown and YAML text.

**Spec:** `openspec/changes/update-workflow/proposal.md` (`skip_specs: true`: no requirement describes the wording).

## Global Constraints

- English only; no names of models, review products or local tools in shipped text.
- The heading `## Change workflow (superpowers × OpenSpec)` stays exactly as it is (`schema.go:agentsHeading`).
- The new plan line is `**Review:** judgment` or `**Review:** checks`, placed after `**Scenarios:**`; the template writes it as `**Review:** <checks|judgment>`.

## Review Focus

- A plan with the new `**Review:**` line must still parse: `dkt lint` and `dkt sync` read this change's own plan.
- The YAML block scalars in `schema.yaml` must keep their indentation, or `openspec instructions` fails for every project.
- The shipped section must still contain `dkt new` and must not contain `## Development` (`TestAgentsSection`).
- The guidance must not contradict a lint rule (task headings, `**Scenarios:**`, Reuse table, Cancelled Tasks).
- Both READMEs must say the same thing.

## Reuse

| Need | Existing (path:symbol) | Decision | Reason | Evidence |
|---|---|---|---|---|
| Cut the workflow section for `dkt init` | `schema.go:AgentsSection` | reuse | Unchanged; only the text it cuts changes | `rg -n "func AgentsSection" schema.go` |
| Ship the schema and templates | `schema.go:Schema` | reuse | Embeds the edited files as they are | `rg -n "go:embed" schema.go` |
| Flag a `**Review:**` placeholder left in a plan | `internal/lint/placeholder.go:checkPlaceholders` | reuse | Already reports any angle-bracket placeholder copied from a shipped template | `rg -n "func checkPlaceholders" internal/lint/` |
| Read tasks from plan.md | `internal/change/plan.go:ParsePlan` | reuse | Ignores lines it does not know, so the new line needs no parser change | `rg -n "func ParsePlan|scenariosPrefix" internal/change/plan.go` |

---

### Task 1: Shipped guidance

**Scenarios:** none

**Review:** checks

**Files:**
- Modify: `AGENTS.md` (the "Change workflow" section)
- Modify: `openspec/schemas/docket/schema.yaml` (`plan` and `apply` instructions)
- Modify: `openspec/schemas/docket/templates/plan.md`
- Test: `schema_test.go`

- [x] **Step 1: Write the failing tests.** In `schema_test.go`: `TestPlanTemplateReviewLine` reads `templates/plan.md` from `Schema()` and requires `**Review:** <checks|judgment>`; `TestAgentsSectionReviewAtBothEnds` requires the section to contain `**Review:** judgment`, `dkt conflicts` and `one pull request`.
- [x] **Step 2: Run them and see them fail.** `go test -run 'TestPlanTemplateReviewLine|TestAgentsSectionReviewAtBothEnds' .` fails on both.
- [x] **Step 3: Edit the text.** The AGENTS.md section: three triage paths, plan review before code, per-task review only for `**Review:** judgment`, mechanical checks on every task, one review of the whole change before merging, one pull request per change and when to split, parallel sessions. The `plan` instruction: slices, merging coupled tasks, the `**Review:**` line, the `**Files:**` block. The `apply` instruction: the same review rules. The plan template: the `**Review:**` line.
- [x] **Step 4: Run the tests.** `go test ./...` passes; `openspec instructions plan --change update-workflow` and `openspec instructions apply --change update-workflow` print the new text; `dkt init` in a temporary repository prints the new section.
- [x] **Step 5: Commit** with the trailers `Change: update-workflow` and `Task: 1`.

### Task 2: READMEs and follow-ups

**Scenarios:** none

**Review:** checks

**Files:**
- Modify: `README.md`, `README.zh-TW.md` (the workflow steps)
- Create: `docs/ideas/2026-10-28-workflow-followups.md`

- [x] **Step 1: Edit both READMEs.** Triage names the three paths; "Implement and review" names the mechanical checks, the judgment-only task review and the single review before merging; one pull request per change.
- [x] **Step 2: Write the follow-up note.** A lint rule for the `**Review:**` line and the `**Files:**` block; a file-overlap check for parallel changes; review findings and sizes recorded in `evidence.md`.
- [x] **Step 3: Commit** with the trailers `Change: update-workflow` and `Task: 2`.
