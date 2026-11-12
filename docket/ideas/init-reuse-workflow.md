---
title: Set up a project's workflow from the user's own workflow template
---

When docket-init sets up a project, `dkt init` writes `docket/workflow.md` from the template, and the agent then fits it to the user's instructions and habits found in its context. A user who runs Docket in several projects on one computer is asked about the same steps every time, and has to gather again how the other projects do them.

The project index already knows those projects: `dkt projects ls --json` lists every registered one, with `ignored` and `missing`.

## Settled (maintainer, 2026-11-13)

These replace the 2026-11-11 decisions where they differ: there is now a user-level workflow file, and a project can follow it.

### Three layers

- Docket's template, then the user template on this computer, then the project's `docket/workflow.md`.
- The user template is one file in the user's config folder, beside dkt's `config.json` (`$XDG_CONFIG_HOME/docket/workflow.md`, or `~/.config/docket/workflow.md`). It has the same headings and `template` field as Docket's template, and holds the user's habits, not one project's tools, hosts or names.
- Only the workflow is covered: not `AGENTS.md`, `openspec/config.yaml`, `docket/domains.md` or the specs.
- Docket's template stays the standard: every file keeps all its steps and headings.
- The project's file is always a whole copy, committed, and never reads the user template when it is used: teammates and CI do not have it.

### Parameters

- `project-owned`, in the front matter of the project's file, committed: the headings of the sections the project decides on its own, which are never compared with the user template. Empty by default; when the field is absent, the list is empty, so Docket's template does not change and its `template` version stays.
- `default`, in the user's settings: `sync` or `independent`, the mode a new project gets. When it is unset, docket-init asks.
- `mode`, one per project, on this computer: `sync` follows the user template, `independent` does not. A project with no record is `independent`; a teammate's projects and older projects fall here.
- `synced`, one per project, on this computer, written by dkt: for each section, a hash of its content at the last sync, when the project's section and the user template's were the same.
- `mode` and `synced` stay out of the repository because they describe one user's template: committed, a teammate's check would compare the project's file with the teammate's own template and carry their habits into the shared file.

### Setting up (docket-init)

1. Start from Docket's template, then look for the user template.
2. The user template has the same `template` version: ask whether to use it. If yes, the project's file takes the user template's sections, and the user is asked for the mode. If no, go on as when there is none.
3. The user template has an older `template` version: ask whether to upgrade it, as docket-doctor upgrades a project's file, then go on as in step 2. When it is not upgraded, go on as when there is none, with the old template as one more reference. When the user template is newer than Docket's, ask the user to upgrade dkt; never downgrade the template.
4. There is no user template: set up the project's file as today. The other registered projects that are neither ignored nor missing and have a `docket/workflow.md` are references beside the user's instructions, read only once the user agrees: section by section, offer the ways those projects do the step, naming which projects do it which way, and let the user decide each. Then ask whether to save the result as the user template; before saving, point out what belongs only to this project and leave it out unless the user chooses it.
5. When `docket/workflow.md` already existed before `dkt init`, nothing changes: it holds the project's shared steps.

### Checking

`dkt inspect` reports and writes nothing:

- When the user template's `template` version is older than Docket's, the user template is upgraded first.
- In `independent` mode, only the `template` version is checked, as today.
- In `sync` mode, each section not in `project-owned` is compared by its user template content, its project content and its `synced` hash:

  | User template | Project | Result |
  |---|---|---|
  | unchanged | unchanged | in step |
  | changed | unchanged | the new version can be taken |
  | unchanged | changed | the project's edit is kept; offer to add the section to `project-owned` |
  | changed | changed | a conflict, for the user to decide |

docket-doctor acts on the report: it asks the user, writes only once they agree, and dkt then records the new `synced`.

## Open

- Where on this computer the per-project records live: in the project's entry in the index, or in the settings file.
- The dkt command docket-doctor runs to record `synced` once the user agrees, and how a section's content is normalized before hashing (whitespace, line endings).
- A reference written for an older `template` version: compare only the sections whose headings match the current template, and say which ones were left out.
- How to present many differences without showing whole files: per section, the template's step and each variant with the projects that use it.
