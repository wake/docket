# materialized-workflow Specification

## Purpose
Defines the workflow template Docket ships, where a user's materialized
workflow lives and what it holds, and what `dkt workflow` reports about it,
so an agent can follow one user's steps for each path in every project.

## Requirements

### Requirement: Workflow file location
A user's materialized workflow SHALL be the file `docket/workflow.md` under the directory named by `XDG_CONFIG_HOME` when that variable holds an absolute path, and under `.config` in the user's home directory otherwise. It SHALL NOT be kept in a repository.

#### Scenario: Workflow in the config home
- **WHEN** `XDG_CONFIG_HOME` is unset, the home directory is `/h`, and a user runs `dkt workflow --json`
- **THEN** `result.path` is `/h/.config/docket/workflow.md`

#### Scenario: Workflow under XDG_CONFIG_HOME
- **WHEN** `XDG_CONFIG_HOME` is `/x` and a user runs `dkt workflow --json`
- **THEN** `result.path` is `/x/docket/workflow.md`

#### Scenario: XDG config home ignored when relative
- **WHEN** `XDG_CONFIG_HOME` is `rel`, the home directory is `/h`, and a user runs `dkt workflow --json`
- **THEN** `result.path` is `/h/.config/docket/workflow.md`

### Requirement: Shipped template
`dkt workflow template` SHALL print the workflow template shipped with the running Docket release. With `--json`, its `result` SHALL hold the text as `text` and its hash as `template`: the first 12 hexadecimal digits of the SHA-256 of the text.

#### Scenario: Template printed
- **WHEN** a user runs `dkt workflow template --json`
- **THEN** `result.text` is the shipped template, `result.template` is the first 12 hexadecimal digits of its SHA-256, and the command exits with 0

### Requirement: Template content
The template SHALL give, under a heading for each of the four paths (full review, light review, check, direct merge), that path's review and verification steps, naming capabilities, never products. A required review SHALL fall back from a reviewer on a different model to the same model in a fresh context, then to a note in the pull request that the work had no independent review.

#### Scenario: Template names capabilities
- **WHEN** the template is printed
- **THEN** it has the headings `## Full review`, `## Light review`, `## Check` and `## Direct merge`, and names no product or model

### Requirement: Workflow status
`dkt workflow` SHALL report the file's path and a status: `missing` when the file does not exist; `incomplete` when it lacks a line `## Full review`, `## Light review`, `## Check` or `## Direct merge`; `stale` when its frontmatter has no `template` or one that differs from the shipped template's hash; `ok` otherwise. `incomplete` SHALL win over `stale`. Only `ok` SHALL exit with 0; the others SHALL exit with 1.

#### Scenario: No workflow
- **WHEN** the workflow file does not exist and a user runs `dkt workflow --json`
- **THEN** `result.status` is `missing`, the findings hold `workflow-missing`, and the command exits with 1

#### Scenario: Complete workflow
- **WHEN** the file has the four headings and `template` equal to the shipped template's hash
- **THEN** `result.status` is `ok`, there are no findings, and the command exits with 0

#### Scenario: A path missing
- **WHEN** the file has `template` equal to an older hash and lacks `## Check`
- **THEN** `result.status` is `incomplete`, `result.missing_sections` is `["Check"]`, one `workflow-incomplete` finding names it, and the command exits with 1

#### Scenario: Made from an older template
- **WHEN** the file has the four headings and `template: 0123456789ab`, which is not the shipped hash
- **THEN** `result.status` is `stale`, `result.recorded` is `0123456789ab`, `result.template` is the shipped hash, the findings hold `workflow-stale`, and the command exits with 1

#### Scenario: No frontmatter
- **WHEN** the file has the four headings and no frontmatter
- **THEN** `result.status` is `stale` and `result.recorded` is null

#### Scenario: Unreadable workflow
- **WHEN** the workflow file exists but cannot be read
- **THEN** it is listed in `unreadable` and the command exits with 3
