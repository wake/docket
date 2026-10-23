# add-init-and-new Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** The `dkt` binary with its shared conventions (output envelope, exit codes, repository identity, OpenSpec runner, workspace scan) and the commands `dkt init` and `dkt new`.

**Architecture:** A Go module at the repository root. The root package only embeds the shipped schema and the AGENTS.md routing section; `cmd/dkt` is the entry point; everything else is under `internal/`. Git and OpenSpec are separate processes started with the environment the CLI was given, so tests control both completely.

**Tech Stack:** Go 1.24 (module floor), standard library, `go.yaml.in/yaml/v3` v3.0.5; `git` and the OpenSpec CLI 1.14.x as external programs; GitHub Actions.

**Spec:** `openspec/changes/add-init-and-new/proposal.md`, `design.md` and `specs/*/spec.md` in this change folder. Read all three; scenario IDs below refer to the delta specs.

## Global Constraints

- Module path `github.com/wake/docket`; `go 1.24` in go.mod; the only third-party dependency is `go.yaml.in/yaml/v3`.
- Public repository: code, comments, messages and docs in English; no hostnames, customer names, personal paths or secrets.
- Docket never fetches, pushes or contacts a network service, keeps no state, writes nothing under `.git/`, and never creates or edits AGENTS.md, CLAUDE.md, `.claude/` or `openspec/config.yaml` in the repository it runs on.
- Every command accepts `--json` and `--default-branch <name>`, before or after positional arguments.
- Exit codes: 0 clean; 1 problems found or request refused; 2 usage error; 3 a needed source could not be read (wins over 1); 4 unusable environment (no git repository, `git` missing, no usable OpenSpec CLI).
- With `--json`, standard output is exactly one JSON object with `tool` ("dkt"), `version`, `command`, `repo`, `ok`, `findings`, `unreadable`, `result`; `findings` and `unreadable` are arrays, never null; `repo` is null when the repository could not be identified. Diagnostics go to standard error.
- OpenSpec is found through `DOCKET_OPENSPEC` (split on whitespace) or `openspec` on the `PATH` of the given environment; only 1.14.x is accepted.
- Child processes (`git`, OpenSpec) receive exactly `Options.Env`; nothing reads `os.Getenv` or `os.Environ` below `cmd/dkt`.
- macOS and Linux only.
- `go test ./...` passes without network access and without an OpenSpec install (a fake stands in).
- Every commit carries `--trailer "Change: add-init-and-new" --trailer "Task: <N>"` and `--trailer "Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"` (use the model that actually wrote the commit).

## Review Focus

1. A repository with no commits yet (`git init` only): `dkt init` must work and report `repo.head` as null; `dkt new` must refuse with exit 1 and say to commit first, before it looks for a default branch, which such a repository does not have yet. Tests in Tasks 3, 6 and 7.
2. Paths containing spaces, for the repository root and for worktrees. Tests in Tasks 5 and 7.
3. `--source` text with YAML-significant or non-ASCII characters (`issue #12: "x"`, `需求 R1#3`) must round-trip exactly through `.openspec.yaml`. Test in Task 7.
4. OpenSpec writing `.openspec.yaml` without a trailing newline: the `x-` keys must still start on their own lines. Test in Task 7.
5. `refs/remotes/origin/HEAD` is a symbolic ref and must not show up as a branch source. Test in Task 5.

---

## Revision 1 (2026-10-24): content precedence across sources

Implementing Task 5 showed that "worktree first" lets the main checkout, which sits on the default branch, shadow an unmerged branch's edits. The `workspace-scan` delta now ranks sources: a worktree whose HEAD the default branch tip does not contain, a local branch, a remote-tracking branch, a worktree whose HEAD the default branch tip contains, the default branch tip. Task 5 gains the scenario `workspace-scan#worktree-with-unmerged-work-comes-first`; no other task changes.

## Reuse

| Need | Existing (path:symbol) | Decision | Reason | Evidence |
|---|---|---|---|---|
| The schema files Docket ships | `openspec/schemas/docket/schema.yaml` | reuse | This repository's schema is the shipped one; `go:embed` reads it in place | design.md "The shipped schema is this repository's own schema" |
| The AGENTS.md routing section `dkt init` prints | `AGENTS.md`: "Change workflow (superpowers × OpenSpec)" | reuse | One source for the text; embedded and cut out by heading | `rg -n "^## Change workflow" AGENTS.md` |
| Creating a change folder and its metadata | external: OpenSpec CLI `openspec new change --schema` | reuse | OpenSpec writes `.openspec.yaml` and checks the schema name | `openspec new change --help` (1.14.3) lists `--schema` |
| Creating `openspec/` | external: OpenSpec CLI `openspec init --tools none` | reuse | Same files as any OpenSpec project | sandbox run 2026-10-24: creates `config.yaml`, `specs/`, `changes/archive/` only |
| Worktrees, refs, trees, merge bases | external: git CLI | reuse | Behaves exactly as the user's git | design.md "Git through the git binary" |
| YAML scalars in `.openspec.yaml` | external: `go.yaml.in/yaml/v3` | reuse | Maintained successor of `gopkg.in/yaml.v3` | `go list -m -versions go.yaml.in/yaml/v3` → v3.0.5 |
| Flag parsing, embedding, JSON | external: Go `flag`, `embed`, `encoding/json` | reuse | Standard library; no CLI framework | design.md "Go, standard library first" |
| Failing CI on new duplicated code only | external: jscpd 5.4.2 | reuse | Compares against a git ref without a committed baseline file | `npx -y jscpd@5.4.2 --help` lists `--baseline-from-ref <REF>` and `--fail-on-new-clones [<N>]` (checked 2026-10-24) |
| Output envelope, exit codes, dispatch | none | new | The repository has no Go code | `git ls-files '*.go'` → empty |
| Workspace scan | none | new | No code yet; no library discovers OpenSpec changes across git worktrees and refs | `git ls-files '*.go'` → empty |
| OpenSpec runner with version gate | none | new | No code yet | `git ls-files '*.go'` → empty |

