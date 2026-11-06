---
title: "Archive guidance defers to dkt sync --check and dkt lint"
kind: prompt
source: "Maintainer request: stop archiving only on dkt's own checks"
---

## What

The first archive guidance line in `shipped/config/operations.yaml` and in this repository's `openspec/config.yaml` now stops archiving only when `dkt sync <change> --check` or `dkt lint <change>` reports a problem. The extra stop conditions are gone.

## Why

The old line said to stop when a task had no commit with `Change: <change>/<N>`. The plan rules allow ticking by hand a task finished without any commit, so the two contradicted each other and an agent could be stuck.

## Verification

`dkt init --force --json` lists `openspec/config.yaml` as `same` with no finding; `dkt inspect --json` reports no `guidance-*` finding; grep shows the new line in both files and the old one in neither; `go vet ./...` and `go test ./...` pass. Not verified: behavior of an agent following the line.

## Review

One adversarial review by codex on the whole diff: approve, no findings (no conflict with the plan rules, no way to get stuck, YAML valid).
