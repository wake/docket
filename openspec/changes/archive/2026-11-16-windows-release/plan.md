# Implementation Plan

**Goal:** A release holds the Windows executables with their digests, the snapshot builds check that from what GoReleaser built, and the specs, the CI job's name and the README say dkt is supported on Windows.

**Architecture:** `.goreleaser.yaml` gains `windows` in `goos`; the binary archives' name template stays `dkt-{{ .Os }}-{{ .Arch }}`, and GoReleaser adds `.exe` to a Windows binary. `TestInstallSnapshot`, which the `release-snapshot` CI job runs on the real GoReleaser output, checks every executable's name and its line in `checksums.txt`, then installs as before. The Windows CI job is renamed, and the README gains Windows.

**Tech Stack:** Go, GoReleaser v2, GitHub Actions.

**Spec:** `openspec/changes/windows-release/proposal.md`, `specs/release/spec.md`, `specs/platform-support/spec.md`.

## Global Constraints

- dkt's behavior does not change on any system; no Go code outside `release_test.go` changes.
- No new dependency. `install.sh` stays POSIX; only its message for a Windows shell changes.
- Asset names and checksums are verified on what GoReleaser built in CI (`release-snapshot`), not by reading the configuration: GoReleaser is not installed locally, and no release is published while releases are on hold.
- Ready only once the fix of `docket/issues/2026-11-16-file-as-folder-windows.md` is on `main` and the skip of `TestDomainsChangesIsAFile` on Windows is gone.

## Review Focus

- A Windows executable without `.exe`, or a version in any name, fails CI.
- `checksums.txt` is checked line by line against the files built, not just for the names.
- The install from the snapshot still runs on the CI machine as before.

## Reuse

| Need | Existing (path:symbol) | Decision | Reason |
|---|---|---|---|
| Windows in the release builds | .goreleaser.yaml:builds[0].goos | extend | `windows` added |
| Checking the snapshot's names and digests | release_test.go:TestInstallSnapshot | extend | It already reads GoReleaser's `artifacts.json` and copies the assets; it gains the Windows names and calls the digest check |
| Checking `checksums.txt` against the files | — | new | `checkSums` in `release_test.go`: nothing in the repository parses `checksums.txt` but `install.sh`, which checks one file; a function of its own gets tests with fixed data, since a snapshot is always consistent |
| The message for a Windows shell | install.sh:main | extend | A case for `MINGW*`, `MSYS*` and `CYGWIN*` before the unsupported-system case |
| The Windows CI job's name | .github/workflows/ci.yml:jobs.windows.name | extend | Renamed; the comment about the name goes |
| Installing on Windows | README.md:Install | extend | Requirements and a Windows install section |

## Revision 1 (2026-11-16): a stricter checksum check, and the release checklist

The pre-merge review on the other model (codex) found that `checkSums` let a later line overwrite an earlier one, so a wrong digest followed by the right one passed, and that extra lines went unchecked; Task 1's acceptance item now asks for exactly one line per file and nothing else. It also found that `docs/release.md` still asked a maintainer to look for four executables after a release; it now names six. Its finding that a renamed check could stall branch protection does not hold: `main` has no branch protection and no rulesets.

---

### Task 1: Build and check the Windows executables

**Scenarios:** release#version-tag-pushed, release#another-tag-pushed, release#tag-outside-the-default-branch, release#two-tags-on-one-commit, release#executable-reports-the-tag, release#snapshot-built-and-installed, release#asset-names-drift, release#checksum-missing, platform-support#build-for-windows

**Review:** judgment

**Files:**
- Modify: `.goreleaser.yaml`
- Modify: `release_test.go`
- Modify: `docs/release.md`

- [x] `TestInstallSnapshot` expects exactly `checksums.txt`, `dkt-darwin-amd64`, `dkt-darwin-arm64`, `dkt-linux-amd64`, `dkt-linux-arm64`, `dkt-windows-amd64.exe` and `dkt-windows-arm64.exe` among the snapshot's assets, and fails on any other set, a Windows name without `.exe` included.
- [x] `checkSums(sums []byte, files map[string][]byte) error` accepts `checksums.txt` text only when its lines are exactly one `<sha256>  <name>` per file, with that file's digest, and nothing else; a test with fixed data covers a valid set, a missing line, a wrong digest, a wrong line before the right one, a line twice, another file, CRLF, one space and a `*` binary marker, and `TestInstallSnapshot` calls it on the snapshot's executables.
- [x] With `windows` in `goos`, the `release-snapshot` CI job passes: `goreleaser check`, the snapshot build and `TestInstallSnapshot`, whose log lists the assets it found; the install from the snapshot still prints the stamped version.
- [x] Other tags, tags off `main`, two tags on one commit and the stamped version keep the checks they have, unchanged: `.github/workflows/release.yml` and `TestPluginVersion` are not modified, and the Windows executables get the same `-X ...cli.Version={{ .Tag }}` stamp from the one build entry.
- [x] Removing `windows` from `goos` makes `TestInstallSnapshot` fail in CI, seen on a throwaway branch.

### Task 2: Say dkt is supported on Windows

**Scenarios:** platform-support#whole-suite-on-windows

**Review:** checks

**Files:**
- Modify: `.github/workflows/ci.yml`
- Modify: `README.md`

- [x] The Windows CI job's name no longer says dkt is unsupported or that the job covers only the build and the protections, and the comment that said the name understated the job is gone; the job still runs `go test ./...`.
- [x] The README lists Windows on Intel and ARM among the supported systems, and its install section tells Windows users to download `dkt-windows-<arch>.exe` and `checksums.txt` from the releases page, compare the digest with PowerShell's `Get-FileHash`, and put the file on `PATH` as `dkt.exe`; it says `install.sh` is for macOS and Linux.

### Task 3: Point a Windows shell to the Windows executables

**Scenarios:** release#windows-shell, release#unsupported-system, release#home-is-not-an-absolute-path, release#unsupported-architecture, release#download-fails, release#target-is-a-directory, release#downloaded-file-does-not-run, release#interrupted-during-a-download

**Review:** checks

**Files:**
- Modify: `install.sh`
- Modify: `release_test.go`

- [x] When `uname -s` reports `MINGW64_NT-10.0`, `MSYS_NT-10.0` or `CYGWIN_NT-10.0`, `install.sh` exits non-zero before downloading anything, and its message says it does not install on Windows and names the system, `dkt-windows-amd64.exe`, `dkt-windows-arm64.exe`, `checksums.txt` and the releases page; the install test's Windows shell case checks this for the three systems.
- [x] `FreeBSD` still gets the unsupported-system message, and the other failure cases of the install tests pass unchanged.
