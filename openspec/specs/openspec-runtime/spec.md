# openspec-runtime Specification

## Purpose
Defines how Docket finds the OpenSpec CLI it delegates to and which versions it
accepts, so that a repository is never changed by an OpenSpec version Docket
was not tested against.

## Requirements

### Requirement: Locating the CLI
Docket SHALL run the OpenSpec command named by the `DOCKET_OPENSPEC` environment variable when it is set and not empty, splitting the value on whitespace into a program and its leading arguments; otherwise it SHALL run `openspec` from `PATH`.

#### Scenario: Override with npx
- **WHEN** `DOCKET_OPENSPEC` is `npx -y @fission-ai/openspec@1.14.3` and a user runs `dkt new`
- **THEN** Docket runs `npx` with the arguments `-y @fission-ai/openspec@1.14.3` followed by the OpenSpec arguments

#### Scenario: Default from PATH
- **WHEN** `DOCKET_OPENSPEC` is unset and a user runs `dkt new`
- **THEN** Docket runs the `openspec` found on `PATH`

### Requirement: Supported versions
Before a command first invokes OpenSpec, Docket SHALL read the CLI's version and SHALL exit with 4, changing nothing, unless that version is in the range this Docket release supports, which for the first release is 1.14.x. The error SHALL name the version found, the accepted range, and `DOCKET_OPENSPEC` as the way to point at another CLI.

#### Scenario: Unsupported version
- **WHEN** the OpenSpec CLI reports version 1.3.0 and a user runs `dkt new add-x --kind feature --source x --title X`
- **THEN** Docket exits with 4, creates nothing, and names 1.3.0 and the accepted range

#### Scenario: CLI missing
- **WHEN** no OpenSpec CLI can be started and a user runs `dkt init`
- **THEN** Docket exits with 4 and says how to install one or set `DOCKET_OPENSPEC`

### Requirement: Failures reported
When an OpenSpec invocation exits with a non-zero status, Docket SHALL report the command it ran, the exit status and OpenSpec's standard error, and SHALL NOT run any later step that depends on that invocation.

#### Scenario: OpenSpec command fails
- **WHEN** `openspec new change` fails during `dkt new`
- **THEN** Docket reports OpenSpec's exit status and error output, writes no `docket.yaml`, and exits with 1
