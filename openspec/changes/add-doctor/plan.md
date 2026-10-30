# Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `dkt inspect` reports every mechanical problem with a project's Docket setup and writes nothing; `dkt init` ships two more skill folders: `docket-setup` (setup, moved out of the `docket` skill) and `docket-doctor` (a command the user invokes, which runs `dkt inspect`, checks what needs judgment and guides the fixes).

**Architecture:** `setup.Init` is split into deciding and writing, and a new `setup.Inspect` returns the decisions without writing. `internal/cli/cmd_inspect.go` puts four parts into one envelope: those decisions; the OpenSpec lookup and version check (`openspec.New`, `Runner.CheckVersion`); the workflow status, from code shared with `dkt workflow`; and a new `archive.Trailers`, which applies archive's trailer test to the active changes on the default branch. The root package embeds three skill folders, and `setup.Init` installs each under both skill roots.

**Tech Stack:** Go 1.26, `go.yaml.in/yaml/v3` (frontmatter and `openai.yaml` in tests), git; headless Claude Code and codex-cli 0.153.4 for the trial.

**Spec:** `openspec/changes/add-doctor/proposal.md`, `openspec/changes/add-doctor/specs/setup-inspection/spec.md`, `openspec/changes/add-doctor/specs/project-setup/spec.md`, `openspec/changes/add-doctor/design.md`.

## Global Constraints

- `dkt inspect [--json] [--default-branch <name>]` takes no arguments. It changes no file anywhere. It starts only git and the OpenSpec CLI's `--version`, and starts OpenSpec at most once.
- The keys of `result` are exactly `files`, `openspec`, `workflow` and `changes`.
  - A `files` entry has exactly `path`, `kind` and `status`:
    - `path` is relative to the repository root;
    - `kind` is `schema`, `skill` or `block`;
    - `status` is `same`, `missing` or `differs`.
  - `openspec` has exactly `version` and `supported`. `version` is null only when no CLI could be started.
  - `workflow` is the `result` of `dkt workflow --json`. It is null when the workflow file is unreadable.
  - A `changes` entry has exactly `slug` and `skipped`. `skipped` holds the reason, or is null when the change was checked. `files` and `changes` are arrays, never null.
- Finding rules:
  - `schema-missing`, `skill-missing`, `agents-block-missing`;
  - `schema-differs`, `skill-differs`, `agents-block-differs`, `agents-block-malformed`;
  - `symlink-refused`;
  - `openspec-missing`, `openspec-unsupported`;
  - `workflow-missing`, `workflow-incomplete`, `workflow-stale`;
  - `no-task-commits`.
- Exit codes:
  - 0: no finding;
  - 1: any finding;
  - 3: anything unreadable; the other checks are still reported;
  - 4: only when the directory is not a git repository or no default branch can be found.
  - A missing or unsupported OpenSpec and a missing home directory are findings (exit 1), never exit 4.
- `dkt init` installs `docket`, `docket-setup` and `docket-doctor` under both `.claude/skills/<name>/` and `.agents/skills/<name>/`.
  - The two copies of each folder are byte-for-byte identical.
  - In each `SKILL.md`, `name` is the folder's name and `description` is at most 1,024 characters.
- `docket-doctor/SKILL.md` has `disable-model-invocation: true` in its frontmatter. `docket-doctor/agents/openai.yaml` holds `policy: allow_implicit_invocation: false`. The other two skills set neither.
- `dkt init`'s output, findings and exit codes do not change, apart from the new skill files it lists.
- Only `internal/openspec` starts OpenSpec and only `internal/gitx` starts git (test helpers excepted). Nothing below `cmd/dkt` reads the process environment.
- Everything in the repository is in English.
- TDD, one item per round. A task's "Write the failing tests" step is its test list. Take one test, see it fail, write the code that makes it pass, then take the next. The steps say what to build, not that every test is written before any code.
- Commits:
  - every commit for a task carries `Change: add-doctor` and `Task: <N>` in one final trailer paragraph (write the message to a file and use `git commit -F`);
  - structural and behavior changes go in separate commits;
  - author `Claude Code <wake+claude-code@protype.tw>`, passed with `git -c user.name=... -c user.email=...`.

## Review Focus

- A `CLAUDE.md` that is a symbolic link to `AGENTS.md`, rather than an `@AGENTS.md` import, is not listed and gives no finding. Test: Task 1 (`TestInspectClaudeMDImportsAgentsMD`, link case).
- A skill directory that is a symbolic link gives one `symlink-refused` naming the link, not one per file under it. The other entries are still reported. Test: Task 1 (`TestInspectSymlinkedSkillDirectory`).
- An OpenSpec whose `--version` exits non-zero gives `openspec-missing`, exit 1, not exit 4. So does a `DOCKET_OPENSPEC` naming a program that does not exist. Test: Task 2 (`TestInspectOpenSpecVersionFails`, `TestInspectNoOpenSpec`).
- These changes are not listed and are not checked:
  - a change on the default branch with no `plan.md`, or a plan with no `### Task`;
  - a change that exists only on a feature branch or in a worktree;
  - an archived change.
  Test: Task 3 (`TestTrailersNotListed`).
- A recorded base that is not a commit (a typo, or a value starting with `-`) is skipped with its reason. It is never passed to git as an option, and never an error. Test: Task 3 (`TestTrailersBaseNotACommit`).

## Reuse

