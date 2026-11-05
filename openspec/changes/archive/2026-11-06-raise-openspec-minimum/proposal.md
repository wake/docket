# Proposal

## Why

The coming `dkt lint` reads a change's deltas through `openspec show --json` instead of parsing them itself. Checked on 2026-11-06 with every release from 1.7.0: the JSON gives requirement and scenario names only from OpenSpec 1.14.0 on; 1.7.0 to 1.13.0 give their text alone, so scenario references cannot be checked. The minimum was set to 1.7.0 on the understanding that it would be raised when a feature needed a newer release.

## What Changes

- The minimum OpenSpec version becomes 1.14.0, in every place that states it: version checks of `dkt init`, `dkt new` and `dkt inspect`, `result.openspec.minimum`, the CI job that runs the real OpenSpec (1.14.0 and the latest), and `AGENTS.md`.
- `TestPlanConfigOpenSpec170Fixture` read a fixture that is not committed and so always skipped; it uses the fake OpenSpec's config text instead.

## Non-Goals

- No upper bound, as before.
- No change to how OpenSpec is located or how its errors are reported.

## Capabilities

### New Capabilities

### Modified Capabilities
- `openspec-runtime`: the minimum version is 1.14.0.
- `setup-inspection`: `result.openspec.minimum` is 1.14.0.

## Impact

- `internal/openspec` (the constant), the tests that give the fake OpenSpec a version, `.github/workflows/ci.yml`, `AGENTS.md`.
- Users with OpenSpec older than 1.14.0 get `openspec-unsupported` and upgrade, as docket-init already offers.
