---
title: "Keep the customizable workflow steps in docket/workflow.md"
kind: change-request
source: "Maintainer request: move the customizable steps into a project workflow file"
specs:
  workflow-file: Project Workflow File
---

# Proposal

## Why

The docket skill mixes two things: the fixed workflow dkt depends on (the triage, the order of the steps, trailers, where plans go) and the steps a project should be able to do its own way (where to work, how to review and verify, when to archive). Those "Default steps" sit in the skill, which `dkt init` replaces on every upgrade, so a project cannot customize them without losing its changes, and an agent has to weigh them against the user's personal instructions each time.

## What Changes

- The docket skill keeps only the fixed workflow. Each customizable step names a section of the project's `docket/workflow.md` by its heading, for example "Plan review" in `docket/workflow.md`; the "Default steps" section and the "(required)" markers are gone, since every step left in the skill is fixed.
- A new shipped template, `docket/workflow.md`: front matter `template: 1`, then one section per customizable step, holding what the skill's default steps said.
- `dkt init` writes the template to `docket/workflow.md` when the file is missing, and never changes it once it exists, with or without `--force`: the file belongs to the project.
- `dkt init` and `dkt inspect` check the file: its template version against the shipped one, and the section headings the template has. They report `workflow-missing` (inspect only), `workflow-outdated`, `workflow-template-unknown` and `workflow-section-missing`; as with other findings, `dkt init`'s next steps then say to run docket-doctor.
- docket-init integrates the user's instructions and habits into a `docket/workflow.md` that `dkt init` just created, keeping the headings and the template version.
- This repository gets its own `docket/workflow.md`, which replaces the AGENTS.md rules about the default steps and archiving.

## Non-Goals

- docket-doctor: merging a newer template into an edited file, or adding a missing section, is its job; it is written later.
- Changelogs between template versions: version 1 is the first, so there is nothing to merge yet.
- Checking what a section says: only its heading is checked.
- Translating headings: the headings are the names the skill uses and stay as shipped.

## Capabilities

### New Capabilities
- `workflow-file`: the project's `docket/workflow.md`: its template version, its sections, when `dkt init` writes it, and how `dkt init` and `dkt inspect` report its state.

### Modified Capabilities
- `project-setup`: `dkt init` also writes `docket/workflow.md` when it is missing and lists it with kind `workflow`; the rules for existing files no longer apply to it.
- `setup-inspection`: `dkt inspect` lists `docket/workflow.md` with the statuses `workflow-file` defines.

## Impact

- New shipped file `shipped/workflow/workflow.md`; `shipped/skill/SKILL.md` and `skills/docket-init/SKILL.md` are rewritten, and `dkt init --force` refreshes the installed copies in this repository.
- `internal/setup`: a `workflow` kind that is created when missing and otherwise only read; `internal/cli`: the findings for it in `dkt init` and `dkt inspect`.
- Tests that pin each template version to its content, and that every section the skill names is a heading of the template.
- `AGENTS.md` points to `docket/workflow.md` instead of the default steps, and lists it among the prompt files.
