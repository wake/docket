# Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** dkt keeps an index of the Docket projects on a machine: `dkt projects add|ls|rm|ignore` edits and lists it, and every other command adds the project it ran in when it is missing.

**Architecture:** `internal/projects` holds the two file locations, the index and settings files, the project path of a directory, and the registration step; it reads the environment from a `[]string`, as `cli.Options.Env` is. `internal/cli/cmd_projects.go` is the command, which needs no repository. `runCommand` calls the registration step after any other command has run with a repository identified, and turns its error into one warning line on standard error.

**Tech Stack:** Go 1.24, standard library only.

**Spec:** `openspec/changes/projects-index/proposal.md`, `design.md`, `specs/project-index/spec.md`, `specs/cli-output/spec.md`.

## Global Constraints

- Index: `$XDG_STATE_HOME/docket/projects.json` when `XDG_STATE_HOME` is absolute, else `$HOME/.local/state/docket/projects.json`. Settings: `$XDG_CONFIG_HOME/docket/config.json` when absolute, else `$HOME/.config/docket/config.json`.
- Index file: `{"projects": [{"path", "origin", "added", "ignored"}]}`, `origin` a string or null, `added` RFC 3339 UTC to the second; written sorted by path, indented two spaces, ending in a newline, mode `0600`, folders `0755`, through `safefile.Install`.
- Settings: only `projects.autoRegister` (boolean, default true). dkt never writes the settings.
- Rules: `not-a-repository`, `not-a-project`, `not-in-index` (exit 1); `no-home` (exit 4); an unreadable index is listed in `unreadable` (exit 3), and never written over.
- Results: `add`, `rm`, `ignore` give `{"state", "project"}`; states `added`, `exists`, `unignored`, `removed`, `ignored`. `ls` gives `{"projects": [...]}`, each entry with `path`, `origin`, `added`, `ignored`, `missing`; never null.
- The envelope's `command` is `projects`. Text output: `<state> <path>` for `add`, `rm`, `ignore`; for `ls` one line per entry, the path then `missing` and `ignored` when they apply, joined by two spaces.
- Registration warning: one line `dkt: project index: <reason naming the file>` on standard error; stdout and the exit code stay as without it.
- No new dependency; no network; OpenSpec is not run. Commits carry `Change: projects-index/<N>` in the last paragraph, with no other trailer.
- `gofmt -l .` prints nothing, `go vet ./...` and `go test ./...` pass after every task.

## Review Focus

- An environment with neither `HOME` nor the XDG variable, which a test or a CI job can give: `dkt projects` exits 4 with `no-home`, any other command warns and keeps its exit code. Pinned in Task 3 (`TestProjectsNoHome`) and Task 5 (`TestRegisterNoHome`).
- A link at the index file: unreadable, not followed and not written over. Pinned in Task 1 (`TestLoadLink`).
- A path argument written with a trailing slash or `..`: it names the same entry. Pinned in Task 4 (`TestProjectsRmGone`).
- Running in a linked worktree whose main worktree is registered: nothing is written. Pinned in Task 5 (`TestRegisterFromLinkedWorktree`).
- macOS temporary folders under `/var`, which git reports as `/private/var`: tests compare against `filepath.EvalSymlinks` of the folder, as the entries hold what git reports.

## Reuse

