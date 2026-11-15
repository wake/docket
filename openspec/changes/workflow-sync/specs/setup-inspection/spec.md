## ADDED Requirements

### Requirement: Workflow modes inspected
Beside the user template, `result.user_workflow` SHALL hold the settings' `default` mode (null when unset), the project's `mode`, and `sections`: in `sync` mode the `heading` and `state` of each section compared as `user-workflow` says, otherwise none. When the index has no location, the mode SHALL be `independent`. A settings file or index that cannot be read SHALL be listed in `unreadable`.

#### Scenario: Independent project
- **WHEN** the project's entry has `mode` `independent` and its "Plan review" text differs from the user template's
- **THEN** `result.user_workflow.mode` is `independent`, `sections` is empty, and no finding is about "Plan review"

#### Scenario: Sync project in step
- **WHEN** the project's entry has `mode` `sync` and every section equals the user template's
- **THEN** `sections` lists each compared section with state `in-step`, and no finding is about them

#### Scenario: Index without a location
- **WHEN** `XDG_CONFIG_HOME` is an absolute path, neither `HOME` nor `XDG_STATE_HOME` is set, and a user runs `dkt inspect --json`
- **THEN** `result.user_workflow` has the user template's path and `mode` `independent`, and no finding or `unreadable` entry is about the index

### Requirement: Section state findings
In `sync` mode, `dkt inspect` SHALL report `workflow-sync-template-changed`, `workflow-sync-project-changed` or `workflow-sync-conflict` for each section in that state, naming `docket/workflow.md` and the section's heading. An `in-step` section SHALL be no finding.

#### Scenario: Changed template reported
- **WHEN** a project in `sync` mode has "Plan review" in state `template-changed`
- **THEN** the finding `workflow-sync-template-changed` names `docket/workflow.md` and "Plan review", and dkt exits with 1
