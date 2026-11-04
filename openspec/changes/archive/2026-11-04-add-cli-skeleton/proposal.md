# Proposal

## Why

Every command of the new dkt must use one output format and one set of exit codes, so that coding agents, scripts and CI can read its results reliably. This skeleton is shared by every command and comes first; the first step of docket-init also relies on `dkt --version` to tell whether dkt is installed.

## What Changes

- Create the Go module `github.com/wake/docket` and the `dkt` binary (`cmd/dkt`).
- Add `dkt --version`: it prints the version and works outside a git repository.
- Add the output rules every command shares: `--json` prints exactly one JSON object, with fixed envelope fields, the repository's identity and an exit code.
- On an unknown command or a usage error, print the usage and exit with 2.
- No real command is added. Locating OpenSpec and checking its version (openspec-runtime) waits for the first change that calls OpenSpec, `dkt init`: no command uses it yet, so a spec for it could not be verified.

## Capabilities

### New Capabilities
- `cli-output`: the output format, envelope fields, repository identity and exit codes of every dkt command.

### Modified Capabilities
(none)

## Impact

- Adds `go.mod`, `cmd/dkt/`, and code and tests under `internal/` for the output and for reading git information.
- Dependencies: only the Go standard library and the system `git`.
