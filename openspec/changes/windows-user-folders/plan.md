# Implementation Plan

**Goal:** On Windows, dkt finds the index, the settings, the user template and the activity records under the folder `USERPROFILE` names (or an absolute `XDG_*`), at the same paths as on other systems, and the Windows CI job proves it; macOS and Linux do not change.

**Architecture:** `internal/projects/files.go` decides once which variable holds the home folder, by `os.UserHomeDir`'s rule (`USERPROFILE` on Windows, `HOME` elsewhere), and reads it, and the `XDG_*` variables, from the environment it is given through `execpath.Getenv`, which compares names without case on Windows. Every user file already goes through `xdgFile`, so `project-index`, `user-workflow` and `session-activity` follow without further code. The no-home messages name the variable of the system. The test environment sets both variables to its temporary home, and tests that remove the home folder remove both.

**Tech Stack:** Go; GitHub Actions.

**Spec:** `proposal.md`, `design.md` and `specs/platform-support/spec.md` in this change folder; the maintainer's decision is in `docket/ideas/windows-user-folders.md`.

## Global Constraints

- The home folder's variable is `USERPROFILE` on Windows and `HOME` on every other system; on Windows `HOME` is never read for dkt's user files.
- An absolute `XDG_CONFIG_HOME` or `XDG_STATE_HOME` wins on every system; paths under the home folder are `.config/docket/...` and `.local/state/docket/...`, joined with the system's separator.
- No `APPDATA` or `LOCALAPPDATA`.
- No change to `dkt activity`'s or the hooks' behavior, to `shipped/`, to `skills/`, or to the README.
- The user files are still read from the environment dkt is given (`Options.Env`), never from the process's own.

## Review Focus

- On macOS and Linux, every user file resolves exactly as before, including with `USERPROFILE` set.
- No test, on any system, reads or writes the runner's own home folder or profile.
- The Windows CI job runs the tests named in each task, and its name is unchanged.

## Reuse

| Need | Existing (path:symbol) | Decision | Reason |
|---|---|---|---|
| The home folder's variable by system | `external: os.UserHomeDir` | extend | Its rule, applied to the given environment; a test ties the two together |
| Reading the variables without case on Windows | `internal/execpath/execpath.go:Getenv` | extend | Replaces `projects.envValue`, which compares names exactly |
| User file lookup | `internal/projects/files.go:xdgFile` | extend | The one place all four files are found |
| No-home messages | `internal/projects/files.go:ErrNoHome`, `internal/projects/register.go:Register` | extend | Name the system's variable |
| Test home folder | `internal/testutil/repo.go:Env` | extend | Sets `USERPROFILE` beside `HOME` |
| Removing the home folder in tests | `internal/testutil/repo.go` | new | Six tests strip `HOME=` by hand; on Windows they must strip `USERPROFILE` too, so one helper does both |
| Windows CI job | `.github/workflows/ci.yml:windows` | extend | Runs the tests of this change |

---

### Task 1: The home folder by system

**Scenarios:** platform-support#index-under-the-user-profile, platform-support#variable-spelled-in-another-case, platform-support#other-systems-unchanged

**Review:** judgment

**Files:**
- Modify: `internal/projects/files.go`
- Modify: `internal/projects/register.go`
- Modify: `internal/projects/files_test.go`

**Interfaces:** `ErrNoHome` stays the sentinel callers test with `errors.Is`; only its message changes. Uses `execpath.Getenv(env []string, key string) (string, bool)` as it is.

- [x] The variable is chosen by a function taking `windows bool`; a test checks it gives `USERPROFILE` for `true` and `HOME` for `false`, on any system.
- [x] A test checks that, for the process's own environment, the chosen variable's value equals what `os.UserHomeDir` returns, whenever `os.UserHomeDir` succeeds.
- [x] `TestIndexPath`, `TestSessionsDir`, `TestSettingsPath` and `TestUserWorkflowPath` use absolute temporary paths and the system's variable, so they pass on macOS, Linux and Windows; each still covers an absolute `XDG_*`, a relative `XDG_*` falling back to the home folder, the last of two entries winning, and no home folder giving `ErrNoHome`.
- [x] On macOS and Linux, an environment holding `USERPROFILE` and `HOME` resolves under `HOME`, and one holding only `USERPROFILE` gives `ErrNoHome` (scenario "Other systems unchanged"); on Windows, one holding only `HOME` gives `ErrNoHome`, one holding `UserProfile` resolves under it, and one holding an absolute `xdg_state_home` or `Xdg_Config_Home` resolves under that folder (scenarios "HOME not read on Windows" and "Variable spelled in another case"). The tests for one system skip on the other, saying why.
- [x] `ErrNoHome`'s message and `Register`'s warning name `USERPROFILE` on Windows and `HOME` elsewhere; `projects.envValue` is gone.

