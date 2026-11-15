## Purpose

Defines the user's own workflow template on one computer, where it is and how its version is read, and how `dkt workflow refs` lists the ways the user's projects and the user template do each step, so that docket-init can set a new project up from them.

## ADDED Requirements

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
