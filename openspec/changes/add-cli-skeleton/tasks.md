# Tasks

Each group is one slice an implementer finishes test-first. Scenario names refer to `specs/cli-output/spec.md`. Ported code is read with `git show v0.15.0:<path>` (see design.md, Layout). Tests run with `go test ./...`; every git call in a test uses `testutil.Env`, never the user's git configuration.

## 1. Module and entry point

**Review:** checks
**Files:** `go.mod`, `cmd/dkt/main.go`, `internal/cli/run.go`

- [ ] 1.1 Create `go.mod` (`module github.com/wake/docket`, `go 1.24`, no requirements), `internal/cli` with `var Version = "dev"`, `type Options struct { Args []string; Dir string; Env []string; Stdout, Stderr io.Writer }` and `func Run(o Options) int`, and `cmd/dkt/main.go`, which fills `Options` from the process and exits with `Run`'s result; verify `go build ./...` and `go vet ./...` pass

## 2. Envelope, version and JSON output

**Review:** judgment (public output format)
**Files:** `internal/cli/envelope.go`, `internal/cli/output.go`, `internal/cli/version.go`, `internal/cli/run.go`, `cmd/dkt/main.go`, `internal/cli/output_test.go`, `internal/cli/version_test.go`, `cmd/dkt/main_test.go`

- [ ] 2.1 Write failing tests for the scenarios JSON on success, Envelope fields present and Working tree build, plus: text mode prints exactly `dkt dev\n`; `findings` and `unreadable` encode as `[]`, never `null`; verify they fail
- [ ] 2.2 Write a failing test for `func ResolveVersion(stamped, buildInfo string) string` in `internal/cli/version.go`: a stamped value other than `dev` wins; otherwise a build-info version matching `^v\d+\.\d+\.\d+$` is used; `v0.0.0-20261104074825-4c775bd664cc`, `v0.16.0+dirty`, `(devel)` and the empty string give `dev`; verify it fails
- [ ] 2.3 Write a failing test for the scenario Release build in `cmd/dkt/main_test.go`: build the binary with `go build -ldflags "-X github.com/wake/docket/internal/cli.Version=v0.16.0"` into the test's temporary directory, run `--version`, expect `dkt v0.16.0\n` and exit 0; verify it fails
- [ ] 2.4 Port the envelope types and writers (`Finding`, `Unreadable`, `Repo`, `Outcome`, `writeJSON`, `writeText`) and implement `--version` in text and JSON mode and `ResolveVersion`, called from `main`; verify every test from 2.1–2.3 passes

## 3. Usage errors and help

**Review:** judgment (error path)
**Files:** `internal/cli/run.go`, `internal/cli/usage.go`, `internal/cli/run_test.go`

- [ ] 3.1 Write failing tests for the scenarios JSON on failure, Envelope on a usage error, Unknown command, No command and Help, plus: `-h` behaves like `--help`; `--json` before the command name (`dkt --json nosuch`) and after it (`dkt nosuch --json`) both give one JSON object; a usage error in JSON mode still prints the usage to standard error; verify they fail
- [ ] 3.2 Port the dispatch and usage output from `v0.15.0:internal/cli/run.go` with an empty command table and without `--default-branch`; verify every test from 3.1 passes

## 4. Repository identity

**Review:** judgment (environment-dependent behavior)
**Files:** `internal/gitx/git.go`, `internal/gitx/git_test.go`, `internal/testutil/repo.go`, `internal/cli/run.go`, `internal/cli/repo_test.go`

- [ ] 4.1 Port from `v0.15.0` the subset of `internal/gitx/git.go` named in design.md and its tests, and `internal/testutil/repo.go` (`Env`, `NewRepo`, `Commit`, and what they need); verify the ported tests pass
- [ ] 4.2 Write failing tests for the scenarios Detached HEAD, No origin remote, No commits yet and Outside a repository, plus: on a branch, `repo.branch` is its name and `repo.origin` is the `origin` URL; with `git` missing from `PATH`, `dkt --version --json` exits 0 with `repo` null; verify they fail
- [ ] 4.3 Make `--version` identify the repository on a best-effort basis (design.md, Repository identity) and put it in the envelope; verify every test from 4.2 passes

## 5. Integration

- [ ] 5.1 Run `go vet ./...`, `go test ./...` and `openspec validate add-cli-skeleton --strict`; build `dkt` into a temporary directory and run `dkt --version`, `dkt --version --json`, `dkt nosuch`, `dkt` and `dkt --help` inside this repository and outside any repository; verify each output and exit code matches the spec

## Workflow follow-up

- Before merging: one adversarial review of the whole change and one fresh review against the spec, in parallel.
- After both reviews are settled, archive the change with OpenSpec as the last commit of the pull request, so `openspec/specs/cli-output/spec.md` exists for the next change; then merge with a merge commit.
