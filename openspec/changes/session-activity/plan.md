# Implementation Plan

**Goal:** Sessions report their state through `dkt activity`, called by coding agent hooks the user installs once through docket-init, and `dkt snapshot` gives each work an `activity` from the records still within the window.

**Architecture:** A new package `internal/activity` writes, removes, prunes and reads per-session records under `docket/sessions/` in the state home, whose folder `internal/projects` locates beside the index. `RunWith` hands `activity` to its own function before the repository is identified, so a hook call runs no git, registers nothing, prints nothing and exits with 0. `snapshot.Collect` reads the live records and maps each to the deepest worktree holding its directory, setting `activity` on that worktree's unmerged, unarchived works. docket-init's last step installs the Claude Code hooks with the user's agreement.

**Tech Stack:** Go; the plugin's skill `docket-init`.

**Spec:** `proposal.md`, `design.md`, `specs/session-activity/spec.md` and `specs/project-snapshot/spec.md` in this change folder; the settled points are in `docket/ideas/session-activity.md`.

## Global Constraints

- States written: `working`, `waiting`; `ended` removes the record. The window is 15 minutes.
- Records: one file per session, `<state home>/docket/sessions/<key>.json`, the key a hash of the session id, or of the directory when there is none; the file holds `session`, `dir` (symbolic links resolved), `state` and `time` (RFC 3339, UTC). Written to a temporary file in the same folder and renamed.
- `dkt activity` prints nothing on standard output or standard error, exits with 0 in every case, starts no process, and does not register the project.
- `result.works[].activity`: `"working"`, `"waiting"` or null, always present.
- Only `dkt activity` writes the sessions folder: `dkt snapshot`, `dkt status` and `dkt inspect` read it or leave it alone.

## Review Focus

- A hook call never prints, never exits with anything but 0, and never starts git, whatever its arguments, input or environment.
- `dkt snapshot` writes nothing in the sessions folder, and removes nothing there.
- A session in a linked worktree inside the main checkout (`.claude/worktrees/<name>`) counts for the linked worktree's works only.
- Two sessions in one directory keep two records; one ending leaves the other's.

## Reuse

| Need | Existing (path:symbol) | Decision | Reason |
|---|---|---|---|
| `projects.SessionsDir` | `internal/projects/files.go:xdgFile` | extend | Built as `IndexPath` is, so the records sit beside the index |
| Per-session state records (`internal/activity`) | | new | Nothing in dkt keeps per-session state on the machine |
| `dkt activity` | `internal/cli/run.go:RunWith` | extend | Handed off before `runCommand`, which identifies the repository with git and registers the project |
| `Options.Stdin` | `internal/cli/run.go:Options` | extend | The process boundary gains standard input, which the hook input arrives on |
| `activity` on a work | `internal/snapshot/result.go:WorkEntry` | extend | One more field on the existing entry |
| Reading the records in `dkt snapshot` | `internal/snapshot/snapshot.go:Collect` | extend | Uses the worktrees it already lists, after the works are built |
| docket-init's hook step | `skills/docket-init/SKILL.md` | extend | A new last step |

---

### Task 1: Per-session state records

**Scenarios:** session-activity#working-recorded, session-activity#later-state-replaces-the-earlier, session-activity#ended-removes-the-record, session-activity#two-sessions-in-one-directory, session-activity#no-hook-input, session-activity#record-past-the-window, session-activity#old-record-removed

**Review:** checks

**Files:**
- Modify: `internal/projects/files.go`
- Modify: `internal/projects/files_test.go`
- Create: `internal/activity/activity.go`
- Create: `internal/activity/activity_test.go`

**Interfaces:**
- Consumes: `projects.ErrNoHome`, `safefile.Install` (temporary file and rename, not following links) and `safefile.ReadNoFollow`, as they are.
- Produces: `projects.SessionsDir(env []string) (string, error)`; `activity.Window` (15 minutes); `activity.Record{Session, Dir, State string; Time time.Time}`; `activity.Write(env []string, now time.Time, state string, in io.Reader, wd string) error`, where `in` is the hook input or nil and `wd` the working directory; `activity.Read(env []string, now time.Time) ([]activity.Record, error)`, the records written within the window before `now`, with a nil error and no records when the folder does not exist.

- [x] `SessionsDir` gives `/s/docket/sessions` for `XDG_STATE_HOME=/s`, `$HOME/.local/state/docket/sessions` when that variable is unset or relative, and `ErrNoHome` with neither.
- [x] `Write` with `working` and the input `{"session_id": "a", "cwd": "/w/app"}` leaves one file holding session `a`, directory `/w/app` resolved, state `working` and `now` in UTC; a later `waiting` from session `a` replaces it in the same file; `ended` from session `a` removes it.
- [x] Two inputs with sessions `a` and `b` and one directory leave two files; `ended` from `a` leaves `b`'s.
- [x] With a nil input, input that is not JSON, or a JSON object without `session_id` or `cwd`, the missing parts come from `wd` (the directory) and from the directory (the key): two writes in one directory leave one file.
- [x] A directory reached through a symbolic link is recorded as its target.
- [x] Each `Write` removes the files in the folder modified more than 15 minutes before `now`, and keeps one modified 5 minutes before.
- [x] An unknown state, no location for the state home, and a folder that cannot be written return an error and write no record.
- [x] `Read` returns the records written 14 minutes before `now`, not one written 16 minutes before; it skips, while still returning the other records, a file that is not a record, a file it cannot read, and a symbolic link; it returns nothing when the folder is missing, and an error when the folder cannot be listed; it writes and removes nothing.