| Need | Existing (path:symbol) | Decision | Reason | Evidence |
|---|---|---|---|---|
| Decide every installed file's status and check its path | internal/setup/setup.go:Init | extend | split into `decideAll` plus the writes, so `Inspect` applies exactly init's rules (symlinks, the `CLAUDE.md` import, block placement) | `rg -n -e "func Init" -e "func decide" -e "func targetsOf" internal/setup/setup.go` |
| Read-only entry point for the decisions | none | new | `setup.Inspect`: nothing else decides without writing; `Init` always writes | `rg -n "^func [A-Z]" internal/setup/` |
| Turn a refusal into findings (one `symlink-refused` per link, `agents-block-malformed`, unreadable) | internal/cli/cmd_init.go:initCommand | extend | extract the loop as `problemFindings` for init and inspect | `rg -n "seenLink" internal/cli/` |
| The assets init installs | internal/cli/cmd_init.go:initCommand | extend | extract `shippedAssets()` so init and inspect install and inspect the same files | `rg -n -F "setup.Assets{" internal/cli/` |
| Embed several skill folders | schema.go:Skill | extend | `Skills()` returns the three folders in order; `Skill()` goes | `rg -n -e "go:embed" -e "func Skill" schema.go` |
| Install each skill under both roots | internal/setup/setup.go:skillDirs | extend | `skillRoots` × `Assets.Skills` in `targetsOf` | `rg -n "skillDirs" internal/` |
| Locate OpenSpec and check its version | internal/openspec/runner.go:New | reuse | `New` and `Runner.CheckVersion` return `*UnavailableError` and `*UnsupportedError`, which map to the two findings; the runner does not change | `rg -n -F -e "&UnavailableError{" -e "&UnsupportedError{" internal/openspec/runner.go` |
| Report the workflow file | internal/cli/cmd_workflow.go:workflowReport | extend | extract `workflowState`, shared by `dkt workflow` and `dkt inspect` | `rg -n "func workflowReport" internal/cli/` |
| Resolve the default branch, exit 4 when none | internal/cli/cmd_new.go:resolveDefaultBranch | reuse | maps `scan.ErrNoDefaultBranch` to `EnvError` as other commands do | `rg -n "func resolveDefaultBranch" -A7 internal/cli/cmd_new.go` |
| Active changes in the default branch's tree | internal/gitx/git.go:ListDirs | reuse | as archive's preconditions do; `scan.Scan` reads every worktree and branch, which the check must not | `rg -n "ListDirs" internal/` |
| Read a file at the tip | internal/gitx/git.go:ShowIfExists | reuse | missing paths are not errors | `rg -n "func (g Git) ShowIfExists" internal/gitx/` |
| Read the base | internal/change/metadata.go:ReadMetadata | reuse | `docket.yaml`, else `x-base` of `.openspec.yaml` | `rg -n "func ReadMetadata" internal/change/` |
| Read the plan's tasks | internal/change/plan.go:ParsePlan | reuse | `Plan.Tasks` | `rg -n "func ParsePlan" internal/change/` |
| Is the base a commit, an ancestor | internal/gitx/git.go:RefExists | reuse | with `IsAncestor`, as `loadChange` and archive's preconditions check | `rg -n -e "RefExists" -e "IsAncestor" internal/archive/` |
| Commits between base and tip, with trailers | internal/gitx/log.go:Commits | reuse | `--full-history`, so a merge's second parent counts | `rg -n "func (g Git) Commits" internal/gitx/` |
| The trailer test | internal/change/trailers.go:NamedTasks | reuse | its `anyTrailer` is exactly archive's `no-task-commits` test | `rg -n -i "anytrailer" internal/` |
| Trailer check outside archiving | none | new | `archive.Trailers`: `planFindings` needs a change loaded from the working tree and a sync result; this reads the default branch's tree for every active change | `rg -n "no-task-commits" internal/` |
| Copy archive findings into the envelope | internal/cli/cmd_sync.go:addArchiveReport | reuse | converts `archive.Finding` and `archive.Unreadable` | `rg -n "func addArchiveReport" internal/cli/` |
| Envelope, exit codes, command table | internal/cli/envelope.go:ExitCode | reuse | add `inspectCommand` to `commands` in `run.go` | `rg -n "var commands" internal/cli/run.go` |
| Test repositories, fake OpenSpec, workflow and snapshot helpers | internal/testutil/repo.go:NewRepo | reuse | with `testutil.FakeOpenSpec`, and in package `cli` `wfConfig`, `wfWrite`, `wfFM`, `wfFour`, `wfHash`, `snapshot`, `sameSnapshot`, `committedRepo`, `runInit` | `rg -n -e "^func wf" -e "^func snapshot" -e "^func sameSnapshot" -e "^func committedRepo" -e "^func runInit" internal/cli/` |
| Parse frontmatter and `openai.yaml` in tests | external: go.yaml.in/yaml/v3 | reuse | already used by `TestSkillFrontmatter` | `rg -n "yaml" schema_test.go` |
| Check which skills Codex shows the model | external: codex | reuse | `codex debug prompt-input` renders the model's input without a model call, as in the design check | `codex debug --help` |
| Run the skills headless in a scratch project | external: claude | reuse | as in the add-workflow trial, with `--setting-sources project,local` | `claude --help` |

---

### Task 1: `dkt inspect` reports the installed files

**Scenarios:** setup-inspection#before-init, setup-inspection#edited-skill, setup-inspection#claude-md-imports-agents-md

**Review:** judgment

**Files:**
- Modify: `internal/setup/setup.go`
- Modify: `internal/cli/cmd_init.go`
- Modify: `internal/cli/run.go`
- Create: `internal/cli/cmd_inspect.go`
- Test: `internal/setup/setup_test.go`
- Create: `internal/cli/cmd_inspect_test.go`

**Interfaces:**
- Produces:
  - **`setup.Inspect(root string, a Assets) ([]File, []Problem, error)`**
    - Returns the files `Init` would install or keep, decided as `Init` decides them without force. They keep `Init`'s statuses: `Written` for a file `Init` would write, `Unchanged`, `Differs`. Each has its `Kind`.
    - Also returns the problems that would make `Init` refuse. Unlike `Init`, it still returns the files that passed.
    - It writes nothing and starts nothing.
    - It leaves out a `CLAUDE.md` that imports `AGENTS.md` (`@AGENTS.md`) or is a symbolic link to it.
    - The error is for failing to read the assets.
  - In package `cli`:
    - `shippedAssets() setup.Assets`;
    - `problemFindings(ps []setup.Problem) ([]Finding, []Unreadable)`, init's mapping of a refusal, unchanged;
    - `InspectResult{Files []InspectFile}`, with JSON `files`;
    - `InspectFile{Path string; Kind setup.FileKind; Status string}`, with JSON `path`, `kind` and `status`.
  - Tasks 2 and 3 add fields to `InspectResult`.

- [ ] **Step 1: Split init's decisions from its writes (structural)**
  - In `setup.go`, move the start of `Init` into `decideAll(root string, a Assets, force bool) ([]decided, []Problem, error)`. That is the `openspec` symlink check, with its early return, then `targetsOf` and the `decide` loop. `Init` calls `decideAll` and keeps the rest.
  - In `cmd_init.go`:
    - extract `shippedAssets()`, the `setup.Assets{...}` literal;
    - extract `problemFindings`, the loop over `ref.Problems`.
  - Run `go test ./internal/setup/ ./internal/cli/` and `go vet ./...`. Expected: PASS, with no test file changed.
  - Commit as `refactor(setup): split init's decisions from its writes`, with the trailers.