| Need | Existing (path:symbol) | Decision | Reason | Evidence |
|---|---|---|---|---|
| Root of the worktree holding a directory | internal/gitx/git.go:Git.Toplevel | reuse | reports `ErrNotRepo` and `ErrGitMissing` as `identify` relies on | `rg -n "func \(g Git\) Toplevel" internal/gitx/` |
| Main worktree | internal/gitx/worktree.go:Git.Worktrees | reuse | lists the main worktree first | `rg -n "func \(g Git\) Worktrees" internal/gitx/` |
| Origin URL | internal/gitx/git.go:Git.RemoteURL | reuse | what `repo.origin` uses | `rg -n "RemoteURL" internal/` |
| Atomic replace of a file | internal/safefile/safefile.go:Install | reuse | a failed write leaves the old file | `rg -n "func Install" internal/safefile/` |
| Read without following a link | internal/safefile/safefile.go:ReadNoFollow | reuse | a link at the index is refused | `rg -n "func ReadNoFollow" internal/safefile/` |
| A value from an environment list | internal/openspec/runner.go:envValue | new | unexported, and importing `openspec` for a lookup would tie the index to the OpenSpec runner; the same seven lines, last value wins | `rg -n "func envValue" internal/` |
| Result, findings, unreadable, exit codes | internal/cli/envelope.go:Outcome, Finding, Unreadable, UsageError, EnvError | reuse | every command's shape | `rg -n "type Outcome" internal/cli/` |
| Identifying the repository without failing | internal/cli/run.go:identify | reuse | as `bestEffortRepo` does for `--version` | `rg -n "func bestEffortRepo" internal/cli/` |
| A command that needs no repository | internal/cli/command.go:Command | extend | a field `NoRepo` | `rg -n "type Command struct" internal/cli/` |
| Registering the command | internal/cli/command.go:commands | extend | one more entry | `rg -n "var commands" internal/cli/` |
| Clock | internal/cli/command.go:Context.Now | reuse | tests fix it through `Options.Now` | `rg -n "Now " internal/cli/run.go` |
| Test repositories and environment | internal/testutil/repo.go:NewRepo, Env | reuse | each test gets its own `HOME` | `rg -n "func Env" internal/testutil/` |
| Init in a test | internal/cli/cmd_init_test.go:doInit | reuse | runs `dkt init` with a fake OpenSpec | `rg -n "func doInit" internal/cli/` |
| The index and settings | none | new | dkt keeps no file outside the repository today | `rg -n "XDG\|\.local/state" internal/` |

---

### Task 1: The index and settings files

**Scenarios:** project-index#index-under-the-state-home, project-index#index-without-a-state-home, project-index#relative-state-home

**Review:** checks

**Files:**
- Create: `internal/projects/files.go`
- Create: `internal/projects/files_test.go`

**Interfaces:**
- Produces, in package `projects`:
  - `var ErrNoHome = errors.New(...)`: neither the XDG variable nor `HOME` is usable.
  - `func IndexPath(env []string) (string, error)`, `func SettingsPath(env []string) (string, error)`: an XDG variable counts only when absolute; `HOME` is used as given.
  - `type Entry struct { Path string \`json:"path"\`; Origin *string \`json:"origin"\`; Added string \`json:"added"\`; Ignored bool \`json:"ignored"\` }`
  - `type Index struct { Projects []Entry \`json:"projects"\` }`
  - `func Load(path string) (*Index, error)`: a missing file gives an empty, non-nil `Projects`; any other failure is an error whose message names `path`.
  - `func (x *Index) Find(path string) int`: the entry's position, or -1.
  - `func (x *Index) Save(path string) error`: sorts by path and replaces the file.
  - `func AutoRegister(settings string) (bool, error)`: true when the file or the key is missing.

- [ ] **Step 1: Write the failing tests**

In `internal/projects/files_test.go`:

- `TestIndexPath`: table over env lists. `XDG_STATE_HOME=/s` gives `/s/docket/projects.json`; `HOME=/h` only gives `/h/.local/state/docket/projects.json`; `XDG_STATE_HOME=state` with `HOME=/h` gives the `HOME` one; `XDG_STATE_HOME=/s` without `HOME` gives `/s/...`; an empty list gives `errors.Is(err, ErrNoHome)`. A later duplicate key wins.
- `TestSettingsPath`: the same table with `XDG_CONFIG_HOME`, `.config` and `config.json`.
- `TestLoadMissing`: `Load` of a path that does not exist returns `Projects` of length 0, not nil, and no error.
- `TestLoadInvalid`: each of `not json`, `{}`, `{"projects":"x"}`, an entry with `"path":"rel"`, an entry with `"added":"yesterday"`, an entry with `"ignored":"no"` gives an error containing the file's path.
- `TestLoadLink`: the index is a symbolic link to a valid index; `Load` returns an error.
- `TestSaveRoundTrip`: `Save` to `<tmp>/a/b/projects.json` creates the folders; entries given as `/w/b`, `/w/a` are written as `/w/a`, `/w/b`; the file ends with `"\n"`, is indented two spaces, and `mode & 0o077 == 0`; `Load` gives back equal entries, with a null origin kept null.
- `TestAutoRegister`: a missing file, `{}` and `{"projects":{}}` give true; `{"projects":{"autoRegister":false}}` gives false; `{` and `{"projects":{"autoRegister":"no"}}` give an error containing the path.

