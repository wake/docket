# Spec Delta

## Purpose

Defines the project's `docket/workflow.md`, which holds the workflow steps a project can do its own way: its template version and sections, when `dkt init` writes it, and how `dkt init` and `dkt inspect` report its state.

## ADDED Requirements

### Requirement: Workflow file created when missing
When `docket/workflow.md` does not exist, `dkt init` SHALL write the workflow template shipped with the running release there, creating `docket/` when needed, and SHALL list it in `files` with kind `workflow` and status `created`.

#### Scenario: Fresh repository gets the workflow file
- **WHEN** a user runs `dkt init --tools claude` in a repository without `openspec/` or `docket/`
- **THEN** `docket/workflow.md` matches the shipped template, `result.files` lists it with kind `workflow` and status `created`, and dkt exits with 0

### Requirement: Workflow file never replaced
When `docket/workflow.md` exists, `dkt init` SHALL NOT change it, with or without `--force`, and SHALL list it with status `kept`. Keeping it is not a finding by itself.

#### Scenario: Edited workflow kept
- **WHEN** `docket/workflow.md` has the shipped template version and every section, with the project's own text in each, and a user runs `dkt init --force`
- **THEN** the file is byte-for-byte unchanged and listed as `kept`, no finding names it, and dkt exits with 0

### Requirement: Template version
The shipped template SHALL start with front matter whose `template` is a positive integer, the template's version. A workflow file's template version is its front matter's `template`, read as a decimal integer; the file has none when it has no front matter, front matter that cannot be read, no `template`, or a value that is not a positive integer.

#### Scenario: Shipped template versioned
- **WHEN** the shipped template is read
- **THEN** its front matter's `template` is a positive integer

#### Scenario: Version that is not a number
- **WHEN** `docket/workflow.md` has `template: one` in its front matter
- **THEN** the file has no template version, and `dkt inspect` reports `workflow-template-unknown` naming it

### Requirement: Workflow sections
Each level-3 heading of the shipped template SHALL name a section the workflow file must have: a Markdown heading of any level, outside fenced code blocks, with the same text. Only the headings are compared, not what the sections hold.

#### Scenario: Section at another heading level
- **WHEN** `docket/workflow.md` has the shipped template version and every section, with `## Plan review` where the template has `### Plan review`
- **THEN** no `workflow-section-missing` finding is reported

#### Scenario: Heading only inside a code block
- **WHEN** the only `### Plan review` line in `docket/workflow.md` is inside a fenced code block
- **THEN** `workflow-section-missing` names "Plan review"

### Requirement: Workflow file status
`dkt inspect` SHALL list `docket/workflow.md` in `result.files` with kind `workflow` and a status: `missing` when it does not exist; `outdated` when its template version is lower than the shipped template's; `differs` when it has no template version, a higher one, or lacks a section; and `same` otherwise, whatever its sections hold.

#### Scenario: Customized but current
- **WHEN** `docket/workflow.md` has the shipped template version and every section, with the project's own text in each
- **THEN** `dkt inspect --json` lists it as `same` and no finding names it

#### Scenario: Lower template version
- **WHEN** the template version of `docket/workflow.md` is lower than the shipped template's
- **THEN** `dkt inspect --json` lists it as `outdated`

### Requirement: Workflow version findings
For `docket/workflow.md`, `dkt inspect` and `dkt init` SHALL report, naming the file: `workflow-missing` when it does not exist (`dkt inspect` only, since `dkt init` writes it); `workflow-outdated`, saying both versions, when its template version is lower than the shipped template's; and `workflow-template-unknown` when it has no template version or a higher one.

#### Scenario: Missing before init
- **WHEN** a user runs `dkt inspect --json` in a repository without `docket/workflow.md`
- **THEN** the findings hold `workflow-missing` naming `docket/workflow.md`, and dkt exits with 1

#### Scenario: Outdated workflow reported by init
- **WHEN** the template version of `docket/workflow.md` is lower than the shipped template's and a user runs `dkt init --json`
- **THEN** the file is unchanged and listed as `kept`, the finding `workflow-outdated` names it with both versions, and the last entry of `result.next_steps` says to run docket-doctor

#### Scenario: Higher template version
- **WHEN** `docket/workflow.md` has a template version higher than the shipped template's
- **THEN** `dkt inspect` reports `workflow-template-unknown` naming it and lists it as `differs`

### Requirement: Workflow section findings
When the template version of `docket/workflow.md` equals the shipped template's, `dkt inspect` and `dkt init` SHALL report `workflow-section-missing` for each section the file lacks, naming the file and, in its message, the heading. With any other version they SHALL NOT check the sections.

#### Scenario: Section removed
- **WHEN** `docket/workflow.md` has the shipped template version and no "Plan review" heading, and a user runs `dkt init --json`
- **THEN** the file is unchanged and listed as `kept`, one `workflow-section-missing` finding names it and "Plan review", and dkt exits with 1

#### Scenario: Sections not checked without a version
- **WHEN** `docket/workflow.md` has no front matter and no headings
- **THEN** `dkt inspect` reports `workflow-template-unknown` naming it, and no `workflow-section-missing` finding
