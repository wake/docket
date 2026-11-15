---
title: Make dkt work on Windows
needs: [docket/ideas/windows-test-fakes.md]
---

The third part of shipping dkt for Windows, split off from `windows-executables` on 2026-11-16. Once it builds, dkt still cannot find git or OpenSpec on Windows, so nearly every command fails:

- `lookPath` in `internal/gitx/git.go` looks for `git`, not `git.exe`, wants the `0o111` permission bits, which Go does not report on Windows, and reads only `PATH=`, while Windows usually spells it `Path=`.
- `resolve` and `isExecutable` in `internal/openspec/runner.go` do the same; npm installs OpenSpec as `openspec.cmd`, found through `PATHEXT` and run through cmd.exe, where Go limits how arguments can be escaped.

Git for Windows also checks files out with CRLF line endings, so the shipped files dkt embeds, and their hashes, differ from those built elsewhere; a `.gitattributes` asking for LF on every system fixes that.

Split again on 2026-11-16, after two runs of the whole suite on Windows (239 failing tests at `main`, 106 with the above fixed): this idea covers finding git and OpenSpec, the LF checkouts, and the tests about finding programs (`internal/openspec`'s runner tests, `internal/gitx`'s path tests, `cmd/dkt`'s build tests), which the Windows CI job then also runs. `windows-paths`, `windows-user-folders` and `windows-suite` cover the rest.

- Kind: a full review, a change request to `openspec-runtime` (Locating the CLI) and `platform-support`.
- The rules each package has for a relative PATH entry stay as they are: git resolves it against the folder it runs in, OpenSpec skips it.
