# Design

## Context

The repository holds no code yet. The previous Docket (tag `v0.15.0`, branch `legacy`) had the same envelope, exit codes and repository identity in `internal/cli` and `internal/gitx`; this change ports only the parts the spec needs, read with `git show v0.15.0:<path>`. Requirements are in `specs/cli-output/spec.md`.

## Goals / Non-Goals

**Goals:**
- One entry point that every later command plugs into, with the envelope, the exit codes and repository identity done once.
- Tests that run the command line in-process, so later changes add a command and its tests without new plumbing.

**Non-Goals:**
- Commands that need a repository, and the exit 4 they give outside one; reading sources, and `unreadable` entries. Both arrive with the first command that needs them.
- The `--default-branch` flag the previous version had.
- CI and release builds.

## Decisions

### Layout
- `cmd/dkt/main.go` only reads the process boundary (arguments, working directory, environment, standard streams) and calls `cli.Run`, which returns the exit code.
- `internal/cli` holds the envelope types, the writers for JSON and text, flag parsing and dispatch. The command table is empty in this change.
- `internal/gitx` runs `git` and reads the root, HEAD, branch and `origin`.
- `internal/testutil` makes temporary repositories with an isolated git environment.

Ported from `v0.15.0`: `internal/cli/envelope.go` and `output.go` (types and writers), the dispatch in `run.go` without the commands and without `--default-branch`, the subset of `internal/gitx/git.go` used here (`Toplevel`, `Head`, `Branch`, `RemoteURL`, executable lookup, `ErrNotRepo`, `ErrGitMissing`) with its tests, and `internal/testutil/repo.go`. Alternative considered: write from scratch; rejected, the old code already meets these requirements and was tested.

### Version
`cli.Version` defaults to `dev`; release builds set it with `-ldflags "-X github.com/wake/docket/internal/cli.Version=<tag>"`. When it is `dev`, `main` takes the main module's version from `runtime/debug.ReadBuildInfo` only when that version is a plain release version, `vMAJOR.MINOR.PATCH`. Since Go 1.24, `go build` in a working tree stamps a pseudo-version such as `v0.16.1-0.20261104…-abcdef123456+dirty`; the spec wants `dev` there, and only `go install …@vX.Y.Z` yields a plain version. Alternative considered: report the pseudo-version; rejected, it contradicts the spec and looks like a release to a reader.

### Flags
The standard `flag` package. `--json`, `--version` and `--help`/`-h` are read before the command name; `--json` is also accepted after it, as before. `--version` and `--help` need no command and no repository.

### Repository identity
`--version` identifies the repository on a best-effort basis: when `git` is missing or the directory is outside a work tree, `repo` is null and the command still succeeds. The root is printed as `git rev-parse --show-toplevel` reports it.

### Tests
Tests call `cli.Run` with buffers and a temporary directory. The git environment is isolated: `HOME` points at a temporary directory, `GIT_CONFIG_NOSYSTEM=1`, and `GIT_CEILING_DIRECTORIES` stops the outside-a-repository test from finding an enclosing repository. One test builds the binary with `go build -ldflags` to check the stamped version.

## Risks / Trade-offs

- [A future Go release changes how it stamps versions] → only a plain `vMAJOR.MINOR.PATCH` is taken from build info; anything else reads as `dev`.
- [A clean working tree checked out exactly at a release tag reports that tag, not `dev`] → accepted: the build is that release.
- [Paths on Windows] → `repo.root` is whatever git prints; this change is not tested on Windows.
