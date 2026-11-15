# Implementation Plan

**Goal:** A user workflow template beside dkt's settings, reported by `dkt inspect`; `dkt workflow refs` listing how the user's projects and the user template do each step; and docket-init and docket-doctor using both.

**Architecture:** `internal/projects` gains the user template's path, built like the settings path. `internal/setup` exports the workflow file's template version and its level-three sections, each section's text normalized for comparison. `dkt inspect` adds `result.user_workflow` and three findings; a new `dkt workflow refs` command groups the sources' sections into variants. The two prompts carry the setup and upgrade steps; dkt only reads.

**Tech Stack:** Go; the plugin's skills `docket-init` and `docket-doctor`.

**Spec:** `proposal.md`, `specs/user-workflow/spec.md`, `specs/setup-inspection/spec.md` and `design.md` in this change folder; the settled setup steps are in `docket/ideas/init-reuse-workflow.md`.

## Global Constraints

- The user template is `<config home>/docket/workflow.md`, with the config home chosen as for `docket/config.json`; a symbolic link there is followed.
- Finding rules: `user-workflow-outdated`, `user-workflow-newer`, `user-workflow-template-unknown`, each with `file` set to the user template's path.
- `dkt workflow refs` JSON result: `template`; `sources` (`path`, `kind` `user`|`project`, `template`, `missing`, `left_out`); `sections` (`heading`, `template`, `variants` with `text`, `same_as_template`, `sources`). Arrays are never null.
- A section's text: the lines under a level-three heading up to the next heading of level three or above, outside fenced code and HTML comments, with trailing whitespace removed from each line and blank lines removed at the start and end, joined with LF.
- A project's `docket/workflow.md` is read inside that project with `os.OpenRoot`, so that a symbolic link leading out of the project is refused and listed in `unreadable`; only the user template follows links.
- dkt writes no workflow file and no user template; the moved text of the prompts follows the idea's settled steps without the mode question.

## Review Focus

- Running `dkt workflow refs` in a worktree of a registered project: that project is left out, as when run in its main checkout.
- A source with no front matter: template 0, still compared on the headings it shares.
- A level-three heading written inside fenced code in a source: not a section.
- A source with the same level-three heading twice: only the first is compared.
- Must not happen: `dkt inspect` failing, or reporting a finding, when there is no user template or no HOME.

## Reuse

| Need | Existing (path:symbol) | Decision | Reason |
|---|---|---|---|
| `projects.UserWorkflowPath` | `internal/projects/files.go:xdgFile` | extend | Builds the path as `SettingsPath` does, so both share a folder |
| `setup.TemplateVersion` | `internal/setup/workflow.go:templateVersion` | extend | Exported for `dkt inspect` and `dkt workflow refs` |
| `setup.Sections` | `internal/setup/workflow.go:headingTexts` | extend | Same heading scan, keeping each section's text |
| `dkt workflow` command | `internal/cli/command.go:commands` | extend | Registered with the other commands |
| `result.user_workflow` | `internal/cli/cmd_inspect.go:inspectResult` | extend | Inspect's result gains the field |
| `missingProject`, whether a registered path is gone | `internal/cli/cmd_projects.go:projectsLs` | extend | Its missing check moves into this helper, which both commands call |
| The result types of `dkt workflow refs` | | new | No existing result lists a section's variants with their sources |

---

### Task 1: dkt inspect reports the user template

**Scenarios:** user-workflow#location-under-the-config-home, setup-inspection#user-template-reported, setup-inspection#older-user-template, setup-inspection#newer-user-template, setup-inspection#no-user-template

**Review:** checks

**Files:**
- Modify: `internal/projects/files.go`
- Modify: `internal/projects/files_test.go`
- Modify: `internal/setup/workflow.go`
- Modify: `internal/setup/workflow_test.go`
- Modify: `internal/cli/cmd_inspect.go`
- Modify: `internal/cli/cmd_inspect_test.go`

**Interfaces:**
- Produces: `projects.UserWorkflowPath(env []string) (string, error)`, returning `projects.ErrNoHome` as `SettingsPath` does; `setup.TemplateVersion(b []byte) int`, the former `templateVersion`.

- [ ] `UserWorkflowPath` gives `/c/docket/workflow.md` for `XDG_CONFIG_HOME=/c`, and `$HOME/.config/docket/workflow.md` when that variable is unset or relative.
- [ ] `dkt inspect --json` with `XDG_CONFIG_HOME=/c` reports `result.user_workflow.path` `/c/docket/workflow.md`; with a user template of template 3 it reports `exists` true and `version` 3 and no finding about it.
- [ ] A user template of template 2 gives `user-workflow-outdated`, one of a version above the shipped one gives `user-workflow-newer`, one with no front matter gives `user-workflow-template-unknown`, each naming the path, and dkt exits with 1.
- [ ] With no user template, `exists` is false, `version` is 0 and no finding is about it; with neither `XDG_CONFIG_HOME` nor `HOME`, `result.user_workflow` is null, there is no finding about it, and inspect still reports the rest.
- [ ] A user template that cannot be read is listed in `unreadable`, `result.user_workflow` still holds its path, and dkt exits with 3; a symbolic link to a readable file is followed.

