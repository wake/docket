---
title: "Find git and OpenSpec on Windows"
kind: feature
source: "docket/ideas/windows-runtime.md"
---

# Proposal

## Why

On Windows dkt finds neither git nor OpenSpec, so nearly every command reports `git-missing` or `openspec-missing`: it looks for a file named exactly `git` or `openspec`, wants an execute bit that Windows files do not have, and reads `PATH` only when spelled in upper case, while Windows names programs `git.exe` and `openspec.cmd` and usually spells the variable `Path`. A run of the whole suite on Windows failed 204 of its 239 failing tests for this reason. And a checkout made by Git for Windows converts the shipped files to CRLF, so a dkt built there embeds different bytes from one built elsewhere.

## What Changes

- On Windows, dkt looks for git, OpenSpec and a program named in `DOCKET_OPENSPEC` under each extension in `PATHEXT` (`.com`, `.exe`, `.bat` and `.cmd` when it is unset), unless the name already has one of them; any file found counts as runnable, and the environment variables `PATH` and `PATHEXT` are read whatever their case.
- The other rules for finding them stay as they are on every system: the last `PATH` wins; git resolves a relative `PATH` entry against the folder it runs in; OpenSpec skips a relative entry, and resolves a relative `DOCKET_OPENSPEC` program against the folder it runs in.
- `.gitattributes` asks for LF line endings in every checkout.
- The `windows-latest` CI job also runs the tests of finding programs (`internal/execpath`, `internal/openspec`, `cmd/dkt`), and the real OpenSpec test runs on Windows too.

## Non-Goals

- The paths git prints with forward slashes on Windows (`windows-paths`).
- Where the user's files go on Windows (`windows-user-folders`).
- The rest of the test suite on Windows (`windows-suite`); dkt stays unsupported on Windows until `windows-release`.
- Any change on macOS or Linux.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `platform-support`: adds Programs found on Windows, Variable names without case on Windows, and Same shipped files from any checkout.

## Impact

- A new package `internal/execpath`, used by `internal/gitx` (`lookPath`) and `internal/openspec` (`resolve`, `isExecutable`, `envValue`).
- `.gitattributes`.
- Tests in `internal/openspec`, `internal/gitx` and `cmd/dkt` that name executables gain `testutil.Exe`.
- `.github/workflows/ci.yml`: the Windows job's test list, and the OpenSpec job on Windows.
