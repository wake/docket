# Tasks

Each group is one slice an implementer finishes test-first. Scenario names refer to `specs/<capability>/spec.md`. Code marked as ported is read with `git show v0.15.0:<path>`. Tests run with `go test ./...`; every git call in a test uses `testutil.Env`, and repositories come from `testutil.NewRepo`.

## Reuse

| Need | Existing (path:symbol) | Decision | Reason | Evidence |
|---|---|---|---|---|
| Trailers of a commit's history | `internal/gitx/log.go:ChangeTrailers` | extend | Take the revision as a parameter; `dkt sync` passes HEAD | `rg -n "ChangeTrailers" internal` |
| Run git, "no answer" on exit 1 | `internal/gitx/git.go:Git.output,outputIn,runOptional` | reuse | Isolated environment and error type every git call uses | `rg -n "func \(g Git\)" internal/gitx/git.go` |
| Folders in a commit's tree | `internal/gitx/refs.go:TreeDirs` | reuse | `internal/scan` lists change folders of a ref the same way | `rg -n "TreeDirs" internal` |
| Default branch, merge base, blobs at a commit, files a branch adds | none (`gitx` has `Head`, `Branch`, `Refs`, `UncommittedUnder`) | new | Port of `v0.15.0`'s `ResolveDefaultBranch` without the flag; the rest is new | `rg -n "merge-base\|cat-file\|origin/HEAD\|diff-filter" internal` (none), `git show v0.15.0:internal/scan/defaultbranch.go` |
| Schema, plan tasks, trailer values | `internal/change/metadata.go:SchemaOf,TaskRefs`, `internal/change/plan.go:ParsePlan` | reuse | Read the default branch's files as `dkt sync` reads the worktree's | `rg -n "^func (SchemaOf\|TaskRefs\|ParsePlan)" internal/change` |
| Main spec titles | `internal/lint/document.go:checkSpecTitles` | extend | Return the findings so `dkt inspect` can share it | `rg -n "checkSpecTitles" internal` |
| Lint input and findings | `internal/lint/lint.go:Input,Run` | extend | Add `Added` and the `routing` rule | `rg -n "type Input" internal/lint` |
| The two commands | `internal/cli/cmd_lint.go:runLint`, `internal/cli/cmd_inspect.go:inspectCommand` | extend | Add `default_branch` and the new checks | `rg -n "func runLint\|func inspectCommand" internal/cli` |

## 1. Trailers of any revision

**Review:** checks (restructuring)
**Files:** `internal/gitx/log.go`, `internal/gitx/log_test.go`, `internal/cli/cmd_sync.go`

- [ ] 1.1 Verify `go test ./internal/gitx/ ./internal/cli/` passes; give `ChangeTrailers` a `rev string` parameter, pass `"HEAD"` from `dkt sync` and the existing tests, and change nothing else; verify `go test ./...` passes and that mutating the revision in `dkt sync` to another value fails a test

## 2. Git calls

**Review:** judgment (environment-dependent)
**Files:** `internal/gitx/branch.go`, `internal/gitx/branch_test.go`, `internal/gitx/blobs.go`, `internal/gitx/blobs_test.go`

- [ ] 2.1 Write failing tests for every scenario of default-branch, plus: `origin/HEAD` pointing outside `refs/remotes/origin/` is ignored; a `main` that exists only as `refs/remotes/origin/main` is found; `MergeBase` of two branches, of unrelated histories (false), and with an unborn HEAD (false); `ReadBlobs` returns the files that exist at a commit, leaves out a missing path and a directory, and keeps a file's bytes exactly (a CRLF line, no trailing newline); `AddedSince` lists a file committed on the branch, a staged one and an untracked one under the given folders, not a file that exists at the base and was changed, not an ignored one, not a file outside the folders, not a committed file deleted from the worktree, each once, sorted, and a path with a space; verify they fail
- [ ] 2.2 Implement `DefaultBranch`, `MergeBase`, `ReadBlobs` and `AddedSince` (design.md, Git calls); verify every test passes

## 3. Routing and shared titles in dkt lint

**Review:** judgment (output format)
**Files:** `internal/lint/lint.go`, `internal/lint/document.go`, `internal/lint/routing.go`, `internal/lint/routing_test.go`, `internal/lint/document_test.go`, `internal/cli/cmd_lint.go`, `internal/cli/cmd_lint_test.go`

- [ ] 3.1 In its own commit, make `checkSpecTitles` into `SpecTitles(root fs.FS) ([]Finding, []Unreadable, int)` that `Run` calls (design.md, Main spec titles shared), changing no test; verify `go test ./...` passes
- [ ] 3.2 Write failing tests for every scenario of change-lint's Design and plan routing and its modified Lint result, plus: `Added` nil reports nothing and empty reports nothing; a routing finding has no change and no line; routing is reported with a change named and without; the text output's skip lines for no default branch and for no merge base; `default_branch` is the name when the rule ran; verify they fail
- [ ] 3.3 Implement the routing rule and `default_branch` (design.md, Routing); verify every test passes

## 4. Trailers and titles in dkt inspect

**Review:** judgment (reads git objects, output format)
**Files:** `internal/cli/cmd_inspect.go`, `internal/cli/cmd_inspect_test.go`

- [ ] 4.1 Write failing tests for every scenario of setup-inspection's Commit trailers on the default branch, Default branch in the inspect result and Main spec titles inspected, plus: a change of another schema on the default branch is not checked; a change whose plan.md is missing on the default branch is not checked; a trailer `Change: add-x/two` (malformed) still counts as naming `add-x`; `Change: add-x-2/1` does not name `add-x`; the trailer check reads the default branch's tip, not the worktree (a plan.md with tasks only in the worktree is not checked); the default branch taken from `origin` when it is ahead of the local branch; the existing Clean setup scenario still gives no finding; verify they fail
- [ ] 4.2 Implement them (design.md, Trailer check, Main spec titles shared); verify every test passes

## 5. Integration

**Review:** checks
**Files:** none beyond earlier groups

- [ ] 5.1 Build dkt into a temporary `GOBIN`; in a clone of a repository set up with `dkt init` (so `origin/HEAD` exists), open a change with a plan with one task, commit it with `Change: <slug>/1`, squash-merge it into `main` with `git merge --squash` and a commit message without the trailer, push, fetch, and check that `dkt inspect` reports `trailer-missing` for it; on a new branch write `docs/superpowers/plans/x.md` and check that `dkt lint` reports `routing` for it; run both in this repository and check that `dkt inspect` names `main` and reports nothing about trailers
- [ ] 5.2 Run `gofmt -l .`, `go vet ./...` and `go test ./...`; verify they are clean

## Workflow follow-up

- Archive the change with OpenSpec as the last commit of the pull request, after the reviews.
