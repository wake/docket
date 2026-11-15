# Implementation Plan

**Goal:** dkt builds for Windows with the protections it has on macOS and Linux (links refused, new files never through a link, the folder lock), and a Windows CI job verifies those protections.

**Architecture:** Per-system files (`_windows.go` and `//go:build !windows`) hold what differs: `safefile` gains the shared flags for creating a new file and a Windows `ReadNoFollow`; a new `internal/lock` package takes `lockDir` from `internal/cli`, with `flock` on macOS and Linux and `LockFileEx` on a lock file named for the folder on Windows; `go.mod` sets `winsymlink=0` so that Windows reports junctions as links to the existing checks. A `windows-latest` job builds, vets, and tests these packages.

**Tech Stack:** Go, standard library only (`syscall`, `kernel32.dll` through `syscall.NewLazyDLL`); GitHub Actions.

**Spec:** `openspec/changes/windows-executables/proposal.md`, `design.md`, `specs/platform-support/spec.md`.

## Global Constraints

- The macOS and Linux code paths do what they do now; the existing tests pass without being changed, except for moving the `Umask` and `Mkfifo` tests into Unix-only files as they are.
- No new dependency in `go.mod`; its only change is the `godebug` line.
- Structural and behavior edits go in separate commits: moving `lockDir` into `internal/lock` comes before the Windows lock.
- Code that runs only on Windows is verified in the Windows CI job: its tests pass there, and one run of the job on a throwaway branch, with each Windows protection disabled, shows each of their tests failing. That branch is deleted afterwards and never merged. The job is added in Task 3, so the Windows items of Tasks 1 and 2 are checked once it exists.

## Review Focus

- On Windows, no path goes on without a protection: a link or junction is refused, and a lock that cannot be taken stops the command before it writes.
- macOS and Linux behave exactly as before.
- The Windows job's name, and the spec, do not suggest that dkt can be used on Windows.

## Reuse

| Need | Existing (path:symbol) | Decision | Reason |
|---|---|---|---|
| The flags for creating a new file without following a link | internal/safefile/safefile.go:InstallIn | extend | The flags `InstallIn` and `openNewFile` both pass become one exported value per system |
| A read that does not follow a link | internal/safefile/safefile.go:ReadNoFollow | extend | Gains a Windows version; the macOS and Linux one stays |
| Opening a new record or proposal | internal/cli/cmd_record.go:openNewFile | extend | Uses the shared flags |
| The folder lock | internal/cli/cmd_new.go:lockDir | extend | Moves to `internal/lock` as `Dir`, so its tests can run on Windows without `internal/cli`'s shell fakes, and gains a Windows version |
| Junctions reported as links | external: Go `winsymlink` setting | extend | One line in `go.mod` makes the existing `ModeSymlink` checks catch junctions |
| `LockFileEx`, `UnlockFileEx` | external: kernel32.dll | extend | Not in `syscall`; loaded with `syscall.NewLazyDLL` instead of adding `golang.org/x/sys` |
| The Windows CI job | .github/workflows/ci.yml | extend | A new job beside `test` |
| Unix-only test helpers | internal/setup/setup_test.go:umask, internal/setup/fix_test.go:TestDecideNonRegularIsUnreadable | extend | Move into files that build only where `Umask` and `Mkfifo` exist |

---

### Task 1: Links refused on Windows

**Scenarios:** platform-support#symbolic-link-read-on-windows, platform-support#junction-on-the-path-on-windows, platform-support#link-at-a-new-file-s-name

**Review:** judgment

**Files:**
- Modify: `go.mod`
- Modify: `internal/safefile/safefile.go`
- Create: `internal/safefile/nofollow_unix.go`
- Create: `internal/safefile/nofollow_windows.go`
- Modify: `internal/safefile/safefile_test.go`
- Create: `internal/safefile/safefile_windows_test.go`
- Create: `internal/safepath/safepath_windows_test.go`
- Modify: `internal/cli/cmd_record.go`
- Modify: `internal/cli/cmd_record_test.go`
- Modify: `internal/cli/cmd_new_test.go`

**Interfaces:**
- `safefile.CreateFlags`: `os.O_WRONLY|os.O_CREATE|os.O_EXCL`, plus `syscall.O_NOFOLLOW` on systems that have it. `InstallIn` and `openNewFile` open with it; Task 2 does not use it.
- `safefile.ReadNoFollow(p string) ([]byte, error)`: unchanged signature; on Windows a link at `p` fails with a `*fs.PathError` wrapping `syscall.ELOOP`.

