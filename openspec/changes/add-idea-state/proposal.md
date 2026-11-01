# Proposal

## Why

Ideas and `docket/order.md` carry their state by hand: the maintainer ticks
an order line in the release pull request, and nothing says which ideas are
in progress, finished or given up. Issues already derive their state from the
records and changes whose source names them; ideas can do the same, so that
no worktree writes a file every change shares and two changes made in
parallel do not meet in `docket/order.md`.

## What Changes

- An idea's state is derived, not written. Its handlers are the records and
  changes whose source names its path, found as an issue's are. The state is
  `dropped`, `pending` (no handler), `in-progress` or `done` (every handler
  on the default branch).
- `docket/order.md` loses its checkboxes. A line is `N. ` and one path in
  backticks; whether it is done comes from its idea. **BREAKING** for this
  repository's `docket/order.md` and for the sentence in `AGENTS.md` that
  reads "the first unticked line"; this change rewrites both.
- `dkt status` reports the order (`result.order`, each line with its idea's
  state and the needed ideas it still waits on), the next piece of work
  (`result.next`) and the ideas outside the order that are pending or in
  progress (`result.ideas`). Ideas and the order are read from the default
  branch tip.
- `dkt lint`, with no name given, checks `docket/order.md` (`order-format`,
  `order-idea`) and every idea's front matter (`idea-front-matter`), and
  reports a change or record whose source names an idea that does not exist
  (`idea-source`).
- A source names the path in its first field, so `docket/ideas/x.md, item 1`
  and `docket/ideas/x.md (order 5)` name `docket/ideas/x.md`. Issues take the
  same rule.
- The `docket` skill: a change's or record's worktree does not edit ideas or
  the order; work that takes an idea passes the idea's path alone as its
  source; one idea is one piece of work; the next piece of work is the one
  `dkt status` reports.

## Non-Goals

- No classification fields (origin, nature, domain). Issues took fixes,
  optimizations and tests out of the ideas; domains stay with
  `docket/ideas/2026-10-30-spec-domains.md`.
- No `to:` field, and no command writes an idea or the order: `dkt sync`
  does not tick the order and has no order check.
- No command that creates an idea or edits the order.
- No check that a line's needed ideas come earlier in the order.
- No reading of ideas or the order on branches or in worktrees for
  `dkt status`: an edit counts once the default branch holds it.
- No edits to the sources of archived changes or merged records.
- No release version per idea (`git tag --contains`).

## Builds On

- `issues` (Issue state, Issue sources checked) and
  `internal/status/issues.go:LoadIssues`: handlers from the sources of
  records and changes, read from the tip or the first source; this change
  shares that pass instead of writing a second one.
- `internal/lint/issue.go:checkIssueSource` and `run.isFile`: a source under
  a folder that names no regular file.
- `docket-metadata` (Title form) and `internal/change:ValidTitle`,
  `change.DecodeMapping`; `internal/record:Split` for front matter.
- `change-lint` (What is linted, Titles): lint already checks a whole-repo
  file set, the main spec titles, whenever no name is given.
- `workspace-scan` and `internal/scan:ResolveDefaultBranch`: the default
  branch tip as a source.
- Searched `internal/` and `openspec/specs/` for `docket/ideas`,
  `order.md`, `ideas/`: only the issue lint reads an idea path; nothing reads
  the order.

## Capabilities

### New Capabilities
- `ideas`: idea files and `docket/order.md`, their lint, how an idea's
  state is derived, how `dkt status` reports the order and the ideas, and
  the `docket` skill's rules for them.

### Modified Capabilities
- `issues`: an issue's handlers and the issue source check use the path a
  source names.
- `docket-metadata`: the path a source names.
- `change-lint`: with no name given, lint also checks the order and the
  ideas.

## Impact

- `internal/status`: idea state and the order, sharing the handler pass with
  issues; `internal/cli` status result and text lines.
- `internal/lint`: the order, idea front matter, idea sources; the issue
  source check takes the source's path.
