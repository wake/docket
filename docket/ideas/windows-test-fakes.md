---
title: Test fakes that do not need /bin/sh
needs: [docket/ideas/windows-executables.md]
---

The second part of shipping dkt for Windows, split off from `windows-executables` on 2026-11-16. The tests stand in for OpenSpec and git with `/bin/sh` scripts, which cannot run on Windows, so CI cannot run most of the tests there.

- Rewrite the fakes in Go: `FakeOpenSpec` and `CountGit` in `internal/testutil`, and those in `internal/cli/cmd_conflicts_gone_test.go`, `internal/cli/repo_test.go` and `internal/gitx/path_test.go`. One way is for the test binary to run itself as the fake.
- macOS and Linux run the same fakes, so the existing tests must pass unchanged; this part can merge without a Windows machine.
- Kind: a light review, `test`.
