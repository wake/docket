---
title: "Build dkt for Windows without dropping its protections"
kind: feature
source: "docket/ideas/windows-executables.md"
specs:
  platform-support: Platform support
---

# Proposal

## Why

dkt does not build for Windows: it opens files with `O_NOFOLLOW` and locks folders with `flock`, which Windows lacks. Making it build by passing a flag of 0 would quietly drop the refusal of symbolic links that several specs require, and leave two runs of `dkt new` or `dkt issue` free to interleave. This change is the first of four parts toward shipping dkt for Windows (`windows-test-fakes`, `windows-runtime` and `windows-release` follow): it makes dkt build there with the same protections, and verifies those protections on Windows in CI.

## What Changes

- dkt builds for Windows.
- Reading a file without following a link refuses a symbolic link on Windows as it does on macOS and Linux.
- On Windows, a junction counts as a symbolic link wherever dkt refuses one, on the file and on the folders leading to it.
- Creating a file that must not exist yet (a new record, a new proposal, the temporary file a write goes through) refuses a link at that name on every system.
- The lock that keeps two runs of `dkt new` or of `dkt issue` with a card from interleaving works on Windows; when it cannot be taken, the command fails and says why, writing nothing.
- A new CI job on `windows-latest` builds and vets dkt and runs the tests of the protections. Its name says that dkt is not yet supported on Windows.

## Non-Goals

- Running dkt on Windows: it cannot yet find git or OpenSpec there (`windows-runtime`).
- Running the whole test suite on Windows: the tests stand in for OpenSpec and git with `/bin/sh` scripts (`windows-test-fakes`).
- Releasing Windows executables or changing `install.sh` (`windows-release`).
- Checking what Windows returns where dkt tests for `ENOTDIR` or `ENOTEMPTY`; that is verified once the whole suite runs there (`windows-runtime`).
- Any change to behavior on macOS or Linux.
- A new dependency.

## Capabilities

### New Capabilities

- `platform-support`: which systems dkt supports, what is verified on Windows before it is supported there, and that a protection the specs require is never dropped on a system that lacks the means for it.

### Modified Capabilities

None.

## Impact

- `internal/safefile`: the no-follow read and the flags for creating a new file, per system.
- `internal/cli/cmd_new.go`, `internal/cli/cmd_issue.go`, `internal/cli/cmd_record.go`: use the shared lock and the shared flags.
- A new package for the folder lock, with a Windows implementation over `LockFileEx` loaded from `kernel32.dll`.
- `go.mod`: a `godebug` line, so that Windows reports junctions as symbolic links.
- `internal/setup` tests that use `Umask` and `Mkfifo` build only on systems that have them.
- `.github/workflows/ci.yml`: the Windows job.
