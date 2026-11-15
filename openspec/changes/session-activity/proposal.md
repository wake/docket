---
title: "Show whether a piece of work is being worked on"
kind: feature
source: "docket/ideas/session-activity.md"
specs:
  session-activity: Session Activity
---

## Why

`dkt snapshot` shows each work's stage, ticked tasks and last commit, but not whether a session is working on it, so work in progress and work that stopped half-way look the same to a viewer. The maintainer settled on 2026-11-11 and 2026-11-16 how sessions report it: each session writes its own state, with the time, to a file under the state home that the user agreed to once, when installing the hook; a state not refreshed within a window counts as absent.

## What Changes

- A new command, `dkt activity <working|waiting|ended>`, records the calling session's state and directory, with the time, in a file of its own under `$XDG_STATE_HOME/docket/sessions/`; `ended` removes it. It takes the session and directory from the JSON a coding agent's hook passes on standard input, or uses its own working directory. It starts no process, prints nothing and always exits with 0, so a hook that calls it never stops or disturbs the agent. Each write removes the records older than the window.
- `dkt snapshot` gives each work an `activity`, `working`, `waiting` or null: a session's state counts for 15 minutes after its last write, and for the works not yet merged that a worktree holds when the session's directory is in that worktree. It reads only.
- docket-init offers, as its last step, to install the Claude Code hooks that call `dkt activity`, in the user's Claude Code settings, explaining that they write on every turn and how to remove them; it writes the settings only once the user agrees.

## Non-Goals

- Hooks for other coding agents, such as Codex; `dkt activity` itself is not tied to Claude Code.
- Showing activity in `dkt status`, `dkt inspect` or the text output of `dkt snapshot`.
- Telling sessions apart in `dkt snapshot`: a work gets one activity, whatever number of sessions are on it.
- Probing for running sessions or processes, or relaying between sessions.
- Writing anything into a repository.

## Capabilities

### New Capabilities

- `session-activity`: the `dkt activity` command, the state records it writes, and the window after which a record counts as absent.

### Modified Capabilities

- `project-snapshot`: each work carries its `activity`.
- `cli-output`: `dkt activity` is an exception to JSON output and exit codes, and its records may be written outside the repository.

## Impact

- New package for the state records; `internal/projects` gives their folder beside the index.
- `internal/cli`: the `activity` command, run before the repository is identified; `cmd/dkt` passes standard input.
- `internal/snapshot`: `activity` on each work.
- Prompt: `skills/docket-init/SKILL.md`.