- [ ] **Step 2: Write the failing tests**

  **`internal/setup/setup_test.go: TestInspect`**, using the package's `assets`, with one sub-test per case:
  - Empty temporary root:
    - every target is `Written`, including `AGENTS.md` and `CLAUDE.md`;
    - no problems;
    - the root is still empty afterwards.
  - After `Init`: every returned file is `Unchanged`, and `CLAUDE.md` is absent, since `Init` wrote `@AGENTS.md`.
  - With `.agents/skills/docket/SKILL.md` rewritten:
    - that file is `Differs` with `KindSkill`;
    - the file is unchanged on disk.
  - `CLAUDE.md` replaced by a symbolic link to `AGENTS.md`: `CLAUDE.md` is absent.
  - `.claude/skills/docket` a symbolic link to another directory:
    - the problems are `ProblemSymlink` for the paths under it;
    - the schema and `.agents/skills/docket/SKILL.md` are still returned;
    - nothing is written in either directory.

  **`internal/cli/cmd_inspect_test.go`** gets two helpers, used by every inspect test in this and later tasks:
  - `runInspect(t *testing.T, dir string, env []string, args ...string) (int, string, string)` runs `Run` with `Args: append([]string{"inspect"}, args...)`.
  - `inspectEnv(t *testing.T) (env []string, workflowPath string)` builds one clean environment, so that Tasks 2 and 3 add no findings to these tests:
    - `testutil.Env(t)`;
    - `wfConfig(t)`'s `XDG_CONFIG_HOME`, with `wfWrite(t, path, wfFM(wfHash())+wfFour)`;
    - `DOCKET_OPENSPEC=` a `testutil.FakeOpenSpec(t, "1.14.3")`.

    A test changes one part by appending a variable, since the last value counts, or by rewriting the workflow file.

  Use `committedRepo`, and `runInit` with `testutil.OpenSpecCmd(t)`.
  - **`TestInspectBeforeInit`**: no `dkt init`.
    - Exit 1.
    - The rules include `schema-missing`, `skill-missing` and `agents-block-missing`.
    - Every `result.files` status is `missing`.
    - The `kind` values seen include `schema`, `skill` and `block`.
  - **`TestInspectEditedSkill`**: `dkt init`, then write `mine\n` to `.claude/skills/docket/SKILL.md`.
    - The `result.files` entry for that path has status `differs` and kind `skill`.
    - Exactly one finding: `skill-differs` with `file` set to that path.
    - The file still holds `mine\n`.
  - **`TestInspectClaudeMDImportsAgentsMD`**: after `dkt init`:
    - case 1: `CLAUDE.md` is `@AGENTS.md\n` (init wrote it);
    - case 2: `CLAUDE.md` replaced by a symbolic link to `AGENTS.md`.
    - In both cases, no `result.files` entry and no finding has the path `CLAUDE.md`.
  - **`TestInspectSymlinkedSkillDirectory`**: `dkt init`, then replace `.claude/skills/docket` with a symbolic link to a temporary directory.
    - Exit 1.
    - Exactly one `symlink-refused` finding, with `file` `.claude/skills/docket`.
    - `result.files` still lists the schema files as `same`.
  - **`TestInspectUnreadableFile`**: skipped as root.
    - `dkt init`, then chmod `.agents/skills/docket/SKILL.md` to `000`.
    - Exit 3.
    - `unreadable` names that path.
    - `result.files` still lists the schema files.
  - **`TestInspectUsage`**: `dkt inspect extra` exits with 2.

- [ ] **Step 3: Run them to see them fail**

  Run: `go test ./internal/setup/ -run TestInspect` and `go test ./internal/cli/ -run TestInspect`.
  Expected: FAIL (`undefined: Inspect`, `unknown command "inspect"`).

- [ ] **Step 4: Implement `setup.Inspect` and the `inspect` command**
  - **`setup.Inspect`**
    - It is `decideAll(root, a, false)`, with each `decided` mapped to a `File`.
    - Mark the two `CLAUDE.md` cases with an unexported field on `decided`, set where `decide` and `decideBlock` already return `Unchanged` for them, and skip those entries. `Init` ignores the field.
  - **`inspectCommand()`** in `cmd_inspect.go`: `Name: "inspect"`, `Usage: "inspect"`. Any argument gives `UsageError{Msg: "inspect takes no arguments"}`. Add it to the end of `commands` in `run.go`.
  - **Status and finding for each file.** `File` is the file's path.

    | Init status | `status` | Rule (kind schema / skill / block) | Message |
    |---|---|---|---|
    | `Written` | `missing` | `schema-missing` / `skill-missing` / `agents-block-missing` | `not installed; run dkt init` (block: `the Docket block is not installed; run dkt init`) |
    | `Unchanged` | `same` | none | |
    | `Differs` | `differs` | `schema-differs` / `skill-differs` / `agents-block-differs` | `differs from the shipped schema; dkt init --force overwrites it` (skill: `... shipped skill ...`; block: `the Docket block differs from the shipped one; dkt init --force replaces it`) |

  - **Problems** go through `problemFindings`.
  - **Human output**: one line `fmt.Sprintf("%-8s %s", status, path)` for each file that is not `same`.
  - `result.files` is `[]InspectFile{}` when empty.

- [ ] **Step 5: Run the tests to see them pass**

  Run: `go test ./internal/setup/ ./internal/cli/` and `go vet ./...`.
  Expected: PASS.

- [ ] **Step 6: Commit**

  Commit as `feat(inspect): report the installed files`, with the trailers.

### Task 2: `dkt inspect` reports OpenSpec and the workflow file

**Scenarios:** setup-inspection#clean-setup, setup-inspection#unsupported-openspec, setup-inspection#no-openspec, setup-inspection#workflow-reported

**Review:** judgment

**Files:**
- Modify: `internal/cli/cmd_workflow.go`
- Modify: `internal/cli/cmd_inspect.go`
- Test: `internal/cli/cmd_inspect_test.go`

**Interfaces:**
- Consumes: Task 1's `InspectResult` and `inspectCommand`.
- Produces:
  - **`workflowState(c *Context) (*WorkflowResult, []Finding, []Unreadable, error)`**
    - Returns what `dkt workflow` reports: the result, nil when the file cannot be read; the findings; and the unreadable sources.
    - The error wraps `workflow.ErrNoHome`, or is another failure to locate the file.
  - `WorkflowResult.Path` becomes `*string`. `dkt workflow` always sets it, so its output does not change.
  - **New fields:**
    - `InspectResult.OpenSpec OpenSpecState`, with JSON `openspec`;
    - `OpenSpecState{Version *string; Supported bool}`, with JSON `version` and `supported`;
    - `InspectResult.Workflow *WorkflowResult`, with JSON `workflow`.

- [ ] **Step 1: Share `dkt workflow`'s report (structural)**
  - Move everything in `workflowReport` except the `ErrNoHome` → `EnvError` mapping into `workflowState`. `workflowReport` keeps that mapping and builds its Human line.
  - Run `go test ./internal/cli/ -run TestWorkflow`. Expected: PASS, with no test changed.
  - Commit as `refactor(workflow): share the workflow report`, with the trailers.