- [x] `GOOS=windows go vet ./internal/safefile/ ./internal/safepath/` passes on macOS; `syscall.O_NOFOLLOW` appears only in files that do not build on Windows. (The whole build for Windows needs Task 2 and is checked in Task 3.)
- [x] On every system, opening with `CreateFlags` a name where a symbolic link to a file outside the folder stands fails with an error matching `fs.ErrExist`, and the file outside is unchanged; covered by a test in `safefile_test.go` that runs on macOS, Linux and Windows.
- [x] On Windows, `ReadNoFollow` of a symbolic link to a regular file fails with `syscall.ELOOP` and returns no bytes, and of a regular file returns its bytes; the existing `TestReadNoFollow` covers both and passes in the Windows job.
- [x] On Windows, `ReadNoFollow` of a junction to a folder fails with `syscall.ELOOP`, and opening with `CreateFlags` at a junction's name fails with an error matching `fs.ErrExist` and leaves the folder it points to unchanged; tests in `safefile_windows_test.go`, with the junction made by `mklink /J`.
- [x] On macOS and Linux, a test in `internal/cli` plants a symbolic link at the new record's name after the checks (through the `openNewFile` seam, then calling the real one) and sees `dkt record` fail with the link's target unchanged; the same holds for a new proposal of `dkt new`. Making `CreateFlags` follow links (dropping both `os.O_EXCL` and `syscall.O_NOFOLLOW`, since either alone refuses a link on macOS and Linux) makes it fail. `InstallIn` and `openNewFile` take their flags only from `CreateFlags`.
- [x] On Windows, `ReadNoFollow` of a missing file returns an error matching `fs.ErrNotExist`, as on macOS and Linux.
- [x] On Windows, `safepath.Check` on a path through a junction (made with `mklink /J`) to a folder inside the root returns a `*safepath.SymlinkError` naming the junction, and `safepath.OpenDir` through it does too; tests in `safepath_windows_test.go`, passing in the Windows job.
- [x] Removing the `godebug` line makes the junction tests fail, and making `ReadNoFollow` on Windows skip its link check makes the link test fail, both seen in the throwaway mutation run of the Windows job.
- [x] On macOS and Linux, `go test ./...` passes with the existing tests unchanged.

### Task 2: The folder lock on Windows

**Scenarios:** platform-support#second-run-waits, platform-support#lock-cannot-be-set-up

**Review:** judgment

**Files:**
- Create: `internal/lock/lock.go`
- Create: `internal/lock/lock_unix.go`
- Create: `internal/lock/lock_windows.go`
- Create: `internal/lock/lock_test.go`
- Create: `internal/lock/lock_windows_test.go`
- Modify: `internal/cli/cmd_new.go`
- Modify: `internal/cli/cmd_new_test.go`
- Modify: `internal/cli/cmd_issue_card_fix_test.go`

**Interfaces:**
- `lock.Dir(root, rel string) (unlock func(), err error)`: blocks until the folder `<root>/<rel>` is free and returns the function that releases it, as `lockDir` does now. `dkt new` locks `openspec`, `dkt issue` with a card locks `.`, as now.
- On Windows the lock file is `<os.TempDir()>\docket\<volume serial>-<file index>.lock`, the two numbers in lower-case hex, from `GetFileInformationByHandle` on the opened folder.

- [x] Moving `lockDir` into `internal/lock` is its own commit, with no change in behavior; `TestIssueConcurrent`, `TestIssueCardConcurrent` and `TestRecordConcurrent` and the rest of `go test ./internal/cli/` pass unchanged on macOS and Linux.
- [x] A test in `lock_test.go`, run on macOS, Linux and Windows: while one `Dir` on a folder is held, a second `Dir` on the same folder through a different spelling of its path (a trailing separator or a `..` step) does not return; once the first is released, it returns.
- [x] A test in `lock_test.go`: `Dir` on two different folders does not wait.
- [x] On macOS and Linux, tests in `internal/cli` make the lock fail (the lock function is a package variable tests replace): `dkt new` and `dkt issue` with a card exit with an error naming the failure, and the working tree has no new or changed file.
- [x] On Windows, when the temporary folder (through `TMP`) is a regular file, `Dir` returns an error whose message names the `docket` folder it tried to create and the reason, and returns no unlock function; test in `lock_windows_test.go`.
- [x] On Windows, a lock file is created empty and is still there after the release; test in `lock_windows_test.go`.
- [x] A comment in `lock_windows.go` says why `golang.org/x/sys` is not used, and that users with different temporary folders do not hold each other off.
- [x] Making the Windows `Dir` return without calling `LockFileEx` makes the waiting test fail, seen in the throwaway mutation run of the Windows job.

### Task 3: Windows CI job

**Scenarios:** platform-support#build-for-windows, platform-support#windows-check-names-its-limit

**Review:** checks

**Files:**
- Modify: `internal/setup/setup_test.go`
- Modify: `internal/setup/fix_test.go`
- Create: `internal/setup/sys_unix_test.go`
- Create: `internal/setup/sys_windows_test.go`
- Modify: `.github/workflows/ci.yml`

- [x] `GOOS=windows GOARCH=amd64 go build ./...`, the same with `GOARCH=arm64`, and `GOOS=windows go vet ./...` pass on macOS: the `Umask` helper and the FIFO case of `TestDecideNonRegularIsUnreadable` are in files that build only on systems that have them, and the tests that use them still run on macOS and Linux.
- [x] The directory case of `TestDecideNonRegularIsUnreadable` stays in `fix_test.go`, built on every system.
- [x] `.github/workflows/ci.yml` has a job on `windows-latest` named `windows: build and protections only (dkt is not supported on Windows yet)`, which runs `go build ./...`, `go build ./...` with `GOARCH=arm64`, `go vet ./...`, and `go test ./internal/safefile/ ./internal/safepath/ ./internal/lock/`; it passes on the pull request.
- [x] The existing `test`, `openspec` and `release-snapshot` jobs are unchanged and pass.