- [ ] **Step 2: Run them and see them fail**

Run: `go test ./internal/projects/`
Expected: FAIL, the package does not build.

- [ ] **Step 3: Implement `internal/projects/files.go`**

`Load` reads with `safefile.ReadNoFollow`, decodes, and checks the form: `projects` present and an array, each `path` absolute, each `added` parsed by `time.Parse(time.RFC3339, ...)`. `Save` sorts by path, `os.MkdirAll(dir, 0o755)`, `json.MarshalIndent(x, "", "  ")` plus a newline, then `safefile.Install(path, data, 0o600, false, path)`.

- [ ] **Step 4: Run them and see them pass**

Run: `go test ./internal/projects/ && go vet ./... && gofmt -l .`
Expected: PASS, nothing printed by `gofmt`.

- [ ] **Step 5: Commit**

`git add internal/projects/` and commit "Read and write the project index and its settings" with the trailer `Change: projects-index/1`.

---

### Task 2: The project of a directory

**Scenarios:** project-index#linked-worktree, project-index#subdirectory

**Review:** checks

**Files:**
- Create: `internal/projects/locate.go`
- Create: `internal/projects/locate_test.go`

**Interfaces:**
- Consumes: `gitx.Git{Dir, Env}`, `Toplevel`, `Worktrees`, `RemoteURL`, `gitx.ErrNotRepo`, `gitx.ErrGitMissing`.
- Produces, in package `projects`:
  - `var ErrNotRepository = errors.New(...)`
  - `type Project struct { Path, Worktree string; Origin *string; Docket bool }`: `Path` is the main worktree, `Worktree` the root of the worktree holding the directory, `Docket` whether that worktree has a regular `docket/workflow.md`.
  - `func Locate(dir string, env []string) (Project, error)`: a directory that does not exist, is not a directory, or is not in a work tree gives an error that `errors.Is` `ErrNotRepository`; `gitx.ErrGitMissing` passes through.
  - `func IsDocket(worktree string) bool`: `os.Lstat` of `docket/workflow.md` is a regular file.
  - `func MainWorktree(g gitx.Git) (string, error)`: the first entry of `g.Worktrees()`.

- [ ] **Step 1: Write the failing tests**

In `internal/projects/locate_test.go`, with real repositories from `testutil.NewRepo` and paths compared after `filepath.EvalSymlinks`:

- `TestLocateSubdirectory`: a repository committing `docket/workflow.md` and `internal/x.go`; `Locate(<repo>/internal)` gives `Path` and `Worktree` the repository, `Docket` true, `Origin` nil.
- `TestLocateLinkedWorktree`: `git worktree add <tmp>/wt -b b` in that repository; `Locate(<tmp>/wt)` gives `Path` the main repository and `Worktree` `<tmp>/wt`.
- `TestLocateOrigin`: after `git remote add origin https://example.invalid/x.git`, `*Origin` is that URL.
- `TestLocateNotRepository`: a plain temporary folder, a path that does not exist, and a regular file each give `errors.Is(err, ErrNotRepository)`.
- `TestIsDocket`: false with no file, with `docket/workflow.md` as a folder, and as a link to a regular file; true for a regular file.

- [ ] **Step 2: Run them and see them fail**

Run: `go test ./internal/projects/ -run 'Locate|IsDocket'`
Expected: FAIL, `Locate` is not defined.

- [ ] **Step 3: Implement `internal/projects/locate.go`**

`Locate` stats `dir` first, then `Toplevel`, `MainWorktree` and `RemoteURL("origin")` on a `gitx.Git` whose `Dir` is the top level.

- [ ] **Step 4: Run them and see them pass**

Run: `go test ./internal/projects/ && go vet ./... && gofmt -l .`
Expected: PASS.

- [ ] **Step 5: Commit**