- [ ] **Step 2: Write the failing tests**

  **Shared setup.** Each test starts from Task 1's `inspectEnv(t)`. It overrides `DOCKET_OPENSPEC` by appending a variable, or rewrites the file at `workflowPath`.
  - **`TestInspectCleanSetup`**:
    - `committedRepo`, then `runInit`;
    - `inspectEnv(t)` as it is;
    - `dkt inspect --json`.
    - Expected:
      - exit 0;
      - `findings` empty;
      - `result.openspec` equals `{"version": "1.14.3", "supported": true}`;
      - `result.workflow.status` is `ok`;
      - `result` has exactly the keys `files`, `openspec`, `workflow` and `changes` once Task 3 lands. Assert the first three now; Task 3 adds `changes` to this assertion.
  - **`TestInspectUnsupportedOpenSpec`**: fake `FakeOpenSpec(t, "1.3.0")`.
    - `result.openspec.version` is `1.3.0` and `supported` is false.
    - A finding `openspec-unsupported` whose message contains `1.3.0` and `1.14.x`.
    - Exit 1.
  - **`TestInspectNoOpenSpec`**: `DOCKET_OPENSPEC=<temp dir>/missing`.
    - `result.openspec.version` is null and `supported` is false.
    - A finding `openspec-missing`.
    - Exit 1, not 4.
  - **`TestInspectOpenSpecVersionFails`**: `DOCKET_OPENSPEC` is a script that prints nothing and exits 3. Same assertions as `TestInspectNoOpenSpec`.
  - **`TestInspectWorkflowReported`**: the workflow file is `wfFM(wfHash())` plus the four headings without `## Check`.
    - `result.workflow.status` is `incomplete`.
    - One finding `workflow-incomplete` whose message contains `"## Check"`.
  - **`TestInspectWorkflowNoHome`**: drop `HOME=` and `XDG_CONFIG_HOME=` from the environment, as `TestWorkflowNoHome` does, and set `UserHome: nil`.
    - Exit 1, not 4.
    - A finding `workflow-missing` whose message is `workflow.ErrNoHome`'s text.
    - `result.workflow` is `path` null, `status` `missing`, `recorded` null, `missing_sections` `[]`.
    - `result.files` is non-empty.
  - **`TestInspectWorkflowUnreadable`**: skipped as root. The workflow file exists with mode `000`.
    - Exit 3.
    - `unreadable` names its path.
    - `result.workflow` is null.
    - `result.files` is non-empty and `result.openspec.version` is `1.14.3`. Task 3 adds `result.changes`.
  - Extend Task 1's `TestInspectUnreadableFile`: `result.openspec.version` is `1.14.3` and `result.workflow.status` is `ok`.

- [ ] **Step 3: Run them to see them fail**

  Run: `go test ./internal/cli/ -run TestInspect`.
  Expected: FAIL (no `openspec` or `workflow` in `result`).

- [ ] **Step 4: Implement**
  - **OpenSpec.** Call `openspec.New(c.Repo.Root, c.Env)`, then `CheckVersion()`.
    - On success: `version` is the result and `supported` is true.
    - `*openspec.UnavailableError`, from either call: `openspec-missing` with the error's text as the message, and `version` null.
    - `*openspec.UnsupportedError`: `openspec-unsupported` with the error's text, and `version` set to `Found`.
    - Any other error is returned.
  - **Workflow.** Call `workflowState`.
    - On `workflow.ErrNoHome`: one finding `{Rule: "workflow-missing", Message: err.Error()}` with no `File`. `result.workflow` is `&WorkflowResult{Path: nil, Status: workflow.Missing, Template: workflow.Hash(docket.Template()), Recorded: nil, MissingSections: []string{}}`.
    - Otherwise, append its findings and unreadable sources.
  - **Human lines:**
    - `openspec <version>`, plus ` (unsupported)` when not supported, or `openspec not found`;
    - `workflow <status> <path>`, with no path when it is null; no line when `result.workflow` is null.

- [ ] **Step 5: Run the tests to see them pass**

  Run: `go test ./internal/cli/` and `go vet ./...`.
  Expected: PASS.

- [ ] **Step 6: Commit**

  Commit as `feat(inspect): report OpenSpec and the workflow file`, with the trailers.

### Task 3: `dkt inspect` checks the trailers on the default branch

**Scenarios:** setup-inspection#squashed-change, setup-inspection#merged-with-a-merge-commit, setup-inspection#base-not-an-ancestor, setup-inspection#nothing-written

**Review:** judgment

**Files:**
- Create: `internal/archive/trailers.go`
- Create: `internal/archive/trailers_test.go`
- Modify: `internal/cli/cmd_inspect.go`
- Test: `internal/cli/cmd_inspect_test.go`

**Interfaces:**
- Consumes:
  - `archive.Env{Git, Root, Default}`;
  - `resolveDefaultBranch(c)`;
  - `addArchiveReport`.
- Produces:
  - `archive.TrailerCheck{Slug string; Skipped *string}`, with JSON `slug` and `skipped`;
  - `archive.Trailers(e Env) ([]TrailerCheck, []Finding, error)`;
  - `InspectResult.Changes []archive.TrailerCheck`, with JSON `changes`, never null.

