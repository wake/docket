## Context

The idea's "Settled (maintainer, 2026-11-13)" section is the design for what this change builds and is followed as written. Two open points are settled here: where the per-project records live, and the command that records them together with how a section is normalized before hashing.

What `init-reuse-workflow` left: `projects.UserWorkflowPath` locates the user template beside the settings; `setup.TemplateVersion` reads a workflow file's version; `setup.Sections` gives the level-three sections with their text normalized (LF, no trailing whitespace, no blank lines at either end); `dkt inspect` reports `result.user_workflow`; `dkt workflow` has the subcommand `refs`. The project index (`internal/projects`) holds one entry per project and is written only by dkt; the settings file is never written by dkt.

## Goals / Non-Goals

**Goals:**
- dkt compares and records; the agent asks the user and writes the workflow files, as in `init-reuse-workflow`.

**Non-Goals:**
- A record of which side of a conflict the user chose: the next `synced` hash is all that is kept.

## Decisions

### `mode` and `synced` live in the project's index entry

The index is dkt's own state, under the state home, with one entry per project, and dkt already rewrites it; the settings file is the user's, and dkt never writes it. So both go on the entry, as optional fields `mode` and `synced`, which the index keeps on every write. A project with no entry, or an entry without `mode`, is `independent`, which covers a teammate's projects and older ones. `workflow.default` stays in the settings, since the user sets it once for all projects.

Considered: a separate file per project under the state home. Rejected: a second place to keep in step with the index when a project is removed or moves.

### One command records both: `dkt workflow mode <sync|independent>`

docket-init runs it once the user picks a mode; docket-doctor runs `dkt workflow mode sync` again after every write the user agreed to. In `sync` mode it hashes each compared section whose two texts are the same variant and keeps the earlier hash of the others; `independent` drops `synced`, so switching back starts from what the two files share then. dkt checks the texts itself, so a hash is only ever one both files had, whatever the agent did.

Considered: a separate `dkt workflow synced <heading>` that records named sections. Rejected: the agent would have to name the right sections, and a hash recorded for texts that differ would hide a change.

### Normalizing and hashing

A section's text is normalized as `user-workflow`'s Same variant says, which `setup.Sections` already does, and hashed with SHA-256 in lowercase hexadecimal. Only edits a reader can see make a section change.

### The section states

| Texts | `synced` hash | State |
|---|---|---|
| the same variant | any, or none | `in-step` |
| differ | the project's text | `template-changed` |
| differ | the user template's text | `project-changed` |
| differ | neither, or none | `conflict` |

An old hash under texts that are the same again is `in-step`: nothing is to be done, and the next `dkt workflow mode sync` records it. With no hash there is no telling which side moved, so the user decides, as for a conflict.

Only sections both files have at level three, outside `project-owned`, are compared, and only when the two files have the same template version: a version gap is upgraded first, through the existing `workflow-outdated` and `user-workflow-outdated` findings, so that an upgrade does not show up as many changed sections.

### Which copy of the project's file

`dkt inspect` and `dkt workflow mode` read `docket/workflow.md` in the worktree they run in, as `dkt inspect` reads every file, and use the entry of the project that worktree belongs to.

### `project-owned` is written like an idea's `needs`

`project-owned: [Where to work, How to merge]`, read with the existing front matter reader and split on commas. A value that is not a bracketed list is `workflow-project-owned-invalid`, and nothing is compared until it is fixed, so that a typo never offers to overwrite a section the project meant to own.

### What `dkt inspect` reports

`result.user_workflow` gains `default` (from the settings, or null), `mode` and `sections` (`heading`, `state`), and a finding for each section not `in-step`, all naming `docket/workflow.md` so that docket-doctor finds them with the rest of the file's findings. Inspect now reads the index and the settings when there is a home, and lists either in `unreadable` when it cannot read it.

### docket-init and docket-doctor

docket-init, once the user agrees to take the user template, proposes `workflow.default` when it is set and otherwise asks for the mode, then runs `dkt workflow mode`. docket-doctor handles the three findings: `workflow-sync-template-changed`, offer to take the user template's section; `workflow-sync-project-changed`, offer to add the heading to `project-owned`, or to copy the project's section into the user template; `workflow-sync-conflict`, show both and let the user pick or own the section. After each agreed write it runs `dkt workflow mode sync` and inspects again.

## Risks / Trade-offs

- [An older dkt that rewrites the index drops `mode` and `synced`] → the project falls back to `independent`, which compares nothing and writes nothing; running `dkt workflow mode sync` again restores it.
- [A section edited on a branch and recorded there makes the main checkout's copy look `project-changed` until the branch is merged] → the record follows the files the user is working on; once merged, the two agree again.
- [A broken index or settings file now makes `dkt inspect` exit with 3] → it names the file, as for every source it cannot read, and registration already warns about the same file.
- [A heading holding a comma cannot be listed in `project-owned`] → none of the template's headings has one; a quoted form can come when one does.

## Migration Plan

Nothing to run: every existing project has no `mode`, so it is `independent` and `dkt inspect` reports what it reports today, with `mode` `independent` and no sections.
