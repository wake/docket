---
title: "Recognize the files earlier releases shipped"
kind: fix
source: "Found while testing an upgrade from v0.17.0: dkt init reports skill-differs for a file a release shipped"
---

## What

`internal/setup/history.go` now holds the hash of every file v0.16.0, v0.17.0 and v0.18.0 shipped, so `dkt init` and `dkt inspect` treat those copies as outdated. One test fails when a shipped file changes and its hash is not recorded, and prints what to add. Another reads every release tag and fails when a release's hash is gone; CI now fetches the tags for it. `AGENTS.md` says what a change to `shipped/` adds.

## Why

`Earlier` was empty: no release step filled it. A project set up by an earlier release got `skill-differs` after an upgrade, as if the user had edited the skill, and `dkt init` kept the old file, against the `project-setup` spec.

## Verification

The tests were seen failing, then passing; a shipped file changed unrecorded, or any release's hash dropped, makes them fail. With no tags they fail in CI and skip elsewhere. End to end: projects set up by the real v0.16.0, v0.17.0 and v0.18.0 executables, then upgraded with this build. Before the fix, the first two gave `skill-differs`. After it, they give `skill-outdated`, `dkt init` updates both copies, and `dkt inspect` is clean.

## Review

An attack review on another model found that the first test let a release's hash be dropped unnoticed. Fixed by reading the tags rather than keeping a second list. Not reviewed again; the fix is small.
