# Implementation Plan

**Goal:** A project in `sync` mode follows the user's workflow template section by section: `dkt inspect` reports each section's state, `dkt workflow mode` records the mode and the `synced` hashes, and docket-init and docket-doctor ask the user before anything is written.

**Architecture:** `internal/setup` reads `project-owned` and compares the user template's and the project's level-three sections into the four states. `internal/projects` keeps `mode` and `synced` on the index entry and reads `workflow.default` from the settings. `dkt inspect` adds `default`, `mode` and `sections` to `result.user_workflow` with a finding per section not in step; `dkt workflow mode` is the only writer of `mode` and `synced`. The two prompts carry the asking.

**Tech Stack:** Go; the plugin's skills `docket-init` and `docket-doctor`.

**Spec:** `proposal.md`, `design.md` and the delta specs `specs/user-workflow/spec.md`, `specs/setup-inspection/spec.md` and `specs/project-index/spec.md` in this change folder; the settled design is in `docket/ideas/workflow-sync.md`.

## Global Constraints

- State names: `in-step`, `template-changed`, `project-changed`, `conflict`. Finding rules: `workflow-sync-template-changed`, `workflow-sync-project-changed`, `workflow-sync-conflict`, `workflow-project-owned-invalid` (each with `file` `docket/workflow.md`), `user-workflow-missing` (with `file` the user template's path).
- A hash is the SHA-256, in lowercase hexadecimal, of a section's text as `setup.Sections` gives it.
- Index entry: `mode` (`sync` or `independent`) and `synced` (heading to hash), both left out of the JSON when unset. Settings: `{"workflow": {"default": "sync" | "independent"}}`.
- `result.user_workflow`: `path`, `exists`, `version`, `default` (string or null), `mode` (`sync` or `independent`), `sections` (array of `heading`, `state`; never null). `dkt workflow mode` result: `path`, `mode`, `synced` (headings; never null).
- dkt writes no workflow file, no user template and no settings file; only `dkt workflow mode` writes `mode` and `synced`.

## Review Focus

- `dkt inspect` writes nothing beyond registration, in `sync` mode with sections to report too.
- A hash in `synced` is only ever one of a text both files had when it was recorded.
- `dkt projects add`, `rm`, `ignore` and automatic registration keep another entry's `mode` and `synced`.
- `dkt inspect` with no home: `result.user_workflow` null, the index and the settings not read, no failure.
- `dkt workflow mode` in a linked worktree: reads that worktree's `docket/workflow.md` and writes the entry of the main checkout's path.

## Reuse

| Need | Existing (path:symbol) | Decision | Reason |
|---|---|---|---|
| `setup.ProjectOwned` | `internal/change/frontmatter.go:ReadFrontMatter` | extend | Reads the field with the front matter reader, as an idea's `needs` is read |
| `setup.CompareSections` and its states | `internal/setup/workflow.go:Sections` | extend | Same sections and normalized text, compared across two files |
| `mode` and `synced` on an entry | `internal/projects/files.go:Entry` | extend | Optional fields on the existing entry, checked in `Load` |
| `projects.DefaultMode` | `internal/projects/files.go:AutoRegister` | extend | Reads the settings file the same way, another key |
| `result.user_workflow` default, mode, sections | `internal/cli/cmd_inspect.go:inspectUserWorkflow` | extend | Inspect's user template report gains the fields and findings |
| `dkt workflow mode` | `internal/cli/cmd_workflow.go:workflowCommand` | extend | Second subcommand of the existing command |
| The `not-a-repository` and `not-a-project` findings for a directory | `internal/cli/cmd_projects.go:projectsAdd` | extend | Its mapping of `projects.Locate`'s results to findings moves into a helper both commands call |
| docket-init's mode question | `skills/docket-init/SKILL.md` | extend | Step 4 gains the mode once the user template is taken |
| docket-doctor's section states | `skills/docket-doctor/SKILL.md` | extend | The three findings join its list of problems |

## Revision 1 (2026-11-16): the inspect fields became a requirement of their own

Found by the author while archiving: modifying `setup-inspection`'s "User template inspected" to hold `default`, `mode` and `sections` made its description longer than `openspec validate --strict` allows in a main spec. The delta now leaves that requirement as it was and adds "Workflow modes inspected" with the same text and scenarios; no behavior changes. Task 3 still lists `setup-inspection#user-template-reported`, which stays in the main spec unchanged and is still tested.

---

### Task 1: Compare a project's sections with the user template

**Scenarios:** user-workflow#template-changed, user-workflow#project-changed, user-workflow#both-changed, user-workflow#never-synced, user-workflow#same-text-with-an-old-hash, user-workflow#different-template-versions, user-workflow#no-template-versions

**Review:** checks

**Files:**
- Modify: `internal/setup/workflow.go`
- Modify: `internal/setup/workflow_test.go`

**Interfaces:**
- Consumes: `setup.Sections`, `setup.TemplateVersion`, `change.ReadFrontMatter`, as they are.
- Produces: `setup.ProjectOwned(b []byte) ([]string, error)`: the headings, nil when the field is absent, an error when the value is not a bracketed list. `setup.SectionState{Heading, State, Hash string}` and `setup.CompareSections(user, project []byte, owned []string, synced map[string]string) []setup.SectionState`: the compared sections in the user template's order, never nil; `Hash` is the hash of the shared text for `in-step` and empty otherwise.

- [x] `ProjectOwned` gives `[Where to work, How to merge]` for `project-owned: [Where to work, How to merge]`, nil with no field or no front matter, an empty list for `[]`, and an error for `project-owned: Where to work`.
- [x] `CompareSections` gives `in-step` for texts that are the same variant whether `synced` holds their hash, another hash or none; `template-changed` when the hash is the project's text; `project-changed` when it is the user template's; `conflict` when it is neither or there is none.
- [x] Only level-three sections in both files and not in `owned` are compared; a heading in one file only, or in `owned`, gives no state.
- [x] Two files with different template versions give no states; so do two files that both have no template version, and one with a version beside one without.
- [x] `Hash` of an `in-step` section is the lowercase hexadecimal SHA-256 of its text as `Sections` gives it.

### Task 2: Mode records in the index and the default in the settings

**Scenarios:** user-workflow#default-read-from-the-settings, project-index#entry-written, project-index#mode-kept-on-another-write, project-index#unknown-mode-in-the-index

**Review:** checks

**Files:**
- Modify: `internal/projects/files.go`
- Modify: `internal/projects/files_test.go`
- Modify: `internal/cli/cmd_projects_test.go`

**Interfaces:**
- Produces: `projects.Entry` fields `Mode string` (`json:"mode,omitempty"`) and `Synced map[string]string` (`json:"synced,omitempty"`); `projects.DefaultMode(settings string) (string, error)`: `sync`, `independent`, or empty when the file or key is missing; an error for another value or a file that is not JSON.

- [x] An index whose entry holds `mode` `sync` and `synced` loads them and saves them unchanged; an entry without them saves without the keys, as before.
- [x] An entry with `mode` `always` makes `Load` fail, and `dkt projects ls --json` lists the index in `unreadable` and exits with 3.
- [x] Each of `dkt projects add` in another Docket project, `dkt projects ignore` and `dkt projects rm` on another path, and automatic registration of another project (by `dkt status` there) keeps the first entry's `mode` and `synced`.
- [x] `DefaultMode` gives `sync` for `{"workflow": {"default": "sync"}}`, empty with no file or no key, and an error for `"always"` and for a file that is not JSON.

### Task 3: dkt inspect reports the modes and the section states

**Scenarios:** user-workflow#project-owned-section-left-out, user-workflow#project-owned-not-a-list, user-workflow#default-read-from-the-settings, user-workflow#unknown-default, user-workflow#project-with-no-record, setup-inspection#user-template-reported, setup-inspection#independent-project, setup-inspection#sync-project-in-step, setup-inspection#index-without-a-location, setup-inspection#changed-template-reported

**Review:** judgment

**Files:**
- Modify: `internal/cli/cmd_inspect.go`
- Modify: `internal/cli/cmd_inspect_test.go`

**Interfaces:**
- Consumes: `setup.ProjectOwned`, `setup.CompareSections` (Task 1); `projects.Entry.Mode`, `projects.Entry.Synced`, `projects.DefaultMode` (Task 2); `projects.IndexPath`, `projects.SettingsPath`, `projects.Load`, `projects.ProjectPath` and `readProjectWorkflow` as they are.

- [x] `result.user_workflow` holds `default` (null when unset), `mode` and `sections` (an empty array, not null, when nothing is compared), beside `path`, `exists` and `version` as before.
- [x] A project with no entry, or an entry without `mode`, is `independent`; in `independent` mode `sections` is empty and no section finding is reported even where texts differ.
- [x] In `sync` mode each compared section is listed with its state, and each of `template-changed`, `project-changed` and `conflict` gives its finding naming `docket/workflow.md` and the heading, with exit 1; all `in-step` gives no finding and exit 0 when nothing else is found.
- [x] A `project-owned` heading is not listed; an invalid `project-owned` gives `workflow-project-owned-invalid` and no sections.
- [x] No sections are listed when the user template does not exist, when `docket/workflow.md` does not exist, or when the two template versions differ.
- [x] Settings with `workflow.default` `sync` give `default` `sync`; `always` or a settings file that is not JSON lists the settings file in `unreadable`, `default` null, exit 3; an index that cannot be read is listed in `unreadable` and the mode is `independent`.
- [x] With neither `HOME` nor `XDG_CONFIG_HOME`, `result.user_workflow` is null, neither the index nor the settings is read, and inspect reports the rest; with an absolute `XDG_CONFIG_HOME` and neither `HOME` nor `XDG_STATE_HOME`, `result.user_workflow` holds the path and `mode` `independent`, and nothing about the index is reported.
- [x] In `sync` mode with sections to report, the index file and the project's files are byte-for-byte unchanged after inspect.

### Task 4: dkt workflow mode records the mode and the synced hashes

**Scenarios:** user-workflow#hash-recorded, user-workflow#sync-recorded, user-workflow#earlier-hash-kept, user-workflow#hashes-kept-across-a-version-gap, user-workflow#independent-recorded, user-workflow#entry-added, user-workflow#no-user-template, user-workflow#unknown-mode, user-workflow#outside-a-repository, user-workflow#index-that-cannot-be-read, user-workflow#mode-without-a-home, user-workflow#user-template-without-a-home

**Review:** judgment

**Files:**
- Modify: `internal/cli/cmd_workflow.go`
- Modify: `internal/cli/cmd_workflow_test.go`
- Modify: `internal/cli/cmd_projects.go`

**Interfaces:**
- Consumes: `setup.ProjectOwned`, `setup.CompareSections` (Task 1); `projects.Entry.Mode`, `projects.Entry.Synced` (Task 2); `projects.Locate`, `projects.Load`, `projects.UserWorkflowPath`, `projectsEnv` and `readProjectWorkflow` as they are.

- [x] `dkt workflow mode sync --json` in a project whose sections equal the user template's sets `mode` `sync`, maps every compared heading to its hash, lists the headings in `result.synced`, gives `result.path` the project path, and exits with 0.
- [x] A section whose texts differ keeps its earlier hash, or gets none when it had none, and is not in `result.synced`; a `project-owned` section and a heading in one file only get no new hash; when the two template versions differ, `synced` is unchanged and `result.synced` is empty.
- [x] `dkt workflow mode independent` sets `mode` `independent` and removes `synced`.
- [x] With no entry for the project, it adds one as `dkt projects add` would (origin, time, `ignored` false) with the mode; an ignored entry stays ignored.
- [x] Run in a linked worktree, it reads that worktree's `docket/workflow.md` and writes the entry of the main checkout's path.
- [x] With `sync`, a missing user template gives `user-workflow-missing` and an invalid `project-owned` gives `workflow-project-owned-invalid`, both exit 1 with the index not written; outside a repository it gives `not-a-repository` and in one without `docket/workflow.md` `not-a-project`, exit 1.
- [x] `dkt workflow mode always`, `dkt workflow mode` and `dkt workflow mode sync extra` are usage errors with exit 2.
- [x] An index holding `not json`, or a user template or project file that cannot be read, is listed in `unreadable`, the index is unchanged, and dkt exits with 3.
- [x] With neither `HOME` nor `XDG_STATE_HOME`, `independent` and `sync` report `no-home` and exit with 4; with an absolute `XDG_STATE_HOME` and neither `HOME` nor `XDG_CONFIG_HOME`, `sync` reports `no-home`, exits with 4 and writes no index, while `independent` records the mode.
- [x] `dkt projects add` reports `not-a-repository` and `not-a-project` exactly as before, through the shared helper.

### Task 5: docket-init asks for the mode and docket-doctor resolves the section states

**Scenarios:** none

**Review:** checks

**Files:**
- Modify: `skills/docket-init/SKILL.md`
- Modify: `skills/docket-doctor/SKILL.md`

- [x] In docket-init's step 4, once the user agrees to take a user template of the same version, the agent proposes `result.user_workflow.default` when it is set and otherwise asks for the mode, explaining `sync` and `independent` in a line each, and runs `dkt workflow mode <mode>` only once the user agrees.
- [x] docket-doctor lists `workflow-sync-template-changed`, `workflow-sync-project-changed`, `workflow-sync-conflict` and `workflow-project-owned-invalid` with the other problems and resolves them one at a time with the user's agreement: take the user template's section; keep the project's edit and offer to add the heading to `project-owned` or to copy it into the user template; for a conflict, show both and let the user take either text or add the heading to `project-owned`; fix the `project-owned` line. After each agreed write it runs `dkt workflow mode sync`, then `dkt inspect` again.
- [x] Neither skill writes `mode`, `synced`, a workflow file or the user template without the user's agreement, and neither changes the settings file.
- [x] Every rule and command the two skills name exists in `internal/cli`.
