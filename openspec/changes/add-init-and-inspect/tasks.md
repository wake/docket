# Tasks

Each group is one slice an implementer finishes test-first. Scenario names refer to `specs/<capability>/spec.md`. Code marked as ported is read with `git show v0.15.0:<path>` (design.md names each piece). Tests run with `go test ./...`; every git call in a test uses `testutil.Env`, and every OpenSpec call in a unit test uses the fake OpenSpec from group 1, never a real CLI.

## Reuse

| Need | Existing (path:symbol) | Decision | Reason | Evidence |
|---|---|---|---|---|
| Envelope, findings, exit codes | `internal/cli/envelope.go:Outcome,Finding,Unreadable,ExitCode,EnvError,UsageError` | reuse | Already the shared output of every command | `rg -n "^type \|^func ExitCode" internal/cli/envelope.go` |
| Repository root | `internal/gitx/git.go:Git.Toplevel` | reuse | Resolves the root and the not-a-repository case | `rg -n "func \(g Git\) Toplevel" internal/gitx` |
| Temporary repositories in tests | `internal/testutil/repo.go:Env,NewRepo` | reuse | Isolated git environment already used by every test | `rg -n "^func " internal/testutil/repo.go` |
| Locate and run OpenSpec | `v0.15.0:internal/openspec/runner.go:Runner,New,CheckVersion,Run` | extend | Port; minimum version instead of a range, telemetry off | `git show v0.15.0:internal/openspec/runner.go` |
| Fake OpenSpec | `v0.15.0:internal/testutil/openspec.go:FakeOpenSpec` | extend | Port; add the `init` behavior | `git show v0.15.0:internal/testutil/openspec.go` |
| Refuse symbolic links | `v0.15.0:internal/safepath/safepath.go:Check,SymlinkError` | reuse | Port as is | `git show v0.15.0:internal/safepath/safepath.go` |
| Ignored paths | `v0.15.0:internal/gitx/ignore.go:Git.Ignored,Ignore` | reuse | Port as is | `git show v0.15.0:internal/gitx/ignore.go` |
| The `schema` line | `v0.15.0:internal/setup/config.go:SchemaKey,NameDocket` | extend | Port; add the `operations` section | `git show v0.15.0:internal/setup/config.go` |
| Write without following links | `v0.15.0:internal/setup/setup.go:install,readNoFollow` | reuse | Port as is | `git show v0.15.0:internal/setup/setup.go` |
| Embed shipped files | `v0.15.0:schema.go` | extend | Same root-package embed, new `shipped/` layout | `git show v0.15.0:schema.go` |
| Decide each file's status | none | new | `v0.15.0`'s `setup.Init`/`setup.Inspect` carry per-machine folders, the `AGENTS.md` block and user skills, none of which remain | `git show v0.15.0:internal/setup/setup.go \| rg -n "^func (Init\|Inspect)"` |

## 1. OpenSpec runner

**Review:** judgment (environment-dependent behavior, error path)
**Files:** `internal/openspec/runner.go`, `internal/openspec/runner_test.go`, `internal/testutil/openspec.go`

- [ ] 1.1 Port `FakeOpenSpec` into `internal/testutil/openspec.go`, extended as design.md (Tests) says, and write failing tests for the scenarios Override with npx, Default from PATH, Every invocation, Too old, Newer version accepted and CLI missing, plus: `1.10.0` is accepted (numbers compare as numbers, not text); `abc` and a non-zero exit of `--version` are unsupported; a failed call returns an error carrying the command, the exit status and standard error; verify they fail
- [ ] 1.2 Port `internal/openspec/runner.go` with the minimum version and `OPENSPEC_TELEMETRY=0` (design.md, OpenSpec runner); verify every test from 1.1 passes

## 2. Command table, help and repository

**Review:** judgment (public interface)
**Files:** `internal/cli/run.go`, `internal/cli/usage.go`, `internal/cli/command.go`, `internal/cli/run_test.go`, `internal/cli/command_test.go`

- [ ] 2.1 Write failing tests for the scenarios Command help, Help ignores --json and Outside a repository (for `dkt init` and `dkt inspect`), plus: `dkt inspect --help` lists `--json`; `dkt init --nosuch` exits with 2 and prints `dkt init`'s usage to standard error; with `git` missing from `PATH`, `dkt inspect --json` reports `git-missing` and exits with 4; `dkt nosuch` still exits with 2; the existing tests in `run_test.go` still pass; verify the new ones fail
- [ ] 2.2 Add the command table with `init` and `inspect` (flags only; their run functions return an error until groups 7 and 8), per-command help, and the repository check (design.md, Commands and help); verify every test from 2.1 passes

## 3. Shipped files

**Review:** checks
**Files:** `shipped/**`, `shipped.go`, `shipped_test.go`, `AGENTS.md`

- [ ] 3.1 Write a failing test that the embedded files are exactly those in design.md's table of shipped files, that `shipped/schema/schema.yaml` names `templates` that all exist, and that `shipped/skill/SKILL.md` and every template start with the front matter or heading their target needs; verify it fails
- [ ] 3.2 Add `shipped/` with the content design.md names (the edited English docket skill and proposal and plan templates, the schema draft, placeholders for the rest) and `shipped.go`; add `shipped/` to the prompt files in `AGENTS.md`; verify the test from 3.1 passes

