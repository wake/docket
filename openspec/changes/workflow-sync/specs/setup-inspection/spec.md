## MODIFIED Requirements

### Requirement: User template inspected
`dkt inspect` SHALL report the user template that `user-workflow` locates in `result.user_workflow`, with its `path`, whether it `exists`, its template `version` (0 when it has none or does not exist), the settings' `default` mode (null when unset), the project's `mode`, and `sections`: in `sync` mode the `heading` and `state` of each section compared as `user-workflow` says, otherwise none. When no location can be found, `result.user_workflow` SHALL be null. A user template, settings file or index that cannot be read SHALL be listed in `unreadable`.

#### Scenario: User template reported
- **WHEN** the user template exists with template 3
- **THEN** `result.user_workflow` has its path, `exists` true and `version` 3

#### Scenario: Independent project
- **WHEN** the project's entry has `mode` `independent` and its "Plan review" text differs from the user template's
- **THEN** `result.user_workflow.mode` is `independent`, `sections` is empty, and no finding is about "Plan review"

#### Scenario: Sync project in step
- **WHEN** the project's entry has `mode` `sync` and every section equals the user template's
- **THEN** `sections` lists each compared section with state `in-step`, and no finding is about them

## ADDED Requirements

### Requirement: Section state findings
In `sync` mode, `dkt inspect` SHALL report `workflow-sync-template-changed`, `workflow-sync-project-changed` or `workflow-sync-conflict` for each section in that state, naming `docket/workflow.md` and the section's heading. An `in-step` section SHALL be no finding.

#### Scenario: Changed template reported
- **WHEN** a project in `sync` mode has "Plan review" in state `template-changed`
- **THEN** the finding `workflow-sync-template-changed` names `docket/workflow.md` and "Plan review", and dkt exits with 1
