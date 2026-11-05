## MODIFIED Requirements

### Requirement: Inspect writes nothing
`dkt inspect` SHALL change no file, inside the repository or outside it, and the only processes it SHALL start are git and the OpenSpec CLI's version query. It SHALL exit with 0 when it reports no finding and with 1 when it reports any.

#### Scenario: Nothing written
- **WHEN** a user runs `dkt inspect` in a repository where `dkt init` never ran
- **THEN** no file inside or outside the repository is created, changed or removed

#### Scenario: Clean setup
- **WHEN** the repository was set up by this release's `dkt init` and not changed since, and the OpenSpec CLI reports 1.14.0
- **THEN** `dkt inspect --json` reports no finding and exits with 0

### Requirement: OpenSpec CLI inspected
`dkt inspect` SHALL locate the OpenSpec CLI as openspec-runtime says, read its version, and report in `result.openspec` the `version` (null when the CLI cannot be started), the `minimum` (`1.14.0`) and `supported`. It SHALL report `openspec-missing` when the CLI cannot be started and `openspec-unsupported` when the version is too old or cannot be read; neither SHALL make it exit with 4.

#### Scenario: Old OpenSpec
- **WHEN** the OpenSpec CLI reports 1.13.2 and a user runs `dkt inspect --json`
- **THEN** `result.openspec.version` is `1.13.2`, `result.openspec.supported` is false, the findings hold `openspec-unsupported`, and dkt exits with 1

#### Scenario: No OpenSpec
- **WHEN** no OpenSpec CLI can be started
- **THEN** `result.openspec.version` is null and the findings hold `openspec-missing`
