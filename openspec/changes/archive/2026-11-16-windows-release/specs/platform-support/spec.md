## ADDED Requirements

### Requirement: Supported on macOS, Linux and Windows
dkt SHALL be supported on macOS, Linux and Windows, on Intel and ARM. CI SHALL run the whole test suite on each of the three systems; a test that cannot run on one of them SHALL skip there and say why in its skip message.

#### Scenario: Build for Windows
- **WHEN** dkt is built for Windows on amd64 or arm64
- **THEN** the build succeeds

#### Scenario: Whole suite on Windows
- **WHEN** the CI job that runs on Windows is listed among a pull request's checks
- **THEN** it runs the whole test suite, and its name does not say that dkt is unsupported on Windows or that the job covers only part of it

## REMOVED Requirements

### Requirement: Supported systems
**Reason**: dkt is now supported on Windows, so the limit this requirement set, and the CI job name it asked for, no longer hold.
**Migration**: Replaced by "Supported on macOS, Linux and Windows".
