# Implementation Plan

**Goal:** dkt finds git and OpenSpec on Windows, and a dkt built from any checkout holds the shipped files as committed; the Windows CI job verifies both.

**Architecture:** A new package `internal/execpath` holds the three Windows rules (variable names without case, `PATHEXT` extensions, no execute bit); `internal/gitx` and `internal/openspec` call it from their own lookups, which keep their other rules. `.gitattributes` asks for LF in every checkout. The Windows CI job runs the tests of finding programs, and the real OpenSpec job gains a Windows entry.

**Tech Stack:** Go, standard library only; GitHub Actions.

**Spec:** `openspec/changes/windows-runtime/proposal.md`, `design.md`, `specs/platform-support/spec.md`.

## Global Constraints

- macOS and Linux behave as before; the existing tests pass there unchanged, apart from naming executables with `testutil.Exe` and joining PATH entries with `os.PathListSeparator`, which change nothing on those systems.
- No new dependency.
- Windows-only behavior is verified in CI; a probe on 2026-11-16 (#179) already ran `TestRealOpenSpec` and `TestRealOpenSpecConflicts` on `windows-latest` with this lookup.

## Review Focus

- A relative `PATH` entry: git still resolves it against the folder it runs in, OpenSpec still skips it, on every system.
- Nothing found on macOS or Linux that was not found before, nor anything missed that was found.

## Reuse

| Need | Existing (path:symbol) | Decision | Reason |
|---|---|---|---|
| The rules Windows changes, shared by both lookups | — | new | `internal/execpath`: no existing package holds program lookup; `exec.LookPath` reads the process's own environment, not the one dkt is given |
| Finding git | internal/gitx/git.go:lookPath | extend | Tries each name `execpath.Names` gives, with `execpath.Getenv` and `execpath.MayRun` |
| Finding OpenSpec | internal/openspec/runner.go:resolve | extend | The same, for PATH lookups and for a program path in `DOCKET_OPENSPEC` |
| The execute check of OpenSpec | internal/openspec/runner.go:isExecutable | extend | Uses `execpath.MayRun` |
| Reading OpenSpec's environment | internal/openspec/runner.go:envValue | extend | Becomes `execpath.Getenv` |
| Refusing unsafe arguments to a batch file | internal/openspec/runner.go:exec | extend | Checks the arguments before starting a `.bat` or `.cmd` program; returns the new `*UnsafeArgError` |
| LF checkouts | — | new | `.gitattributes`; the repository has none |
| The Windows CI job, the OpenSpec job | .github/workflows/ci.yml | extend | Gain the tests above and a Windows entry |

---

### Task 1: Finding programs on Windows

**Scenarios:** platform-support#git-on-windows, platform-support#openspec-installed-by-npm-on-windows, platform-support#extension-given, platform-support#extensions-in-order, platform-support#name-alone-on-other-systems, platform-support#path-spelled-as-windows-does

**Review:** judgment

**Files:**
- Create: `internal/execpath/execpath.go`
- Create: `internal/execpath/execpath_test.go`
- Modify: `internal/gitx/git.go`
- Modify: `internal/gitx/path_test.go`
- Modify: `internal/openspec/runner.go`
- Modify: `internal/openspec/runner_test.go`
- Create: `internal/gitx/path_windows_test.go`
- Create: `internal/openspec/runner_windows_test.go`
- Modify: `cmd/dkt/main_test.go`

**Interfaces:**
- `execpath.Getenv(env []string, key string) (string, bool)`: the last value of `key`; names compare without case on Windows only.
- `execpath.Names(name string, env []string) []string`: on Windows, `name` plus each `PATHEXT` extension in order (lower-cased, a leading dot added where missing; `.com;.exe;.bat;.cmd` when unset or empty), or `[name]` when `name` already ends in one of them, compared without case; elsewhere `[name]`.
- `execpath.MayRun(fi fs.FileInfo) bool`: true on Windows; elsewhere whether any execute bit is set.
- Each has an unexported form taking whether the system is Windows, so the tests cover both rules on every system.

- [x] `execpath_test.go`, on every system, covers both rules: `Names` with `PATHEXT` unset, empty, set in mixed case, with an entry written without its dot, and with the name already carrying an extension in another case; `Getenv` with `Path` against `PATH`, and with two entries (the last wins); `MayRun` with and without execute bits.
- [x] On macOS and Linux, the existing tests of `internal/gitx` and `internal/openspec` pass unchanged apart from the `testutil.Exe` and `os.PathListSeparator` edits, including those of a relative `PATH` entry: `TestRelativePathEntryResolvesAgainstDir` and `TestEmptyPathEntryResolvesAgainstDir` (git resolves it against its folder), and the OpenSpec tests that a relative entry is skipped and a relative `DOCKET_OPENSPEC` program resolves against its folder.
- [x] `internal/openspec` and `internal/gitx`'s path tests pass on Windows, with the programs they make named with `testutil.Exe`: `TestDefaultFromPath`, `TestRelativeOverrideResolvesAgainstDir`, `TestUnstartableProgramIsUnavailable`, `TestRelativePathEntryResolvesAgainstDir`, `TestEmptyPathEntryResolvesAgainstDir`.
- [x] On Windows, through the real lookups: git is found from an environment spelling `Path` (`path_windows_test.go`); with two spellings of `Path`, the last one decides, for git and for OpenSpec; with `PATHEXT=.CMD;.EXE` and both `openspec.exe` and `openspec.cmd` in one folder, OpenSpec resolves to `openspec.cmd`, and to `openspec.exe` once the order is reversed; `DOCKET_OPENSPEC` naming an absolute `openspec.exe` resolves to that file, and naming `./bin/openspec` (no extension) resolves to `bin\openspec.exe` beside it (`runner_windows_test.go`).
- [x] `cmd/dkt`'s tests pass on Windows, running the `dkt` they build as `dkt` plus `testutil.Exe`.
- [x] Making `Names` return the name alone on Windows makes `execpath_test.go` and `TestDefaultFromPath` on Windows fail; making `Getenv` compare names exactly does the same for its test.

### Task 2: Plain arguments to a batch file

**Scenarios:** platform-support#change-name-with-a-command-separator, platform-support#plain-arguments-pass

**Review:** judgment

**Files:**
- Modify: `internal/openspec/runner.go`
- Create: `internal/openspec/batch_test.go`
- Modify: `internal/openspec/runner_windows_test.go`

**Interfaces:**
- `openspec.UnsafeArgError{Program, Arg string}`: its message names the argument, says that `cmd.exe` would read it, and says that a `.bat` or `.cmd` OpenSpec takes only letters, digits and `-_.,/@+=:`.

- [x] `batch_test.go`, on every system, through the check taking whether the system is Windows: each of `& | < > ^ % ! ( ) " '`, a space and a non-ASCII letter in an argument is refused for a `.cmd` and a `.BAT` program on Windows; `add-x`, `identity/user-auth`, `claude,codex`, `@fission-ai/openspec@1.14.0`, `--deltas-only` pass; nothing is refused for an `.exe` program, nor on other systems.
- [x] On Windows, a `Runner` whose program is a `.cmd` fake returns an `*UnsafeArgError` for `show add-x&calc`, and the fake's log shows it was not started (`runner_windows_test.go`).
- [x] Removing the check makes `batch_test.go` and the Windows test fail.
- [x] `TestRealOpenSpec` and `TestRealOpenSpecConflicts` still pass on Windows through `npx.cmd` (Task 4's job).

### Task 3: LF checkouts

**Scenarios:** platform-support#built-from-a-windows-checkout

**Review:** checks

**Files:**
- Create: `.gitattributes`

- [x] `.gitattributes` holds `* text=auto eol=lf`, with a comment saying why; `git ls-files --eol` shows no file whose index copy has CRLF, so no file changes on renormalizing.
- [x] On Windows, `TestShippedStarts` (root package) and `TestEarlierHoldsShippedFiles` (`internal/setup`) pass, which failed there with CRLF checkouts.

### Task 4: Windows CI

**Scenarios:** platform-support#openspec-installed-by-npm-on-windows, platform-support#built-from-a-windows-checkout

**Review:** checks

**Files:**
- Modify: `.github/workflows/ci.yml`

- [x] The Windows job fetches the whole history (as the `test` job does, for the release tags) and also runs `go test` of `internal/execpath`, `internal/openspec`, `cmd/dkt` and the root package, `-run 'PathEntry'` of `internal/gitx`, and `-run 'TestEarlierHolds'` of `internal/setup`; its name is unchanged.
- [x] The `openspec` job runs on `ubuntu-latest` and `windows-latest` for each OpenSpec version, and `TestRealOpenSpec` and `TestRealOpenSpecConflicts` pass on both.
- [x] Every other job is unchanged and passes.
