---
title: "Release dkt on a version tag and install it with install.sh"
kind: feature
source: "Rebuild plan, step 6: release process and install.sh"
specs:
  release: Release
---

# Proposal

## Why

The docket-init skill installs dkt with `curl -fsSL https://raw.githubusercontent.com/wake/docket/main/install.sh | sh`, and the plugin is meant to be installed from this repository, but `install.sh`, the release build and the marketplace file do not exist. Docket cannot be installed by anyone until they do.

## What Changes

- Pushing a tag `vX.Y.Z` on `main` runs a GitHub Actions workflow that builds dkt with GoReleaser for macOS and Linux, on Intel and ARM, and publishes the executables and `checksums.txt` as a GitHub release. The file names hold no version. The executables print the tag as their version.
- New `install.sh`: downloads the executable for the machine and `checksums.txt` from the latest release, or the one `DKT_VERSION` names, verifies the SHA-256, and installs to `~/.local/bin/dkt`.
- New `.claude-plugin/marketplace.json`: this repository is the plugin's marketplace, and its source points at the tag of the current version. `.claude-plugin/plugin.json` gets the same version, starting at `0.16.0`, the next version after v0.15.0.
- A test fails when the plugin's version and the marketplace's source disagree. The release workflow runs it with the tag, so a tag that disagrees publishes nothing.
- CI builds a snapshot of every target on each pull request, publishing nothing, and installs it with `install.sh`.
- New `docs/release.md`: the steps of a release.

## Non-Goals

- Pushing a tag or publishing a release: the first public release is the maintainer's decision.
- Windows executables: dkt does not build for Windows today, and its test suite cannot run there. See design.md.
- A way to choose the install directory, a self-update command, Homebrew or other package managers.
- Signing or notarizing the executables, or signing `checksums.txt`.
- A tool that makes the version bump: it is two lines in two files.
- A plugin format for Codex: not settled, and not tested.

## Capabilities

### New Capabilities
- `release`: how a version of dkt is published, how the plugin follows that version, and how `install.sh` installs a published dkt.

### Modified Capabilities

## Impact

- New: `.goreleaser.yaml`, `.github/workflows/release.yml`, `install.sh`, `.claude-plugin/marketplace.json`, `release_test.go`, `docs/release.md`.
- Changed: `.github/workflows/ci.yml`, `.claude-plugin/plugin.json`, `.gitignore`.
- CI gains a job that downloads GoReleaser. The release workflow holds `contents: write`; nothing else does.
- `install.sh` fails until the first release that has executables: the current latest release, v0.15.0, has none.
- No change to dkt's code, and no new Go dependency.
