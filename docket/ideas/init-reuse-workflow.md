---
title: Use the user's other workflows when setting up a project
---

When docket-init sets up a project, `dkt init` writes `docket/workflow.md` from the template, and the agent then fits it to the user's instructions and habits found in its context. A user who already runs Docket in other projects has usually tuned these steps there (reviewers, verification, merging, the tools they name), and today none of that is looked at when a new project is set up.

The project index already knows those projects: `dkt projects ls --json` lists every registered one, with `ignored` and `missing`.

## Settled (maintainer, 2026-11-11)

- Only `docket/workflow.md` is covered.
- The template stays the standard: the new file starts from it and keeps every step, as the integration step does now.
- The other projects' `docket/workflow.md` files, when there are any, are references beside the user's instructions. Section by section, the agent compares them with the template and, where they differ, offers the ways those projects do the step, naming which projects do it which way; the user decides each.
- docket-init lists the other registered projects that are neither ignored nor missing and have a `docket/workflow.md`, and reads them only once the user agrees, as with user-level files today.
- What belongs only to a source project (its own tools, hosts, names) is pointed out and not carried over unless the user chooses it: the new project may be public when the source is not.
- Nothing is copied whole or kept in step: the result is the new project's own file, as there is no user-level workflow file by design.
- When `docket/workflow.md` already existed before `dkt init`, nothing changes: it holds the project's shared steps.

## Open

- A reference written for an older `template` version: compare only the sections whose headings match the current template, and say which ones were left out.
- How to present many differences without showing whole files: per section, the template's step and each variant with the projects that use it.
- Whether dkt needs anything; probably not, since the agent only reads files the index points to.