- [ ] **Step 1: Write the failing tests**

  **`internal/archive/trailers_test.go`** builds repositories on `main` with `testutil.NewRepo` and `Repo.Commit`. The plan is `"# Plan\n\n### Task 1: One\n\n- [x] **Step 1: a**\n\n### Task 2: Two\n\n- [x] **Step 1: b**\n"`. `docket.yaml` is `title: X\nkind: feature\nsource: s\nbase: <sha>\n`. The `Env` comes from `scan.ResolveDefaultBranch(g, "main")`.
  - **`TestTrailersSquashed`**:
    - commit `c1`;
    - then one commit `squashed (#1)` with no trailers, adding `openspec/changes/add-x/docket.yaml` (base `c1`) and its `plan.md`.
    - Expected:
      - checks `[{add-x, nil}]`;
      - one finding, `no-task-commits`, with `Change` `add-x` and `File` `openspec/changes/add-x/plan.md`;
      - the message contains `squash`.
  - **`TestTrailersMergeCommit`**:
    - commit `c1`;
    - branch `work` from `c1` with a commit whose message ends with `Change: add-x\nTask: 1` and adds the change;
    - another commit on `main`;
    - `git merge --no-ff work` on `main`.
    - Expected: no finding; checks `[{add-x, nil}]`.
  - **`TestTrailersBaseNotAncestor`**: the base is a commit on branch `other`, which is not merged into `main`.
    - No finding.
    - `Skipped` contains `is not an ancestor of main`.
  - **`TestTrailersBaseNotACommit`**: two changes, with bases `0123456789abcdef` and `-x`.
    - Both are skipped with `is not a commit in this repository`.
    - No error.
  - **`TestTrailersNoBase`**: `docket.yaml` has no `base`. Skipped with `no base recorded`.
  - **`TestTrailersNotListed`**: none of the following appears in the checks.
    - a change with no `plan.md`;
    - a change whose plan has no `### Task`;
    - a change committed only on an unmerged branch;
    - `openspec/changes/archive/2026-01-01-add-y/` with a plan and no trailers.

  **`internal/cli/cmd_inspect_test.go`:**
  - **`TestInspectSquashedChange`**: as `TestTrailersSquashed`, after `dkt init` and a commit of the installed files.
    - The findings include `no-task-commits` with `change` `add-x`.
    - `result.changes` equals `[{"slug": "add-x", "skipped": null}]`.
  - **`TestInspectNothingWritten`**:
    - **Setup:**
      - `committedRepo` and `dkt init`;
      - overwrite every installed file under `openspec/schemas/docket/`, `.claude/skills/` and `.agents/skills/` with `mine\n`;
      - replace the text between the Docket markers in `AGENTS.md`;
      - commit, then commit a squashed change as above;
      - no workflow file;
      - `DOCKET_OPENSPEC` is a fresh `FakeOpenSpec(t, "1.14.3")`, which returns its own log;
      - build the environment once, since every `testutil.Env` call makes a new `HOME`.
    - **Snapshot** with `snapshot(t, dir)`:
      - the repository, but not `.git/`;
      - the `XDG_CONFIG_HOME` directory;
      - the `HOME` directory.
    - **Run** `dkt inspect --json`.
    - **Expected:**
      - exit 1;
      - the rules include `schema-differs`, `skill-differs`, `agents-block-differs`, `workflow-missing` and `no-task-commits`;
      - there is no `openspec-*` finding;
      - every snapshot is unchanged;
      - `git status --porcelain` is unchanged;
      - the fake's log holds exactly one line, `--version`.
  - **`TestInspectNoDefaultBranch`**: a repository whose only branch is `work`, with no `main`, `master` or origin. Exit 4.
  - **`TestInspectDefaultBranchFlag`**:
    - **Setup:** `main` holds no change. Branch `trunk`, made from `main`, holds a squashed change `add-x`.
    - **Run 1:** `dkt inspect --json --default-branch trunk`. The findings include `no-task-commits` for `add-x`.
    - **Run 2:** without the flag, no finding names `add-x`.
  - Extend `TestInspectCleanSetup`: add `changes` to the key assertion, as `[]`.
  - Extend `TestInspectUnreadableFile` and `TestInspectWorkflowUnreadable`: `result.changes` is present, as `[]`.

- [ ] **Step 2: Run them to see them fail**

  Run: `go test ./internal/archive/ -run TestTrailers` and `go test ./internal/cli/ -run TestInspect`.
  Expected: FAIL (`undefined: Trailers`, no `changes` in `result`).

- [ ] **Step 3: Implement `archive.Trailers`**

  For each name `e.Git.ListDirs(e.Default.Commit, "openspec/changes")` returns, except `archive`, in that order, with `dir` = `openspec/changes/<name>`:
  1. Read `<dir>/plan.md` with `ShowIfExists` at the tip. When it is missing, or `change.ParsePlan` finds no task, the change is not listed.
  2. Read `<dir>/docket.yaml` and `<dir>/.openspec.yaml` at the tip into a map and call `change.ReadMetadata`.
     - On an error, skip with `cannot read the base: <err>`.
     - With an empty base, skip with `no base recorded`.
  3. When the base starts with `-`, or `RefExists(base+"^{commit}")` is false, skip with `the base <base> is not a commit in this repository`.
  4. When `IsAncestor(base, tip)` is false, skip with `the base <base> is not an ancestor of <default branch name>`.
  5. Call `Commits(base, tip)`, then `change.NamedTasks(plan, name, commits)`. When `anyTrailer` is false, add a finding:
     - `Rule: "no-task-commits"`, `Change: name`, `File: dir+"/plan.md"`;
     - `Message: "no commit on <default branch name> since the base carries Change: <name> with a Task: trailer; was the pull request squashed? dkt archive will refuse this change"`.

  Git failures are returned as the error.

  **In `inspectCommand`:**
  - call `resolveDefaultBranch(c)` first, so exit 4 comes before any other check;
  - after the other parts, call `archive.Trailers(archive.Env{Git: c.Git, Root: c.Repo.Root, Default: db})`;
  - add its findings with `addArchiveReport`;
  - add a Human line `skipped  <slug>: <reason>` for each skipped change.

- [ ] **Step 4: Run the tests to see them pass**

  Run: `go test ./internal/archive/ ./internal/cli/` and `go vet ./...`.
  Expected: PASS.

- [ ] **Step 5: Commit**

  Commit as `feat(inspect): check the trailers of changes on the default branch`, with the trailers.

### Task 4: The setup and doctor skills, installed with the docket skill

**Scenarios:** project-setup#fresh-repository-gets-the-skill, project-setup#skill-copies-agree, project-setup#other-configuration-unchanged, project-setup#symlinked-schema-file-refused, project-setup#symlinked-skill-directory-refused, project-setup#setup-before-the-first-work, project-setup#setup-invoked-by-the-user, setup-inspection#doctor-not-run-on-its-own, setup-inspection#doctor-invoked, setup-inspection#drift-found, setup-inspection#problems-listed-first, setup-inspection#edited-skill-fixed-with-consent

**Review:** judgment

**Files:**
- Create:
  - `.agents/skills/docket-setup/SKILL.md`
  - `.agents/skills/docket-doctor/SKILL.md`
  - `.agents/skills/docket-doctor/agents/openai.yaml`
  - `.claude/skills/docket-setup/SKILL.md`, `.claude/skills/docket-doctor/SKILL.md` and `.claude/skills/docket-doctor/agents/openai.yaml` (written by `dkt init`)
- Delete: `.agents/skills/docket/setup.md`, `.claude/skills/docket/setup.md`
- Modify:
  - `.agents/skills/docket/SKILL.md`, `.claude/skills/docket/SKILL.md`
  - `schema.go`, `schema_test.go`
  - `internal/setup/setup.go`, `internal/setup/setup_test.go`, `internal/setup/rewrite_test.go`
  - `internal/cli/cmd_init.go`, `internal/cli/cmd_init_test.go`, `internal/cli/cmd_init_agents_test.go`, `internal/cli/cmd_inspect_test.go`
  - the last four only where they name `docket.Skill()`, `Assets.Skill` or `setup.md`

**Interfaces:**
- Produces:
  - `setup.Skill{Name string; Files fs.FS}`.
  - `setup.Assets.Skills []setup.Skill`, which replaces `Assets.Skill`.
  - `docket.Skills() []setup.Skill`, which replaces `docket.Skill()`. It returns `docket`, `docket-setup` and `docket-doctor`, in that order, each with paths relative to its folder.
  - In `setup.go`, `skillRoots = []string{".claude/skills", ".agents/skills"}` replaces `skillDirs`.
  - `targetsOf` order: the schema; then, for each root, each skill in order; then `AGENTS.md` and `CLAUDE.md`.

