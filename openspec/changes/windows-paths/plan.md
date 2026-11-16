# Implementation Plan

**Goal:** On Windows, the absolute paths dkt reads from git are written with backslashes, so every absolute path dkt prints or compares is spelled one way; the tests that held such paths as raw JSON text escape them; the Windows CI job verifies both.

**Architecture:** `internal/gitx` converts the two absolute paths it reads from git, the repository root (`Toplevel`) and each worktree path (`Worktrees`), with `filepath.FromSlash` as soon as it reads them; every other package gets them from there. `filepath.FromSlash` changes nothing on macOS and Linux, where the separator is already `/`. Tests that build JSON by joining strings, or search a JSON file's text for a path, escape the path as JSON does, which is the same text on macOS and Linux for the temporary paths they use.

**Tech Stack:** Go, standard library only; GitHub Actions.

**Spec:** `openspec/changes/windows-paths/proposal.md`, `specs/platform-support/spec.md`.

## Global Constraints

- macOS and Linux behave as before; the existing tests pass there unchanged in what they check.
- No new dependency.
- Windows-only behavior is verified in CI. Probes on 2026-11-16 (#182) ran these packages on `windows-latest` with this change: `internal/scan` and `internal/conflicts` pass whole; in `internal/cli`, `internal/snapshot` and `internal/gitx`, the failures left rely on `chmod`, on file names Windows forbids, on permission bits, on links, or on a program named without an extension, and belong to `windows-suite`.

## Review Focus

- No absolute path dkt reads from git for the repository's root or worktrees is left in git's spelling: `Toplevel` and `Worktrees` are the only places that read them. A remote's URL (`RemoteURL`, printed as `origin`) stays as configured.
- Paths relative to the repository root keep forward slashes.
- No test changes what it checks on macOS and Linux.

## Reuse

| Need | Existing (path:symbol) | Decision | Reason |
|---|---|---|---|
| The repository root in the system's spelling | internal/gitx/git.go:Toplevel | extend | Converts git's output with `filepath.FromSlash` |
| Worktree paths in the system's spelling | internal/gitx/worktree.go:Worktrees | extend | Converts each `worktree` field with `filepath.FromSlash`, after unquoting |
| A path as JSON text in the `internal/cli` tests | internal/cli/cmd_projects_test.go:prReal | new | `prJSON`, beside `prReal`: no test helper escapes a string as JSON; `json.Marshal` without the quotes |
| The Windows CI job | .github/workflows/ci.yml | extend | Gains the tests below |

---

### Task 1: Write the paths read from git as the system writes them

**Scenarios:** platform-support#repository-root-on-windows, platform-support#worktree-path-compared-with-one-dkt-builds, platform-support#relative-path-keeps-forward-slashes

**Review:** checks

**Files:**
- Modify: `internal/gitx/git.go`
- Modify: `internal/gitx/worktree.go`
- Modify: `internal/gitx/path_windows_test.go`
- Modify: `internal/cli/repo_test.go`

- [x] On Windows, `Toplevel` of a linked worktree holds no `/`, equals that worktree's path from `Worktrees`, and equals the worktree folder resolved with `filepath.EvalSymlinks`; `Worktrees` lists both worktrees with no `/`. A new test in `internal/gitx/path_windows_test.go` covers it and fails on Windows without the conversion.
- [x] `TestRepoRootFromSubdirectory` expects `repo.root` to be `filepath.FromSlash` of `git rev-parse --show-toplevel`: backslashes on Windows, git's output unchanged on macOS and Linux.
- [x] Paths relative to the repository root keep forward slashes on Windows: the tests of `internal/scan` and `internal/conflicts`, which check such paths in their results, pass on `windows-latest` (run by Task 2's CI job).

### Task 2: Escape paths in test JSON, and run the tests on Windows

**Scenarios:** platform-support#worktree-path-compared-with-one-dkt-builds, platform-support#relative-path-keeps-forward-slashes

**Review:** checks

**Files:**
- Modify: `internal/cli/cmd_projects_test.go`
- Modify: `internal/cli/register_test.go`
- Modify: `internal/snapshot/activity_test.go`
- Modify: `.github/workflows/ci.yml`

- [x] Every test in `internal/cli/cmd_projects_test.go`, `internal/cli/register_test.go` and `internal/snapshot/activity_test.go` that writes a path into JSON, or looks for one in a JSON file's text, escapes it as JSON does (`prJSON` in `internal/cli`, `json.Marshal` in `internal/snapshot`), the checks that a path is absent included, such as `TestProjectsRm`'s, which would otherwise pass on Windows whatever the file holds.
- [x] The `windows-latest` CI job also runs, each reporting tests run and none skipped by name except as listed: `go test ./internal/scan/ ./internal/conflicts/`; `internal/gitx` with `PathsFromGit` added to the tests it runs; `internal/snapshot`'s `TestActivity` tests; and `internal/cli`'s `TestRepoRoot`, `TestRegister`, `TestInspectInNewProject`, `TestProjects` and `TestWorkflow` tests, skipping only `TestProjectsLsUncheckable`, which relies on `chmod`, with the reason in a comment. The job's comment names the paths read from git among what it verifies.
- [x] On macOS and Linux, `go test ./...` passes, and no test changed in this task checks anything different there.
