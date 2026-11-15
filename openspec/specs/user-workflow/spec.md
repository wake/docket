# User Workflow Specification

## Purpose
Defines the user's own workflow template on one computer, where it is and how its version is read, and how `dkt workflow refs` lists the ways the user's projects and the user template do each step, so that docket-init can set a new project up from them.

## Requirements

### Requirement: User template location
The user template SHALL be the file `docket/workflow.md` under `$XDG_CONFIG_HOME` when that variable is an absolute path, and under `$HOME/.config` otherwise, beside the settings file. Its template version SHALL be read as the Template version requirement of `workflow-file` defines for a workflow file. A symbolic link at the file SHALL be followed.

#### Scenario: Location under the config home
- **WHEN** `XDG_CONFIG_HOME` is `/c` and a user runs `dkt inspect --json`
- **THEN** `result.user_workflow.path` is `/c/docket/workflow.md`

### Requirement: Workflow references
`dkt workflow refs` SHALL list, for each level-three heading of the shipped workflow template in order, the heading, the template's text under it, and the variants of that section among the sources: each variant's text, whether it equals the template's text, and the paths of the sources that have it. It SHALL write nothing, and SHALL run outside a repository too.

#### Scenario: Projects that do a step the same way
- **WHEN** two registered projects have the same text under "Plan review" and a third has another text
- **THEN** "Plan review" lists two variants, one with the paths of the two projects and one with the path of the third

### Requirement: Same variant
Two texts SHALL be the same variant, and a text SHALL equal the template's, when they are equal after line endings become LF, trailing whitespace is removed from each line, and blank lines are removed at the start and the end.

#### Scenario: Only whitespace differs
- **WHEN** two projects' "Reviewers" texts differ only in CRLF line endings and a trailing blank line
- **THEN** they are one variant

### Requirement: Reference sources
The sources SHALL be the user template when it exists, and each registered project that is not ignored, exists, holds `docket/workflow.md`, and is not the repository the command runs in. A project's `docket/workflow.md` reached through a symbolic link that leads out of the project SHALL NOT be read, and SHALL be listed in `unreadable`.

#### Scenario: Link out of a project
- **WHEN** a registered project's `docket/workflow.md` is a symbolic link to a file outside that project
- **THEN** the file is not read, its text is in no variant, and `unreadable` names it

#### Scenario: Projects that are not references
- **WHEN** one registered project is ignored, one no longer exists, one has no `docket/workflow.md`, and the command runs in a fourth
- **THEN** none of the four is a source

#### Scenario: User template as a source
- **WHEN** the user template exists
- **THEN** it is a source, with kind `user` and its path

#### Scenario: Unknown workflow subcommand
- **WHEN** a user runs `dkt workflow nosuch`
- **THEN** it is a usage error and dkt exits with 2

### Requirement: References from another template version
For each source, `dkt workflow refs` SHALL give its path, its kind (`user` or `project`), its template version, the shipped level-three headings it lacks as `missing`, and its own level-three headings the shipped template lacks as `left_out`. Only sections whose headings match the shipped template SHALL be compared.

#### Scenario: Source on an older template
- **WHEN** a project's `docket/workflow.md` has template 2, a section "Old step" the shipped template lacks, and no "Reviewers" section
- **THEN** that source has template 2, `left_out` holds "Old step", `missing` holds "Reviewers", and no variant comes from "Old step"

### Requirement: Unreadable references
When the index or a source cannot be read, `dkt workflow refs` SHALL list it in `unreadable`, list the sources it could read, and exit with 3.

#### Scenario: One project's workflow cannot be read
- **WHEN** one registered project's `docket/workflow.md` cannot be read and another's can
- **THEN** `unreadable` names the first, the second is a source, and dkt exits with 3

### Requirement: Project-owned sections
A project's `docket/workflow.md` MAY list in its front matter, as `project-owned`, the headings of the sections the project decides on its own, written `project-owned: [<heading>, <heading>]`. When the field is absent the list SHALL be empty. A value not of that form SHALL be reported as `workflow-project-owned-invalid`, naming the file.

#### Scenario: Project-owned section left out
- **WHEN** a project in `sync` mode has `project-owned: [Where to work]` and its "Where to work" text differs from the user template's
- **THEN** `dkt inspect --json` lists no state for "Where to work" and no finding names it

#### Scenario: Project-owned not a list
- **WHEN** a project in `sync` mode has `project-owned: Where to work` in its front matter
- **THEN** the finding `workflow-project-owned-invalid` names `docket/workflow.md`, no section state is listed, and dkt exits with 1

### Requirement: Default mode setting
The settings file SHALL be able to hold `workflow.default`, `sync` or `independent`: the mode docket-init proposes for a new project. dkt SHALL NOT write it. Any other value SHALL make the settings file unreadable to `dkt inspect`.

#### Scenario: Default read from the settings
- **WHEN** the settings file holds `{"workflow": {"default": "sync"}}` and a user runs `dkt inspect --json`
- **THEN** `result.user_workflow.default` is `sync`

#### Scenario: Unknown default
- **WHEN** the settings file holds `{"workflow": {"default": "always"}}` and a user runs `dkt inspect --json`
- **THEN** `unreadable` names the settings file, `result.user_workflow.default` is null, and dkt exits with 3

### Requirement: Project mode
A project's mode SHALL be the `mode` of its entry in the project index, `sync` or `independent`, and `independent` when it has no entry or the entry has no `mode`. In `sync` mode the project follows the user template; in `independent` mode it does not. `mode` and `synced` SHALL NOT be written to the repository.

#### Scenario: Project with no record
- **WHEN** the project index has no entry for the project and a user runs `dkt inspect --json`
- **THEN** `result.user_workflow.mode` is `independent` and no section state is listed

