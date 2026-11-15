---
title: "Have another reviewer decide a must-fix finding the author turns down"
kind: prompt
source: "Review of fence-in-comment: the author alone turned down a critical finding from plan review; the maintainer settled the text on 2026-11-15"
---

## What

"Reviewers" in the shipped `docket/workflow.md` template, and in this repository's copy, gains a paragraph: the author writes down why a finding is not taken up, and another reviewer, with high reasoning effort where it can be set, decides a turned-down must-fix finding or two reviewers' opposite conclusions; "Plan review" points to it. The template moves to version 2. Tests no longer assume the shipped version is 1.

## Why

Nothing said what to do with a finding the author disagrees with, so a must-fix finding could be turned down on the author's reading alone.

## Verification

Both copies of the changed sections are identical. Tests pass with the template at 2 and at 3. `dkt init --force` and `dkt inspect` here report nothing. In a scratch repository, `dkt init` writes version 2, and a file at version 1 gets `workflow-outdated`. Not verified: docket-doctor bringing the paragraph into an existing project.

## Review

Codex (gpt-5.6-sol), a different model, reviewed the whole diff: one important finding, "Plan review" still said to fix every finding. Fixed by pointing it to "Reviewers"; a small fix, not reviewed again.
