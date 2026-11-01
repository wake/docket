# Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship a user-invoked `docket-init` command that `dkt init` installs once per machine, have `dkt inspect` report it, and let `dkt inspect` run in a repository without a default branch.

**Architecture:** The command is a skill folder embedded from `user-skills/docket-init/`. `internal/setup` installs it as it installs the per-machine schema: targets whose `base` is an absolute directory (each agent's user skills directory) and whose `rel` starts with `docket-init/`. The CLI resolves those directories from the environment (`setup.UserSkillDirs`, `userdirs.ClaudeConfig`) and passes them in a new `setup.Dirs` value. `dkt inspect` treats a missing default branch as a skipped trailer check.

**Tech Stack:** Go 1.24 standard library, `gopkg.in/yaml.v3` (already a dependency, used in tests).

**Spec:** `openspec/changes/add-docket-init/proposal.md`, `design.md`, `specs/init-command/spec.md`, `specs/project-setup/spec.md`, `specs/setup-inspection/spec.md`.

## Global Constraints

- go.mod stays at `go 1.24`; no new dependency.
- Public repository: English only; no hostnames, personal paths or secrets in code, tests or docs.
- No test and no manual run writes the real `~/.claude`, `~/.agents` or `$CLAUDE_CONFIG_DIR`: tests take their environment from `testutil.Env` (a temporary `HOME`) or set `HOME` explicitly; a manual run sets `HOME`, `XDG_DATA_HOME` and `CLAUDE_CONFIG_DIR` to new temporary directories. Never `go install` a branch build.
- Exact names: skill `docket-init`; source folder `user-skills/docket-init/` holding `SKILL.md` and `agents/openai.yaml`; file kind `user-skill`; findings `user-skill-missing`, `user-skill-differs`; JSON field `result.default_branch`.
- Destinations: `<claude>/skills/docket-init/` and `<home>/.agents/skills/docket-init/`, where `<claude>` is `CLAUDE_CONFIG_DIR` when it is set and not empty, else `<home>/.claude`. A relative `CLAUDE_CONFIG_DIR` is `&userdirs.RelativeError{"CLAUDE_CONFIG_DIR", value}`.
- Messages: `dkt init` `user-skill-differs`: `differs from the shipped skill; use --force to overwrite`. `dkt inspect` `user-skill-missing`: `not installed; run dkt init`; `user-skill-differs`: `differs from the shipped skill; dkt init --force overwrites it`. Inspect's skipped trailer check prints the human line `skipped  trailers: <error text>`.
- Commits: identity `git -c user.name="Claude Code" -c user.email=wake+claude-code@protype.tw`; message through `-F <file>`; the trailers `Change: add-docket-init`, `Task: <N>` and `Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>` together in the final paragraph. A restructuring and a behavior change are separate commits.
- Codex copies and Claude Code copies of a skill are byte-for-byte identical.

## Review Focus

- `CLAUDE_CONFIG_DIR` set to `<home>/.agents`: both destinations are the same directory; the files are decided and listed once, not twice. Pinned in Task 2 (`UserSkillDirs` dedupes).
- A file the user added under a `docket-init` folder, or a skill of their own beside it, is left alone and not reported. Pinned in Task 2.
- A refused project path (a linked `.claude/skills/docket`) also stops the user-level writes: nothing is written anywhere. Pinned in Task 2.
- `dkt inspect --default-branch nope` naming a branch that does not exist is a skipped trailer check, not exit 4. Pinned in Task 3.
- A regular file where the `docket-init` folder should be: `dkt init` refuses and writes nothing. Pinned in Task 2.

## Reuse

| Need | Existing (path:symbol) | Decision | Reason | Evidence |
|---|---|---|---|---|
| Install a file outside the project with Init's statuses | internal/setup/setup.go:target | extend | its `base` already gives the per-machine schema absolute-path targets; user skills are more such targets, decided, written and installed by the same `decide`, `write` and `install` | `rg -n "base" internal/setup/setup.go` |
| List every target | internal/setup/setup.go:targetsOf | extend | add the user skill targets after the schema's | `rg -n "func targetsOf" internal/setup` |
| Pass the per-machine destinations | internal/setup/setup.go:Init | extend | `schemaDir string` becomes `Dirs{Schema, UserSkills}` in `Init` and `Inspect`; a second loose parameter would grow every call | `rg -n 'setup\.(Init\|Inspect)\(' internal --type go` and `rg -n '\b(Init\|Inspect)\(' internal/setup --glob '*_test.go'` (33 call sites, in setup tests and two commands) |
| Refuse links at or below the owned folder | internal/safepath/safepath.go:Check | reuse | already gives the schema's rule when called with `base` | `rg -n "safepath.Check" internal/setup` |
| Report a link with an absolute path | internal/setup/setup.go:problemForTarget | reuse | already rewrites the link path of a target under a `base` | `rg -n "problemForTarget" internal/setup` |
| Report one finding per linked directory | internal/cli/cmd_init.go:problemFindings | reuse | already dedupes by the link's path | `rg -n "seenLink" internal/cli` |
| Home directory | internal/userdirs/userdirs.go:Home | reuse | same rules as for the data directory | `rg -n "func Home" internal/userdirs` |
| Claude Code's configuration directory | internal/userdirs/userdirs.go:DataHome | extend | a `ClaudeConfig` beside it, reading `CLAUDE_CONFIG_DIR` by the same rules; nothing reads that variable yet | `rg -n CLAUDE_CONFIG_DIR internal` (no match) |
| The user skill directories | internal/setup/schema.go:SchemaDir | extend | a `UserSkillDirs` beside it naming the two agents' directories | `rg -n "func SchemaDir" internal/setup` |
| Embed the command | schema.go:Skills | extend | a `UserSkills` beside it with its own embed; adding docket-init to `Skills` would install it into every project | `rg -n "go:embed" schema.go` |
| Check skill frontmatter | schema_test.go:checkSkillFrontmatter | extend | run it on `UserSkills` too; docket-init is invoked only by the user, as docket-doctor is | `rg -n "docket-doctor" schema_test.go` |
| Check the Codex policy | schema_test.go:TestDoctorHiddenFromCodex | extend | cover docket-init | `rg -n "allow_implicit" schema_test.go` |
| Map init statuses to findings | internal/cli/cmd_init.go:initCommand | extend | one more kind in the differs switch | `rg -n "KindSkill" internal/cli` |
| Map inspect statuses to findings | internal/cli/cmd_inspect.go:inspectStatus | extend | one more kind | `rg -n "func inspectStatus" internal/cli` |
| Default branch | internal/scan/defaultbranch.go:ResolveDefaultBranch | reuse | inspect only changes how it reacts to `ErrNoDefaultBranch` | `rg -n "ErrNoDefaultBranch" internal` |
| Trailer check | internal/archive/trailers.go:Trailers | reuse | skipped, not changed | `rg -n "func Trailers" internal/archive` |
| OpenSpec version for the command's check | internal/cli/cmd_inspect.go:openSpecState | reuse | the command reads `result.openspec` from `dkt inspect --json` | `rg -n "openSpecState" internal/cli` |
| Setup steps in the command | .agents/skills/docket-setup/SKILL.md | reuse | the command reads and follows it | `ls .agents/skills` |
| Doctor steps in the command | .agents/skills/docket-doctor/SKILL.md | reuse | the command reads and follows it | `ls .agents/skills` |
| Test environment for init | internal/cli/cmd_init_test.go:runInit | extend | appends `HOME=` + `userHome(dir)`, as it appends `XDG_DATA_HOME` | `rg -n "func runInit" internal/cli` |
| Test environment for inspect | internal/cli/cmd_inspect_test.go:runInspect | extend | the same `HOME` as `runInit` | `rg -n "func runInspect" internal/cli` |

---

### Task 1: The docket-init skill

**Scenarios:** init-command#not-run-on-its-own, init-command#invoked, init-command#environment-first, init-command#missing-openspec, init-command#missing-superpowers, init-command#not-a-git-repository, init-command#new-project, init-command#differing-files, init-command#files-to-commit

**Review:** judgment

**Files:**
- Create: `user-skills/docket-init/SKILL.md`
- Create: `user-skills/docket-init/agents/openai.yaml`
- Modify: `schema.go`
- Modify: `schema_test.go`

**Interfaces:**
- Produces: `docket.UserSkills() []setup.Skill`, returning exactly one `setup.Skill{Name: "docket-init", Files: <user-skills/docket-init>}`.

- [ ] **Step 1: Write the failing tests** in `schema_test.go`:
  - `TestUserSkillNames`: the names of `UserSkills()` equal `[]string{"docket-init"}`.
  - `TestSkillFrontmatter` also runs `checkSkillFrontmatter` for each of `UserSkills()`; `checkSkillFrontmatter` requires `disable-model-invocation: true` for `docket-doctor` and `docket-init`, and its absence for the others.
  - `TestDoctorHiddenFromCodex` (renamed `TestInvokedOnlyHiddenFromCodex`) covers `Skills()` and `UserSkills()`: `docket-doctor` and `docket-init` have `agents/openai.yaml` with `policy.allow_implicit_invocation: false`; no other skill has the file.
- [ ] **Step 2: Run** `go test -run 'TestUserSkillNames|TestSkillFrontmatter|TestInvokedOnlyHiddenFromCodex' .` — expect FAIL (`UserSkills` undefined).
- [ ] **Step 3: Implement `UserSkills()`** in `schema.go` with its own `//go:embed user-skills/docket-init` variable, and write `agents/openai.yaml` as docket-doctor's (`policy:` / `  allow_implicit_invocation: false`).
- [ ] **Step 4: Write `user-skills/docket-init/SKILL.md`** in the style of `.agents/skills/docket-doctor/SKILL.md` (numbered `##` steps, short imperative sentences). Frontmatter:

  ```yaml
  name: docket-init
  description: >-
    Set this project up for Docket: check that git, dkt, the OpenSpec CLI
    and superpowers are in place, run dkt init, write the user's workflow
    file when it is missing, check the setup, and list the files to
    commit. Runs only when the user invokes it.
  disable-model-invocation: true
  ```

  The body must say, in this order:
  1. *Check the environment, writing nothing.* Run every check below and collect what is missing before saying anything; a check that cannot run because an earlier one failed is listed as not checked yet, with the reason. Then, when anything is missing, report all of it together, with how to install each, and stop. Install nothing.
     - git: `git rev-parse --show-toplevel` fails → open the project's repository or create one with `git init`.
     - `dkt`: it cannot be started → install it (the README's `go install github.com/wake/docket/cmd/dkt@latest`).
     - OpenSpec, only inside a work tree: `dkt inspect --json`. `openspec-missing` or `openspec-unsupported` → install OpenSpec 1.14.x (`npm install -g @fission-ai/openspec@1.14`) or set `DOCKET_OPENSPEC` to a command that runs it, such as `npx -y @fission-ai/openspec@1.14.3`. Another exit with 2 or 4 → report what it said.
     - superpowers: check the agent's own skill list for brainstorming, writing-plans and subagent-driven-development (Claude Code lists them as `superpowers:brainstorming` and so on); missing → Claude Code `/plugin install superpowers@claude-plugins-official`, Codex `/plugins` then search for superpowers, other agents https://github.com/obra/superpowers; then start a new session.
  2. *Run `dkt init --json`.* It writes the project's files, the per-machine schema and this command's user-level copies. For any finding it reports, do what `docket-doctor` says for that finding (read its `SKILL.md`, see step 4): show how a differing file differs and run `dkt init --force` only after the user agrees. When it exits with 2, 3 or 4, tell the user what it said and stop.
  3. *The workflow file.* Run `dkt workflow --json`. On `missing` or `incomplete`, read `docket-setup/SKILL.md` from this repository's `.claude/skills/` (Claude Code) or `.agents/skills/` (Codex) and follow it; say that skills `dkt init` added during this session may not be loaded yet, which is why it is read from the file. On `stale`, say so and offer setup. On `ok`, go on.
  4. *Check the setup.* Read `docket-doctor/SKILL.md` the same way and follow it.
  5. *Report.* List what `git status --short` shows as created or changed by the setup, for the user to commit (`openspec/`, `.claude/skills/`, `.agents/skills/`, `AGENTS.md`, `CLAUDE.md`); say that the workflow file, the per-machine schema and the `docket-init` copies are outside the repository. Do not commit. In a repository with no commit, say that the first commit gives it the default branch `dkt new` needs. In Claude Code, when `.claude/skills/` did not exist before this run, say to type `/reload-skills`; in Codex, to restart when `$docket` does not show. End by saying development or a refactoring can start: the user describes the work, and the `docket` skill triages it.

  The intro says the command writes only through `dkt init`, `docket-setup` and `docket-doctor`, each with the consent it asks for, and never installs or commits.
- [ ] **Step 5: Run** `go test .` — expect PASS.
- [ ] **Step 6: Commit** `schema.go`, `schema_test.go`, `user-skills/docket-init/` with the trailers (`Task: 1`).

### Task 2: setup installs the user-level skills

**Scenarios:** project-setup#fresh-machine-gets-the-command, project-setup#claude-config-dir, project-setup#edited-command-copy, project-setup#copies-agree, project-setup#other-configuration-unchanged, project-setup#symlinked-schema-file-refused, project-setup#symlinked-skill-directory-refused, project-setup#symlinked-command-folder-refused, project-setup#user-skills-directory-linked, project-setup#personal-skills-unchanged

**Review:** judgment

**Files:**
- Modify: `internal/setup/setup.go`
- Modify: `internal/setup/schema.go`
- Modify: `internal/setup/setup_test.go`, `internal/setup/create_test.go`, `internal/setup/rewrite_test.go`, `internal/setup/schema_test.go`
- Modify: `internal/userdirs/userdirs.go`, `internal/userdirs/userdirs_test.go`
- Modify: `internal/cli/cmd_init.go`, `internal/cli/cmd_inspect.go` (call sites only)

**Interfaces:**
- Produces: `type Dirs struct { Schema string; UserSkills []string }` — an empty `Schema` leaves the per-machine schema out, an empty `UserSkills` leaves the user-level skills out.
- Produces: `func Init(root string, dirs Dirs, a Assets, force bool, initOpenSpec func() error) (Result, error)` and `func Inspect(root string, dirs Dirs, a Assets) (Report, error)`.
- Produces: `Assets.UserSkills []Skill` (each installed under `<dir>/<name>` for each of `Dirs.UserSkills`); `KindUserSkill FileKind = "user-skill"`.
- Produces: `func userdirs.ClaudeConfig(env []string, userHome func() (string, error)) (string, error)`; `func setup.UserSkillDirs(env []string, userHome func() (string, error)) ([]string, error)` returning `<claude>/skills` then `<home>/.agents/skills`, with a duplicate (after `filepath.Clean`) dropped.

- [ ] **Step 1: Restructure (no behavior change).** Replace `schemaDir string` in `Init` and `Inspect` with `dirs Dirs` holding only `Schema`; update every call: the tests use a helper `dirs(root) Dirs { return Dirs{Schema: mdir(root)} }`, `cmd_init.go` and `cmd_inspect.go` pass `setup.Dirs{Schema: schemaDir}`. Run `go test ./...` — expect PASS. Commit (`Task: 2`, message `refactor(setup): pass destinations as Dirs`).
- [ ] **Step 2: Write the failing tests.**
  - `internal/userdirs`: `TestClaudeConfig` table: `HOME=/h` → `/h/.claude`; `HOME=/h` `CLAUDE_CONFIG_DIR=/c` → `/c`; `CLAUDE_CONFIG_DIR=` → `/h/.claude`; `CLAUDE_CONFIG_DIR=c` → `&RelativeError{"CLAUDE_CONFIG_DIR", "c"}`; only `CLAUDE_CONFIG_DIR=/c` → `/c`; empty env → `ErrNoHome`.
  - `internal/setup`: `TestUserSkillDirs`: `HOME=/h` → `[/h/.claude/skills /h/.agents/skills]`; with `CLAUDE_CONFIG_DIR=/c` → `[/c/skills /h/.agents/skills]`; with `CLAUDE_CONFIG_DIR=/h/.agents` → `[/h/.agents/skills]`; only `CLAUDE_CONFIG_DIR=/c` → `ErrNoHome`.
  - `internal/setup`: add `UserSkills: []Skill{{"docket-init", MapFS{"SKILL.md": "# init\n", "agents/openai.yaml": "policy: {}\n"}}}` to the test `assets`, and a helper `udirs(root) []string` returning `root+".claude/skills"` and `root+".agents/skills"`. Tests, each with `Dirs{Schema: mdir(root), UserSkills: udirs(root)}`:
    - `TestInitUserSkillsFresh`: both files under both directories are `Written` at their absolute paths with kind `KindUserSkill`, and hold the shipped bytes; the two folders are identical.
    - `TestInitUserSkillDiffers`: an edited `<udirs[1]>/docket-init/SKILL.md` is `Differs` and unchanged without force, `Overwritten` with the shipped bytes with force.
    - `TestInitUserSkillFolderLinked`: `<udirs[0]>/docket-init` a link to another directory; `Init(force)` returns `*Refused` with a `ProblemSymlink` whose error names `<udirs[0]>/docket-init`; nothing is written under the link target, under `<udirs[1]>`, or in the repository.
    - `TestInitUserSkillsDirLinked`: `<udirs[0]>` itself a link to another directory; the files are written under its target and Init succeeds.
    - `TestInitUserSkillsLeaveOthers`: `<udirs[0]>/docket-init/notes.md`, `<udirs[0]>/mine/SKILL.md` and `root+".claude/settings.json"` exist; all three are byte-for-byte unchanged and in no `Result.Files` entry.
    - `TestInitProjectRefusalWritesNoUserSkill`: `.claude/skills/docket` is a link; Init refuses and neither `<udirs[i]>/docket-init` exists.
    - `TestInitUserSkillFolderIsFile`: `<udirs[0]>/docket-init` is a regular file; Init returns `*Refused` and writes nothing.
    - `TestInspectUserSkills`: before Init, `Inspect` lists the four files with kind `KindUserSkill` and status `Written`, and creates neither directory.
    - `TestInitNoUserSkillDirs`: with `Dirs{Schema: mdir(root)}`, no result file has kind `KindUserSkill`.
- [ ] **Step 3: Run** `go test ./internal/setup ./internal/userdirs` — expect FAIL (`ClaudeConfig`, `UserSkillDirs`, `Dirs.UserSkills`, `KindUserSkill` undefined).
- [ ] **Step 4: Implement.** `ClaudeConfig` mirrors `DataHome`. `UserSkillDirs` goes in `internal/setup/schema.go` beside `SchemaDir`. In `targetsOf`, after the schema, add for each directory in `dirs.UserSkills` and each skill in `a.UserSkills` the skill's files with `rel` `path.Join(sk.Name, p)`, kind `KindUserSkill`, `base` the directory. Nothing else in `decide`, `write` or `problemForTarget` should need to change; if it does, say why in the report.
- [ ] **Step 5: Run** `go test ./...` — expect PASS.
- [ ] **Step 6: Commit** (`Task: 2`, message `feat(setup): install the user-level docket-init skill`).

### Task 3: dkt inspect without a default branch

**Scenarios:** setup-inspection#repository-without-commits, setup-inspection#default-branch-found

**Review:** judgment

**Files:**
- Modify: `internal/cli/cmd_inspect.go`
- Modify: `internal/cli/cmd_inspect_test.go`

**Interfaces:**
- Produces: `InspectResult.DefaultBranch *string` with JSON key `default_branch`.

- [ ] **Step 1: Write the failing tests** in `cmd_inspect_test.go`:
  - `TestInspectNoDefaultBranch` (replaces the test of that name that expects exit 4; the branch `work` repository stays): exit is 1, `result.default_branch` is nil, `result.changes` is `[]`, no finding has rule `environment`, and the text output (`runInspect` without `--json`) has the line `skipped  trailers: no default branch found`.
  - `TestInspectRepositoryWithoutCommits`: `testutil.NewRepo` with no commit; exit is not 4; `default_branch` nil; `changes` `[]`; `result.openspec` and `result.files` present.
  - `TestInspectDefaultBranchFound`: `committedRepo` → `default_branch` is `"main"`.
  - `TestInspectDefaultBranchFlag` also asserts `default_branch` is `"trunk"`, and that `--default-branch nope` gives `default_branch` nil and an exit other than 4.
  - `TestInspectCleanSetup`'s expected keys become `changes default_branch files openspec workflow`.
- [ ] **Step 2: Run** `go test ./internal/cli -run 'TestInspect'` — expect FAIL.
- [ ] **Step 3: Implement.** In `inspectCommand`, call `scan.ResolveDefaultBranch(c.Git, c.DefaultBranch)`; on `scan.ErrNoDefaultBranch`, skip `archive.Trailers` and add the human line; otherwise set `res.DefaultBranch` to `db.Name`. Other errors still return.
- [ ] **Step 4: Run** `go test ./internal/cli` — expect PASS.
- [ ] **Step 5: Commit** (`Task: 3`, message `feat(inspect): run without a default branch`).

### Task 4: dkt init and dkt inspect report the command

**Scenarios:** project-setup#fresh-machine-gets-the-command, project-setup#claude-config-dir, project-setup#edited-command-copy, project-setup#symlinked-command-folder-refused, setup-inspection#nothing-written, setup-inspection#clean-setup, setup-inspection#before-init, setup-inspection#edited-skill, setup-inspection#machine-copy-differs, setup-inspection#command-not-installed, setup-inspection#claude-md-imports-agents-md, setup-inspection#no-home-directory

**Review:** judgment

**Files:**
- Modify: `internal/cli/cmd_init.go`, `internal/cli/cmd_inspect.go`
- Modify: `internal/cli/cmd_init_test.go`, `internal/cli/cmd_inspect_test.go`, and any other `internal/cli/*_test.go` whose expected output lists every file `dkt init` writes

**Interfaces:**
- Consumes: `setup.UserSkillDirs`, `setup.Dirs`, `setup.KindUserSkill`, `docket.UserSkills()` (Tasks 1, 2).

- [ ] **Step 1: Give init and inspect one home in tests.** Add `func userHome(dir string) string { return dir + ".home" }` beside `dataHome`, and have `runInit` and `runInspect` append `HOME=`+`userHome(dir)`. Run `go test ./internal/cli` — expect PASS (nothing writes there yet).
- [ ] **Step 2: Write the failing tests.**
  - `TestInitUserSkill`: fresh run; the text output has `written` lines for `userHome/.claude/skills/docket-init/SKILL.md`, `.../agents/openai.yaml` and the same under `userHome/.agents/skills/docket-init/`; `--json` lists them with those absolute paths; exit 0.
  - `TestInitClaudeConfigDir`: `CLAUDE_CONFIG_DIR=<temp>/claude`; the copy is under `<temp>/claude/skills/docket-init/`, `userHome/.claude` does not exist, the Codex copy is under `userHome/.agents/skills/docket-init/`.
  - `TestInitRelativeClaudeConfigDir`: `CLAUDE_CONFIG_DIR=claude` → exit 4, the message contains `CLAUDE_CONFIG_DIR is not an absolute path`, and nothing was written: no `AGENTS.md` in the repository, no per-machine schema directory, no `userHome/.agents`.
  - `TestInspectRelativeClaudeConfigDir`: `CLAUDE_CONFIG_DIR=claude` → exit 1 (not 4), a `user-skill-missing` finding whose message contains `CLAUDE_CONFIG_DIR is not an absolute path`, and the other files still listed.
  - `TestInspectUserSkillDiffers`: after init, edit `userHome/.claude/skills/docket-init/SKILL.md`; `result.files` lists that absolute path as `differs` with kind `user-skill`; the finding is `user-skill-differs` with message `differs from the shipped skill; dkt init --force overwrites it`.
  - `TestInspectSymlinkedCommandFolder`: `userHome/.agents/skills/docket-init` a link to another directory → exit 1, one `symlink-refused` finding whose `file` is that absolute path, no `unreadable` entry for it, and the link target unchanged.
  - `TestInitUserSkillDiffers`: an edited `userHome/.agents/skills/docket-init/SKILL.md` → exit 1, finding `user-skill-differs` with that absolute `file` and the message from Global Constraints; `--force` → listed `overwritten`, exit 0.
  - `TestInitSymlinkedCommandFolder`: `userHome/.claude/skills/docket-init` a link → exit 1, one `symlink-refused` finding whose `file` is that absolute path, nothing written.
  - `TestInspectCommandNotInstalled`: after init, remove `userHome/.agents/skills/docket-init`; `result.files` lists its two files as `missing` with kind `user-skill`; findings hold `user-skill-missing` for each with message `not installed; run dkt init`.
  - `TestInspectNoHomeForSchema` also expects `user-skill-missing` with the `ErrNoHome` text and no `file`.
  - `TestInspectNothingWritten`: after init, remove both user-level `docket-init` folders; snapshot `userHome(r.Dir)` before and after inspect as the other directories are; `user-skill-missing` is among the rules.
  - `TestInspectCleanSetup` still reports no finding.
- [ ] **Step 3: Run** `go test ./internal/cli` — expect FAIL.
- [ ] **Step 4: Implement.** `shippedAssets` adds `UserSkills: docket.UserSkills()`. `dkt init` resolves `setup.UserSkillDirs` after `SchemaDir`, an error being an `EnvError` as there, and maps a differing `KindUserSkill` file to `user-skill-differs`. `dkt inspect` adds a `user-skill-missing` finding with the error's text when `UserSkillDirs` fails, and `inspectStatus` names kind `KindUserSkill` `user-skill` with "the shipped skill". Update expected file lists in existing tests that now include the four files.
- [ ] **Step 5: Run** `go vet ./... && go test ./...` — expect PASS.
- [ ] **Step 6: Commit** (`Task: 4`, message `feat(init): install and inspect the docket-init command`).

### Task 5: The doctor, docs and prompt paths

**Scenarios:** none

**Review:** checks

**Files:**
- Modify: `.agents/skills/docket-doctor/SKILL.md`, `.claude/skills/docket-doctor/SKILL.md`
- Modify: `README.md`, `README.zh-TW.md`
- Modify: `AGENTS.md` (Development section only, outside the Docket block)
- Modify: `docs/ARCHITECTURE.md`

- [ ] **Step 1: Doctor** (both copies, identical): step 1 lists `default_branch` (the branch whose trailers were checked; null when none was found, as in a repository with no commit, and `changes` is then empty); step 4's installed-file problems add `user-skill-missing` and `user-skill-differs`, and say the `docket-init` copies, like the schema's files, are per-machine and named by absolute path; the comparison run sets `HOME`, `XDG_DATA_HOME` and `CLAUDE_CONFIG_DIR` to new temporary directories, so nothing on the machine is written, and compares per-machine files under them.
- [ ] **Step 2: README (both languages):** the paragraph on skills says that in any project the user can type `/docket-init` (`$docket-init` in Codex) to check the environment and set the project up; the files-written sentence adds the `docket-init` command; the per-machine paragraph says `dkt init` also writes `~/.claude/skills/docket-init/` (under `CLAUDE_CONFIG_DIR` when set) and `~/.agents/skills/docket-init/`.
- [ ] **Step 3: AGENTS.md:** the prompt paths add `user-skills/docket-init/`, and the sentence on what `dkt init` ships adds that it installs `docket-init` at user level.
- [ ] **Step 4: ARCHITECTURE.md:** the `schema.go`, `internal/setup` and `internal/userdirs` rows mention the command and `CLAUDE_CONFIG_DIR`; the capability table adds `setup.UserSkillDirs` and `userdirs.ClaudeConfig`.
- [ ] **Step 5: Verify.** `diff .agents/skills/docket-doctor/SKILL.md .claude/skills/docket-doctor/SKILL.md` prints nothing; `go test .` passes (`TestInstalledSkillMatchesShipped`); `rg -n "user-skill-missing|CLAUDE_CONFIG_DIR" .agents/skills/docket-doctor/SKILL.md` finds both; each edited file reads through. Commit (`Task: 5`, message `docs: the docket-init command`).
