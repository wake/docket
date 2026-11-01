# change-archive Specification

## Purpose
Defines the checks `dkt archive` runs before it lets OpenSpec merge a change
into the current specs, the evidence it leaves behind, and how `dkt rebase`
accepts newer specs as a change's starting point.

## Requirements

### Requirement: Preconditions
`dkt archive` SHALL refuse with exit 1, changing nothing, unless the working tree has no uncommitted changes, HEAD contains the default branch tip, every named change exists at the default branch tip, and the `docket` schema is found as `openspec-runtime` says. When the schema is not found, it SHALL report `schema-missing` telling the user to run `dkt init`.

#### Scenario: Uncommitted changes
- **WHEN** the working tree has an uncommitted edit and a user runs `dkt archive add-x`
- **THEN** the command refuses and exits with 1

#### Scenario: Branch behind the default branch
- **WHEN** HEAD does not contain the default branch tip
- **THEN** `dkt archive add-x` refuses and exits with 1

#### Scenario: Change not merged yet
- **WHEN** `add-x` exists in the worktree but not at the default branch tip
- **THEN** `dkt archive add-x` refuses and exits with 1

#### Scenario: Schema not installed
- **WHEN** neither the project's `openspec/schemas/docket/` nor the per-machine directory holds the schema and a user runs `dkt archive add-x`
- **THEN** no OpenSpec command is run, nothing is changed, the findings hold `schema-missing` naming `dkt init`, and the command exits with 1

### Requirement: Steps in order
For each named change, in the order given, `dkt archive` SHALL run the plan check, lint, the base check, `openspec validate <slug> --strict`, the reading cross-check, `openspec archive <slug> -y --json`, the spec title step, and the evidence step. It SHALL stop at the first change that fails a step and report the step, leaving changes archived earlier in the same run archived in the working tree.

#### Scenario: Second of two fails
- **WHEN** a user runs `dkt archive add-x add-y`, `add-x` passes every step, and `add-y` fails lint
- **THEN** `add-x` is archived in the working tree, `add-y` is untouched, the lint finding is reported, and the command exits with 1

### Requirement: Plan check
The plan check SHALL fail when `dkt sync <slug> --check` would exit with 1, when any step of a `### Task` section in plan.md is unticked, or when no commit reachable from HEAD and not from the change's base carries both `Change: <slug>` and a `Task:` trailer.

#### Scenario: Unticked task
- **WHEN** Task 5 of `add-x` has an unticked step and no commit names it
- **THEN** `dkt archive add-x` stops at the plan check and names Task 5

#### Scenario: Squashed pull request
- **WHEN** every step of `add-x` is ticked but no commit since its base carries `Change: add-x` with a `Task:` trailer
- **THEN** `dkt archive add-x` stops at the plan check

### Requirement: Base check
For every requirement the change modifies, removes or renames, the base check SHALL compare that requirement's block in the main spec at the change's base with its block in the working tree, which includes changes archived earlier in the same run, and SHALL fail, showing the difference, when they differ. For every requirement the change adds, it SHALL fail when the working tree's main spec has a requirement of that name that the main spec at the base did not.

#### Scenario: Requirement changed since base
- **WHEN** `add-y` modifies `booking` / "Cancel booking", and `add-x`, archived after `add-y`'s base, changed that requirement
- **THEN** `dkt archive add-y` stops at the base check and shows the requirement's text at the base and now

#### Scenario: Two changes in one run touch one requirement
- **WHEN** `add-x` and `add-y` share a base, both modify `booking` / "Cancel booking", and a user runs `dkt archive add-x add-y`
- **THEN** `add-x` is archived and `add-y` stops at the base check

#### Scenario: Other requirement in the same capability
- **WHEN** `add-y` modifies `booking` / "Cancel booking", and since its base only `booking` / "Refund points" has changed
- **THEN** the base check passes

#### Scenario: Added name now taken
- **WHEN** `add-y` adds `booking` / "Waitlist cap" and that requirement appeared in the main spec after `add-y`'s base
- **THEN** `dkt archive add-y` stops at the base check

### Requirement: Reading cross-check
Before running `openspec archive`, `dkt archive` SHALL compare Docket's reading of the change's deltas with the output of `openspec show <slug> --type change --json --deltas-only`, and SHALL fail when the capabilities, operations, requirement names or scenario names differ.

#### Scenario: Readings disagree
- **WHEN** Docket reads a requirement in `add-x` that OpenSpec's output does not list
- **THEN** `dkt archive add-x` stops before `openspec archive` and reports both readings

### Requirement: Evidence
After the spec title step, `dkt archive` SHALL write `evidence.md` in the archived change folder, recording the date, the Docket and OpenSpec versions, the URL of the `origin` remote or none, the HEAD SHA, the change's title or none, its kind, source and base, its schema, the steps skipped for it, for each task the commits whose trailers name it, and the results of the validate and base check steps.

