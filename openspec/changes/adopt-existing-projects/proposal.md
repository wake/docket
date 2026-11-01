# Proposal

## Why

A trial on a copy of an existing project (`docket/ideas/2026-10-31-existing-project-adoption.md`)
showed that Docket can look set up there and still not work. `dkt init`
reported ok, but git ignored the skills it installed. The changes the
project had already made with OpenSpec's `spec-driven` schema were linted
under Docket's rules and shown as `design 0/0 tasks`. Its main specs, which
had no titles, failed every `dkt lint`. A project should be able to use
Docket correctly from the day it adopts it, and keep its earlier work as
it is unless it chooses to convert it.

## What Changes

- **Ignored install paths.** `dkt init` and `dkt inspect` report
  `install-ignored` for each file they install or keep in the repository
  that git ignores, naming the ignore rule. Neither changes an ignore file.
  `docket-doctor` shows what changing the rule would let in, and edits the
  ignore file only after the user agrees.
- **Main spec titles.** `dkt inspect` reports `title-missing` for each main
  spec without a title, as `dkt lint` does. `docket-doctor` drafts every
  missing title from the spec's Purpose, lists them, and writes them only
  after the user agrees.
- **Changes made with another schema.** Docket reads `schema` in a change's
  `.openspec.yaml`. A change that names a schema other than `docket`:
  - is linted only for the titles of the capabilities it adds, and is
    listed by `dkt lint` as made with that schema;
  - is shown by `dkt status` with its schema, its phase and step progress
    taken from the checkboxes of `tasks.md`, and no tasks;
  - is archived by `dkt archive`, which skips the plan check and the base
    check for it and records that in `evidence.md`;
  - is reported by `dkt inspect` as `other-schema`, with its step progress;
  - is refused by `dkt sync` and `dkt rebase` with `other-schema`, since it
    has no plan or base for them to work on.
  `docket-doctor` proposes converting such a change to the `docket` schema
  only when none of its steps is ticked, and converts it only after the
  user agrees; a change with progress is left to finish in its own format.
- The `docket` skill tells an agent that a change of another schema follows
  that schema's files and is still archived with `dkt archive`.
- The README gets steps for adopting Docket in an existing project,
  including what to do with OpenSpec content the project kept out of git.

## Non-Goals

- `dkt` never writes `.gitignore`, `.git/info/exclude` or any other ignore
  file.
- No `dkt` command converts a change; conversion is the doctor's step, done
  by the agent with the user.
- No conversion is proposed for a change that has a ticked step, and no
  archived change is rewritten.
- `dkt conflicts` does not change: it already compares only delta specs.
- No survey of the project's code, interface list or capability draft
  (`2026-10-31-codebase-decomposition.md`, `2026-10-24-add-index.md`).
- `dkt init` does not detect or edit project rules that disagree with the
  workflow; `docket-doctor` already reports them.

## Builds On

- `project-setup` and `setup-inspection`: `dkt init` and `dkt inspect` decide
  every installed file through `internal/setup` (`Init`, `Inspect`,
  `Report`); the new findings are added there and mapped in
  `internal/cli/cmd_init.go` and `cmd_inspect.go`. git calls go through
  `internal/gitx` (`Git.Run`).
- `docket-metadata`: `internal/change` (`ReadMetadata`, `parseMetadata`)
  already reads `.openspec.yaml` for `x-` keys and `created`; the schema is
  one more key there.
- `change-lint` `Titles` and `internal/lint` (`checkTitles`,
  `checkSpecTitles`): the title rules are reused, both for the changes of
  another schema and for `dkt inspect`'s `title-missing`.
- `change-status` (`internal/status`: `Load`, `Change`, `Progress`): progress
  for a change of another schema is counted the way `Progress` counts steps.
- `change-archive` (`internal/archive`: `Archive`, the plan check, the base
  check, `evidence.md`): the steps stay in order; two are skipped by schema.
- `setup-inspection` doctor requirements and `.agents/skills/docket-doctor/`:
  the new findings get their own fixes there; `docket-init` already follows
  the doctor, so it needs no new step.
- Searched `rg -n 'check-ignore|schema' internal --type go`: nothing reads
  a change's `schema` or asks git whether a path is ignored.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `docket-metadata`: a change's schema is read from `.openspec.yaml`.
