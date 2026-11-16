---
title: Find the user's folders on Windows
kind: change-request
source: docket/ideas/windows-user-folders.md
---

# Proposal

## Why

dkt keeps the project index, the settings, the user's workflow template and the session activity records under `$XDG_STATE_HOME` or `$XDG_CONFIG_HOME`, or under `HOME` when those are not set. Windows does not set `HOME`, so there dkt reports "cannot locate the user folders: set HOME" and finds none of these files. On 2026-11-16 the maintainer settled that Windows follows the same rules as the other systems, with the home folder taken from `USERPROFILE` (`docket/ideas/windows-user-folders.md`).

## What Changes

- On Windows, the home folder dkt puts its user files under is the folder `USERPROFILE` names, at the same paths as on other systems (`.config/docket`, `.local/state/docket`). `HOME` is not read there, as Go's `os.UserHomeDir` does not read it on Windows.
- An absolute `XDG_CONFIG_HOME` or `XDG_STATE_HOME` keeps its meaning on Windows.
- On Windows, these variables are read whatever the case of their names, as `PATH` already is.
- The message for a missing home folder names `USERPROFILE` on Windows.
- The Windows CI job also runs the tests that find the user folders.
- macOS and Linux do not change.

## Non-Goals

- No `%APPDATA%` or `%LOCALAPPDATA%`: the maintainer chose the same paths as on other systems.
- No change to what `dkt activity` writes, when it writes, or to the hooks docket-init offers; only where its folder is found.
- No change to docket-init's or the README's text about these folders (`~/.config`, `~/.local/state`), which already reads right for a home folder on any system.
- Not the other Windows failures: paths git prints with forward slashes (`windows-paths`), and tests that assume a Unix system (`windows-suite`). dkt is still not described as supported on Windows.

## Capabilities

### New Capabilities

### Modified Capabilities
- `platform-support`: a new requirement, User folders on Windows, saying where the home folder is on Windows for every requirement that places files under `HOME` (in `project-index`, `user-workflow` and `session-activity`), so that those specs stay as they are.

## Impact

- `internal/projects/files.go` and `register.go`: the home folder's variable and the no-home messages.
- `internal/testutil/repo.go`: the test environment sets the home folder on every system.
- Tests in `internal/projects`, `internal/cli`, `internal/snapshot` and `internal/activity` that remove the home folder or give Unix paths.
- `.github/workflows/ci.yml`: the Windows job runs those tests.
