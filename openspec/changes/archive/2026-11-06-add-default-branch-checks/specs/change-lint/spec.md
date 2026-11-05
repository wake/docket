## ADDED Requirements

### Requirement: Design and plan routing
`dkt lint` SHALL report `routing` for each file under `docs/superpowers/specs/` or `docs/superpowers/plans/` that is in the worktree but not at the merge base of HEAD and the default branch's tip, whether committed on the branch, staged, or untracked and not ignored, naming the file. It SHALL do so whether or not changes are named. When there is no default branch or no merge base, it SHALL skip this rule.

#### Scenario: Plan written to superpowers' folder
- **WHEN** the current branch adds `docs/superpowers/plans/2026-11-06-x.md`, not yet committed, and a user runs `dkt lint add-x`
- **THEN** the finding `routing` names `docs/superpowers/plans/2026-11-06-x.md`

#### Scenario: File already on the default branch
- **WHEN** `docs/superpowers/specs/old.md` exists at the merge base and the branch changes it
- **THEN** no `routing` finding names it

#### Scenario: Ignored file
- **WHEN** an untracked `docs/superpowers/plans/x.md` is ignored by git
- **THEN** no `routing` finding names it

#### Scenario: Routing without a default branch
- **WHEN** no default branch can be found and a user runs `dkt lint --json`
- **THEN** there is no `routing` finding and `result.default_branch` is null

## MODIFIED Requirements

### Requirement: Lint result
The `result` of `dkt lint` SHALL hold `changes`, the slugs linted with their schema (`{"slug", "schema"}`), `specs`, the number of main specs whose titles were checked (0 when a change was named), and `default_branch`, the name of the default branch the routing rule used or null when it was skipped. Every finding SHALL name its change in `change` when it is about one, and its file and line when it has them. Without `--json`, the output SHALL list the findings.

#### Scenario: Clean change
- **WHEN** `add-x` breaks no rule, the default branch is `main`, and a user runs `dkt lint add-x --json`
- **THEN** `result.changes` is `[{"slug": "add-x", "schema": "docket"}]`, `result.specs` is 0, `result.default_branch` is `main`, there is no finding, and dkt exits with 0