### Task 2: dkt workflow refs lists the references

**Scenarios:** user-workflow#projects-that-do-a-step-the-same-way, user-workflow#only-whitespace-differs, user-workflow#link-out-of-a-project, user-workflow#projects-that-are-not-references, user-workflow#user-template-as-a-source, user-workflow#unknown-workflow-subcommand, user-workflow#source-on-an-older-template, user-workflow#one-project-s-workflow-cannot-be-read

**Review:** judgment

**Files:**
- Modify: `internal/setup/workflow.go`
- Modify: `internal/setup/workflow_test.go`
- Create: `internal/cli/cmd_workflow.go`
- Create: `internal/cli/cmd_workflow_test.go`
- Modify: `internal/cli/command.go`
- Modify: `internal/cli/cmd_projects.go`

**Interfaces:**
- Consumes: `projects.UserWorkflowPath`, `setup.TemplateVersion` (Task 1); `projects.Load` for the index; `projects.ProjectPath`, which maps a worktree to its project's main checkout, to leave out the repository the command runs in.
- Produces: `setup.Section{Heading, Text string}` and `setup.Sections(b []byte) []setup.Section`, level-three sections in file order with their normalized text, the first of a repeated heading only.

- [ ] `Sections` gives each level-three section's text normalized as Global Constraints say; a heading inside fenced code or an HTML comment is not a section; a repeated heading gives only its first section.
- [ ] With two projects sharing a "Plan review" text and a third differing, "Plan review" has two variants with those project paths; texts differing only by CRLF, trailing whitespace or blank lines at the ends are one variant, and a variant equal to the template's text has `same_as_template` true.
- [ ] An ignored project, a missing one, one without `docket/workflow.md`, and the repository the command runs in, reached from its main checkout or a worktree, are not sources; the user template is a source with kind `user` when it exists.
- [ ] A source on template 2 with a section the shipped template lacks and without "Reviewers" has template 2, that heading in `left_out`, "Reviewers" in `missing`, and contributes no variant from the extra section; a source with no front matter has template 0 and is still compared.
- [ ] An unreadable source is listed in `unreadable` while the other sources are listed, and dkt exits with 3; an index that cannot be read is listed in `unreadable` too; a project's `docket/workflow.md` that links out of the project is not read, is in no variant, and is listed in `unreadable`.
- [ ] `dkt workflow refs --json` outside a repository gives one result envelope with `repo` null, empty arrays rather than null in `findings`, `unreadable`, `sources`, `sections` and each `variants`, and exits with 0; it writes nothing in any project.
- [ ] `dkt workflow nosuch` and `dkt workflow` with no subcommand are usage errors with exit 2, and with `--json` give the `usage` finding; without `--json`, `dkt workflow refs` prints per section its heading and each variant's sources.
- [ ] `dkt projects ls` reports missing projects exactly as before, through `missingProject`.

### Task 3: docket-init and docket-doctor use the user template

**Scenarios:** none

**Review:** checks

**Files:**
- Modify: `skills/docket-init/SKILL.md`
- Modify: `skills/docket-doctor/SKILL.md`

- [ ] docket-init's step for a created `docket/workflow.md` follows the idea's settled setup steps 1 to 5 without the mode question: same version, asks to use the user template and copies it; older, offers the upgrade first and otherwise uses it as one more reference; newer, asks for a newer dkt; none, reads the other projects' ways through `dkt workflow refs` only once the user agrees, offers them section by section with the projects that use each, then offers to save the result as the user template at the path `dkt inspect` reports, pointing out first what belongs only to this project.
- [ ] In that step, Docket's template stays the starting point with every step and heading kept, the user's instructions and habits in context are still integrated as today, and the user template and the other projects are references beside them.
- [ ] When `docket/workflow.md` existed before `dkt init`, the step still leaves it unchanged and points to docket-doctor for adjustments.
- [ ] docket-doctor handles `user-workflow-outdated` as it handles `workflow-outdated`, on the user template and only once the user agrees, asks for a newer dkt on `user-workflow-newer`, and asks the user about `user-workflow-template-unknown`; the three go through its existing order: listed with the other problems, fixed one at a time with the user's agreement, then `dkt inspect` again.
- [ ] Every rule and command the two skills name exists in `internal/cli` (`dkt workflow refs`, the three finding rules).
