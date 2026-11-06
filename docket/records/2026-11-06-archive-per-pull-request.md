---
title: "Archive each change in its own pull request"
kind: prompt
source: "Maintainer decision: keep archiving each change in its own pull request"
---

## What

`docket/workflow.md`, under "When to archive": dropped the "Until this repository has a release process" condition, so the sentence now reads "Archive each change in its own pull request, as its last commit."

## Why

The release process now exists, so the old wording would have silently switched the rule to archiving at release time. The maintainer decided to keep archiving per pull request.

## Verification

`dkt inspect --json` lists `docket/workflow.md` as `same` with no `workflow-` finding; grep shows the new sentence and not the old one. No code changed, so no build or tests.

## Review

Adversarial review by codex found one issue: the skill lists archive (step 9) after merge (step 8), so "last commit of the pull request" looks unexecutable. Not changed: step 9 says when to archive follows "When to archive" in `docket/workflow.md`, so the skill defers to this sentence; the sentence is the same as the one it replaces and as AGENTS.md says. No contradiction with docs/release.md found.