- [ ] **Step 1: Take the skills by name (structural)**
  - **`internal/setup/setup.go`**:
    - add `Skill`, `Assets.Skills` and `skillRoots`, in place of `Assets.Skill` and `skillDirs`;
    - `targetsOf` uses the order above.
  - **`schema.go`**: `Skills()` replaces `Skill()`. It still embeds only `.agents/skills/docket` and returns `[]setup.Skill{{"docket", ...}}`, made with `fs.Sub`. It panics on an error, as `Skill()` did.
  - **`cmd_init.go`**: `shippedAssets()` passes `docket.Skills()`.
  - **Tests.** Adapt every test that names `docket.Skill()`, `Assets.Skill` or `skillRels` to the new interfaces, without weakening it:
    - `TestSkillFrontmatter` and `TestInstalledSkillMatchesShipped` loop over `Skills()`;
    - `TestFreshRepositoryGetsTheSkill` loops over `docket.Skills()` and both roots, checking every shipped file byte-for-byte.
    - In `internal/setup/setup_test.go`:
      - `assets` gets `Skills: []Skill{{"docket", fstest.MapFS{"SKILL.md": ...}}, {"docket-setup", fstest.MapFS{"SKILL.md": ..., "agents/x.yaml": ...}}}`;
      - `TestInitInstallsSkill` checks all six paths, for the two skills under each of the two roots, as `Written` with `KindSkill` and the given bytes. A rerun gives `Unchanged`.
  - **Verify.** Run `go test ./...` and `go vet ./...`. Expected: PASS. `dkt init`'s tests show its output unchanged.
  - Update the doc comments that say "the skill" where they now mean the skills.
  - Commit as `refactor(setup): take the skills by name`, with the trailers.

- [ ] **Step 2: Write the failing tests**

  **`schema_test.go`:**
  - **`TestSkillNames`**: the names of `Skills()` are `[docket docket-setup docket-doctor]`.
  - **`TestSkillFrontmatter`**: also reads `disable-model-invocation` as a `*bool`. It is `true` for `docket-doctor` and absent for the other two.
  - **`TestDoctorHiddenFromCodex`**:
    - `agents/openai.yaml` of `docket-doctor` decodes to `policy.allow_implicit_invocation`, read as a `*bool`, equal to `false`;
    - neither of the other two skills has `agents/openai.yaml`.
  - **`TestInstalledSkillMatchesShipped`**: also checks that every regular file under `.claude/skills/<name>/` is a shipped file. This catches a stale `setup.md`.

  **`internal/cli/cmd_init_test.go`:**
  - **`TestSkillCopiesAgree`**: after `dkt init`, for each skill:
    - the regular files under `.claude/skills/<name>/` and `.agents/skills/<name>/` are the same set of relative paths, with byte-identical contents;
    - `SKILL.md`'s frontmatter `name` equals `<name>`;
    - its `description` is at most 1,024 characters.

- [ ] **Step 3: Run them to see them fail**

  Run: `go test . -run 'TestSkill|TestDoctor|TestInstalled'`.
  Expected: FAIL. `TestSkillNames` sees only `docket`; `TestDoctorHiddenFromCodex` finds no `docket-doctor`.