### Task 2: The dkt activity command

**Scenarios:** session-activity#unknown-state, session-activity#folder-that-cannot-be-written, session-activity#no-home, cli-output#activity-with-json, cli-output#activity-usage-error, cli-output#json-on-success, cli-output#json-on-failure, cli-output#unknown-command, cli-output#no-command, cli-output#help

**Review:** judgment

**Files:**
- Modify: `internal/cli/run.go`
- Create: `internal/cli/cmd_activity.go`
- Create: `internal/cli/cmd_activity_test.go`
- Modify: `cmd/dkt/main.go`

**Interfaces:**
- Consumes: `activity.Write` (Task 1).
- Produces: `cli.Options.Stdin io.Reader`, nil meaning no input.

- [x] `Run` with `activity working` and the hook input on `Stdin` writes the record, exits with 0, and leaves standard output and standard error empty.
- [x] `activity busy`, `activity` with no state, `activity working extra`, `activity working --json`, input that is not JSON, no `HOME` or `XDG_STATE_HOME`, and a sessions folder that cannot be written each exit with 0 with standard output and standard error empty; the first three write nothing.
- [x] With `PATH` empty, so that no git can start, `activity working` still writes the record.
- [x] Run inside a Docket project not in the index, `activity working` leaves the index unwritten.
- [x] Standard input that is a terminal is not read: `cmd/dkt` passes it only when it is not a character device; and when the working directory cannot be read, `cmd/dkt` still exits with 0 for `activity`.
- [x] `dkt --help` lists `activity`, and `dkt activity --help` prints its help and exits with 0; `dkt --json activity working` prints nothing and exits with 0.
- [x] The existing tests of JSON output and exit codes for the other commands pass unchanged (`cli-output`'s MODIFIED requirements change only what `dkt activity` does).

### Task 3: dkt snapshot reports each work's activity

**Scenarios:** project-snapshot#session-working-on-a-change, project-snapshot#session-waiting, project-snapshot#merged-work, project-snapshot#worktree-inside-another, project-snapshot#worktree-path-through-a-link, project-snapshot#prunable-worktree, project-snapshot#no-records, project-snapshot#folder-that-cannot-be-listed, session-activity#record-past-the-window, cli-output#inspect-in-a-new-project, cli-output#snapshot-leaves-the-session-records

**Review:** judgment

**Files:**
- Modify: `internal/snapshot/result.go`
- Modify: `internal/snapshot/snapshot.go`
- Modify: `internal/snapshot/snapshot_test.go`
- Modify: `internal/cli/cmd_snapshot_test.go`

**Interfaces:**
- Consumes: `activity.Read` and `activity.Record` (Task 1); the worktrees `scan.Scan` lists, as they are. `snapshot.Collect(g gitx.Git)` keeps its signature: it reads the records with `g.Env` at the current time, and tests write records with times relative to the current time.

- [x] Every work's JSON holds `activity`, null when the state home has no sessions folder, and nothing is reported about the folder.
- [x] A `working` record from 2 minutes ago with a directory in a linked worktree holding the unmerged `add-x` gives `add-x` `working` and every other work null; a `waiting` one gives `waiting`; a `working` and a `waiting` record for the same worktree give `working`.
- [x] A record whose directory is a subdirectory of the worktree counts; with the linked worktree inside the main worktree, a record in the linked one counts only for the linked worktree's works.
- [x] A merged work, and a work whose only location in that worktree is archived, get null.
- [x] A record written 16 minutes ago gives null.
- [x] A record whose directory reaches the worktree through a symbolic link counts, and so does a worktree that git reports through a link (a repository under the test's temporary folder, which on macOS is reached through `/var`).
- [x] A session in the folder of a prunable worktree, recreated as a plain folder, and a bare repository's path give no activity.
- [x] `dkt inspect` in a Docket project not in the index still writes no file in the repository and adds the project to the index, as before.
- [x] A sessions folder that cannot be listed is named in `unreadable`, every `activity` is null, and dkt exits with 3; a file in it that is not a record is skipped.
- [x] After `dkt snapshot`, the sessions folder's files are byte-for-byte unchanged, a record past the window included.

### Task 4: docket-init offers the Claude Code hooks

**Scenarios:** none

**Review:** checks

**Files:**
- Modify: `skills/docket-init/SKILL.md`

- [x] A new last step, for Claude Code only: when no hook in the user's `~/.claude/settings.json` already calls `dkt activity`, it offers the hooks, saying that they write a small state file under the state home on every prompt, tool call and turn end of every Claude Code session on the computer, never in a repository, so that `dkt snapshot` can show which works a session is on; it writes the settings only once the user agrees.
- [x] The step names the events and states (`UserPromptSubmit` and `PreToolUse` with `working`, `Notification` and `Stop` with `waiting`, `SessionEnd` with `ended`), the command as the absolute path `command -v dkt` gives followed by `2>/dev/null || true`, keeps the settings and hooks already in the file, and says how to remove the hooks.
- [x] The step covers its other branches: the user declines (nothing is written); the settings file does not exist (it is created with the hooks alone); it is not valid JSON, or its `hooks` is not an object (nothing is written, and the user is told why); `command -v dkt` finds nothing (the step stops).
- [x] The hooks object the step describes, written into a scratch settings file, parses as JSON and has the five events with those commands; the check is done by hand and noted in the pull request.
- [x] Every command and field the step names exists in `internal/cli` and `internal/snapshot`.
