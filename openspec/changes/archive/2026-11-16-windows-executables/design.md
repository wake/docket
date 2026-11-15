# Design

## Context

What blocks the Windows build, as checked on the current `main` with `GOOS=windows go vet ./...`: `syscall.O_NOFOLLOW` in `safefile.InstallIn`, `safefile.ReadNoFollow` and `openNewFile` in `cmd_record.go`; `syscall.Flock` in `lockDir` in `cmd_new.go`, which `dkt new` and `dkt issue` with a card use; and, in tests, `syscall.Umask` (`setup_test.go`) and `syscall.Mkfifo` (`fix_test.go`). With these out of the way, the rest of the code and the tests build for Windows.

The checks for links on the folders of a path (`safepath.Check`, `safepath.OpenDir`, and every caller that relies on them) test `fs.ModeSymlink` from `Lstat`. Since Go 1.23, Windows reports a junction as a folder with `fs.ModeIrregular`, not as a symbolic link, unless the `winsymlink` setting is `0`.

## Goals / Non-Goals

**Goals:**
- Every protection the code has on macOS and Linux has a Windows counterpart, or fails with a message where it cannot be had.
- The macOS and Linux code paths stay as they are; their tests pass unchanged.

**Non-Goals:**
- No Windows-specific behavior beyond the protections; the commands cannot run on Windows yet, since git and OpenSpec are not found there.

## Decisions

### Junctions as links: `godebug winsymlink=0` in `go.mod`

The `godebug` block of `go.mod` applies to every main package and test of this module, so `Lstat` and `File.Stat` report junctions as `fs.ModeSymlink` everywhere dkt checks for links, with no change to those checks. It is read only on Windows.

Alternative: a Windows-only check of the reparse tag next to each `ModeSymlink` test. It would touch every check, and the standard library exposes the tag only through `Lstat`'s mode. Not chosen. The setting is Go's documented way back to the earlier behavior; if Go removes it, the junction tests in the Windows job fail.

### Creating a new file: one flag set, shared

`O_CREATE|O_EXCL` already refuses a link at the name: POSIX requires `open` to fail with `EEXIST` when the name is a symbolic link, and Go on Windows opens with `FILE_OPEN_REPARSE_POINT` whenever both flags are set. On macOS and Linux the code keeps `O_NOFOLLOW` as it is. `safefile` exports the flags for opening a new file for writing, with `O_NOFOLLOW` on systems that have it and nothing added on Windows, so that the bare 0 never stands apart from `O_EXCL`. `InstallIn` and `openNewFile` use it.

### Reading without following a link on Windows

Open with `CreateFile` and `FILE_FLAG_OPEN_REPARSE_POINT`, so that a link at the name is opened as itself rather than followed; then `File.Stat` on that handle: a symbolic link, or a junction once `winsymlink=0` applies, fails with `ELOOP` in a `*fs.PathError`, as `O_NOFOLLOW` does on macOS and Linux. Any other file is read through the same handle, so nothing can be swapped between the check and the read. Like `O_NOFOLLOW`, this covers the last part of the path only; the folders on the way are the callers' checks.

### The folder lock

A new package holds the lock, so its tests can run on Windows without the rest of `internal/cli`, which needs the `/bin/sh` fakes. `Dir(root, rel)` keeps `lockDir`'s contract: block until the folder `<root>/<rel>` is free, return the function that releases it.

- macOS and Linux: `flock` on the folder, as now.
- Windows: `LockFileEx` cannot lock a folder, so the lock is taken on a file that stands for it. The folder is opened, and `GetFileInformationByHandle` gives its volume serial number and file index, which identify it however its path is spelled, as the inode does for `flock`. The lock file is `docket\<volume>-<index>.lock` in the user's temporary folder (`os.TempDir`), opened for reading and writing with `O_CREATE` and locked exclusively with `LockFileEx`, which blocks; the release calls `UnlockFileEx` and closes it.
- `LockFileEx` and `UnlockFileEx` are not in `syscall`; they are loaded from `kernel32.dll` with `syscall.NewLazyDLL`, rather than adding `golang.org/x/sys`, since `go.mod` has no dependencies and this is a few lines.
- When the temporary folder cannot be found, its `docket` folder cannot be created, the lock file cannot be opened, or `LockFileEx` fails, `Dir` returns an error naming the path and the reason; the command stops before writing. It never goes on unlocked.

### Tests that need a Unix system

The `Umask` test and the `Mkfifo` case move into files built only where those calls exist (`//go:build !windows`), unchanged, so `go vet ./...` builds the tests on Windows.

### The Windows CI job

A job on `windows-latest`, named `windows: build and protections only (dkt is not supported on Windows yet)`, runs `go build ./...`, the same for `GOARCH=arm64`, `go vet ./...`, and `go test` of `internal/safefile`, `internal/safepath` and the lock package. The runner's account may create symbolic links, so the existing tests of these packages run there as written; the junction cases are made with `mklink /J` in Windows-only tests. Later parts widen the job to the whole suite.

## Risks / Trade-offs

- [The temporary folder is per user] → Two users running dkt on the same worktree at the same time on Windows do not hold each other off. Known limit, said in the lock's comment; one worktree is normally used by one user.
- [Lock files are left behind] → Each is an empty file named for one folder, created once and never removed; the system's cleaning of temporary files removes them. Removing one while a run holds it cannot happen, since an open file on Windows cannot be deleted by default.
- [`winsymlink` is removed in a later Go] → The junction tests on Windows fail, and the checks then need the reparse tag directly.
- [Paths longer than 260 characters] → `CreateFile` without Go's long path handling fails on them; the read fails with an error, not silently. Recorded with a `shortcut:` comment, to be lifted in `windows-runtime` if a test meets it.
- [What a non-link reparse point reads as, such as a cloud placeholder] → It is read through the handle; when that fails, the read returns the error.