Commit "Find the project of a directory" with the trailer `Change: projects-index/2`.

---

### Task 3: `dkt projects add` and `ls`

**Scenarios:** project-index#index-under-the-state-home, project-index#index-that-is-not-json, project-index#new-project, project-index#project-already-in-the-index, project-index#ignored-project-added, project-index#repository-without-docket, project-index#not-a-repository, project-index#project-on-an-unmounted-disk, project-index#ignored-entries-listed, project-index#empty-index, project-index#unknown-subcommand, project-index#list-outside-a-repository, project-index#no-home

**Review:** judgment

**Files:**
- Create: `internal/cli/cmd_projects.go`
- Create: `internal/cli/cmd_projects_test.go`
- Modify: `internal/cli/command.go` (the field `NoRepo`, the command list)
- Modify: `internal/cli/run.go` (`runCommand` with `NoRepo`)
- Modify: `README.md` (the command table)

**Interfaces:**
- Consumes: Task 1 and Task 2.
- Produces, in package `cli`:
  - `Command.NoRepo bool`: `runCommand` identifies the repository as `bestEffortRepo` does and runs the command with `Repo` nil when there is none.
  - `func projectsCommand() Command`: `Name: "projects"`, `Args: -1`, `Usage: "projects <add [<path>]|ls|rm <path>|ignore <path>> [--json]"`, `NoRepo: true`.
  - `type projectState struct { State string \`json:"state"\`; Project projects.Entry \`json:"project"\` }`
  - `type projectList struct { Projects []listedProject \`json:"projects"\` }`, `type listedProject struct { projects.Entry; Missing bool \`json:"missing"\` }`
  - `func projectsEnv(c *Context) (index string, err error)`: `IndexPath(c.Env)`, with `ErrNoHome` as `EnvError{Rule: "no-home"}`.
  - `func absArg(c *Context, arg string) string`: `arg` made absolute against `c.Dir` and cleaned.

- [ ] **Step 1: Write the failing tests**

In `internal/cli/cmd_projects_test.go`, with a helper `prRun(t, env []string, dir string, args ...string) (code int, out projectsOut, stdout, stderr string)` that runs `projects <args> --json` with `Options.Now` fixed at `2026-11-08T09:10:11Z`, and a helper writing a Docket repository (`docket/workflow.md` committed):

- `TestProjectsAddNew`: exit 0, `state` `added`, `project.path` the repository, `ignored` false, `added` `2026-11-08T09:10:11Z`; the index under `HOME/.local/state/docket/` holds that entry.
- `TestProjectsAddUnderStateHome`: with `XDG_STATE_HOME=<tmp>/s`, the entry is in `<tmp>/s/docket/projects.json`.
- `TestProjectsAddExists`: a second `add` gives `exists`, and the file's bytes and modification time (set to an hour ago with `os.Chtimes` before) are unchanged.
- `TestProjectsAddUnignores`: an index written by the test with the entry ignored and `added` `2026-01-01T00:00:00Z` gives `unignored`, `ignored` false, `added` unchanged.
- `TestProjectsAddNotProject`: a repository without `docket/workflow.md`: exit 1, `not-a-project`, no index file.
- `TestProjectsAddNotRepository`: `add <plain folder>`: exit 1, `not-a-repository`.
- `TestProjectsUnreadableIndex`: the index holds `not json`; `add` exits 3, `unreadable[0].source` is the index path, the file still holds `not json`.
- `TestProjectsLs`: an index with an ignored entry for an existing folder and an entry for a path that does not exist: both listed, sorted by path, `missing` true only on the second, `ignored` true only on the first; the index bytes are unchanged; the text output is exactly those two lines in the Global Constraints form.
- `TestProjectsLsEmpty`: no index: stdout holds `"projects":[]`, exit 0, no file created.
- `TestProjectsLsOutsideRepo`: run in a plain folder: `repo` is null, exit 0.
- `TestProjectsUsage`: `projects`, `projects prune`, `projects ls x`, `projects add a b`, `projects rm`: each exits 2 with the finding `usage`.
- `TestProjectsNoHome`: an environment with only `PATH`: `ls` exits 4 with `no-home`.

