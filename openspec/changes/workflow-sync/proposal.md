---
title: "Keep a project's workflow in step with the user's workflow template"
kind: feature
source: "docket/ideas/workflow-sync.md"
---

## Why

`init-reuse-workflow` lets docket-init set a project's `docket/workflow.md` up from the user's own workflow template, but once set up the two drift apart: a step the user improves in the template stays old in every project, and nothing tells the user. The maintainer settled on 2026-11-13 how a project can keep following the user template, section by section, with `dkt inspect` reporting the drift and docket-doctor resolving it with the user.

## What Changes

- A project's `docket/workflow.md` can list in its front matter, as `project-owned`, the headings of the sections the project decides on its own; they are never compared with the user template. Absent means empty, so Docket's template and its `template` version do not change.
- The user's settings can hold `workflow.default`, `sync` or `independent`: the mode docket-init proposes for a new project.
- Each project's entry in the project index on this computer can hold `mode` (`sync` or `independent`) and `synced` (per section, a hash of the text the project and the user template last shared). A project with no `mode` is `independent`. Neither goes into the repository.
- `dkt inspect` reports the default mode and the project's mode, and in `sync` mode compares each section that is not project-owned with the user template and its `synced` hash, reporting one of four states: in step, the template changed, the project changed, or a conflict. It still writes nothing.
- A new command, `dkt workflow mode <sync|independent>`, records the project's mode; in `sync` mode it records `synced` for the sections the project and the user template now share. It is the only writer of `mode` and `synced`.
- docket-init asks for the mode, or proposes the default, after the project takes the user template, and records it once the user agrees. docket-doctor acts on the section states: it asks the user, writes only once they agree, and records the new `synced`.

## Non-Goals

- Any change to Docket's workflow template or its `template` version.
- Writing a project's `docket/workflow.md` or the user template from dkt: the agent writes them, only once the user agrees.
- Syncing on its own: nothing is compared or written unless `dkt inspect` or `dkt workflow mode` is run.
- Writing the settings file from dkt: `workflow.default` is set by the user.
- Comparing sections when the project and the user template follow different template versions: upgrading comes first.

## Capabilities

### New Capabilities

### Modified Capabilities

- `user-workflow`: the `project-owned`, `workflow.default`, `mode` and `synced` parameters, the section comparison, and `dkt workflow mode`.
- `setup-inspection`: `dkt inspect` reports the modes and, in `sync` mode, each section's state.
- `project-index`: an entry can hold `mode` and `synced`.

## Impact

- `internal/setup`: reads `project-owned` and compares a project's sections with the user template's.
- `internal/projects`: the index entry's `mode` and `synced`, and `workflow.default` from the settings.
- `internal/cli`: `dkt inspect` reports the modes and section states; `dkt workflow mode`.
- Prompts: `skills/docket-init/SKILL.md` and `skills/docket-doctor/SKILL.md`.
