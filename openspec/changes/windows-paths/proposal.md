---
title: "Print absolute paths on Windows as Windows writes them"
kind: change-request
source: "docket/ideas/windows-paths.md"
---

# Proposal

## Why

Git for Windows prints absolute paths with forward slashes (`C:/Users/me/repo`), and dkt passes on the two it reads from git, the repository root and the worktree paths, as they are, while the paths dkt builds itself use backslashes. The output then mixes both spellings, and a path from git never equals the same path built by dkt. A run of the suite on Windows failed 49 tests whose output holds such paths, among them every test of `internal/scan` and `internal/conflicts` that failed there.

## What Changes

- On Windows, the repository root (`git rev-parse --show-toplevel`) and each worktree path (`git worktree list --porcelain`) are written with backslashes as soon as dkt reads them, so every absolute path dkt prints or compares is spelled as Windows spells it, with the drive letter as git gives it.
- Paths relative to the repository root keep forward slashes on every system, as the specs already show them.
- Tests that write a path into JSON, or look for one in a JSON file, by its raw text now escape it as JSON does, since a Windows path holds backslashes: those of `dkt projects`, of registering a project, and of session activity in `dkt snapshot`.
- The `windows-latest` CI job also runs `internal/scan`, `internal/conflicts`, and those tests.

## Non-Goals

- Short 8.3 names (`RUNNER~1`): dkt keeps the long names git prints and does not rewrite paths the user gives it.
- Tests that give Unix paths as test data, rely on `chmod`, or name a program without an extension (`windows-suite`), among them `internal/activity`'s own tests; and where the user's files go on Windows (`windows-user-folders`).
- Any change on macOS or Linux, where git already writes paths as the system does.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `platform-support`: adds Absolute paths as Windows writes them.

## Impact

- `internal/gitx`: `Toplevel` and `Worktrees`.
- Tests: a Windows test in `internal/gitx`; `internal/cli`'s `TestRepoRootFromSubdirectory`, which expected git's own spelling; the JSON text of paths in `internal/cli`'s `dkt projects` and registration tests and `internal/snapshot`'s activity tests.
- `.github/workflows/ci.yml`: the Windows job's test list.