- [ ] **Step 2: Run them and see them fail**

Run: `go test ./internal/cli/ -run TestProjects`
Expected: FAIL, `projectsCommand` is not defined.

- [ ] **Step 3: Add `Command.NoRepo` and the command**

In `runCommand`, when `cmd.NoRepo`, ignore `identify`'s error and keep `ctx.Repo` nil. `add` uses `absArg` on its path or `c.Dir`, then `projects.Locate`: `ErrNotRepository` is a finding `not-a-repository`, `gitx.ErrGitMissing` an `EnvError{Rule: "git-missing"}`, `Docket` false a finding `not-a-project`. A `Load` error is an `Unreadable{Source: <index path>, Reason}`. `ls` stats each path with `os.Lstat`: `fs.ErrNotExist` sets `missing`; any other error adds an `Unreadable` and leaves `missing` false.

- [ ] **Step 4: Add the README row**

After `dkt domains` in the command table: `| \`dkt projects\` | Adds, lists, removes or ignores the Docket projects in this machine's index |`.

- [ ] **Step 5: Run the tests and see them pass**

Run: `go test ./internal/cli/ -run TestProjects && go vet ./... && gofmt -l .`
Expected: PASS.

- [ ] **Step 6: Commit**

Commit "Add dkt projects add and ls" with the trailer `Change: projects-index/3`.

---

### Task 4: `dkt projects rm` and `ignore`

**Scenarios:** project-index#path-that-is-gone, project-index#remove-an-entry, project-index#remove-an-unknown-path, project-index#ignore-a-registered-project, project-index#ignore-before-registering

**Review:** checks

**Files:**
- Modify: `internal/cli/cmd_projects.go`
- Modify: `internal/cli/cmd_projects_test.go`

**Interfaces:**
- Consumes: Task 3's `absArg`, `projectState`, `prRun`.
- Produces: `func pathArg(c *Context, arg string) (path string, origin *string)`: the project path and origin when `projects.Locate(absArg(c, arg))` succeeds, else `absArg(c, arg)` and nil.

- [ ] **Step 1: Write the failing tests**

- `TestProjectsRm`: a registered repository; `rm <repo>` gives `removed`, `project` the removed entry, and the index no longer holds it.
- `TestProjectsRmSubdirectory`: `rm <repo>/internal` removes the repository's entry.
- `TestProjectsRmGone`: an index holding `<tmp>/old`, which does not exist; `rm <tmp>/old/` and, separately, `rm <tmp>/x/../old` remove it.
- `TestProjectsRmUnknown`: `rm <tmp>/none`: exit 1, `not-in-index`, the index bytes unchanged.
- `TestProjectsIgnoreRegistered`: `ignore <repo>` on a registered entry gives `ignored` and `ignored` true; a second run gives `exists` and leaves the bytes unchanged.
- `TestProjectsIgnoreNew`: a Docket repository with an origin, not in the index: `ignore <repo>` adds an entry with `ignored` true, that origin and the fixed time.
- `TestProjectsIgnoreNotRepository`: `ignore <plain folder>` adds an entry with that path and a null origin.

- [ ] **Step 2: Run them and see them fail**

Run: `go test ./internal/cli/ -run 'TestProjects(Rm|Ignore)'`
Expected: FAIL.

- [ ] **Step 3: Implement `rm` and `ignore` in `internal/cli/cmd_projects.go`**

Both use `pathArg`, then `Find`; only a change in the index is saved.

- [ ] **Step 4: Run the tests and see them pass**

Run: `go test ./internal/cli/ -run TestProjects && go vet ./... && gofmt -l .`
Expected: PASS.

- [ ] **Step 5: Commit**

Commit "Add dkt projects rm and ignore" with the trailer `Change: projects-index/4`.

---

### Task 5: Automatic registration

**Scenarios:** project-index#first-command-in-a-project, project-index#project-already-registered, project-index#init-registers-the-project, project-index#status-in-a-repository-without-docket, project-index#registration-turned-off, project-index#no-settings-file, project-index#settings-that-are-not-json, project-index#index-that-cannot-be-written, project-index#ignore-before-registering, cli-output#inspect-in-a-new-project