- [ ] **Step 4: Write the skill files**
  - In `schema.go`, embed `.agents/skills/docket .agents/skills/docket-setup .agents/skills/docket-doctor`. `Skills()` returns the three, in that order.
  - Create `.agents/skills/docket-setup/SKILL.md` with the text below. It is today's `setup.md`, with the changes this change asks for.
  - Create `.agents/skills/docket-doctor/SKILL.md` and `.agents/skills/docket-doctor/agents/openai.yaml` with the text below.
  - In `.agents/skills/docket/SKILL.md`:
    - replace "`missing` or `incomplete`: read `setup.md` in this skill's directory and run setup first; then start the work." with "`missing` or `incomplete`: run the `docket-setup` skill first; then start the work.";
    - replace "and that setup can update it." with "and that the `docket-setup` skill can update it.";
    - keep the line wrapping of the surrounding text.
  - Delete `.agents/skills/docket/setup.md` with `git rm`.

  `.agents/skills/docket-setup/SKILL.md`:

  ````markdown
  ---
  name: docket-setup
  description: >-
    Write or revise the user's Docket workflow file. Use when the user asks
    to set up, update or change their Docket workflow, or when the docket
    skill sends you here because `dkt workflow` reports `missing` or
    `incomplete`.
  ---

  # Workflow setup

  Setup writes the user's workflow file: Docket's template, with each step
  said the way this user does it. Run it:

  - when the `docket` skill sends you here because `dkt workflow` reports
    `missing` or `incomplete` at the start of development work; finish
    setup before that work;
  - when the user asks to update a `stale` workflow;
  - when the user asks to change their workflow, whatever its status.

  1. Run `dkt workflow --json` for the file's path (`result.path`) and its
     status (`result.status`), and `dkt workflow template --json` for the
     template (`result.text`) and its hash (`result.template`).
  2. Draft the file from the template, step by step, saying for each step
     how this user does it:
     - Look first at the user's instructions already in your context, such
       as their global agent instructions and memory: the reviewers, models,
       commands and habits they name for planning, reviews, verification and
       merging.
     - Then at the tools you can use: other models you can call for a
       review, subagent types, plugins, skills and commands.
     - Do not read, list or search user-level files that are not already in
       your context, such as global instruction files or another agent's
       configuration, unless the user agrees first.
     - Keep every step of the template. Where the user's instructions say
       how to do a step, write it their way, naming the tool and its
       settings; where they add a step, add it; where they contradict a
       step, keep the template's step and note the conflict for the user to
       settle.
     - Keep the headings exactly as the template has them: `## Full review`,
       `## Light review`, `## Check`, `## Direct merge` and `## Reviews`.
     - For `incomplete`, keep the sections the file already has and draft
       only the missing ones; the draft is still the whole file. When
       `result.recorded` from step 1 is not the template's hash, the kept
       sections are also out of date: treat them as for `stale`. For
       `stale`, keep the user's tools and bring in what the current
       template does differently.
     - For a change the user asks for, start from the file as it is and
       change only what they asked, along with what `stale` or `incomplete`
       calls for when the file has that status.
  3. Show the user the whole file as it will be written, in one block with
     every section in full, not a summary or only the new part. Then list
     each point where it departs from the template or from their
     instructions, and the conflicts to settle. Revise it until they agree.
  4. Only after the user agrees, write the file at `result.path`, creating
     its directory when missing. If the path is a symbolic link, write
     through it and keep the link. The file starts with the hash from
     step 1:

     ```
     ---
     template: "<hash>"
     ---
     ```

  5. Run `dkt workflow`. It must report `ok`; then go on with the work, if
     there is any.

  If the user does not want to set up now, follow the template's steps for
  this piece of work, and ask again at the next one.
  ````

  `.agents/skills/docket-doctor/SKILL.md`:

  ````markdown
  ---
  name: docket-doctor
  description: >-
    Check this project's Docket setup (the files dkt init installs, the
    OpenSpec CLI, the user's workflow file, merged changes that lost their
    commit trailers, and what needs judgment), then help the user fix what
    is wrong. Runs only when the user invokes it.
  disable-model-invocation: true
  ---

  # Docket doctor

  Check this project's Docket setup, list every problem, then fix them with
  the user one at a time. Change nothing the user has not agreed to.

  ## 1. Run `dkt inspect`

  Run `dkt inspect --json` before anything else. It writes nothing. Its
  `result` holds:

  - `files`: each file `dkt init` installs, as `same`, `missing` or
    `differs`;
  - `openspec`: the OpenSpec CLI's `version` (null when none could be
    started) and whether it is `supported`;
  - `workflow`: the user's workflow file, as `dkt workflow` reports it;
  - `changes`: the active changes on the default branch whose commit
    trailers were checked, and those skipped, with the reason.

  Its `findings` and `unreadable` are the mechanical problems. When it exits
  with 2 or 4 it checked nothing: tell the user what it said and stop.

  ## 2. Check what needs judgment

  Read the workflow file at `result.workflow.path` when it exists, then
  check:

  - What each step needs: for each step of the workflow file, whether this
    environment has what it names, such as another model to review with,
    subagents, or a command, skill or plugin. Where a required review's
    reviewer is missing, say which level of the fallback under the file's
    `## Reviews` applies: a reviewer on a different model, else the same
    model in a fresh context, else a note in the pull request that the work
    had no independent review.
  - Drift: where the workflow file and the user's instructions already in
    your context (their global agent instructions, memory, this project's
    `AGENTS.md` and `CLAUDE.md`) say different things about the same step,
    such as another reviewer, model or effort. Quote both.
  - Conflicts with the framework: instructions in your context that
    Docket's rules do not allow, such as squash merges (`dkt sync` reads
    each commit's trailers), or rules that open a change or a release where
    the triage would not.
  - superpowers: whether its skills (brainstorming, writing-plans,
    subagent-driven-development) are available; the full review path uses
    them.

  Do not read, list or search user-level files that are not already in your
  context, such as global instruction files or another agent's
  configuration, unless the user agrees first.

  ## 3. List every problem

  Before fixing anything, list every problem from steps 1 and 2, each with
  what it is, why it matters and the fix you propose. When there is none,
  say so and stop.

  ## 4. Fix them one at a time

  Take the problems in the order listed. For each, say what you will do,
  and do it only after the user agrees; when they decline, go to the next.

  - The workflow file (`workflow-missing`, `workflow-incomplete`,
    `workflow-stale`, or a step this environment cannot do): run the
    `docket-setup` skill.
  - Drift: ask which side the user wants to keep. The workflow file changes
    through `docket-setup`; for their own instructions, propose the
    wording.
  - An installed file (`schema-missing`, `skill-missing`,
    `agents-block-missing`, `schema-differs`, `skill-differs`,
    `agents-block-differs`): for a differing file, first show how it differs
    from the shipped one. In a new temporary directory, run `git init`,
    create an empty `openspec` directory, run `dkt init` there, and compare
    the file at the same path. Then run `dkt init` for a missing file, or
    `dkt init --force` for a differing one; `--force` replaces every
    differing file, so name them all before asking.
  - `symlink-refused` or `agents-block-malformed`: explain what stops
    `dkt init` and propose how to fix it; do not change the link or the
    markers yourself.
  - `openspec-missing` or `openspec-unsupported`: say which versions Docket
    supports (the finding names them) and that `DOCKET_OPENSPEC` can name
    the command that runs one. Install nothing.
  - A conflict with the framework: propose the wording for the user's
    instructions; never edit their files.
  - `no-task-commits`: explain that the change's commits on the default
    branch lost their `Change` and `Task` trailers, as a squashed pull
    request does, and that `dkt archive` will refuse the change. Leave the
    decision to the user.
  - An unreadable source, or no superpowers: say what is wrong; the fix is
    the user's.

  ## 5. Check again

  Run `dkt inspect` again and report what is left.
  ````

  `.agents/skills/docket-doctor/agents/openai.yaml`:

  ```yaml
  policy:
    allow_implicit_invocation: false
  ```

- [ ] **Step 5: Regenerate the repository's copies**
  - Run: `go run ./cmd/dkt init --force`.
    - Expected: it reports `overwritten` for `.claude/skills/docket/SKILL.md`, and `written` for the docket-setup and docket-doctor files under `.claude/skills/`. It never asks for OpenSpec, since `openspec/` exists.
  - Run: `git rm .claude/skills/docket/setup.md`.
  - Run: `rg -n "setup\.md" .agents .claude/skills`.
    - Expected: no output.

- [ ] **Step 6: Run the tests to see them pass**

  Run: `go test ./...` and `go vet ./...`.
  Expected: PASS.

- [ ] **Step 7: Commit**

  Commit as `feat(skill): setup and doctor skills, installed by dkt init`, with the trailers. It holds the skill files, the deletions, the embedding, the new tests and the regenerated copies.

### Task 5: Docket's docs

**Scenarios:** none

**Review:** checks

**Files:**
- Modify:
  - `README.md`, `README.zh-TW.md`
  - `docs/ARCHITECTURE.md`
  - `AGENTS.md`
  - `docket/ideas/2026-10-29-framework-skill.md`
  - `docket/order.md`
- Create: `docket/ideas/2026-10-30-process-review.md` (written during design; commit it)

