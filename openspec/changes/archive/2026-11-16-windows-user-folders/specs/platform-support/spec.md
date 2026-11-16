# Spec Delta

## ADDED Requirements

### Requirement: User folders on Windows
Wherever a requirement places dkt's files under `$HOME`, as for the index, the settings, the user template and the activity records, on Windows dkt SHALL use the folder `USERPROFILE` names, at the same paths under it, and SHALL NOT read `HOME`; a message that names `HOME` as missing SHALL name `USERPROFILE` instead. An absolute `XDG_CONFIG_HOME` or `XDG_STATE_HOME` SHALL be used as on other systems. These variables SHALL be read whatever the case of their names.

#### Scenario: Index under the user profile
- **WHEN** on Windows, `XDG_STATE_HOME` is not set, `USERPROFILE` is `C:\Users\me`, and a project is added
- **THEN** the entry is written to `C:\Users\me\.local\state\docket\projects.json`

#### Scenario: User template under the user profile
- **WHEN** on Windows, `XDG_CONFIG_HOME` is not set, `USERPROFILE` is `C:\Users\me`, and a user runs `dkt inspect --json`
- **THEN** `result.user_workflow.path` is `C:\Users\me\.config\docket\workflow.md`

#### Scenario: HOME not read on Windows
- **WHEN** on Windows, `HOME` is `C:\h`, neither `USERPROFILE` nor `XDG_STATE_HOME` is set, and a user runs `dkt projects ls --json`
- **THEN** the findings hold `no-home`, whose message names `USERPROFILE`, and dkt exits with 4

#### Scenario: State home kept on Windows
- **WHEN** on Windows, `XDG_STATE_HOME` is `D:\s` and a session in a folder runs `dkt activity working`
- **THEN** a file in `D:\s\docket\sessions\` holds the session's record

#### Scenario: Variable spelled in another case
- **WHEN** on Windows, the environment holds `UserProfile` set to `C:\Users\me` and no `USERPROFILE`, and a project is added
- **THEN** the entry is written to `C:\Users\me\.local\state\docket\projects.json`

#### Scenario: Other systems unchanged
- **WHEN** on macOS or Linux, `HOME` is `/h`, `USERPROFILE` is `/u`, `XDG_STATE_HOME` is not set, and a project is added
- **THEN** the entry is written to `/h/.local/state/docket/projects.json`
