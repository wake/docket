# Tasks

Each group is one slice an implementer finishes test-first. Scenario names refer to `specs/<capability>/spec.md`. Code marked as ported is read with `git show v0.15.0:<path>` (design.md names each piece). Tests run with `go test ./...`; every git call in a test uses `testutil.Env`.

## Reuse

| Need | Existing (path:symbol) | Decision | Reason | Evidence |
|---|---|---|---|---|
| Read without following a link; write keeping the mode | `internal/setup/setup.go:readNoFollow,install,rename` | extend | Move to `internal/safefile` unchanged (with the `rename` test hook), so `dkt sync` writes plan.md the way `dkt init` writes | `rg -n -e "func install" -e "func readNoFollow" -e "rename" internal/setup` |
| Refuse symbolic links | `internal/safepath/safepath.go:Check,SymlinkError` | reuse | Already refuses a link anywhere on a path under the root | `rg -n "^func Check" internal/safepath` |
| Command table, positional slug | `internal/cli/command.go:Command` (`Args`) | reuse | `dkt new` already takes a positional | `rg -n "Args +int" internal/cli` |
| Envelope, findings, exit codes | `internal/cli/envelope.go:Outcome,Finding,Unreadable,ExitCode` | reuse | Shared output of every command | `rg -n -e "^type " -e "^func ExitCode" internal/cli/envelope.go` |
| Run git | `internal/gitx/git.go:Git.output` | reuse | Isolated environment and error type | `rg -n "func \(g Git\) output" internal/gitx` |
| Markdown lines, fences, headings, comments | `v0.15.0:internal/markdown/markdown.go:Lines,FenceMask,Headings,Body,BlankComments` | extend | Port only these, with their tests; `dkt lint` adds the rest | `git show v0.15.0:internal/markdown/markdown.go` |
| Tasks and steps of plan.md | `v0.15.0:internal/change/plan.go:ParsePlan,Task`, `v0.15.0:internal/change/tick.go:Step,Tick,lineStarts` | extend | Port reduced to tasks, steps and cancelled numbers; `Tick` as is | `git show v0.15.0:internal/change/plan.go` |
| Commit trailers | `v0.15.0:internal/gitx/log.go:Git.Commits,splitValues`, test pattern `v0.15.0:internal/gitx/log_test.go:TestCommitsPaths` (a `--no-ff` merge) | extend | Port reduced to the `Change` trailer over the whole history of HEAD | `git show v0.15.0:internal/gitx/log.go`, `git show v0.15.0:internal/gitx/log_test.go` |
| Schema of a change; task references in a trailer | none (`internal/change/metadata.go` holds only `ValidLine`, `CheckSlug`, `Suffixed`, `FrontMatter`) | new | Add `SchemaOf` and `TaskRefs` to `internal/change` | `rg -n -e "Schema" -e "Trailer" -e "openspec.yaml" internal` |

## 1. Shared file helpers

**Review:** checks (restructuring)
**Files:** `internal/safefile/safefile.go`, `internal/safefile/safefile_test.go`, `internal/setup/setup.go`, `internal/setup/setup_test.go`

- [ ] 1.1 Verify `go test ./internal/setup/` passes; move `readNoFollow` and `install` to `internal/safefile` as `ReadNoFollow` and `Install` (design.md, Shared file helpers) and call them from `internal/setup`, changing no test except the one that replaces the `rename` hook; verify `go test ./...` passes and that mutating `Install`'s mode keeping fails a test. Then, in its own commit, add `internal/safefile/safefile_test.go` with a test that `ReadNoFollow` fails on a symbolic link at the file itself, and verify that removing its `O_NOFOLLOW` fails that test

## 2. Plan parsing

**Review:** checks
**Files:** `internal/markdown/markdown.go`, `internal/markdown/markdown_test.go`, `internal/change/plan.go`, `internal/change/tick.go`, `internal/change/plan_test.go`, `internal/change/tick_test.go`

- [ ] 2.1 Port the tests of `Lines`, `FenceMask`, `Headings`, `Body`, `BlankComments` and `Tick`, and write failing tests for `ParsePlan`: tasks in order with number, name, heading line and steps (ticked and not); a step inside a fenced block or an HTML comment is not a step; a task's body ends at the next heading of level 1 to 3 but not at a level-4 heading; `## Cancelled Tasks` bullets give their numbers; a heading `### Task 2 - x` (no colon) is not a task; CRLF and a BOM give the same lines; verify they fail
- [ ] 2.2 Port and implement them (design.md, Plan parsing); verify every test passes

## 3. Trailers and schema

**Review:** checks
**Files:** `internal/gitx/log.go`, `internal/gitx/log_test.go`, `internal/change/metadata.go`, `internal/change/metadata_test.go`

- [ ] 3.1 Write failing tests: `ChangeTrailers` returns oldest first each commit's SHA and its `Change` values, reads a lower-case `change:` key, ignores a `Change:` line outside the last paragraph, unfolds a folded value, returns nothing for a repository whose HEAD is unborn when called that way, parses a history where a commit with no trailer sits between two with trailers (the alternating NUL framing), and reads the commits of both sides of a `--no-ff` merge; `TaskRefs` for `add-x`, `add-x/1`, `add-x/1,2`, `add-x/2, 3`, `add-x-2/1` (not mine), `add-x/` `add-x/0` `add-x/two` `add-x/1,,2` (malformed); `SchemaOf` for the scenarios Schema read and No schema key (change-metadata), a quoted value, a value with a comment, and an indented `schema:` line (not top-level, gives `docket`); verify they fail
- [ ] 3.2 Implement them (design.md, Trailers, Change schema); verify every test passes

## 4. dkt sync

**Review:** judgment (output format, error paths)
**Files:** `internal/cli/cmd_sync.go`, `internal/cli/cmd_sync_test.go`, `internal/cli/command.go`

- [ ] 4.1 Write failing tests for every scenario of plan-sync and Sync outside a repository (cli-output), plus: a plan with a BOM keeps it; a plan file mode 0600 keeps its mode; a value `Change: add-x` alone ticks nothing and is no problem; `trailer-unknown-task` for a cancelled task's number; an unreadable plan.md (mode 000) and an unreadable `.openspec.yaml` are each listed in `unreadable` with exit 3 and nothing written (skip when running as root); a task named by commits whose steps are all ticked gives no finding with `--check`; an unborn HEAD gives `task-no-commit` for every unfinished task with `--check` and ticks nothing without it; `plan-missing`; the text output; verify they fail
- [ ] 4.2 Implement `dkt sync` (design.md, Command); verify every test passes

## 5. Integration

**Review:** checks
**Files:** none beyond earlier groups

- [ ] 5.1 Build dkt into a temporary `GOBIN`; in a new repository set up with `dkt init --tools claude`, open a change with `dkt new`, write a plan.md with three tasks, make one commit carrying `Change: <slug>/1` and one carrying `Change: <slug>/2,3`, and check that `dkt sync <slug> --check` exits 1 naming the tasks, `dkt sync <slug>` ticks them and leaves `git diff` showing only checkbox changes, and `dkt sync <slug> --check` then exits 0
- [ ] 5.2 Run `gofmt -l .`, `go vet ./...` and `go test ./...`; verify they are clean