### Task 2: Tests find the home folder on every system

**Scenarios:** platform-support#index-under-the-user-profile, platform-support#user-template-under-the-user-profile, platform-support#home-not-read-on-windows, platform-support#state-home-kept-on-windows, platform-support#other-systems-unchanged

**Review:** judgment

**Files:**
- Modify: `internal/testutil/repo.go`
- Modify: `internal/cli/cmd_inspect_test.go`
- Modify: `internal/cli/register_test.go`
- Modify: `internal/cli/cmd_workflow_test.go`
- Modify: `internal/cli/cmd_projects_test.go`

**Interfaces:** `testutil.WithoutHome(env []string) []string`: a copy of env without its `HOME` and `USERPROFILE` entries.

- [x] `testutil.Env` sets `USERPROFILE` to the same temporary folder as `HOME`, on every system.
- [x] `TestInspectUserWorkflowNoHome`, `TestInspectIndexWithoutLocation`, `TestRegisterNoHome`, `TestRegisterNoStateHome` and the no-home cases of `TestWorkflowModeNoHome` remove the home folder with `testutil.WithoutHome`, and still pass on macOS and Linux; the first four still pass on Windows.
- [x] A `dkt projects ls --json` test whose environment holds only the other system's variable (`USERPROFILE` on macOS and Linux, `HOME` on Windows) reports `no-home` with a message naming the system's variable, and exits with 4.
- [x] A `dkt projects add --json` test with no `XDG_STATE_HOME`, and `HOME` and `USERPROFILE` set to two different temporary folders, writes the index at `.local/state/docket/projects.json` under the system's variable's folder and creates nothing under the other.
- [x] A `dkt inspect --json` test with no `XDG_CONFIG_HOME`, and `HOME` and `USERPROFILE` set to two different folders, reports `result.user_workflow.path` as `.config/docket/workflow.md` under the system's variable's folder, with `exists` true for a template written there.
- [x] The `internal/cli` tests of `dkt activity`, which use a temporary `XDG_STATE_HOME`, pass on Windows (scenario "State home kept on Windows").
- [x] The helpers that work out an expected path from `HOME` (`prIndexPath` in `cmd_projects_test.go`, `regHome` in `register_test.go`) and `TestInspectNothingWritten`, which watches the `HOME` folder, stay as they are: `testutil.Env` gives `HOME` and `USERPROFILE` the same folder, so they point at the folder dkt uses on every system. Which variable dkt reads is told apart only by the tests above that give the two different folders.

### Task 3: The Windows CI job runs these tests

**Scenarios:** none

**Review:** checks

**Files:**
- Modify: `.github/workflows/ci.yml`

- [x] The Windows job runs, with an anchored `-run` pattern since the rest of each package still fails there: in `internal/projects`, the path tests and the home-variable tests of Task 1; in `internal/cli`, `TestInspectUserWorkflow`, `TestInspectNoUserWorkflow`, `TestInspectUserWorkflowNoHome`, `TestInspectIndexWithoutLocation`, `TestProjectsAddUnderStateHome`, `TestProjectsNoHome`, the `dkt projects add` test of Task 2, `TestRegisterNoHome`, `TestRegisterNoStateHome`, the `TestActivity` tests and the new tests of Task 2. Its name is unchanged.
- [x] The inspect workflow sync tests and the activity tests of `internal/snapshot` and `internal/activity` are not added: they fail on Windows for path forms and Unix test paths (see design.md, Context), which `windows-paths` and `windows-suite` handle.
- [x] The pull request's Windows job passes.
