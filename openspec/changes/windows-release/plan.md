# Implementation Plan

**Goal:** A release holds the Windows executables with their digests, the snapshot builds check that from what GoReleaser built, and the specs, the CI job's name and the README say dkt is supported on Windows.

**Architecture:** `.goreleaser.yaml` gains `windows` in `goos`; the binary archives' name template stays `dkt-{{ .Os }}-{{ .Arch }}`, and GoReleaser adds `.exe` to a Windows binary. `TestInstallSnapshot`, which the `release-snapshot` CI job runs on the real GoReleaser output, checks every executable's name and its line in `checksums.txt`, then installs as before. The Windows CI job is renamed, and the README gains Windows.

**Tech Stack:** Go, GoReleaser v2, GitHub Actions.

**Spec:** `openspec/changes/windows-release/proposal.md`, `specs/release/spec.md`, `specs/platform-support/spec.md`.

## Global Constraints

- dkt's behavior does not change on any system; no Go code outside `release_test.go` changes.
- No new dependency. `install.sh` does not change.
- Asset names and checksums are verified on what GoReleaser built in CI (`release-snapshot`), not by reading the configuration: GoReleaser is not installed locally, and no release is published while releases are on hold.
- Ready only once the fix of `docket/issues/2026-11-16-file-as-folder-windows.md` is on `main` and the skip of `TestDomainsChangesIsAFile` on Windows is gone.

## Review Focus

- A Windows executable without `.exe`, or a version in any name, fails CI.
- `checksums.txt` is checked line by line against the files built, not just for the names.
- The install from the snapshot still runs on the CI machine as before.

## Reuse

| Need | Existing (path:symbol) | Decision | Reason |
|---|---|---|---|
| Windows in the release builds | .goreleaser.yaml | extend | `windows` in `goos` |
| Checking the snapshot's names and digests | release_test.go:TestInstallSnapshot | extend | It already reads GoReleaser's `artifacts.json` and copies the assets; it gains the Windows names and a digest check of each against `checksums.txt` |
| The Windows CI job's name | .github/workflows/ci.yml | extend | Renamed; the comment about the name goes |
| Installing on Windows | README.md | extend | Requirements and a Windows install section |

---

### Task 1: Build and check the Windows executables

**Scenarios:** release#version-tag-pushed, release#another-tag-pushed, release#tag-outside-the-default-branch, release#two-tags-on-one-commit, release#executable-reports-the-tag, release#snapshot-built-and-installed, release#asset-names-drift, release#checksum-missing, platform-support#build-for-windows

**Review:** judgment

**Files:**
- Modify: `.goreleaser.yaml`
- Modify: `release_test.go`

- [ ] `TestInstallSnapshot` expects exactly `checksums.txt`, `dkt-darwin-amd64`, `dkt-darwin-arm64`, `dkt-linux-amd64`, `dkt-linux-arm64`, `dkt-windows-amd64.exe` and `dkt-windows-arm64.exe` among the snapshot's assets, and fails on any other set, a Windows name without `.exe` included.
- [ ] `TestInstallSnapshot` checks that `checksums.txt` has one line per executable, whose digest is the SHA-256 of the file built, and fails when a line is missing or differs.
- [ ] With `windows` in `goos`, the `release-snapshot` CI job passes: `goreleaser check`, the snapshot build and `TestInstallSnapshot`, whose log lists the assets it found; the install from the snapshot still prints the stamped version.
- [ ] Other tags, tags off `main`, two tags on one commit and the stamped version keep the checks they have, unchanged: `.github/workflows/release.yml` and `TestPluginVersion` are not modified, and the Windows executables get the same `-X ...cli.Version={{ .Tag }}` stamp from the one build entry.
- [ ] Removing `windows` from `goos`, or editing a digest in the snapshot's `checksums.txt`, makes `TestInstallSnapshot` fail (seen in CI on a throwaway branch, or locally on the downloaded snapshot).

### Task 2: Say dkt is supported on Windows

**Scenarios:** platform-support#whole-suite-on-windows

**Review:** checks

**Files:**
- Modify: `.github/workflows/ci.yml`
- Modify: `README.md`

- [ ] The Windows CI job's name no longer says dkt is unsupported or that the job covers only the build and the protections, and the comment that said the name understated the job is gone; the job still runs `go test ./...`.
- [ ] The README lists Windows on Intel and ARM among the supported systems, and its install section tells Windows users to download `dkt-windows-<arch>.exe` and `checksums.txt` from the releases page, compare the digest with PowerShell's `Get-FileHash`, and put the file on `PATH` as `dkt.exe`; it says `install.sh` is for macOS and Linux.
