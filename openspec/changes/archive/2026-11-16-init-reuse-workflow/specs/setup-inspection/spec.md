## ADDED Requirements

### Requirement: User template inspected
`dkt inspect` SHALL report the user template that `user-workflow` locates in `result.user_workflow`, with its `path`, whether it `exists`, and its template `version`, 0 when it has none or does not exist. When no location can be found, `result.user_workflow` SHALL be null. A user template that cannot be read SHALL be listed in `unreadable`.

#### Scenario: User template reported
- **WHEN** the user template exists with template 3
- **THEN** `result.user_workflow` has its path, `exists` true and `version` 3

### Requirement: User template findings
`dkt inspect` SHALL report `user-workflow-outdated` when the user template's version is lower than the shipped template's, `user-workflow-newer` when it is higher, and `user-workflow-template-unknown` when the file exists with no template version, each naming the file. A missing user template SHALL be no finding.

#### Scenario: Older user template
- **WHEN** the user template has template 2 and the shipped template is 3
- **THEN** the finding `user-workflow-outdated` names the user template, and dkt exits with 1

#### Scenario: Newer user template
- **WHEN** the user template has a template version higher than the shipped one
- **THEN** the finding `user-workflow-newer` names the user template

#### Scenario: No user template
- **WHEN** the user template does not exist
- **THEN** `result.user_workflow.exists` is false and no finding is about it
