---
title: Offer another project's workflow when setting up a project
---

When docket-init sets up a project, `dkt init` writes `docket/workflow.md` from the template, and the agent then fits it to the user's instructions and habits found in its context. A user who already runs Docket in other projects has usually tuned these steps there (reviewers, verification, merging, the tools they name), and today redoes the same integration in every new project, or copies the file by hand.

The project index already knows those projects: `dkt projects ls --json` lists every registered one, with `added`, `ignored` and `missing`.

Directions:

- docket-init: when `dkt init` reports `docket/workflow.md` as created, list the other registered projects that are neither ignored nor missing and have a `docket/workflow.md`, and ask the user whether to start from one of them instead of the template. Read another project's files only once the user agrees, as with user-level files today.
- Show the main points of the chosen file and where it differs from the template, as the integration step does now, and leave out what belongs only to the source project (its own tools, hosts, names) before writing: the new project may be public when the source is not.
- It is a one-time copy: from then on the new project owns its file, and nothing keeps the two in step, as there is no user-level workflow file by design.
- When `docket/workflow.md` already existed before `dkt init`, nothing changes: it holds the project's shared steps.

Open:

- What else can be offered besides `docket/workflow.md`: the `## Issues` section of `AGENTS.md` (where issues are filed), OpenSpec's `context` and `rules` in `openspec/config.yaml`. `docket/domains.md` and the specs belong to their project and are not offered.
- A source whose `template` value differs from the template `dkt init` installs: offer only sources at the same version, or copy one and bring it up to date with docket-doctor.
- With several candidates, how to order them: by `added`, by a shared origin host, or just by path.
- Whether the user's instructions from context are still integrated on top of the copied file, or only their differences from it are shown.
- Whether dkt helps, for example `dkt init --workflow-from <path>`, which would check the template version and refuse symbolic links, or the agent copies the file itself.