**Review:** judgment

**Files:**
- Create: `internal/projects/register.go`
- Create: `internal/cli/register_test.go`
- Modify: `internal/cli/run.go`
- Modify: `README.md`

**Interfaces:**
- Consumes: Tasks 1 and 2.
- Produces, in package `projects`: `func Register(root string, g gitx.Git, env []string, now time.Time) error`: nil when it added the project or had nothing to do; otherwise an error whose message names the file it could not read or write.

- [ ] **Step 1: Write the failing tests**

In `internal/cli/register_test.go`, each test with its own `testutil.Env`; "the same as without registration" means the exit code and stdout equal those of the same run in a fresh environment whose settings hold `{"projects":{"autoRegister":false}}`:

- `TestRegisterFirstCommand`: `status --json` in a Docket repository: the index holds the repository with `ignored` false; code and stdout the same as without registration; stderr has no `project index` line.
- `TestRegisterAlreadyRegistered`: after `projects add` and `os.Chtimes` an hour back, `lint --json` leaves the index bytes and modification time unchanged.
- `TestRegisterFromLinkedWorktree`: the main worktree registered; `status --json` in a linked worktree leaves the index unchanged.
- `TestRegisterIgnoredKept`: an ignored entry; `status --json` leaves the index unchanged.
- `TestRegisterNoDocket`: a repository without `docket/workflow.md`: no index file after `status --json`.
- `TestRegisterOff`: `autoRegister` false: no index file.
- `TestRegisterInit`: `doInit` with `--tools claude` in a repository without Docket: the index holds it afterwards.
- `TestRegisterNotForVersionHelpUsage`: `--version`, `--help`, `status --help` and `status --bogus` write no index file.
- `TestRegisterNotForProjects`: `projects rm <repo>` run inside the registered repository leaves no entry for it.
- `TestRegisterBadSettings`: the settings hold `{`: stderr has one line starting `dkt: project index:` and naming the settings path; no index file; code and stdout the same as without registration.
- `TestRegisterUnwritable`: `XDG_STATE_HOME` set to a folder under a regular file: stderr names the index path; code the same as without registration.
- `TestRegisterNoHome`: an environment with only `PATH` and the git variables of `testutil.Env` but no `HOME`: stderr has the warning, the code is the same as without registration.
- `TestInspectInNewProject`: `inspect --json` in a committed Docket repository: `git status --porcelain` prints nothing afterwards, and the index holds the repository.

- [ ] **Step 2: Run them and see them fail**

Run: `go test ./internal/cli/ -run 'TestRegister|TestInspectInNewProject'`
Expected: FAIL, the index is never written by other commands.

- [ ] **Step 3: Implement `Register` and call it**

`Register`, in this order: `SettingsPath`, `AutoRegister` (false: return nil); `IndexPath`, `Load`; an entry for `root`: return; `IsDocket(root)` false: return; `MainWorktree(g)`, an entry for it: return; `RemoteURL("origin")`; append the entry with `now.UTC()` to the second; `Save`. In `runCommand`, after `cmd.Run` and before `finish`, when `!cmd.NoRepo && ctx.Repo != nil`, call it with `ctx.Repo.Root`, `ctx.Git`, `ctx.Env`, `ctx.Now()`, and print `dkt: project index: <err>` on `o.Stderr` when it fails.

- [ ] **Step 4: Run the whole suite**

Run: `go test ./...`
Expected: PASS. An existing test that now fails builds its own environment without `HOME` and checks stderr: give it the `HOME` of `testutil.Env`, and do not change what it asserts. Report each such test.

- [ ] **Step 5: Say it in the README**

Change the `dkt inspect` row to "writes nothing in the project", and add after the sentence under the table: "Every command run in a Docket project also adds it to this machine's project index, which `dkt projects ls` lists; set `projects.autoRegister` to `false` in `~/.config/docket/config.json` to stop that."

- [ ] **Step 6: Check and commit**

Run: `gofmt -l . && go vet ./... && go test ./...`
Expected: nothing printed by `gofmt`, PASS.

Commit "Register a project when a command runs in it" with the trailer `Change: projects-index/5`.
