---
title: Ship dkt for Windows
---

The first releases ship macOS and Linux only. The maintainer chose that on 2026-11-06, when a GoReleaser snapshot showed that dkt does not build for Windows. This is what adding Windows takes, as found at `c83d025`; nothing has been run on Windows.

## What does not build

- `syscall.O_NOFOLLOW`: `Install` and `ReadNoFollow` in `internal/safefile/safefile.go`, `internal/cli/cmd_new.go`, `internal/cli/cmd_record.go`.
- `syscall.Flock`, `LOCK_EX`, `LOCK_UN`: `lockOpenSpec` in `internal/cli/cmd_new.go`, which locks the `openspec` directory so that two runs of `dkt new` cannot interleave.

These need a port, not only a build fix. The specs require the no-follow reads and the refusal of symbolic links (`change-creation`, `records`, `project-setup`, `plan-sync`), so a flag of 0 would drop that protection without saying so. The lock needs another way to hold off a second run.

## What builds but is unverified

`syscall.ENOTDIR` and `syscall.ENOTEMPTY` exist on Windows, but what Windows returns in those cases has not been checked: `internal/scan/scan.go`, `internal/scan/records.go`, `internal/lint/record.go`, `internal/cli/cmd_new.go`.

## Why CI could not verify a port today

- These do not build for Windows: `syscall.Umask` in `internal/setup/setup_test.go`, `syscall.Mkfifo` in `internal/setup/fix_test.go`.
- These fakes are `/bin/sh` scripts: `FakeOpenSpec` in `internal/testutil/openspec.go`, and those in `internal/cli/cmd_conflicts_gone_test.go` and `internal/gitx/path_test.go`. `devbin_test.go` and `release_test.go` already skip on Windows.

A port needs fakes that run there and a `windows-latest` CI job.

## Once it builds

- Add `windows` to `goos` in `.goreleaser.yaml`, and check that the asset name gets `.exe`, which has not been tried.
- Add the targets to `TestInstallSnapshot` and to the `release` spec.
- `install.sh` stays POSIX and keeps refusing Windows shells with a pointer to the releases page, so Windows users need their own way to install.
