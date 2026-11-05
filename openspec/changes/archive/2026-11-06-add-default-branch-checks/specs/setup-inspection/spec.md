## ADDED Requirements

### Requirement: Commit trailers on the default branch
`dkt inspect` SHALL read the default branch's tip, not the worktree. For each folder under its `openspec/changes/` other than `archive` whose schema is `docket` and whose `plan.md` has a task, some commit reachable from the tip SHALL carry a `Change` trailer whose value is the change's name or starts with the name and `/`, read as `dkt sync` reads trailers. Otherwise it SHALL report `trailer-missing`, naming the change and its `plan.md`.

#### Scenario: Squash merge
- **WHEN** the default branch's tip holds `openspec/changes/add-x/` with a plan.md that has a task, and no commit reachable from the tip carries a `Change` trailer naming `add-x`
- **THEN** `dkt inspect --json` reports `trailer-missing` for `add-x` and exits with 1

#### Scenario: Merged with its trailers
- **WHEN** a commit reachable from the default branch's tip carries `Change: add-x/1`
- **THEN** no `trailer-missing` finding names `add-x`

#### Scenario: Change not on the default branch
- **WHEN** `add-y` exists only in the worktree, on a branch that is not the default branch
- **THEN** no `trailer-missing` finding names `add-y`

#### Scenario: Plan without tasks
- **WHEN** the default branch's tip holds `add-z` whose plan.md has no task, and no commit names `add-z`
- **THEN** no `trailer-missing` finding names `add-z`

### Requirement: Default branch in the inspect result
`dkt inspect` SHALL report in `result.default_branch` the name of the default branch the trailer check read, or null when it did not run. It SHALL skip that check, say why in the text output, and not fail for it, when there is no default branch or when the repository is a shallow clone, whose history may not hold the commits that carry the trailers.

#### Scenario: Inspect without a default branch
- **WHEN** no default branch can be found and a user runs `dkt inspect --json`
- **THEN** `result.default_branch` is null and no finding is `trailer-missing`

#### Scenario: Shallow clone
- **WHEN** the repository was cloned with `--depth 1`, and the default branch's tip holds `add-x`, with a plan.md that has a task, whose `Change` trailers are on earlier commits only
- **THEN** `dkt inspect --json` reports no `trailer-missing`, `result.default_branch` is null, and the text output says the check was skipped for a shallow clone

### Requirement: Main spec titles inspected
`dkt inspect` SHALL report `title-missing` for each main spec in the worktree that has no title, as change-lint's Main spec titles requirement says, naming its `spec.md`.

#### Scenario: Main spec with OpenSpec's first line
- **WHEN** `openspec/specs/booking/spec.md` starts with `# booking Specification` and a user runs `dkt inspect --json`
- **THEN** the findings hold `title-missing` naming `openspec/specs/booking/spec.md`, and dkt exits with 1