Rows marked `external:` name a dependency outside the repository; there is no path to check.

---

### Task 1: Module and embedded assets

**Scenarios:** none

**Files:**
- Create: `go.mod`, `schema.go`
- Test: `schema_test.go`

**Interfaces:**
- Produces (package `docket`, import `github.com/wake/docket`):
  - `func Schema() fs.FS` — the files of `openspec/schemas/docket/`, with paths relative to it (`schema.yaml`, `templates/proposal.md`, …).
  - `func AgentsSection() string` — the AGENTS.md section from the line `## Change workflow (superpowers × OpenSpec)` up to, not including, the next line starting with `## `, with trailing whitespace trimmed.

- [ ] **Step 1: Write the failing tests** in `schema_test.go`:
  - `TestSchemaHasSchemaFile`: `fs.ReadFile(Schema(), "schema.yaml")` succeeds and contains `name: docket`.
  - `TestSchemaHasTemplates`: `templates/proposal.md`, `templates/spec.md`, `templates/design.md`, `templates/plan.md` all readable; no `templates/tasks.md`.
  - `TestAgentsSection`: starts with `## Change workflow (superpowers × OpenSpec)`, contains `dkt new`, does not contain `## Development`.
- [ ] **Step 2: Run** `go test ./...` — expected: build failure, `Schema` undefined.
- [ ] **Step 3: Implement** `go.mod` (`module github.com/wake/docket`, `go 1.24`) and `schema.go` with `//go:embed openspec/schemas/docket` and `//go:embed AGENTS.md`; `Schema` uses `fs.Sub`. `AgentsSection` panics if the heading is missing (a build-time invariant, guarded by the test).
- [ ] **Step 4: Run** `go test ./...` — expected: PASS.
- [ ] **Step 5: Commit** `feat: embed the docket schema and the routing section`.

### Task 2: Git access and test repositories

**Scenarios:** none

**Files:**
- Create: `internal/gitx/git.go`, `internal/gitx/worktree.go`, `internal/testutil/repo.go`
- Test: `internal/gitx/git_test.go`

**Interfaces:**
- Produces (package `gitx`):
  - `type Git struct { Dir string; Env []string }`; every method runs `git` in `Dir` with exactly `Env`.
  - `var ErrGitMissing, ErrNotRepo error`
  - `type ExitError struct { Args []string; Code int; Stderr string }` (implements `error`).
  - `func (g Git) Run(args ...string) (string, error)` — stdout with the trailing newline trimmed.
  - `func (g Git) Toplevel() (string, error)` — `ErrNotRepo` outside a work tree, `ErrGitMissing` when no `git` on the `PATH` in `Env`.
  - `func (g Git) Head() (sha string, ok bool, err error)` — `ok` false while HEAD is unborn.
  - `func (g Git) Branch() (name string, ok bool, err error)` — `ok` false when detached.
  - `func (g Git) RemoteURL(name string) (string, bool, error)`
  - `func (g Git) RefExists(ref string) (bool, error)`
  - `func (g Git) SymbolicRef(ref string) (string, bool, error)`
  - `func (g Git) MergeBase(a, b string) (string, bool, error)` — `ok` false when there is no common ancestor.
  - `func (g Git) IsAncestor(a, b string) (bool, error)`
  - `type Ref struct { Name, SHA string }`; `func (g Git) Refs(patterns ...string) ([]Ref, error)` — full refnames; symbolic refs are left out.
  - `type Worktree struct { Path, Head, Branch string; Detached, Bare, Prunable bool }`; `func (g Git) Worktrees() ([]Worktree, error)` — parsed from `git worktree list --porcelain`.
  - `func (g Git) ListDirs(rev, dir string) ([]string, error)` — names of the directories directly under `dir` in `rev`'s tree; a missing `dir` gives an empty list and no error.
  - `func (g Git) Show(rev, path string) ([]byte, error)` — the blob at `rev:path`, via `git show`.
