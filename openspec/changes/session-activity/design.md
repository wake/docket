## Context

The idea's two "Settled" sections are followed as written: each session reports for itself, nothing probes or relays; a session writes its state with the time, and readers only read; a state not refreshed within a window is absent; the user agrees once, when a setup step they run installs the hook; the file lives under `$XDG_STATE_HOME/docket/`, mapped to the work by its working directory, and nothing is written in a repository. This design settles the rest: the states and the window, how `dkt snapshot` carries them, and who writes and how the hook is installed.

What exists: `internal/projects` locates the index under the state home (`xdgFile`, `IndexPath`); `scan.Scan` lists the worktrees, and `dkt snapshot` gives each work its `locations` (a worktree location's `source` is the worktree's path), `merged` and archived state. Every command goes through `runCommand`, which identifies the repository with four or five git runs and registers the project afterwards.

## Goals / Non-Goals

**Goals:**
- A hook call costs one short process start and a few file operations, and can never stop, slow down noticeably, or talk to the coding agent.

**Non-Goals:**
- Knowing which work a session means when its worktree holds several unmerged works: all of them get the activity.

## Decisions

### Two states, and `ended` removes the record

`working` and `waiting` (for the user) are the states a reader sees; `dkt activity ended` deletes the session's record, since an ended session and an absent one look the same to a viewer. The idea named the three; the third needs no record of its own.

### A 15-minute window

While an agent works, its hooks fire at least once per tool call. The longest stretch without a hook is one tool call, and Claude Code stops a foreground command after 10 minutes at most, so a working session writes at least every 10 minutes or so; 15 minutes leaves room. A session waiting for the user writes nothing, so it counts as absent 15 minutes after its turn ended, as the settled rule says.

Considered: 5 minutes, which would drop a session running one long test suite; an hour, which would keep a closed laptop's session for an hour.

### One file per session, keyed by the session

Each session writes `docket/sessions/<key>.json` under the state home, where the key is a hash of the session id, or of the directory when there is none. The file holds `session`, `dir`, `state` and `time` (RFC 3339, UTC), written to a temporary file and renamed into place. Parallel sessions never write the same file, so no lock is needed.

Considered: one shared file. Rejected: hooks of parallel sessions would race on it.

### Session and directory from the hook's input, not from Claude Code alone

Claude Code passes every hook a JSON object on standard input with `session_id` and `cwd`. `dkt activity` uses those two fields when standard input holds such an object, and otherwise its own working directory, keyed by that directory. Another coding agent can call the same command with the same fields, or with none. The directory is stored with its symbolic links resolved, so that it compares with the worktree paths git reports.

### Fast and silent: run before everything else

`RunWith` hands `activity` to its own function before any repository is identified: no git run, no registration, no output, exit 0 whatever happens. In Claude Code, exit code 2 from a hook blocks the tool call or the prompt, and what a `UserPromptSubmit` hook prints goes into the agent's context; a hook also runs on every tool call. `cmd/dkt` passes standard input through `Options.Stdin`; a terminal is never read, so running the command by hand does not wait for input.

### Old records removed on write

Each write removes the files in `docket/sessions/` whose modification time is older than the window. Records of sessions that crashed, or of directories outside any Docket project, so last at most until another session writes. Readers ignore them past the window either way.

### `dkt snapshot` gives each work one `activity`

`result.works[].activity` is `working`, `waiting` or null. `dkt snapshot` reads the records (from the state home in the environment it runs with), drops those past the window, and maps each record's directory to the deepest worktree holding it, so that a session in a linked worktree inside the main checkout counts for the linked one. A record counts for the works that have a location in that worktree that is not archived, and that are not merged: those are the works being done there, while the main checkout also holds every merged work. `working` wins over `waiting`. dkt decides what counts, so a viewer shows the value as it is and gets no times to compare.

A missing folder means no activity; a folder that cannot be listed is unreadable, as every unreadable source is; a record that cannot be parsed is skipped, since it is another session's and may be written by a newer dkt.

### docket-init installs the Claude Code hooks, in the user's settings

The hook is a setup step the user runs on purpose, so it goes in docket-init, as its last step, offered once the project is set up. The prompt first looks for a hook that already calls `dkt activity` and skips the step when there is one, so that a second project does not ask again. It explains that the hooks write a small state file under the state home on every prompt, tool call and turn end of every Claude Code session on the computer, and never in a repository; it writes the settings only once the user agrees; and it says how to remove them.

They go in the user's `~/.claude/settings.json`: a project's `.claude/settings.json` is committed, which would put the hook on teammates, and `.claude/settings.local.json` stays in the checkout it was made in, so a session started in a linked worktree would not have it. The events: `UserPromptSubmit` and `PreToolUse` write `working`; `Notification` (a permission prompt or an idle reminder) and `Stop` write `waiting`; `SessionEnd` writes `ended`. The command is the absolute path that `command -v dkt` gives, so it does not depend on the hooks' PATH.

Considered: docket-doctor. Rejected: it fixes problems `dkt inspect` reports, and a missing hook is not one.

## Risks / Trade-offs

- [One tool call longer than 15 minutes makes a working session look absent until its next hook] → rare; it corrects itself on the next tool call or turn end.
- [A session waiting longer than 15 minutes counts as absent] → this is the settled rule; the session reappears on the user's next prompt.
- [A future Claude Code changes the hook input] → the command falls back to its working directory, keyed by that directory.
- [`dkt` removed or moved after the hook is installed] → the hook fails with a code other than 2, which Claude Code does not treat as blocking; docket-init's removal steps say how to take the hooks out.
- [Sessions outside Docket projects write records too] → they are never read, and are removed after the window by the next write; the consent text says the hooks run in every session.

## Migration Plan

Nothing to run: without the hook there are no records, and every work's `activity` is null.
