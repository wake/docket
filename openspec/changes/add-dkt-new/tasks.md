# Tasks

Each group is one slice an implementer finishes test-first. Scenario names refer to `specs/<capability>/spec.md`. Code marked as ported is read with `git show v0.15.0:<path>` (design.md names each piece). Tests run with `go test ./...`; every git call in a test uses `testutil.Env`, and every OpenSpec call in a unit test uses the fake OpenSpec, never a real CLI.

## Reuse

| Need | Existing (path:symbol) | Decision | Reason | Evidence |
|---|---|---|---|---|
| Envelope, findings, exit codes | `internal/cli/envelope.go:Outcome,Finding,Unreadable,ExitCode,EnvError,UsageError` | reuse | Shared output of every command | `rg -n -e "^type " -e "^func ExitCode" internal/cli/envelope.go` |
| Command table and flag parsing | `internal/cli/command.go:Command`, `internal/cli/run.go:runCommand` | extend | Add `Args` and alternating parsing for the first positional argument | `rg -n -e "type Command" -e "NArg" internal/cli` |
| Locate and run OpenSpec | `internal/openspec/runner.go:New,Runner.CheckVersion,Runner.Run,FailedError` | reuse | Same runner and errors as `dkt init` | `rg -n "^func" internal/openspec/runner.go` |
| OpenSpec errors to exit 4 | `internal/cli/cmd_init.go:runInit` (the `UnavailableError`/`UnsupportedError` switch) | extend | Move the switch into a helper both commands call | `rg -n "openspec-missing" internal/cli` |
| Refuse symbolic links | `internal/safepath/safepath.go:Check,SymlinkError` | reuse | Already refuses a link anywhere on a path under the root | `rg -n "^func Check" internal/safepath` |
| Run git | `internal/gitx/git.go:Git.Run,Git.output` | reuse | Isolated environment and error type already handled | `rg -n "^func \(g Git\)" internal/gitx/git.go` |
| Worktree list | `v0.15.0:internal/gitx/worktree.go:Git.Worktrees,unquotePorcelainPath` | reuse | Port as is: parses `--porcelain -z` entry by entry (`internal/gitx/added.go:splitNUL` drops empty fields and sorts, so it cannot) | `git show v0.15.0:internal/gitx/worktree.go`, `rg -n -A6 "func splitNUL" internal/gitx` |
| Fake OpenSpec | `internal/testutil/openspec.go:FakeOpenSpec` | extend | Add `new change` | `rg -n "init\)" internal/testutil/openspec.go` |
| Test repositories | `internal/testutil/repo.go:NewRepo,Env` | reuse | Isolated git environment used by every test | `rg -n "^func " internal/testutil/repo.go` |
| Slug rules, suffixes, kinds, title form | `v0.15.0:internal/change/new.go:ValidateSlug,Suffixed,Kinds`, `v0.15.0:internal/change/docket.go:ValidTitle` | extend | Port; keep only the date and `archive` rules; the form check also covers the source and valid UTF-8 | `git show v0.15.0:internal/change/new.go`, `git grep -n "func ValidTitle" v0.15.0 -- internal/change` |
| Lock against concurrent runs | `v0.15.0:internal/cli/cmd_new.go:lockOpenSpec` | reuse | Port as is | `git show v0.15.0:internal/cli/cmd_new.go \| rg -n "func lockOpenSpec" -A 18` |
| Remove a failed change folder | `v0.15.0:internal/cli/cmd_new.go:cleanup` (closure in `newCommand`) | reuse | Port as is | `git show v0.15.0:internal/cli/cmd_new.go \| rg -n "cleanup :="` |
| Workspace scan | `v0.15.0:internal/scan/scan.go:Scan,Inventory`, `v0.15.0:internal/scan/trees.go`, `v0.15.0:internal/gitx/worktree.go` | extend | Reduced port: no default branch, no ordering, no records or issues; refs at one commit read once | `git show v0.15.0:internal/scan/scan.go \| rg -n "^func"` |
| Metadata file | `v0.15.0:internal/change/docket.go:CreateDocket` | new | Replaced by the front matter of `proposal.md`; nothing to port | `git show v0.15.0:internal/change/docket.go` |

## 1. Positional arguments

**Review:** judgment (public interface)
**Files:** `internal/cli/command.go`, `internal/cli/run.go`, `internal/cli/command_test.go`

