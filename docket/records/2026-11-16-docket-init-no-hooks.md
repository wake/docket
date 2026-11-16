---
title: "Stop offering the session activity hooks in docket-init"
kind: prompt
source: "docket/ideas/session-activity-hooks.md"
---

## What

docket-init's last step, which offered five Claude Code hooks calling `dkt activity` in `~/.claude/settings.json`, is removed. `dkt activity`, `dkt snapshot`'s `activity`, their specs and the state records are unchanged; without hooks there is no data. Nothing else told users to install the hooks (README, docket-doctor, docket-upgrade, docs: checked).

## Why

The maintainer wants to discuss again how often the hooks run (each prompt, tool call and turn end) and where (user level, every Claude Code session). Until that is settled (`docket/ideas/session-activity-hooks.md`), docket-init does not offer them. No spec requires the step, so this is a prompt edit.

## Verification

- The skill still loads: its front matter is unchanged, steps 0–5 remain, and nothing refers to a step 6.
- `skills/docket-init/SKILL.md` is not in `shipped/`, so no hash changes.
- The prompt page's zh, zh.base and en copies match after the change.

## Review

Codex (gpt-5.6-sol, adversarial) approved with no findings: no spec requires the step, nothing else refers to it or tells users to install the hooks, and no code, spec or test changed.
