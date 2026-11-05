## MODIFIED Requirements

### Requirement: Locating the CLI
When the environment variable `DOCKET_OPENSPEC` is set and not empty, dkt SHALL run the command it names, split on whitespace into a program and the arguments that go before OpenSpec's own. Otherwise dkt SHALL run `openspec` from `PATH`.

#### Scenario: Override with npx
- **WHEN** `DOCKET_OPENSPEC` is `npx -y @fission-ai/openspec@1.14.0` and a user runs `dkt init`
- **THEN** dkt runs `npx` with the arguments `-y @fission-ai/openspec@1.14.0` followed by the OpenSpec arguments

#### Scenario: Default from PATH
- **WHEN** `DOCKET_OPENSPEC` is unset and a user runs `dkt init`
- **THEN** dkt runs the `openspec` found on `PATH`

### Requirement: Minimum version
Before a command first calls OpenSpec, dkt SHALL read the CLI's version. When it is below 1.14.0, or no version can be read from the output, dkt SHALL report `openspec-unsupported`, naming the version or output found, the minimum 1.14.0, and `DOCKET_OPENSPEC` as the way to use another CLI; it SHALL change nothing and exit with 4. There is no upper bound.

#### Scenario: Too old
- **WHEN** the OpenSpec CLI reports version 1.13.2 and a user runs `dkt init --tools claude`
- **THEN** dkt exits with 4, writes nothing, and the finding `openspec-unsupported` names 1.13.2 and 1.14.0

#### Scenario: Newer version accepted
- **WHEN** the OpenSpec CLI reports version 2.0.0 and a user runs `dkt init`
- **THEN** dkt runs as usual and reports no version problem
