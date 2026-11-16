---
title: "Release and support dkt on Windows"
kind: change-request
source: "docket/ideas/windows-release.md"
---

# Proposal

## Why

dkt now finds its programs, paths and user folders on Windows, keeps its protections there, and passes the whole test suite on `windows-latest`, but no release holds a Windows executable, and the specs still say dkt is not supported on Windows, with a CI job named to match.

## What Changes

- A release also holds `dkt-windows-amd64.exe` and `dkt-windows-arm64.exe`, and `checksums.txt` holds their digests.
- The snapshot CI builds on every pull request also check the names of every executable and that `checksums.txt` holds the digest of each, from what GoReleaser actually built.
- `platform-support`: dkt is supported on Windows, as on macOS and Linux; the Windows CI job runs the whole test suite and its name no longer says otherwise.
- The README lists Windows among the supported systems and tells Windows users to download the executable from the releases page and check it against `checksums.txt` with PowerShell's `Get-FileHash`.

## Non-Goals

- `install.sh` installing on Windows: it stays POSIX and keeps refusing Windows shells, but its message there now says it does not install on Windows and names the Windows executables, instead of calling the system unsupported.
- The plugin skills that install or update dkt (`docket-init`, `docket-upgrade`) run `install.sh`; telling Windows users there to download by hand is prompt text, left for a change of its own.
- Publishing a release: the next one is on hold, so this change is verified on the snapshot builds.
- Any change to how dkt behaves on any system.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `platform-support`: Supported systems is replaced by Supported on macOS, Linux and Windows (OpenSpec does not let a modified requirement drop a scenario, and the old scenario asked for the name this change removes).
- `release`: Publishing a version tag, Snapshot on every pull request, Failing without installing (a Windows shell gets its own scenario and message).

## Impact

- `.goreleaser.yaml`: `windows` in `goos`.
- `release_test.go`: `TestInstallSnapshot` expects the Windows executables and checks `checksums.txt` through a helper with tests of its own; the install tests' Windows shell case.
- `install.sh`: the message for a Windows shell.
- `.github/workflows/ci.yml`: the Windows job's name and comment.
- `README.md`: requirements and installing on Windows.
