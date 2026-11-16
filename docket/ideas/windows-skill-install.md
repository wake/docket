---
title: Install and update dkt from the plugin skills on Windows
needs: [docket/ideas/windows-release.md]
---

Found by the pre-merge review of `windows-release` (#193), and to be merged before any release that holds the Windows executables. `docket-init` (step 2) and `docket-upgrade` (step 2) install or update `dkt` only by running `install.sh`, which installs only on macOS and Linux; in a Windows shell it stops and names `dkt-windows-<arch>.exe` and `checksums.txt`. A Windows user following either skill stops there.

## What the skills should say on Windows

- How to tell Windows: the coding agent's environment, or `uname -s` naming `MINGW`, `MSYS` or `CYGWIN`.
- The architecture: `PROCESSOR_ARCHITECTURE` (`AMD64` or `ARM64`) picks `dkt-windows-amd64.exe` or `dkt-windows-arm64.exe`.
- Download it and `checksums.txt` with PowerShell (`Invoke-WebRequest`) from `releases/latest/download/`, or from `releases/download/<tag>/` when updating to a chosen version.
- Check the file with `Get-FileHash -Algorithm SHA256` against its line in `checksums.txt`, compared without regard to case; on a mismatch, delete the download and stop.
- Where it goes: as `dkt.exe` in a folder on `PATH`, the same one on every update, such as `%LOCALAPPDATA%\Programs\dkt`; say how to add that folder to the user's `PATH` and that a new shell is needed to see it. Replacing it while a `dkt.exe` runs fails on Windows; say so.
- `docket-upgrade`: finding the `dkt` in use with `Get-Command dkt` instead of `command -v dkt`.
- The README's Windows install steps say the same.

## Done when

- Both plugin skills (`skills/docket-init/SKILL.md`, `skills/docket-upgrade/SKILL.md`) give the Windows steps, and macOS and Linux keep `install.sh`.
- Kind: a light review (`prompt`).
