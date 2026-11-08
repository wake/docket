## MODIFIED Requirements

### Requirement: Inspect writes nothing
`dkt inspect` SHALL change no file, inside the repository or outside it, apart from adding the project to the project index as `project-index` defines for every command, and the only processes it SHALL start are git and the OpenSpec CLI's version query. It SHALL exit with 0 when it reports no finding and with 1 when it reports any.

#### Scenario: Nothing written
- **WHEN** a user runs `dkt inspect` in a repository where `dkt init` never ran
- **THEN** no file inside or outside the repository is created, changed or removed

#### Scenario: Clean setup
- **WHEN** the repository was set up by this release's `dkt init` and not changed since, and the OpenSpec CLI reports 1.14.0
- **THEN** `dkt inspect --json` reports no finding and exits with 0
