---
title: Release dkt for Windows
needs: [docket/ideas/windows-runtime.md]
---

The last part of shipping dkt for Windows, split off from `windows-executables` on 2026-11-16.

- Add `windows` (amd64 and arm64) to `goos` in `.goreleaser.yaml`, and check that the assets are named `dkt-windows-<arch>.exe`.
- Add the targets to `TestInstallSnapshot` and to the `release` spec.
- `install.sh` stays POSIX and keeps refusing Windows shells; the README tells Windows users to download from the releases page and check the file against `checksums.txt` with PowerShell's `Get-FileHash`.
- Kind: a full review, a change request to `release`.
