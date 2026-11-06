---
title: "Say in the docket skill how the order is written and what to do when nothing is next"
kind: prompt
source: "Order format settled in add-dkt-status; the maintainer approved this skill text on 2026-11-06"
---

## What

The "Order" bullet of the docket skill now gives the line form of `docket/order.md` (ideas and local issues, one per line), says that issues kept in the project's tool are not listed and that `dkt status` reads the default branch, and says what to do for each `result.reason` when `result.next` is null. Changed in `shipped/skill/SKILL.md` and, through `dkt init --force`, in `.claude/skills/docket/` and `.agents/skills/docket/`.

## Why

`dkt status` shipped with a line form and four reasons the skill did not mention, so an agent could not write the order or act on an empty `next` without guessing.

## Verification

Mechanical: the three copies are byte for byte the same, the front matter still parses, and the tests that read the shipped files pass (`go test . ./internal/setup/ ./internal/cli/` filtered to the shipped, skill, init and workflow tests). The text matches the add-dkt-status specs: the two line forms, the four reasons, `waiting_on`. No agent was run on it.

## Review

An adversarial review on another model found one problem: `dkt status` can name a `result.next` while `unreadable` lists a worktree or branch it could not read, where that item may already be taken, so doing it at once could duplicate work. Fixed: the skill now says to report `unreadable` to the user, who decides whether to go on. The maintainer wrote that sentence. The fix was not reviewed again; it adds one sentence.
