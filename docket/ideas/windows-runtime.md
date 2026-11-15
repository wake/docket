---
title: Make dkt work on Windows
needs: [docket/ideas/windows-test-fakes.md]
---

The third part of shipping dkt for Windows, split off from `windows-executables` on 2026-11-16. Once it builds, dkt still cannot find git or OpenSpec on Windows, so nearly every command fails:

- `lookPath` in `internal/gitx/git.go` looks for `git`, not `git.exe`, wants the `0o111` permission bits, which Go does not report on Windows, and reads only `PATH=`, while Windows usually spells it `Path=`.
- `resolve` and `isExecutable` in `internal/openspec/runner.go` do the same; npm installs OpenSpec as `openspec.cmd`, found through `PATHEXT` and run through cmd.exe, where Go limits how arguments can be escaped.

Then widen the `windows-latest` CI job to `go test ./...` and fix what it finds: path separators, CRLF line endings, and the tests that check permission bits (17 files) or make symbolic links, which either hold on Windows too or skip there with the reason.

- Kind: a full review, under the `platform-support` capability.
- How large it is depends on what CI finds; split it again past about 800 lines or 20 files.