### Requirement: Synced hashes
An entry's `synced` SHALL map a section's heading to the SHA-256, in lowercase hexadecimal, of the section's text normalized as Same variant says, recorded when the project's section and the user template's had that text.

#### Scenario: Hash recorded
- **WHEN** a project's and the user template's "Reviewers" texts are both `A reviewer.` and a user runs `dkt workflow mode sync`
- **THEN** the entry's `synced` maps "Reviewers" to the SHA-256 of `A reviewer.`

### Requirement: Section states
In `sync` mode, each level-three section that both the project's file and the user template have, and that is not project-owned, SHALL have one state: `in-step` when the two texts are the same variant; otherwise `template-changed` when the project's text has the `synced` hash, `project-changed` when the user template's has it, and `conflict` when neither has it or there is none. Sections SHALL be compared only when both files have a template version and it is the same.

#### Scenario: Template changed
- **WHEN** the `synced` hash of "Plan review" is the project's text, and the user template's text differs
- **THEN** "Plan review" is `template-changed`

#### Scenario: Project changed
- **WHEN** the `synced` hash of "Plan review" is the user template's text, and the project's text differs
- **THEN** "Plan review" is `project-changed`

#### Scenario: Both changed
- **WHEN** the project's and the user template's "Plan review" texts differ from each other and from the `synced` hash
- **THEN** "Plan review" is `conflict`

#### Scenario: Never synced
- **WHEN** the two "Plan review" texts differ and `synced` has no hash for it
- **THEN** "Plan review" is `conflict`

#### Scenario: Same text with an old hash
- **WHEN** the two "Plan review" texts are the same variant and differ from the `synced` hash
- **THEN** "Plan review" is `in-step`

#### Scenario: Different template versions
- **WHEN** a project in `sync` mode has template 3 and the user template has template 2
- **THEN** no section state is listed

#### Scenario: No template versions
- **WHEN** a project in `sync` mode and the user template both have no template version
- **THEN** no section state is listed

### Requirement: Record the mode
`dkt workflow mode <sync|independent>`, run in a Docket project, SHALL set the mode on the project's entry in the index, adding the entry as `dkt projects add` would when there is none. With `sync`, `synced` SHALL hold the hash of each `in-step` section and keep every other earlier hash; with `independent`, the entry SHALL hold no `synced`. `result` SHALL give the project's `path`, the `mode`, and the `in-step` headings as `synced`.

#### Scenario: Sync recorded
- **WHEN** a user runs `dkt workflow mode sync --json` in a project whose sections all equal the user template's
- **THEN** the entry has `mode` `sync`, `synced` has a hash for every compared section, `result.synced` lists their headings, and dkt exits with 0

#### Scenario: Earlier hash kept
- **WHEN** the project's "Plan review" differs from the user template's, `synced` holds a hash for it, and a user runs `dkt workflow mode sync`
- **THEN** that hash is unchanged and "Plan review" is not in `result.synced`

#### Scenario: Hashes kept across a version gap
- **WHEN** the project has template 3, the user template has template 2, `synced` holds hashes, and a user runs `dkt workflow mode sync --json`
- **THEN** `synced` is unchanged and `result.synced` is empty

#### Scenario: Independent recorded
- **WHEN** the entry has `mode` `sync` and `synced`, and a user runs `dkt workflow mode independent`
- **THEN** the entry has `mode` `independent` and no `synced`

#### Scenario: Entry added
- **WHEN** the project index has no entry for the project and a user runs `dkt workflow mode independent`
- **THEN** the index holds an entry for the project with `ignored` false and `mode` `independent`

### Requirement: Mode command refusals
`dkt workflow mode` SHALL write nothing and exit with 1 when it runs outside a git repository (`not-a-repository`), in a repository without `docket/workflow.md` (`not-a-project`), with `sync` when the user template does not exist (`user-workflow-missing`), or with `sync` when `project-owned` is invalid (`workflow-project-owned-invalid`). A mode other than `sync` or `independent` SHALL be a usage error.

#### Scenario: No user template
- **WHEN** the user template does not exist and a user runs `dkt workflow mode sync --json` in a Docket project the index already holds
- **THEN** the findings hold `user-workflow-missing`, the index is not written, and dkt exits with 1

#### Scenario: Unknown mode
- **WHEN** a user runs `dkt workflow mode always --json`
- **THEN** the findings hold `usage` and dkt exits with 2

#### Scenario: Outside a repository
- **WHEN** a user runs `dkt workflow mode independent --json` outside any git repository
- **THEN** the findings hold `not-a-repository` and dkt exits with 1

### Requirement: Mode command environment
When the index, the user template or the project's `docket/workflow.md` cannot be read, `dkt workflow mode` SHALL list it in `unreadable`, write nothing and exit with 3. When the index or the user template has no location, it SHALL report `no-home` and exit with 4.

#### Scenario: Index that cannot be read
- **WHEN** the index holds `not json` and a user runs `dkt workflow mode sync --json` in a Docket project
- **THEN** `unreadable` lists the index, the file still holds `not json`, and dkt exits with 3

#### Scenario: Mode without a home
- **WHEN** neither `HOME` nor `XDG_STATE_HOME` is set and a user runs `dkt workflow mode independent --json` in a Docket project
- **THEN** the findings hold `no-home` and dkt exits with 4

#### Scenario: User template without a home
- **WHEN** `XDG_STATE_HOME` is an absolute path, neither `HOME` nor `XDG_CONFIG_HOME` is set, and a user runs `dkt workflow mode sync --json` in a Docket project
- **THEN** the findings hold `no-home`, the index is not written, and dkt exits with 4