- [ ] **Step 1: Edit the docs**
  - **`AGENTS.md`**, outside the Docket block:
    - the status line lists `dkt inspect` with the other commands;
    - the prompt paths become `.claude/skills/docket/`, `.claude/skills/docket-setup/`, `.claude/skills/docket-doctor/`, the same three under `.agents/skills/`, and the rest as they are;
    - "`dkt init` ships `openspec/schemas/docket/`, the `docket` skill and the Docket block" becomes "... the `docket`, `docket-setup` and `docket-doctor` skills and the Docket block".
  - **`README.md`**:
    - In the paragraph on the `docket` skill, add after the workflow file's sentence: "Setup is its own skill, `docket-setup`, which the user can also invoke to revise the file (`/docket-setup` in Claude Code, `$docket-setup` in Codex). To check a project's setup, the user invokes `docket-doctor`: it runs `dkt inspect`, checks what needs judgment, lists every problem and then fixes them with the user one at a time."
    - In the command list:
      - add `dkt inspect [--json]               # the project's Docket setup; writes nothing`;
      - the `dkt init` comment becomes `# install the docket schema, the skills and the Docket block`.
  - **`README.zh-TW.md`**: the same edits in Traditional Chinese.
  - **`docs/ARCHITECTURE.md`**:
    - **Codemap:**
      - `schema.go` embeds "the `docket`, `docket-setup` and `docket-doctor` skills";
      - `internal/setup` "Installs the schema, the skills and the Docket block (`dkt init`), and decides what it would do without writing (`dkt inspect`)";
      - `internal/archive` adds "the trailer check of `dkt inspect`".
    - **Capability index**, new rows:
      - `| Decide what init would install, without writing | \`setup.Inspect\` |`
      - `| Check the trailers of the changes on the default branch | \`archive.Trailers\` |`
  - **`docket/ideas/2026-10-29-framework-skill.md`**:
    - **Status:** "All three changes are done: `add-framework-skill` (the skill and the block), `add-workflow` (the materialized workflow) and `add-doctor` (`dkt inspect` and the `docket-doctor` command)."
    - Remove the open question on the doctor's checks; `add-doctor`'s design answers it.
  - **`docket/order.md`**: tick line 2.
  - **`docket/ideas/2026-10-30-process-review.md`**: "Not part of `docket doctor`" becomes "Not part of the doctor (`docket-doctor`)".

- [ ] **Step 2: Verify**

  Run: `rg -n "setup\.md|docket doctor" README.md README.zh-TW.md docs AGENTS.md docket/ideas/2026-10-29-framework-skill.md docket/ideas/2026-10-30-process-review.md`.
  Expected: no output.

  Run: `go test . -run 'TestBlock|TestInstalledSkill'`.
  Expected: PASS. The Docket block is unchanged.

- [ ] **Step 3: Commit**

  Commit as `docs: dkt inspect, the setup and doctor skills`, with the trailers.

### Task 6: Trial of the doctor and setup in a scratch project

**Scenarios:** setup-inspection#doctor-not-run-on-its-own, setup-inspection#doctor-invoked, setup-inspection#drift-found, setup-inspection#problems-listed-first, setup-inspection#edited-skill-fixed-with-consent, project-setup#setup-before-the-first-work, project-setup#setup-invoked-by-the-user

**Review:** checks

**Files:**
- Modify, only if the trial calls for new wording: `.agents/skills/docket/SKILL.md`, `.agents/skills/docket-setup/SKILL.md`, `.agents/skills/docket-doctor/SKILL.md`, and their copies under `.claude/skills/`

The lead runs this task.
- Transcripts and a results table go in the scratchpad; the summary goes in the pull request.
- Every Claude Code run:
  - is `claude -p` with `--output-format stream-json --verbose`, `--setting-sources project,local`, and `--plugin-dir` pointing at the installed superpowers;
  - runs on two models, the session's default and Sonnet;
  - has the branch's `dkt`, built into the scratchpad, first on `PATH`;
  - has `XDG_CONFIG_HOME` set to a scratch directory.
- So the user's own instructions and workflow file are never read or written.

- [ ] **Step 1: Set up a scratch project**
  - A new git repository on `main` holding:
    - a small Go program (`main.go` printing `hello`) and a README;
    - a script `review-light.sh` that appends its arguments to a log in the scratchpad and prints `no findings`.
  - Run the branch's `dkt init` and commit.
  - Write `<XDG_CONFIG_HOME>/docket/workflow.md` with the current template's hash and text, with Light review step 2 replaced by "Have the whole diff reviewed once by running `./review-light.sh --effort low`."

- [ ] **Step 2: The doctor does not run on its own**

  Prompts "Add a --name flag to hello." and "What does main.go do?", with `--max-turns 8`.
  Expected: no Skill call to `docket-doctor`, and no Bash command containing `dkt inspect`.

- [ ] **Step 3: The doctor lists every problem first**

  **Setup:**
  - append a line to `.claude/skills/docket/SKILL.md`;
  - set the workflow file's frontmatter to `template: "0123456789ab"`;
  - replace the workflow file's Full review step 5 with "Before merging, have the whole change reviewed once by running `./review-other-model.sh`." That script does not exist.
  - append to the project's `CLAUDE.md`:
    - "For a light review, run `./review-light.sh --effort high`."
    - "Merge pull requests with squash."
  - commit;
  - record checksums of the project's files and of the workflow file.

  **Run:** prompt `/docket-doctor`, with `--max-turns 25` and Read, Glob, Grep, Skill, and Bash for `dkt`, `git`, `mktemp`, `mkdir` and `diff` allowed.

  **Expected:**
  - the first Bash command is `dkt inspect --json`;
  - the last message lists the following, each with its reason and proposed fix:
    - `skill-differs`;
    - `workflow-stale`;
    - the drift, quoting both `--effort low` and `--effort high`;
    - the squash rule, as a conflict with the framework;
    - the missing `./review-other-model.sh`, with the fallback level that applies;
  - every checksum is unchanged.

  **Run once more on Sonnet, without `--plugin-dir`.** Expected: the list also says that superpowers is not available.

- [ ] **Step 4: The fix waits for consent**
  1. Resume Step 3's session with "Start with the edited skill."
     - Expected: the agent shows how `.claude/skills/docket/SKILL.md` differs from the shipped file and asks before `dkt init --force`.
     - The file is unchanged.
  2. Resume with "Yes, go ahead."
     - Expected: it runs `dkt init --force`.
     - The file equals the shipped one.
     - The workflow file is unchanged.

- [ ] **Step 5: Codex**
  1. In the scratch project, run `codex debug prompt-input` (no model call). Expected: the skills listed to the model include `docket` and `docket-setup`, and not `docket-doctor`.
  2. Run `codex exec --model gpt-5.6-sol --sandbox read-only '$docket-doctor'` once, on a copy of Step 3's project made before Step 4.
     - Its first command is `dkt inspect --json`.
     - Its last message lists `skill-differs`, `workflow-stale` and the drift before any fix.
     - Every checksum is unchanged.

- [ ] **Step 6: Setup invoked by the user, and before the first work**
  1. Restore the workflow file to the current hash. Prompt `/docket-setup Change the light review reviewer to ./review-other.sh.`
     - Expected: the whole revised file is shown with the new reviewer.
     - The workflow file is unchanged.
  2. Delete the workflow file. Prompt "Add a --name flag to hello."
     - Expected: the `docket` skill sends the agent to `docket-setup`, and a draft of the four path sections is shown before any other step of the work.
     - No workflow file exists afterwards.

- [ ] **Step 7: Adjust and rerun on failure**

  When a run fails its expectation:
  1. change the wording;
  2. install it with `go run ./cmd/dkt init --force`;
  3. run `go test ./...`;
  4. rerun every prompt.

  Commit any change, with the trailers.
