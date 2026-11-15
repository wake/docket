---
title: Print paths on Windows as Windows writes them
needs: [docket/ideas/windows-runtime.md]
---

Part of shipping dkt for Windows, split off from `windows-runtime` on 2026-11-16. Git for Windows prints absolute paths with forward slashes (`C:/Users/me/repo`), and dkt passes them on as they are, while the paths dkt makes itself use backslashes. The two kinds then fail to compare, and the output mixes both.

## Where the paths come from

- `git rev-parse --show-toplevel` (`internal/gitx/git.go`, `Toplevel`): the repository root, printed as `repo.root` in every command's JSON and used to build other paths.
- `git worktree list --porcelain` (`internal/gitx/worktree.go`, `Worktrees`): each worktree's path, printed as a `source` in `dkt conflicts`, `dkt lint`, `dkt status` and others, and as a worktree's path in `dkt snapshot`, and compared with paths dkt builds.

## What the Windows probes found

On 2026-11-16, a whole run of `go test ./...` on `windows-latest`, with git and OpenSpec found and checkouts in LF (`windows-runtime`), failed 49 tests whose output holds such paths: `internal/cli` 34 (conflicts, domains, lint, inspect, issue, ...), `internal/scan` 12, `internal/conflicts` 3. Every `internal/cli` output holds `repo.root`, so some of those 34 may fail for another reason; the rest of the run's failures belong to `windows-user-folders` and `windows-suite`.

## What the spec should say

An absolute path that dkt prints or compares SHALL be written as the system writes it: on Windows, with backslashes and the drive letter as git gives it. Paths relative to the repository root keep forward slashes on every system, as the specs already show them. macOS and Linux do not change.

## Done when

- `internal/scan`, `internal/conflicts`, and the `internal/cli` tests above pass on Windows.
- Kind: a full review, a change request to `cli-output` (or `platform-support`), with macOS and Linux output unchanged.
