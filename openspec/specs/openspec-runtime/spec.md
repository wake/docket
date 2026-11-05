# OpenSpec Runtime Specification

## Purpose
Defines how dkt commands deal with the OpenSpec environment they depend on: how the CLI is called, which versions are accepted, and how a failed call is handled.

## Requirements

### Requirement: Locating the CLI
When the environment variable `DOCKET_OPENSPEC` is set and not empty, dkt SHALL run the command it names, split on whitespace into a program and the arguments that go before OpenSpec's own. Otherwise dkt SHALL run `openspec` from `PATH`.

#### Scenario: Override with npx
- **WHEN** `DOCKET_OPENSPEC` is `npx -y @fission-ai/openspec@1.14.0` and a user runs `dkt init`
- **THEN** dkt runs `npx` with the arguments `-y @fission-ai/openspec@1.14.0` followed by the OpenSpec arguments

#### Scenario: Default from PATH
- **WHEN** `DOCKET_OPENSPEC` is unset and a user runs `dkt init`
- **THEN** dkt runs the `openspec` found on `PATH`

### Requirement: Telemetry off
Every time dkt runs OpenSpec, it SHALL set the environment variable `OPENSPEC_TELEMETRY` to `0` and pass the rest of the environment through unchanged.

#### Scenario: Every invocation
- **WHEN** `OPENSPEC_TELEMETRY` is `1` and a user runs `dkt init`
- **THEN** every OpenSpec process dkt starts sees `OPENSPEC_TELEMETRY` as `0`

### Requirement: Minimum version
Before a command first calls OpenSpec, dkt SHALL read the CLI's version. When it is below 1.14.0, or no version can be read from the output, dkt SHALL report `openspec-unsupported`, naming the version or output found, the minimum 1.14.0, and `DOCKET_OPENSPEC` as the way to use another CLI; it SHALL change nothing and exit with 4. There is no upper bound.

#### Scenario: Too old
- **WHEN** the OpenSpec CLI reports version 1.13.2 and a user runs `dkt init --tools claude`
- **THEN** dkt exits with 4, writes nothing, and the finding `openspec-unsupported` names 1.13.2 and 1.14.0

#### Scenario: Newer version accepted
- **WHEN** the OpenSpec CLI reports version 2.0.0 and a user runs `dkt init`
- **THEN** dkt runs as usual and reports no version problem

### Requirement: Missing CLI
When no OpenSpec CLI can be started, a command that needs one SHALL report `openspec-missing`, saying how to install OpenSpec or set `DOCKET_OPENSPEC`; it SHALL change nothing and exit with 4.

#### Scenario: CLI missing
- **WHEN** no OpenSpec CLI can be started and a user runs `dkt init`
- **THEN** dkt exits with 4 and the finding `openspec-missing` says how to install OpenSpec or set `DOCKET_OPENSPEC`

### Requirement: Failures reported
When OpenSpec exits with a non-zero status, dkt SHALL report `openspec-failed`, naming the command it ran, the exit status and OpenSpec's standard error; it SHALL NOT run any later step that depends on that call, and SHALL exit with 1.

#### Scenario: openspec init fails
- **WHEN** the project has no `openspec/`, a user runs `dkt init --tools nosuch`, and `openspec init` fails because it does not know that value
- **THEN** the finding `openspec-failed` names `openspec init --tools nosuch`, its exit status and its error output, none of Docket's own files is written, and dkt exits with 1