- Produces (package `testutil`, test helpers only):
  - `func Env(t testing.TB) []string` — `os.Environ()` plus a temporary `HOME`, `GIT_CONFIG_NOSYSTEM=1`, and `GIT_CONFIG_GLOBAL` pointing to a temporary file with `user.name`, `user.email`, `init.defaultBranch=main`, `commit.gpgsign=false`.
  - `type Repo struct { T testing.TB; Dir string; Env []string }`
  - `func NewRepo(t testing.TB) *Repo` (in a temporary directory), `func NewRepoAt(t testing.TB, dir string) *Repo`
  - `func (r *Repo) Git(args ...string) string` — fails the test on error.
  - `func (r *Repo) Write(rel, content string)`, `func (r *Repo) Commit(msg string, files map[string]string) string` (returns the SHA)
  - `func (r *Repo) AddWorktree(path, newBranch string) *Repo`
  - `func NewBare(t testing.TB) string` and `func (r *Repo) CloneTo(t testing.TB, dir string) *Repo` for remote scenarios.

- [ ] **Step 1: Write the failing tests** in `internal/gitx/git_test.go`, each on a `testutil` repository: `TestToplevelOutsideRepo` (`ErrNotRepo`), `TestGitMissing` (`Env` with an empty `PATH` → `ErrGitMissing`), `TestHeadUnborn` (`ok` false), `TestBranchDetached`, `TestRemoteURLMissing`, `TestRefsSkipsSymbolic` (a clone's `refs/remotes/origin/HEAD` is absent from `Refs("refs/remotes")`), `TestWorktreesPrunable` (add a worktree, delete its directory, entry has `Prunable` true), `TestListDirsMissing` (empty, nil), `TestShow` (returns the committed bytes of a file at a given commit), `TestMergeBaseUnrelated` (`ok` false for two unrelated roots).
- [ ] **Step 2: Run** `go test ./internal/gitx/` — expected: build failure.
- [ ] **Step 3: Implement** with `os/exec`; look `git` up in the `PATH` from `Env`, not the process's. `ListDirs` uses `git ls-tree -d --name-only <rev> -- <dir>/` and returns base names.
- [ ] **Step 4: Run** `go test ./internal/gitx/` — expected: PASS.
- [ ] **Step 5: Commit** `feat(gitx): run git with a given environment`.

### Task 3: CLI core: envelope, exit codes, dispatch, repository identity

**Scenarios:** `cli-output#envelope-fields-present`, `cli-output#detached-head`, `cli-output#no-origin-remote`, `cli-output#not-a-git-repository`

**Files:**
- Create: `internal/cli/envelope.go`, `internal/cli/run.go`, `internal/cli/output.go`, `cmd/dkt/main.go`
- Test: `internal/cli/run_test.go`

**Interfaces:**
- Consumes: `gitx.Git` (Task 2).
- Produces (package `cli`):
  - `var Version = "dev"` (set with `-ldflags "-X github.com/wake/docket/internal/cli.Version=…"`).
  - `type Finding struct { Rule, Message, Change, File string; Line int }` — JSON `rule`, `message`, `change,omitempty`, `file,omitempty`, `line,omitempty`.
  - `type Unreadable struct { Source, Reason string }` — JSON `source`, `reason`.
  - `type Repo struct { Root string; Head, Branch, Origin *string }` — JSON `root`, `head`, `branch`, `origin`.
  - `type Outcome struct { Findings []Finding; Unreadable []Unreadable; Result any; Human []string }` — `Human` lines are printed in text mode only.
  - `type UsageError struct{ Msg string }`, `type EnvError struct{ Msg string }` (both `error`).
  - `type Context struct { Dir string; Env []string; JSON bool; DefaultBranch string; Git gitx.Git; Repo *Repo }`
  - `type Command struct { Name, Usage string; Define func(fs *flag.FlagSet); Run func(c *Context, args []string) (Outcome, error) }` — `Define` binds the command's own flags; the dispatcher adds `--json` and `--default-branch` to the same set.
  - `type Options struct { Args []string; Dir string; Env []string; Stdout, Stderr io.Writer }`
  - `func Run(o Options) int` — the real command set; `func RunWith(o Options, cmds []func() Command) int` — the same with a given set, for tests.
  - `func ExitCode(o Outcome, err error) int` — `UsageError` 2, `EnvError` 4, any other error 1 (reported as rule `internal`), unreadable 3, findings 1, else 0.
- Produces (`cmd/dkt`): `main` calls `cli.Run` with `os.Args[1:]`, the working directory, `os.Environ()`, stdout and stderr, and exits with its result.

- [ ] **Step 1: Write the failing tests** in `internal/cli/run_test.go`, using `RunWith` and a test command `probe` whose `Run` returns an `Outcome` chosen by each test:
  - `TestEnvelopeFieldsPresent`: `probe --json` with an empty outcome → output parses as one object with exactly the eight keys; `findings` and `unreadable` are `[]`, not `null`; `ok` true; exit 0.
  - `TestDetachedHead`: repository checked out at a commit SHA → `repo.branch` null, `repo.head` equals the 40-hex SHA.
  - `TestNoOriginRemote`: `repo.origin` null, exit 0.
  - `TestNotAGitRepository`: `probe --json` in an empty temporary directory → exit 4, `repo` null, one finding with rule `environment`.
  - `TestUnbornHeadIdentity`: freshly initialized repository → `repo.head` null, `repo.branch` "main".
  - `TestFlagsAfterPositionals`: `probe one --json two` gives the probe the positionals `one two` and JSON output.
  - `TestUnknownCommand`: exit 2, usage on stderr.
  - `TestExitCodePrecedence`: table over `ExitCode` — unreadable plus findings → 3; findings only → 1; `UsageError` → 2; `EnvError` → 4.
- [ ] **Step 2: Run** `go test ./internal/cli/` — expected: build failure.
- [ ] **Step 3: Implement.** Flags may follow positionals: parse in a loop, taking one positional each time `flag.Parse` stops. Flag errors become `UsageError`. Identify the repository before running a command (`EnvError` when that fails). Text output: `Human` lines, then one line per finding (`<rule>: <message>`, with `file:line` and `change` when present), then one line per unreadable source, then `dkt <command>: ok` when clean.
- [ ] **Step 4: Run** `go test ./internal/cli/` — expected: PASS.
- [ ] **Step 5: Commit** `feat(cli): result envelope, exit codes and dispatch`.

### Task 4: OpenSpec runner

**Scenarios:** `openspec-runtime#override-with-npx`, `openspec-runtime#default-from-path`

**Files:**
- Create: `internal/openspec/runner.go`, `internal/testutil/openspec.go`
- Test: `internal/openspec/runner_test.go`

**Interfaces:**
- Produces (package `openspec`):
  - `const SupportedRange = "1.14.x"`
  - `type Runner struct { Cmd []string; Dir string; Env []string }`
  - `func New(dir string, env []string) (*Runner, error)` — `Cmd` from `DOCKET_OPENSPEC` split on whitespace when set and non-empty, else `["openspec"]`; a bare program name is looked up on the `PATH` in `env`. Returns `*UnavailableError` when the program cannot be found; its message names `DOCKET_OPENSPEC`.
  - `func (r *Runner) CheckVersion() (string, error)` — runs `<Cmd> --version` once per `Runner`; the trimmed output must match `^1\.14\.(0|[1-9][0-9]*)$`, else `*UnsupportedError{Found, Range}`.
  - `func (r *Runner) Run(args ...string) (string, error)` — calls `CheckVersion` first; a non-zero exit gives `*FailedError{Command []string; Code int; Stderr string}`.
- Produces (package `testutil`):
  - `func FakeOpenSpec(t testing.TB, version string) (cmd, log string)` — writes a POSIX `sh` script and returns its path and the path of a log file that gets one line per invocation (arguments joined by a tab). Behavior: `--version` prints `version`; `init --tools none` creates `openspec/config.yaml` (`schema: spec-driven`), `openspec/specs/.gitkeep`, `openspec/changes/archive/.gitkeep`; `new change <slug> --schema <s>` creates `openspec/changes/<slug>/.openspec.yaml` with `schema: <s>` and `created: 2026-01-02`, without a trailing newline when `FAKE_OPENSPEC_NO_EOL=1`, and as invalid YAML (`schema: [`) when `FAKE_OPENSPEC_BAD_YAML=1`; when `FAKE_OPENSPEC_FAIL=<subcommand>`, that subcommand first does its usual file work, then prints `fake failure` to stderr and exits 3.
  - `func OpenSpecCmd(t testing.TB) string` — the value of `DOCKET_TEST_OPENSPEC` when set (the real CLI, used by CI), else a fake reporting `1.14.3`.

- [ ] **Step 1: Write the failing tests:**
  - `TestOverrideSplitsWords`: `DOCKET_OPENSPEC="<fake> -y @fission-ai/openspec@1.14.3"`; `Run("new","change","x","--schema","docket")` → the log's line is `-y`, `@fission-ai/openspec@1.14.3`, `new`, `change`, `x`, `--schema`, `docket` (after the `--version` line).
  - `TestDefaultFromPath`: `DOCKET_OPENSPEC` unset, `PATH` holds a directory with the fake copied to `openspec` → `Run("--help")` reaches it.
  - `TestUnsupportedVersion`: fake reports `1.3.0` → `*UnsupportedError{Found:"1.3.0", Range:"1.14.x"}` and the log shows only `--version`.
  - `TestVersionMatching`: table — `1.14.0`, `1.14.3`, `1.14.12` accepted; `1.15.0`, `1.14`, `1.14.bad`, `v1.14.3`, `1.14.3-beta`, empty output rejected.
  - `TestMissingProgram`: `DOCKET_OPENSPEC=/nonexistent/openspec` → `*UnavailableError` whose message contains `DOCKET_OPENSPEC`.
  - `TestFailureCarriesStderr`: `FAKE_OPENSPEC_FAIL=new` → `*FailedError` with `Code` 3 and `Stderr` containing `fake failure`.
- [ ] **Step 2: Run** `go test ./internal/openspec/` — expected: build failure.
- [ ] **Step 3: Implement** with `os/exec`, working directory `Dir`, environment exactly `Env`.
- [ ] **Step 4: Run** `go test ./internal/openspec/` — expected: PASS.
- [ ] **Step 5: Commit** `feat(openspec): runner with a version gate`.

If the diff of Tasks 1–4 exceeds 800 lines or 20 files, open the first pull request here and continue Tasks 5–8 in a second one.

### Task 5: Default branch and workspace scan

**Scenarios:** `workspace-scan#change-only-in-another-worktree`, `workspace-scan#change-only-on-a-remote-tracking-branch`, `workspace-scan#merged-branch-skipped`, `workspace-scan#origin-head-set`, `workspace-scan#no-default-branch`, `workspace-scan#merged-change-also-on-a-branch`, `workspace-scan#worktree-with-unmerged-work-comes-first`, `workspace-scan#archived-slug-found`, `workspace-scan#worktree-on-an-unmounted-volume`, `workspace-scan#scan-offline`

**Files:**
- Create: `internal/scan/defaultbranch.go`, `internal/scan/scan.go`
- Test: `internal/scan/scan_test.go`

**Interfaces:**
- Consumes: `gitx.Git`, `gitx.Ref`, `gitx.Worktree`, `Git.ListDirs`, `Git.Show`, `Git.IsAncestor` (Task 2).
- Produces (package `scan`):
  - `type DefaultBranch struct { Name, Ref, Commit string }` — `Ref` is the full refname of the tip.
  - `var ErrNoDefaultBranch error`
  - `func ResolveDefaultBranch(g gitx.Git, flag string) (DefaultBranch, error)` — the name from `flag`, else the target of `refs/remotes/origin/HEAD`, else `main`, else `master` (a name exists if `refs/remotes/origin/<n>` or `refs/heads/<n>` does); the tip is `refs/remotes/origin/<name>` when it exists, else `refs/heads/<name>`.
  - `type Kind string` with `KindWorktree = "worktree"`, `KindLocalBranch = "local-branch"`, `KindRemoteBranch = "remote-branch"`, `KindDefaultBranch = "default-branch"`.
  - `type Source struct { Kind Kind; Name, Commit string }` — `Name` is the worktree path or the full refname; JSON `kind`, `name`, `commit`.
  - `type Change struct { Slug string; Sources []Source }` — sources ordered: worktree whose HEAD the default tip does not contain, local branch, remote-tracking branch, worktree whose HEAD the default tip contains, default branch tip; then by `Name`. `Sources[0]` is where content is read from.
  - `type Archived struct { Slug, Folder string; Sources []Source }` — `Folder` is the archive directory name, e.g. `2026-10-24-add-x`.
  - `type Unreadable struct { Source Source; Reason string }`
  - `type Inventory struct { DefaultBranch DefaultBranch; Changes []Change; Archived []Archived; Unreadable []Unreadable }` — `Changes` and `Archived` sorted by slug.
  - `func Scan(g gitx.Git, db DefaultBranch) (Inventory, error)` — returns an error only for failures that are not tied to one source (`Worktrees`, `Refs`); a source that cannot be read goes into `Unreadable` and the scan continues.
  - `func (inv Inventory) Uses(slug string) (active *Change, archived []Archived)`
  - `func ReadFile(g gitx.Git, src Source, rel string) ([]byte, error)` — `rel` is relative to the repository root (e.g. `openspec/changes/add-x/.openspec.yaml`); a worktree source reads the file on disk, a branch source reads `src.Commit:rel` with `Git.Show`. This is how later changes read a change's content from `Sources[0]`.

- [ ] **Step 1: Write the failing tests** in `internal/scan/scan_test.go`:
  - `TestChangeOnlyInAnotherWorktree`: worktree B has an uncommitted `openspec/changes/add-x/.openspec.yaml`; `Scan` from A lists `add-x` with `Sources[0] = {worktree, <B path>}`.
  - `TestChangeOnlyOnRemoteTrackingBranch`: a second clone pushes `feature-y` with `add-y`; the first fetches; `add-y` has source `{remote-branch, refs/remotes/origin/feature-y}`.
  - `TestMergedBranchSkipped`: a local branch merged into `main` appears in no source.
  - `TestOriginHeadSet`: `git symbolic-ref refs/remotes/origin/HEAD refs/remotes/origin/trunk` → `DefaultBranch{Name:"trunk", Ref:"refs/remotes/origin/trunk"}`.
  - `TestNoDefaultBranch`: only a branch `dev`, no remote → `ErrNoDefaultBranch`.
  - `TestMergedChangeAlsoOnBranch`: `add-x` on `main` with `x-kind: feature` (so the main worktree, checked out on `main`, also holds it), edited on unmerged `feature` to `x-kind: fix` → one `Change`, three sources, `Sources[0]` the local branch `feature`, and `ReadFile(g, Sources[0], "openspec/changes/add-x/.openspec.yaml")` contains `x-kind: fix`.
  - `TestWorktreeWithUnmergedWorkComesFirst`: worktree B on branch `b` with one commit `main` lacks, and an uncommitted edit to `add-x` (`x-kind: change-request`) → `Sources[0]` is worktree B and `ReadFile` on it contains `x-kind: change-request`.
  - `TestUnreadableBranchTree`: branch `broken` adds `add-z`; delete the loose object of `git rev-parse broken:openspec/changes` from `.git/objects` → `Scan` returns no error, `refs/heads/broken` is in `Unreadable`, and changes from other sources are still listed.
  - `TestArchivedSlugFound`: `openspec/changes/archive/2026-10-24-add-x/` on `main` → `Archived{Slug:"add-x", Folder:"2026-10-24-add-x"}`; a folder without a date prefix is not counted.
  - `TestWorktreeOnUnmountedVolume`: a worktree whose directory was removed → listed in `Unreadable` with its path; other sources still scanned.
  - `TestScanOffline`: `origin` set to `ssh://invalid.invalid/x.git` and existing remote-tracking refs → `Scan` succeeds.
  - `TestOriginHEADNotASource` (Review Focus 5) and `TestPathsWithSpaces` (Review Focus 2): a repository under a directory named `with space` and a worktree under `another dir` both scan.
- [ ] **Step 2: Run** `go test ./internal/scan/` — expected: build failure.
- [ ] **Step 3: Implement.** Worktrees are read from the filesystem (`os.ReadDir` of `openspec/changes` and `openspec/changes/archive`; a missing `openspec/` means no changes, an unreadable or missing worktree directory or a `Prunable` entry means unreadable); bare entries are skipped. Branch sources are read with `ListDirs`; a `ListDirs` error for one branch makes that branch unreadable. A branch is skipped when `IsAncestor(branchSHA, db.Commit)`. Archive folders match `^\d{4}-\d{2}-\d{2}-(.+)$`; the directory `archive` is never an active change.
- [ ] **Step 4: Run** `go test ./internal/scan/` — expected: PASS.
- [ ] **Step 5: Commit** `feat(scan): find changes in worktrees and branches`.

### Task 6: `dkt init`

**Scenarios:** `project-setup#fresh-repository`, `project-setup#existing-openspec-project`, `project-setup#locally-edited-template`, `project-setup#re-run-with-no-differences`, `project-setup#routing-section-printed`, `cli-output#json-on-success`, `openspec-runtime#cli-missing`

**Files:**
- Create: `internal/setup/setup.go`, `internal/cli/cmd_init.go`
- Modify: `internal/cli/run.go` (add `init` to the command set)
- Test: `internal/setup/setup_test.go`, `internal/cli/cmd_init_test.go`

**Interfaces:**
- Consumes: `docket.Schema`, `docket.AgentsSection` (Task 1); `openspec.New`, `Runner.Run` (Task 4); `cli.Command`, `cli.Outcome`, `cli.EnvError` (Task 3).
- Produces (package `setup`):
  - `type FileStatus string` with `Written = "written"`, `Unchanged = "unchanged"`, `Differs = "differs"`, `Overwritten = "overwritten"`.
  - `type File struct { Path string; Status FileStatus }` — `Path` relative to the repository root; JSON `path`, `status`.
  - `type Result struct { OpenSpecInitialized bool; Files []File; AgentsSection string }` — JSON `openspec_initialized`, `files`, `agents_section`.
  - `func Init(root string, schema fs.FS, force bool, initOpenSpec func() error) (Result, error)` — calls `initOpenSpec` only when `<root>/openspec` does not exist.
- The `init` command: flag `--force`; the OpenSpec `Runner` is created inside the `initOpenSpec` callback, never before, so a repository that already has `openspec/` needs no OpenSpec CLI; finding rule `schema-differs` for each `Differs` file; an `*openspec.UnavailableError` or `*UnsupportedError` becomes `EnvError`; text mode prints each file with its status, then `Add this section to AGENTS.md:` and the section.

- [ ] **Step 1: Write the failing tests:**
  - `TestFreshRepository` (cli level, `testutil.OpenSpecCmd`): `dkt init` in a committed repository without `openspec/` → `openspec/config.yaml` exists, every file of `docket.Schema()` is byte-identical under `openspec/schemas/docket/`, exit 0.
  - `TestJSONOnSuccess`: same with `--json` → one object, `ok` true, `result.openspec_initialized` true.
  - `TestExistingOpenSpecProject`: `openspec/` present → the fake's log has no `init` line; schema written.
  - `TestLocallyEditedTemplate`: `templates/proposal.md` edited, no `--force` → file unchanged, finding `schema-differs` naming `openspec/schemas/docket/templates/proposal.md`, exit 1.
  - `TestForceOverwrites`: same with `--force` → file equals the shipped one, status `overwritten`, exit 0.
  - `TestRerunNoDifferences`: second run → every status `unchanged`, no file's modification time changed, exit 0.
  - `TestRoutingSectionPrinted`: existing AGENTS.md → output contains `docket.AgentsSection()`, AGENTS.md byte-for-byte unchanged, CLAUDE.md and `openspec/config.yaml` untouched.
  - `TestCLIMissing`: `DOCKET_OPENSPEC=/nonexistent/openspec`, fresh repository → exit 4, message contains `DOCKET_OPENSPEC`.
  - `TestExistingOpenSpecWithoutCLI`: `openspec/` present and `DOCKET_OPENSPEC=/nonexistent/openspec` → exit 0, schema written.
  - `TestInitUnbornRepository` (Review Focus 1): `git init` only → exit 0, `repo.head` null.
- [ ] **Step 2: Run** `go test ./internal/setup/ ./internal/cli/` — expected: FAIL.
- [ ] **Step 3: Implement** `setup.Init` (compare bytes before writing; directories 0755, files 0644) and the command.
- [ ] **Step 4: Run** `go test ./...` — expected: PASS.
- [ ] **Step 5: Commit** `feat: dkt init installs the docket schema`.

### Task 7: `dkt new`

**Scenarios:** `change-creation#date-prefix-rejected`, `change-creation#uppercase-rejected`, `change-creation#slug-active-in-another-worktree`, `change-creation#slug-archived`, `change-creation#unreadable-worktree-blocks-creation`, `change-creation#missing-source`, `change-creation#unknown-kind`, `change-creation#change-created`, `change-creation#schema-not-installed`, `cli-output#json-on-failure`, `cli-output#usage-error`, `cli-output#unreadable-source-with-other-findings`, `openspec-runtime#unsupported-version`, `openspec-runtime#openspec-command-fails`

**Files:**
- Create: `internal/change/new.go`, `internal/cli/cmd_new.go`
- Modify: `internal/cli/run.go` (add `new`)
- Test: `internal/change/new_test.go`, `internal/cli/cmd_new_test.go`

**Interfaces:**
- Consumes: `scan.ResolveDefaultBranch`, `scan.Scan`, `Inventory.Uses` (Task 5); `openspec.New`, `Runner.Run` (Task 4); `Git.MergeBase`, `Git.Head` (Task 2); `cli` types (Task 3).
- Produces (package `change`):
  - `var Kinds = []string{"feature", "change-request", "fix", "baseline"}`
  - `func ValidateSlug(slug string) error` — accepts `^[a-z0-9]+(-[a-z0-9]+)*$`, at most 200 characters, not starting with `\d{4}-\d{2}-\d{2}`; the date-prefix error says archiving adds the date.
  - `func AppendMetadata(path, kind, source, base string) error` — appends `x-kind`, `x-source`, `x-base` (each value YAML-encoded with `go.yaml.in/yaml/v3`), first adding a newline if the file does not end with one, then parses the whole file and checks the three values read back exactly.
  - `type NewResult struct { Slug, Path, Kind, Source, Base string }` — JSON `slug`, `path`, `kind`, `source`, `base`.
- The `new` command: one positional slug, flags `--kind` and `--source`. Checks, in this order, each stopping the command: argument and flag errors (usage, 2; an unknown kind lists the four kinds) → slug rules (finding `slug-invalid`, 1) → HEAD has a commit (finding `no-commits`, "commit first", 1; checked before the default branch, which an empty repository does not have yet) → `openspec/schemas/docket/schema.yaml` exists (finding `schema-missing` pointing to `dkt init`, 1) → default branch (4) → scan: any unreadable source stops with 3 and creates nothing, with `slug-used` findings for any uses found; a use without unreadable sources gives `slug-used` naming every source and archive folder, 1 → merge base of HEAD and the default tip (finding `no-merge-base`, 1) → `openspec new change <slug> --schema docket` (OpenSpec version errors → 4; `*FailedError` → finding `openspec-failed` with its exit code and stderr, 1) → `AppendMetadata` (error → finding `metadata-failed`, 1).
- Creates nothing on failure: `openspec/changes/<slug>/` does not exist before `openspec new change` runs (the scan has just shown the slug unused), so when that command or `AppendMetadata` fails, the command removes the directory before returning.

Tests that only need a successful OpenSpec (`TestChangeCreated`, the slug and scan tests) use `testutil.OpenSpecCmd`, so CI also runs them against the real CLI; tests that inject a version or a failure use `testutil.FakeOpenSpec`.

- [ ] **Step 1: Write the failing tests:**
  - `change` package: `TestValidateSlug` (table: `add-x` ok; `Add-X`, `add_x`, `-x`, `x-`, `a--b`, 201 characters, `2026-10-24-add-x` rejected, the last with a message containing `archiv`); `TestAppendMetadataRoundTrip` (sources `issue 12`, `issue #12: "x"`, `需求 R1#3` read back exactly; `issue 12` is written plain, as `x-source: issue 12`); `TestAppendMetadataNoTrailingNewline` (Review Focus 4).
  - cli level: `TestDatePrefixRejected` (exit 1); `TestUppercaseRejected` (exit 1); `TestJSONOnFailure` (`new Bad_Slug --kind feature --source x --json` → `ok` false, finding `slug-invalid` mentioning `Bad_Slug`, exit 1); `TestUsageError` (`new` with no slug → exit 2, usage on stderr; with `--json`, finding rule `usage`); `TestMissingSource` (2); `TestUnknownKind` (2, stderr lists all four kinds); `TestSlugActiveInAnotherWorktree` (exit 1, message names worktree B's path); `TestSlugArchived` (exit 1, names `2025-01-02-add-x`); `TestUnreadableWorktreeBlocksCreation` (exit 3, no `openspec/changes/add-x/`); `TestUnreadableSourceWithOtherFindings` (worktree B uses the slug and worktree C is gone → a `slug-used` finding and an unreadable entry, exit 3); `TestChangeCreated` (`--source "issue 12"` on a branch made from `main` → `.openspec.yaml` has `schema: docket`, the `created` line, `x-kind: feature`, `x-source: issue 12`, `x-base` equal to `git merge-base HEAD main`); `TestSchemaNotInstalled` (exit 1, message contains `dkt init`); `TestUnsupportedVersion` (fake `1.3.0` → exit 4, no change folder, message names `1.3.0` and `1.14.x`); `TestOpenSpecCommandFails` (`FAKE_OPENSPEC_FAIL=new` → exit 1, finding `openspec-failed` with `3` and `fake failure`, and `openspec/changes/add-x/` does not exist afterwards); `TestMetadataFailureRollsBack` (`FAKE_OPENSPEC_BAD_YAML=1` → exit 1, finding `metadata-failed`, no `openspec/changes/add-x/`); `TestNewUnbornRepository` (Review Focus 1: `git init` only → exit 1, finding `no-commits`); `TestNoMergeBase` (HEAD on an orphan branch with one commit, `main` elsewhere → exit 1, finding `no-merge-base`, nothing created); `TestNewInPathWithSpaces` (Review Focus 2).
- [ ] **Step 2: Run** `go test ./internal/change/ ./internal/cli/` — expected: FAIL.
- [ ] **Step 3: Implement** `change` and the command.
- [ ] **Step 4: Run** `go test ./...` — expected: PASS.
- [ ] **Step 5: Commit** `feat: dkt new opens a change with its kind, source and base`.

### Task 8: Architecture map, CI, install note

**Scenarios:** none

**Files:**
- Create: `docs/ARCHITECTURE.md`, `.github/workflows/ci.yml`
- Modify: `README.md`, `README.zh-TW.md` (an Install section), `AGENTS.md` (drop "(once it exists)")
- Test: the CI run itself

**Interfaces:**
- Consumes: the packages of Tasks 1–7.

- [ ] **Step 1: Write `docs/ARCHITECTURE.md`**, one page: a codemap (`schema.go`, `cmd/dkt`, `internal/{cli,gitx,openspec,scan,setup,change,testutil}`, one line each); a capability index naming the function for each shared job ("run git" → `gitx.Git.Run`, "find changes everywhere" → `scan.Scan`, "start OpenSpec" → `openspec.Runner.Run`, "report a result" → `cli.Outcome`, "build a test repository" → `testutil.NewRepo`, "stand in for OpenSpec" → `testutil.FakeOpenSpec`); and invariants: only `internal/openspec` starts OpenSpec; only `internal/gitx` starts git; nothing below `cmd/dkt` reads the process environment; readers will take bytes, not paths.
- [ ] **Step 2: Write `.github/workflows/ci.yml`** with three jobs on `push` and `pull_request`:
  - `test` on `ubuntu-latest` and `macos-latest`: `actions/setup-go` (`go-version: stable`), `go vet ./...`, `go test ./...`.
  - `openspec` on `ubuntu-latest`: setup-go and `actions/setup-node` (Node 22); `go test ./...` with `DOCKET_TEST_OPENSPEC="npx -y @fission-ai/openspec@1.14.3"` and `OPENSPEC_TELEMETRY=0`.
  - `duplicates` on `ubuntu-latest`: checkout with `fetch-depth: 0`, setup-node, then `npx -y jscpd@5.4.2 --format go --ignore "**/testdata/**" --baseline-from-ref origin/main --fail-on-new-clones .`, which fails only on clones absent from `origin/main`.
- [ ] **Step 3: Add the Install section** to both READMEs: `go install github.com/wake/docket/cmd/dkt@latest`, plus that `dkt init` and `dkt new` need the OpenSpec CLI 1.14.x (or `DOCKET_OPENSPEC`).
- [ ] **Step 4: Verify** `go vet ./...` and `go test ./...` pass locally, and the workflow file parses: `ruby -ryaml -e 'YAML.load_file(".github/workflows/ci.yml")'` exits 0.
- [ ] **Step 5: Commit** `docs: architecture map, CI and install note`.
