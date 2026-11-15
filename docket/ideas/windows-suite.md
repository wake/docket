---
title: Run the whole test suite on Windows
needs: [docket/ideas/windows-paths.md, docket/ideas/windows-user-folders.md]
---

Part of shipping dkt for Windows, split off from `windows-runtime` on 2026-11-16. After `windows-runtime`, `windows-paths` and `windows-user-folders`, what is left failing on Windows are tests that assume a Unix system. Fix or skip them, with the reason, and widen the `windows-latest` CI job to `go test ./...`, renaming it once dkt runs there.

## What the Windows probes found

On 2026-11-16, besides the failures of the other two ideas:

- Files made unreadable with `chmod 000`, which Windows ignores: `internal/lint` 7, several in `internal/cli`, `internal/gitx` `TestToplevelUnreadableGitDir`, `internal/activity` `TestReadMissingAndUnlistable`.
- File names Windows forbids (a tab, a double quote): `internal/gitx` `TestUncommittedUnderOddName`, `TestStatusRecords`, `TestWorktreesUnusualPaths`.
- Permission bits: `internal/setup` `TestWriteModes`, `internal/projects` `TestSaveRoundTrip`.
- Unix absolute paths in test data (`/w/a`, `/p`): `internal/projects` 4; `internal/activity` 3 with `D:\x`-style results.
- `internal/snapshot` `TestSnapshotWritesNothing` times files from the parent's directory listing (`DirEntry.Info`), which NTFS updates lazily; with `os.Lstat` it passes on Windows (5 runs of 5). Not a write by `dkt snapshot`.
- About 14 more in `internal/cli` with no single cause yet (`dkt projects`, `dkt init` listing what OpenSpec changed).

## Done when

- `go test ./...` passes on `windows-latest`, and the job's name no longer says dkt is unsupported there; macOS and Linux tests still pass unchanged in what they check.
- Kind: a light review (`test`); a product bug found on the way becomes an issue, or a change of its own.
