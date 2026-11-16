---
title: Settle how the session activity hooks are installed
---

`dkt activity` and the `activity` of `dkt snapshot` (`session-activity`, from `docket/ideas/session-activity.md`) only have data when a coding agent's hooks call `dkt activity`. How those hooks get installed is open again.

## What was shipped, and then taken out

docket-init's last step offered, with the user's agreement, five Claude Code hooks in the user's `~/.claude/settings.json`:

- `UserPromptSubmit` and `PreToolUse`, which record `working`;
- `Notification` and `Stop`, which record `waiting`;
- `SessionEnd`, which records `ended`.

So `dkt activity` ran on every prompt sent, every tool call and every turn end, in every Claude Code session on the machine, Docket project or not.

On 2026-11-16 the maintainer, who had agreed to "write once per turn", asked to discuss the frequency and the scope again. Until that is settled, docket-init does not offer the hooks. Hooks a user already installed keep working; they can be removed by deleting the entries that call `dkt activity` from `~/.claude/settings.json`.

## Open

- Scope: every session on the machine, or only sessions in a Docket project, decided by the hook or by `dkt activity` itself.
- Frequency: fewer events, such as dropping `PreToolUse`, or writing at most once in a while.
- Where the hooks are installed: docket-init, another step the user runs on purpose, or a project's own settings.
- Other coding agents, such as Codex.
