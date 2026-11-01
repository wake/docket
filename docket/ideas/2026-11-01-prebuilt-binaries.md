---
title: Ship prebuilt dkt binaries with each release
---

Recorded 2026-11-01. The maintainer: asking users to install Go and compile
`dkt` themselves is the wrong design; a release ships a binary that runs.

## Problem

The README's only way to install is
`go install github.com/wake/docket/cmd/dkt@latest`, written on 2026-10-24
and not revisited since. The releases have no files attached (v0.10.0 and
v0.11.0 checked). Someone without Go cannot install `dkt`, and every
install compiles it.

## Direction, not decided

- The release step builds `dkt` for macOS and Linux, on arm64 and amd64,
  and attaches the archives and their checksums to the GitHub release:
  GoReleaser, or a job that runs on the tag.
- Users install with one command: a Homebrew tap, an install script, or
  both. `go install` stays for those who build from source.
- `dkt --version` (`2026-10-24-dkt-version.md`) lets the user and
  `docket-doctor` tell which release is installed.
- The files in a Claude Code plugin's `bin/` are on the PATH of the agent's
  shell, so a plugin could bring `dkt` with it
  (`2026-11-01-docket-init-command.md`).

## Open questions

- Which install channels, and whether Windows is built.
- Whether macOS binaries need signing and notarization for the chosen
  channel.
- How the release notes and the archive step (`dkt archive`) fit with a
  release job.

## Related

- `2026-10-24-dkt-version.md`
- `2026-11-01-docket-init-command.md`
