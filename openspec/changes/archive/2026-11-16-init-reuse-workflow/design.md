## Context

The idea's "Settled (maintainer, 2026-11-13)" section is the design for what this change builds and is followed as written; its sync part moved to `docket/ideas/workflow-sync.md`. This change has two of the idea's open points to settle, how a reference on an older template version is compared and how many differences are shown, and the shape of the command that lists references.

The pieces that exist: `internal/projects` finds dkt's index and settings files under the XDG folders (`xdgFile`, `IndexPath`, `SettingsPath`) and loads the index; `dkt projects ls` works out which projects are missing; `internal/setup/workflow.go` reads a workflow file's template version and headings.

## Goals / Non-Goals

**Goals:**
- docket-init can start a project's workflow from the user template, or from the user's other projects, with dkt doing the reading and comparing and the agent doing the asking.

**Non-Goals:**
- dkt writing the user template or any project's workflow: the agent writes, once the user agrees.
- Keeping projects in step with the user template afterwards (`workflow-sync`).

## Decisions

### The user template's path sits beside the settings file

`projects.UserWorkflowPath(env)`, built with `xdgFile` like `SettingsPath`, so the two always share a folder. A symbolic link there is followed, as for the settings: the file is the user's own, and dotfile managers link it.

### `dkt inspect` reports the user template, and docket-doctor upgrades it

`result.user_workflow` holds `path`, `exists` and `version`, or null when there is no HOME. Three findings, `user-workflow-outdated`, `user-workflow-newer` and `user-workflow-template-unknown`, make inspect exit 1 so that docket-init and docket-upgrade send the user to docket-doctor; a missing user template is not a finding, since it is optional. A user template that cannot be read goes in `unreadable`, as every unreadable source does.

### `dkt workflow refs` lists the references; the agent presents them

A `workflow` command with one subcommand, `refs`, leaves room for the commands `workflow-sync` adds. It needs no repository; inside one, that project is left out, since it is the one being set up. The JSON result:

- `template`: the shipped template version.
- `sources`: each with `path`, `kind` (`user` or `project`), `template`, `missing` (shipped headings it lacks) and `left_out` (its headings the shipped template lacks).
- `sections`: for each shipped level-three heading in order, `heading`, `template` (the shipped text) and `variants`, each with `text`, `same_as_template` and `sources` (paths).

This settles both open points. A reference on another template version is compared on the headings it shares with the shipped template, and its `missing` and `left_out` say what was not compared. Many differences are presented per section, as the template's step and each variant with the sources that use it, so no whole file is shown; the text output lists each section's variants by their sources only, and the JSON carries the texts.

Reading other projects' files needs the user's consent, and that stays in docket-init's prompt: the command reads only when it is run.

### Sections and their comparison

A section is the text under a level-three heading up to the next heading of level three or above, found outside fenced code and HTML comments as `docket/workflow.md` headings are found today. Before comparing, line endings become LF, trailing whitespace is removed from each line, and blank lines are removed at the start and end, so that only edits a reader can see make a new variant. `workflow-sync` can reuse the same normalization for its hashes.

### docket-init and docket-doctor

docket-init's step 4 becomes the idea's setup steps 1 to 5, without the question about the mode, which belongs to `workflow-sync`. To use the user template, the agent copies it to `docket/workflow.md`; to save the user template, it writes the path `dkt inspect` reports. docket-doctor treats `user-workflow-outdated` as it treats `workflow-outdated`, on the user template, and asks for a newer dkt on `user-workflow-newer`.

## Risks / Trade-offs

- [A project's workflow text names its own tools or hosts, and docket-init offers it to another project] → the prompt has the agent point out what belongs to one project before the user saves a user template, as the idea says.
- [Many registered projects make the output long] → variants with the same text are grouped, so the output grows with the number of distinct ways, not with the number of projects.

## Migration Plan

Nothing to run: without a user template, `dkt inspect` reports none, and docket-init offers the references.