- A package for idea files and the order line format.
- Prompt text: `.claude/skills/docket/SKILL.md` and
  `.agents/skills/docket/SKILL.md`, shipped by `dkt init`.
- Docs: `README.md`, `README.zh-TW.md`, `docs/ARCHITECTURE.md`; this
  repository's `AGENTS.md` and `docket/order.md`.

## Implementation Contract

### Observable Behavior

- `dkt status --json` in this repository lists lines 1 to 9 of the order as
  `done`, line 10 as `in-progress` with the change `add-idea-state`, and
  `result.next` as 10; the text output has a line per order line not done
  and one naming the next.
- `dkt lint` fails on an order line with a checkbox, a gap in the numbers, a
  path to a missing or dropped idea, or the same idea twice, and on an idea
  whose front matter lacks a title.

### Interfaces

- An idea such as `docket/ideas/2026-11-02-x.md`: front matter `title`,
  optional `dropped: true`, optional `needs:` (a list of idea paths); then
  free text.
- `docket/order.md`: `# Order`, then lines such as
  ``10. `docket/ideas/2026-10-29-ideas-and-order.md` ``, numbered from 1.
- `dkt status --json`: `result.order[]` (`number`, `path`, `title`, `state`,
  `handlers`, `waiting_on`), `result.next` (a line number or null),
  `result.ideas[]` (`path`, `title`, `state`, `handlers`). A handler is
  `{kind, slug, merged}`, as for issues.
- Lint rules: `order-format`, `order-idea`, `idea-front-matter`,
  `idea-source`.

### Failure Modes

- No `docket/order.md` or no `docket/ideas/` at the tip: `result.order` and
  `result.ideas` are empty, `result.next` is null, and nothing is reported.
- An order line that does not parse is skipped by status and reported by
  lint.
- An order line whose idea is missing at the tip has state `missing` and a
  null title; `result.next` skips it and dropped lines.
- A record or change that cannot be read is listed as unreadable and the
  command exits with 3, as for issues.
- `docket/order.md` or an idea that is a symbolic link is not followed: lint
  reports it and status does not read it.

### Acceptance Criteria

- `ideas#idea-file-read`, `ideas#order-file-read`: the two formats.
- `ideas#order-line-with-a-checkbox`, `ideas#gap-in-the-order`,
  `ideas#order-names-a-missing-idea`, `ideas#same-idea-twice`,
  `ideas#dropped-idea-in-the-order`, `ideas#linked-order-file`: lint checks
  the order.
- `ideas#idea-without-a-title`, `ideas#need-names-no-idea`: lint checks idea
  front matter.
- `ideas#source-names-no-idea`: lint checks idea sources.
- `ideas#idea-pending`, `ideas#handler-on-a-branch`, `ideas#every-handler-merged`,
  `ideas#one-handler-of-two-merged`, `ideas#dropped-idea`,
  `ideas#idea-on-a-branch-only`: the derived state.
- `ideas#order-listed`, `ideas#next-skips-done-dropped-and-missing`,
  `ideas#waiting-on-a-need`, `ideas#no-order-file`,
  `ideas#ideas-outside-the-order`, `ideas#order-not-listed-with-slugs`,
  `ideas#unparsed-order-line`, `ideas#order-in-the-text-output`: status
  reports them.
- `ideas#change-takes-an-idea`, `ideas#idea-found-during-a-change`,
  `ideas#idea-with-a-further-part`, `ideas#agent-picks-the-next-work`: the
  skill's rules.
- `docket-metadata#path-alone`, `docket-metadata#path-with-a-note`: the path
  a source names.
- `issues#handler-with-a-note`, `issues#source-with-a-note-names-no-issue`:
  issues take that path.
- `change-lint#order-and-ideas-linted`, `change-lint#order-not-linted-with-names`:
  what lint reads.

## Change Log

| Date | Source | Change | Impact |
|---|---|---|---|
