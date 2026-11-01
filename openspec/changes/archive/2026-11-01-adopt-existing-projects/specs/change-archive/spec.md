## ADDED Requirements

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

## MODIFIED Requirements

### Requirement: Evidence
After the spec title step, `dkt archive` SHALL write `evidence.md` in the archived change folder, recording the date, the Docket and OpenSpec versions, the URL of the `origin` remote or none, the HEAD SHA, the change's title or none, its kind, source and base, its schema, the steps skipped for it, for each task the commits whose trailers name it, and the results of the validate and base check steps.

#### Scenario: Evidence written
- **WHEN** `dkt archive add-x` succeeds on 2026-11-02 and `add-x`'s title is "Booking waitlist"
- **THEN** `openspec/changes/archive/2026-11-02-add-x/evidence.md` exists, gives the title "Booking waitlist", and lists the commits for every task
