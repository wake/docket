---
title: Keep a project's workflow in step with the user's workflow template
needs: [docket/ideas/init-reuse-workflow.md]
---

`init-reuse-workflow` gives a user a workflow template of their own and lets docket-init set a project up from it. This idea is the part split off from it: once set up, a project can keep following that template, and `dkt inspect` tells the user when the two have drifted.

## Settled (maintainer, 2026-11-13)

Copied word for word from `docket/ideas/init-reuse-workflow.md`, where the maintainer settled them on 2026-11-13.

### Parameters

- `project-owned`, in the front matter of the project's file, committed: the headings of the sections the project decides on its own, which are never compared with the user template. Empty by default; when the field is absent, the list is empty, so Docket's template does not change and its `template` version stays.
- `default`, in the user's settings: `sync` or `independent`, the mode a new project gets. When it is unset, docket-init asks.
- `mode`, one per project, on this computer: `sync` follows the user template, `independent` does not. A project with no record is `independent`; a teammate's projects and older projects fall here.
- `synced`, one per project, on this computer, written by dkt: for each section, a hash of its content at the last sync, when the project's section and the user template's were the same.
- `mode` and `synced` stay out of the repository because they describe one user's template: committed, a teammate's check would compare the project's file with the teammate's own template and carry their habits into the shared file.

### Setting up

Step 2 of docket-init's setup, whose question about the mode belongs here: "The user template has the same `template` version: ask whether to use it. If yes, the project's file takes the user template's sections, and the user is asked for the mode. If no, go on as when there is none."

### Checking

`dkt inspect` reports and writes nothing:

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

- Where on this computer the per-project records live: in the project's entry in the index, or in the settings file. Leaning to the project's entry in the index, since `mode` and `synced` are one record per project, and the index already holds one entry per project.
- The dkt command docket-doctor runs to record `synced` once the user agrees, and how a section's content is normalized before hashing. Leaning to line endings made LF, trailing whitespace dropped from each line, and blank lines dropped at the start and end of the section, so that only edits a reader can see count as changes.

Split from `init-reuse-workflow` on 2026-11-15, when that idea proved too large for one change.