- `project-setup`: `dkt init` reports installed files that git ignores.
- `setup-inspection`: `dkt inspect` reports ignored files, main specs
  without titles and changes of another schema; the doctor handles them.
- `change-lint`: a change of another schema is linted only for the titles
  of the capabilities it adds.
- `change-status`: a change of another schema is shown with its schema and
  `tasks.md` progress.
- `change-archive`: a change of another schema is archived without the plan
  and base checks, and `dkt rebase` refuses it.
- `plan-sync`: `dkt sync` refuses a change of another schema.

## Impact

- `internal/change`, `internal/setup`, `internal/gitx`, `internal/lint`,
  `internal/status`, `internal/archive`, `internal/cli`.
- `dkt lint --json` gains `result.other_schema`; `dkt status --json` gains
  `schema` on each change; `dkt inspect` and `dkt init` gain findings.
- Prompts: `docket-doctor` (both copies, and a new conversion guide beside
  it), the `docket` skill (both copies). README in both languages,
  `docs/ARCHITECTURE.md`.

## Implementation Contract

### Observable Behavior

- In a repository whose `.gitignore` has `.claude/`, `dkt init` writes the
  skills as before, then reports `install-ignored` for each file under
  `.claude/skills/`, naming `.gitignore` and the line of the rule, and
  exits with 1.
- In a project with `openspec/specs/booking/spec.md` and no title beside
  it, `dkt inspect` reports `title-missing` naming
  `openspec/specs/booking/docket.yaml`.
- For an active change `add-x` whose `.openspec.yaml` says
  `schema: spec-driven`, with `tasks.md` holding three checkboxes of which
  one is ticked: `dkt lint` lists it as made with `spec-driven` and reports
  no proposal or plan finding for it; `dkt status` shows `schema:
  spec-driven`, phase `implementing` and 1 of 3 steps; `dkt archive add-x`
  archives it once merged.

### Interfaces

- Findings: `install-ignored` (`dkt init`, `dkt inspect`), `title-missing`
  (`dkt inspect`), `other-schema` (`dkt inspect`, `dkt sync`, `dkt rebase`).
  `install-ignored` names the file in `file` and the rule in its message,
  as `ignored by <source>:<line>: <pattern>`.
- `dkt lint --json`: `result.other_schema`, a list of `{slug, schema}`.
- `dkt status --json`: `schema` on every change (`docket` for Docket's own).
- `evidence.md`: the change's schema, and the checks skipped for it.
- Doctor: a conversion guide `convert.md` in the `docket-doctor` folder.

### Failure Modes

- git cannot answer whether a path is ignored: the paths are listed as
  unreadable, and nothing else changes.
- A `.openspec.yaml` whose `schema` is not a string: the change is listed as
  unreadable by the command reading it.
- A change of another schema that adds a capability without a title beside
  its delta: `dkt lint` reports `title-missing`, and `dkt archive` stops at
  its lint step, so no main spec is left without a title.

### Acceptance Criteria

- `docket-metadata#schema-read`, `docket-metadata#schema-missing`.
- `project-setup#ignored-skill-reported`,
  `project-setup#tracked-file-not-reported`, `project-setup#nothing-ignored`.
- `setup-inspection#ignored-file-inspected`,
  `setup-inspection#main-spec-title-inspected`,
  `setup-inspection#change-of-another-schema-inspected`,
  `setup-inspection#ignore-rule-changed-with-consent`,
  `setup-inspection#titles-written-with-consent`,
  `setup-inspection#unstarted-change-converted`,
  `setup-inspection#started-change-left`.
- `change-lint#change-of-another-schema`,
  `change-lint#new-capability-of-another-schema`,
  `change-lint#change-of-another-schema-has-no-title`.
- `change-status#change-of-another-schema-shown`,
  `change-status#no-tasks-file`, `change-status#docket-change`.
- `change-archive#change-of-another-schema-archived`,
  `change-archive#new-capability-without-a-title-stops-the-archive`,
  `change-archive#rebase-refused-for-another-schema`.
- `plan-sync#sync-refused-for-another-schema`.

## Change Log

| Date | Source | Change | Impact |
|---|---|---|---|
