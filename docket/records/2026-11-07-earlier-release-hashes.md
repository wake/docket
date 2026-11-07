---
title: "Recognize the files earlier releases shipped"
kind: fix
source: "Found while testing an upgrade from v0.17.0: dkt init reports skill-differs for a file a release shipped"
---

## What

`internal/setup/history.go` now holds the hash of every file v0.16.0, v0.17.0 and v0.18.0 shipped, taken from the tags, so `dkt init` and `dkt inspect` treat those copies as outdated. A new test fails when a shipped file changes and its hash is not recorded, and prints what to add. `AGENTS.md` says so too.

## Why

`Earlier` was empty: no release step filled it. A project set up by an earlier release got `skill-differs` after an upgrade, as if the user had edited the skill, and `dkt init` kept the old file, against the `project-setup` spec.

## Verification

The regression test was seen failing, then passing; five mutants make it fail. End to end: projects set up by the real v0.16.0, v0.17.0 and v0.18.0 executables, then upgraded with this build. Before the fix, the first two gave `skill-differs`. After it, they give `skill-outdated`, `dkt init` updates both copies, and `dkt inspect` is clean. No test holds the older releases' hashes, since CI has no tags to compare with; only the end-to-end run checked them.

## Review

Not reviewed yet.
