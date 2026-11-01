# Proposal

## Why

Every piece of pending work in a Docket project is written as an idea, so
work that needs design sits with fixes, optimizations and tests whose
expected behavior is already settled. In this repository about half of the
fifty ideas are such work, and requests from other people have no place at
all. Issues give that work its own form: in the tool a project names, or as
files when it names none.

## What Changes

- An issue is work whose expected behavior is already settled and that is
  not done now: a fix that restores specified behavior, an optimization, a
  test, a request from someone else. It pairs with a record, as an idea
  pairs with a change: the issue says what is wrong, the record what was
  done.
- Docket ships an issue template with four sections: `## Current`,
  `## Expected` (with where the expectation is settled), `## Unchanged` and
  `## Found`. The same sections go into an issue in an outside tool and into
  a file issue.
- A project names its outside issue tool, when it has one, in a
  `## Issues` section of its `AGENTS.md`. Without one, issues are files
  `docket/issues/<date>-<name>.md`.
- `dkt issue <name> --title <text> --source <text>` writes a file issue from
  the template; `dkt issue template` prints the template's sections.
- `dkt lint` checks the file issues the branch adds (front matter, sections,
  word limits, template text left) and reports a record or change whose
  source names a file issue that does not exist.
- `dkt status` lists the file issues that are pending or in progress. The
  state is derived from the records and changes whose source names the
  issue; only `dropped: true` and `idea: <path>`, for an issue that will not
  be done or that became an idea, are written by people.
- The `docket` skill sends work found but not done now to an issue, an issue
  whose expectation cannot be pointed to to an idea, and has a record for an
  issue name it as its source.

## Non-Goals

- `dkt` does not read, write or call an outside issue tool; filing an issue
  there is the agent's step, with the user's agreement.
- No forge template file such as `.github/ISSUE_TEMPLATE/` is installed, and
  `dkt init` writes nothing new.
- No user-level setting for an issue tool: the workflow file, its template
  and `docket-setup` do not change.
- `dkt status` does not list handled issues, and neither `dkt status` nor
  `dkt lint` takes an issue's slug as an argument: a dated name still names
  a record.
- Issues have no kind and no tasks.
- The ideas in this repository that are issues are not moved here; that
  follows once `dkt issue` has shipped.

## Builds On

- `records` and `internal/record` (`Create`, `Words`, `Parse`,
  `ValidateName`, `Slug`): a dated slug never reused across scanned sources,
  no write through a link, nothing left after a failed write, section and
  word checks. File issues follow the same rules and reuse that code.
- `workspace-scan` `Records found`: file issues are found in the same
  sources, in the same way.
- `change-lint` `What is linted` and `Routing`: the issues a branch adds are
  linted as its records are.
- `docket-metadata` `Title form` for an issue's title, and the `source` of a
  change's `docket.yaml` and of a record.
- `materialized-workflow` `dkt workflow template`, the model for
  `dkt issue template`.
- Searched `rg -n -i '\bissues?\b' internal cmd schema.go openspec/specs`:
  no issue concept exists; `issue 12` appears only as example `source` text.

## Capabilities

### New Capabilities

- `issues`: the issue template, file issues and `dkt issue`, how `dkt lint`
  checks them, how `dkt status` lists them, and how the `docket` skill uses
  issues.

### Modified Capabilities

- `workspace-scan`: a scan also finds file issues.
- `change-lint`: lint also covers the file issues the branch adds.

## Impact

- New command `dkt issue`, a new embedded template `issue/template.md`, and
  file issues under `docket/issues/`.
- `internal/record` code shared with issues; `internal/scan`,
  `internal/lint` and `internal/status` read issues.
- `dkt status` JSON gains `result.issues`; existing fields do not change.
- The `docket` skill in `.claude/skills/` and `.agents/skills/`, the README
  in both languages, and `docs/ARCHITECTURE.md`.

## Implementation Contract

### Observable Behavior

- `dkt issue lint-timeout --title "Lint timeout" --source "review of #40"`
  on 2026-11-02 writes `docket/issues/2026-11-02-lint-timeout.md` with that
  front matter and the template's four sections, and reports its slug.
- `dkt issue template` prints the four sections an agent pastes into an
  outside tool.
- `dkt status` shows each file issue that is pending or in progress, with
  the records and changes that name it.
- `dkt lint` reports a file issue the branch adds that is missing a section,
  is too long, or still holds template text.

### Interfaces

- `dkt issue <name> --title <text> --source <text> [--json]`; result
  `slug` and `path`.
- `dkt issue template [--json]`; result `text`.
- File issue: front matter `title`, `source`, optional `dropped: true`,
  optional `idea: docket/ideas/<file>.md`; sections `## Current`,
  `## Expected`, `## Unchanged`, `## Found`.
- `dkt status --json`: `result.issues`, each with `slug`, `title`,
  `source`, `state` (`pending` or `in-progress`), `handlers` (each with
  `kind` `record` or `change`, `slug` and `merged`) and `sources`.
- Lint findings `issue-front-matter`, `issue-section`, `issue-too-long`,
  `placeholder`, `issue-source`; `dkt lint --json` lists the linted issues
  in `result.issues`.
- A record or change names a file issue when its source is the issue's
  path, `docket/issues/<slug>.md`.

### Failure Modes

- A missing or invalid flag, or a name with a date, writes nothing and
  exits with 2 or 1, as `dkt record` does.
- An unreadable scanned source makes `dkt issue` write nothing and exit
  with 3; a link at `docket`, `docket/issues` or the issue's path makes it
  write nothing and exit with 1.
- A failed write leaves no partial file and no folder the run created.
- An issue whose front matter cannot be read is listed with null fields.

### Acceptance Criteria

- `issues#issue-written`, `issues#missing-source-writes-nothing`,
  `issues#same-name-same-day`, `issues#symlinked-issues-folder-refused`,
  `issues#failed-write-leaves-nothing`: file issues are written as records
  are.
- `issues#template-printed`: the template prints.
- `issues#issue-section-empty`, `issues#issue-section-over-the-limit`,
  `issues#issue-front-matter-invalid`, `issues#source-names-no-issue`: lint
  checks file issues and the sources that name them.
- `issues#pending-issue-listed`, `issues#issue-in-progress`,
  `issues#handled-issue-hidden`, `issues#dropped-and-moved-hidden`,
  `issues#issues-not-listed-with-slugs`: status derives the state.
- `issues#bug-found-out-of-scope`, `issues#issue-in-the-project-tool`,
  `issues#record-names-its-issue`: the skill's rules.
- `workspace-scan#issue-on-a-branch`,
  `change-lint#lint-issues-the-branch-adds`: issues are found and linted.

## Change Log

| Date | Source | Change | Impact |
|---|---|---|---|
| 2026-11-01 | plan review (codex) | A handler the default branch tip holds is read from the tip; two state scenarios added; the command flow, the YAML decoder and the front-matter framing shared with records | Tasks 1 and 5 grow; no new task |
| 2026-11-01 | pre-merge review (codex, fresh-context spec review) | Only the tip's copies of a change the tip holds count as handlers; the READMEs and the architecture notes say what lint and status do with issues; `dropped: yes` noted in the design | One fix commit each to Tasks 5 and 6; no new task |
