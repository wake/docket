---
title: Find the user's folders on Windows
needs: [docket/ideas/windows-runtime.md]
---

Part of shipping dkt for Windows, split off from `windows-runtime` on 2026-11-16. dkt finds its user files from `HOME` and the `XDG_*` variables: the project index and the settings (`project-index`), the user's workflow template (`user-workflow`), and the session activity records (`session-activity`). Windows does not set `HOME`, so on Windows dkt reports "cannot locate the user folders: set HOME".

## What the Windows probes found

On 2026-11-16, a whole run of `go test ./...` on `windows-latest` failed for this reason: `internal/projects` 4 (`TestIndexPath`, `TestSessionsDir`, `TestSettingsPath`, `TestUserWorkflowPath`), `internal/cli` 7 (`dkt inspect`'s workflow sync, which finds no user template), `internal/snapshot` 5 (activity). Some tests also give Unix paths such as `/s` as `XDG_*` values, which are not absolute on Windows.

## Settled (maintainer, 2026-11-16)

- Windows follows the same rules as the other systems: an `XDG_*` variable that is set is used as it is; without one, the folders are under the user's home folder, which on Windows is `USERPROFILE`, at the same paths as on other systems (`.config/docket`, `.local/state/docket`).
- `%APPDATA%` and `%LOCALAPPDATA%` are not used.

## Done when

- The tests above pass on Windows, with macOS and Linux unchanged.
- Kind: a full review, changing `project-index`, `user-workflow` and `session-activity` (or one requirement they share).
