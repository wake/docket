---
title: Show whether a piece of work is being worked on
---

`dkt snapshot` shows each change's and record's stage, how many plan tasks are ticked and when the last commit was made, but not whether a session is working on it: work in progress and work that stopped half-way look the same. Moved here from docket-workbench on 2026-11-11, whose viewer needs it; it belongs in Docket because dkt is what reads the work and hands it to viewers.

## Settled (maintainer, 2026-11-11)

- Each development session reports for itself: no central relay between sessions, and nothing probes for running sessions or processes.
- A session writes its state to a file, with the time it last wrote it. Readers (dkt, and viewers through `dkt snapshot`) only read it.
- A state counts only while its write time keeps being refreshed within some window; past the window without a write, it is treated as absent.

## Open

- Where the file goes: in the work's worktree (ignored by git), or at a fixed place on the machine, such as under `$XDG_STATE_HOME/docket/` next to the projects index, mapped to the work by its working directory.
- Who writes it: the coding agent's hooks (a prompt sent, a tool used, a turn ended, the session ended) call a small command; whether dkt provides that command, and how it works with both Claude Code and Codex.
- Which states (working, waiting for the user, ended), and how long the window is.
- How `dkt snapshot` carries it.
- Consent: a hook that writes on every turn is a write the user does not ask for each time, so setting up the hook, with the user's agreement, has to be what allows it; the file never lands in what is committed.