## 4. File statuses

**Review:** judgment (the rules both commands share)
**Files:** `internal/setup/setup.go`, `internal/setup/history.go`, `internal/setup/setup_test.go`, `internal/safepath/safepath.go`, `internal/safepath/safepath_test.go`

- [ ] 4.1 Port `internal/safepath` with its tests; verify they pass
- [ ] 4.2 Write failing tests, at the level of the decision table (design.md, One decision table), for: every target `missing` in an empty repository; every target `same` after writing the shipped files; a file equal to an injected earlier-release hash is `outdated` with action update; an edited file is `differs` with action keep, and overwrite with force; the two skill targets are decided separately; a symbolic link on `.claude/skills/docket` is a refusal naming it; an unreadable `SKILL.md` is a refusal listing it as unreadable; a file the user added under `.claude/skills/docket/` is not a target; verify they fail
- [ ] 4.3 Implement the decision table for the schema and skill files, with the earlier-release table and the ported `install` and `readNoFollow`; verify every test from 4.2 passes

## 5. config.yaml

**Review:** judgment (parser of user files)
**Files:** `internal/setup/config.go`, `internal/setup/config_test.go`, `internal/setup/setup.go`

- [ ] 5.1 Write failing tests for the `schema` line and the `operations` section (design.md, config.yaml), from the scenarios Default schema replaced, Other schema kept, Section appended, Edited guidance kept, Config file created and Unreadable schema key, plus: 1.7.0's exact `config.yaml` (design.md, Context) becomes `schema: docket`, the same comments, and the shipped section; `schema: "spec-driven"` and `schema: spec-driven # default` are replaced too; two `schema:` lines, `operations: {}` and an `operations:` section indented under another key are unreadable or not top-level as design.md says; a file without a final newline gets one before the section; `--force` changes neither another schema nor other guidance; verify they fail
- [ ] 5.2 Port `SchemaKey` and `NameDocket`, add the `operations` section, and give `openspec/config.yaml` its place in the decision table with the statuses and findings of the specs; verify every test from 5.1 passes

## 6. Ignored paths

**Review:** checks (ported with its tests)
**Files:** `internal/gitx/ignore.go`, `internal/gitx/ignore_test.go`

- [ ] 6.1 Port `gitx.Ignored` with its tests, and add tests that a tracked file is never reported and that a path that does not exist yet is still reported when a rule ignores it; verify they pass

## 7. dkt inspect

**Review:** judgment (output format)
**Files:** `internal/cli/cmd_inspect.go`, `internal/cli/cmd_inspect_test.go`

- [ ] 7.1 Write failing tests for the scenarios Nothing written, Clean setup, Statuses listed, Before init, Edited skill, File from an earlier release (setup-inspection), Before init config, Other schema reported, Linked skill folder inspected, Old OpenSpec, No OpenSpec, Ignored OpenSpec folder and Unreadable installed file, plus: the text output has the OpenSpec line and one line per file; verify they fail
- [ ] 7.2 Implement `dkt inspect` on the decision table, the runner and the ignore check (design.md, Output); verify every test from 7.1 passes

## 8. dkt init

**Review:** judgment (error paths, output format)
**Files:** `internal/cli/cmd_init.go`, `internal/cli/cmd_init_test.go`, `internal/setup/openspecinit.go`, `internal/setup/openspecinit_test.go`

- [ ] 8.1 Write failing tests for every scenario of project-setup, and for Too old, CLI missing and openspec init fails (openspec-runtime) run through `dkt init`, plus: `openspec/config.yaml` created by `openspec init` is listed once, with kind `config` and status `created`; an untracked file that existed before `openspec init` is not listed; a run that keeps a file and writes others still writes them; the text output lists the files and the next steps; verify they fail
- [ ] 8.2 Implement `dkt init`: checks before writing, `openspec init` with the before-and-after listing (design.md, Files openspec init created), writes, the ignore check, the result and the next steps; verify every test from 8.1 passes

## 9. Real OpenSpec

**Review:** checks (check settings: the review looks for a check that got weaker)
**Files:** `internal/cli/realopenspec_test.go`, `.github/workflows/ci.yml`

- [ ] 9.1 Write `TestRealOpenSpec` (design.md, Tests); verify it is skipped without `DOCKET_TEST_OPENSPEC=1`, and that it passes locally with `DOCKET_OPENSPEC="npx -y @fission-ai/openspec@1.7.0"` and with `@latest`
- [ ] 9.2 Add the `openspec` job to `.github/workflows/ci.yml`; verify it runs and passes for both versions on the pull request

## 10. Integration

**Review:** checks
**Files:** none beyond earlier groups

- [ ] 10.1 Build dkt with `go install ./cmd/dkt` into a temporary `GOBIN`, then in a new repository follow `skills/docket-init/SKILL.md` steps 3 and 4 by hand: `dkt init --json --tools claude`, commit exactly the files it lists, and `dkt inspect --json`; verify `dkt inspect` reports no finding and `git status` is clean after the commit
- [ ] 10.2 Run `gofmt -l .`, `go vet ./...` and `go test ./...`; verify they are clean

## Workflow follow-up

- Archive the change with OpenSpec as the last commit of the pull request.
- Replace each placeholder in `shipped/` once its prompt is edited; each replacement is a light review.