#### Scenario: Evidence written
- **WHEN** `dkt archive add-x` succeeds on 2026-11-02 and `add-x`'s title is "Booking waitlist"
- **THEN** `openspec/changes/archive/2026-11-02-add-x/evidence.md` exists, gives the title "Booking waitlist", and lists the commits for every task

### Requirement: No commit
`dkt archive` SHALL NOT commit, push or create branches. Its results SHALL stay in the working tree for the user to commit.

#### Scenario: Nothing committed
- **WHEN** `dkt archive add-x` succeeds
- **THEN** HEAD is the same commit as before and `git status` shows the archive's changes as uncommitted

### Requirement: Rebase
`dkt rebase <slug>` SHALL set the change's base to the full SHA of the merge base of HEAD and the default branch tip, where `docket-metadata` keeps it, changing no other line, and SHALL list every requirement the change touches whose main-spec block differs between the old and the new base. It SHALL NOT require the OpenSpec CLI.

#### Scenario: Rebase after merging the default branch
- **WHEN** the default branch was merged into the change's branch and its archived changes modified a requirement `add-y` touches
- **THEN** `dkt rebase add-y` moves `base` in `openspec/changes/add-y/docket.yaml` to the new merge base and lists that requirement

#### Scenario: Rebase a change made before docket.yaml
- **WHEN** `add-y` has no `docket.yaml` and its `.openspec.yaml` holds `x-base`
- **THEN** `dkt rebase add-y` moves `x-base` to the new merge base and creates no `docket.yaml`

#### Scenario: Nothing to move
- **WHEN** the merge base of HEAD and the default branch tip equals the change's current base
- **THEN** `dkt rebase add-y` changes nothing and exits with 0

### Requirement: Spec titles carried
After `openspec archive` succeeds, for every capability with a `docket.yaml` beside its delta, the spec title step SHALL set `title` in `openspec/specs/<capability>/docket.yaml` to that delta's title, creating the file if needed and changing no other line. A file it cannot read SHALL be listed as unreadable, with exit 3; any other failure SHALL be `spec-title-failed` for the capability, with exit 1. Either way the run stops.

#### Scenario: Title carried to a new spec
- **WHEN** `add-x` adds the capability `booking`, `openspec/changes/add-x/specs/booking/docket.yaml` holds `title: Booking`, and a user runs `dkt archive add-x`
- **THEN** `openspec/specs/booking/docket.yaml` holds `title: Booking`

#### Scenario: Title replaced on an existing spec
- **WHEN** `openspec/specs/booking/docket.yaml` holds `title: Bookings` and another line, and `add-x`'s delta for `booking` has a `docket.yaml` holding `title: Booking`
- **THEN** after `dkt archive add-x` that file holds `title: Booking` and its other line unchanged

### Requirement: Schema reported on archive
`dkt archive` SHALL report in `result.schema` the directory it found the schema in, as an absolute path, and whether it matches the shipped schema byte for byte. A copy that differs SHALL be reported and SHALL NOT stop the archive.

#### Scenario: Archive with a differing schema
- **WHEN** the per-machine `schema.yaml` differs from the shipped file and `add-x` passes every step
- **THEN** `add-x` is archived, `result.schema.matches` is false, the output says the copy differs, and the command exits with 0

### Requirement: Changes of another schema archived
For a change of another schema, `dkt archive` SHALL skip the plan check and the base check and run every other step in order, its lint step applying the rules lint applies to such a change. It SHALL NOT require a `plan.md`, a `docket.yaml` or a commit trailer for it.

#### Scenario: Change of another schema archived
- **WHEN** `add-x` has `schema: spec-driven`, a `tasks.md` and no `plan.md` or `docket.yaml`, is in the default branch tip, and a user runs `dkt archive add-x`
- **THEN** `add-x` is archived through `openspec archive`, its `evidence.md` names the schema `spec-driven` and says the plan and base checks were skipped, and the command exits with 0

#### Scenario: New capability without a title stops the archive
- **WHEN** the same `add-x` adds the capability `waitlist` with no `docket.yaml` beside its delta
- **THEN** `dkt archive add-x` stops at the lint step with `title-missing` for `waitlist`, before any OpenSpec command that writes

### Requirement: Rebase of a change of another schema
`dkt rebase <slug>` for a change of another schema SHALL report `other-schema`, naming the change's `.openspec.yaml` and its schema, change nothing, and exit with 1.

#### Scenario: Rebase refused for another schema
- **WHEN** `add-x` has `schema: spec-driven` and a user runs `dkt rebase add-x`
- **THEN** the findings hold `other-schema` naming `openspec/changes/add-x/.openspec.yaml`, no file changes, and the command exits with 1
