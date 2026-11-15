---
title: "Set up a project's workflow from the user's own workflow template"
kind: feature
source: "docket/ideas/init-reuse-workflow.md"
specs:
  user-workflow: User Workflow
---

## Why

When docket-init sets up a project, the agent fits `docket/workflow.md` to the user's habits found in its context. A user who runs Docket in several projects on one computer is asked about the same steps every time, and has to gather again how the other projects do them. The maintainer settled on 2026-11-13 that the user keeps a workflow template of their own, and that a new project can start from it or from how the user's other projects do each step.

## What Changes

- The user template is one file beside dkt's settings, `$XDG_CONFIG_HOME/docket/workflow.md` or `~/.config/docket/workflow.md`, with the same headings and `template` field as Docket's template.
- `dkt inspect` reports where the user template is and its version: `user-workflow-outdated` when it is older than Docket's template, `user-workflow-newer` when it is newer than this dkt knows, and `user-workflow-template-unknown` when it has no template version. It writes nothing.
- A new command, `dkt workflow refs`, lists how the user's other registered projects, and the user template, do each section of Docket's template: the template's text and each variant with the sources that use it. A source written for another template version is compared only on the sections whose headings match, and the sections left out are named. It only reads.
- docket-init sets up `docket/workflow.md` from the user template when it has the same version, offers to upgrade an older one first, asks for a newer dkt when it is newer, and otherwise offers the other projects' ways section by section, once the user agrees to their being read; then it offers to save the result as the user template.
- docket-doctor upgrades an older user template as it upgrades a project's file.

## Non-Goals

- Following the user template after setup, and the `project-owned`, `default`, `mode` and `synced` parameters: `docket/ideas/workflow-sync.md`.
- Any change to Docket's workflow template or its `template` version.
- Any other file than the workflow: not `AGENTS.md`, `openspec/config.yaml`, `docket/domains.md` or the specs.
- Writing the user template from dkt: the agent writes it, only once the user agrees.

## Capabilities

### New Capabilities

- `user-workflow`: the user template's location and version, and the references `dkt workflow refs` lists.

### Modified Capabilities

- `setup-inspection`: `dkt inspect` reports the user template.

## Impact

- `internal/projects` gains the user template's path; `internal/setup` gives a workflow file's sections and version to other packages.
- `internal/cli`: `dkt inspect` reports the user template, and the new `dkt workflow` command.
- Prompts: `skills/docket-init/SKILL.md` and `skills/docket-doctor/SKILL.md`.
