# Tasks

Each group is one slice an implementer finishes test-first. Scenario names refer to `specs/<capability>/spec.md`. Tests run with `go test ./...`; every OpenSpec call in a unit test uses the fake OpenSpec.

## Reuse

| Need | Existing (path:symbol) | Decision | Reason | Evidence |
|---|---|---|---|---|
| The minimum version | `internal/openspec/runner.go:Minimum` | extend | The one constant every check and `result.openspec.minimum` read | `rg -n "Minimum" internal` |
| OpenSpec's config text in tests | `internal/testutil/openspec.go:OpenSpecConfig` | reuse | Replaces the uncommitted fixture | `rg -n "OpenSpecConfig" internal` |

## 1. Minimum 1.14.0

**Review:** checks
**Files:** `internal/openspec/runner.go`, `internal/openspec/runner_test.go`, `internal/cli/*_test.go`, `internal/setup/config_test.go`, `internal/testutil/openspec.go`, `.github/workflows/ci.yml`, `AGENTS.md`

- [ ] 1.1 Write failing tests for the scenarios Too old (openspec-runtime) and Old OpenSpec (setup-inspection) with 1.13.2, plus: 1.14.0 is accepted and `result.openspec.minimum` is `1.14.0`; verify they fail
- [ ] 1.2 Set `Minimum` to `1.14.0`; give the fake OpenSpec `1.14.0` wherever a test means a supported version (the tests of an old version keep their own); make `TestPlanConfigOpenSpec170Fixture` use `testutil.OpenSpecConfig` instead of the uncommitted fixture, renamed to say it is OpenSpec's default config; verify every test passes
- [ ] 1.3 Change the CI job's matrix to `['1.14.0', 'latest']`, `TestRealOpenSpec`'s comment, and the line `Requires OpenSpec 1.7.0 or later.` in `AGENTS.md` to 1.14.0; run `TestRealOpenSpec` locally with `npx -y @fission-ai/openspec@1.14.0` and with `@latest`; verify both pass, and that the CI job passes on the pull request
