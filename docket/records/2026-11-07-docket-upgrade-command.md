---
title: "Add docket-upgrade, a command that upgrades Docket"
kind: prompt
source: "Maintainer request: a command that upgrades dkt and the plugin, then adjusts the project; the maintainer settled the text on 2026-11-07"
---

## What

New plugin command `skills/docket-upgrade/`. It compares versions, updates `dkt` and the plugin once the user agrees, then runs `dkt inspect` in the project: files from an earlier release are updated with `dkt init`, and it suggests docket-doctor for the rest. One sentence of `skills/docket-doctor/SKILL.md`, on exit codes 2 and 4, is reworded. The README gets an Upgrade section and a command row.

## Why

An upgrade is three steps in order across two tools, and the plugin manager covers one of them, off by default for a third-party marketplace. The README only covered installing.

## Verification

Mechanical: both skills equal the settled drafts byte for byte, and `claude plugin validate .` passes. The prompt's commands were followed by hand in a throwaway home: `dkt` from v0.17.0 to v0.18.0, the plugin from 0.17.0 to 0.18.0 through a local clone of the marketplace, and a project from `skill-outdated` to clean, with the build of the earlier-release-hashes fix. Not verified: an agent running the prompt.

## Review

An attack review on another model found two problems in step 4. A `dkt` the user declined to update only compares with its own older files, so it would report the project as up to date. And gating on `openspec/schemas/docket/` skipped the case where that folder is missing. The maintainer rewrote the step; it now needs the latest `dkt` and looks for `docket/workflow.md`. Not reviewed again; one step changed.