- [x] 1.1 Write failing tests, through `RunWith` with a test command that takes one positional: the positional before, between and after flags is received and the flags keep their values; none, or two, is a usage error (exit 2) naming the command's usage; `init` and `inspect` still refuse any positional; verify they fail
- [x] 1.2 Add `Args` to `Command` and alternate flag parsing with positionals (design.md, Positional arguments); verify every test passes

## 2. Change metadata

**Review:** checks
**Files:** `internal/change/metadata.go`, `internal/change/metadata_test.go`

- [x] 2.1 Write failing tests for the scenario Front matter written (change-metadata), and for the form check: blank, only whitespace, a `\n`, a `\r`, invalid UTF-8 are refused; leading spaces are kept; a title holding `"`, `\`, `:`, `#`, a tab and a non-ASCII letter is written so that it reads back as the same string (decode the quoted value with `strconv.Unquote`); the slug rules: `2026-10-24-add-x` and `archive` refused, `add-2026-10-24` and `archived-x` accepted; `Suffixed`; verify they fail
- [x] 2.2 Implement them (design.md, Front matter); verify every test passes

## 3. Workspace scan

**Review:** judgment (environment-dependent behavior, error paths)
**Files:** `internal/scan/scan.go`, `internal/scan/scan_test.go`, `internal/gitx/worktree.go`, `internal/gitx/refs.go`, `internal/gitx/*_test.go`

- [x] 3.1 Write failing tests for every scenario of workspace-scan, plus: a bare repository's entry and `refs/remotes/origin/HEAD` add no source; two refs at one commit are both sources of its uses; an archived folder whose name does not start with a date, and a file named like a change, are not uses; a link entry inside a worktree's `openspec/changes` is not a use; a branch whose tree holds a file, a symbolic link or a submodule at `openspec/changes/<slug>` reports no use for it; a worktree directory that cannot be read (mode 000) is unreadable with the reason; verify they fail
- [x] 3.2 Implement `internal/scan` and the git helpers (design.md, Workspace scan); verify every test passes

## 4. dkt new

**Review:** judgment (error paths, output format)
**Files:** `internal/cli/cmd_new.go`, `internal/cli/cmd_new_test.go`, `internal/cli/runner.go`, `internal/cli/command.go`, `internal/cli/cmd_init.go`, `internal/testutil/openspec.go`

- [x] 4.1 Extend the fake OpenSpec with `new change` (design.md, Tests), and write failing tests for every scenario of change-creation, the scenarios Kind outside the list, Blank title and Source on two lines (change-metadata), and New outside a repository (cli-output), plus: in the scenario Changes folder is a link, the fake's log is empty (not even `--version` ran); OpenSpec missing and too old exit 4 with nothing created; the fake's log shows `new change <slug> --schema docket`; a `config.yaml` naming another schema still gets `--schema docket`; a stray file `openspec/changes/add-x` in this worktree makes the slug `add-x-2`; a link appearing on the path while OpenSpec runs leaves the folder and gives `symlink-refused`; the text output names the folder and, after a suffix, the used folders with at most three sources; verify they fail
- [x] 4.2 Implement `dkt new` (design.md, Order of steps, Lock, OpenSpec failure, Result); verify every test passes

## 5. Real OpenSpec

**Review:** checks
**Files:** `internal/cli/realopenspec_test.go`

- [x] 5.1 Change `TestRealOpenSpec` to open its change with `dkt new add-x --json --kind feature --source x --title "Add x"`: exit 0, `result.slug` is `add-x`, `.openspec.yaml` names `docket`, `proposal.md` holds the front matter; the `instructions` and `status` calls stay; verify it is skipped without `DOCKET_TEST_OPENSPEC=1` and passes locally with OpenSpec 1.7.0 and the latest
- [ ] 5.2 Verify the `openspec` CI job passes for both versions on the pull request

## 6. Integration

**Review:** checks
**Files:** none beyond earlier groups

- [x] 6.1 Build dkt into a temporary `GOBIN`; in a new repository set up with `dkt init --tools claude` and committed, run `dkt new add-x --kind feature --source x --title "Add x"` twice and check the second creates `add-x-2` and names the first; add a second worktree holding an uncommitted `add-y` and check `dkt new add-y` in the first takes `add-y-2`
- [x] 6.2 Run `gofmt -l .`, `go vet ./...` and `go test ./...`; verify they are clean
