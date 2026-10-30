# Design

## Context

See proposal.md for why. What exists today:

- `dkt new` validates a slug (`change.ValidateSlug`), scans every
  worktree, local and remote-tracking branch and the default branch
  (`scan.Scan`), and refuses a slug that any of them uses, active or
  archived (`Inventory.Uses`). It holds a lock on `openspec/` while it
  creates the folder, so two runs in one worktree cannot interleave.
- `change.Kinds` is `feature`, `change-request`, `fix`, `baseline`; only
  `dkt new` checks the kind.
- `dkt lint` lints change folders and main spec titles. It already knows
  the files the branch adds under its routed folders (`addedFiles`,
  `lint.Input.Added`). It has the plan's task heading rules and a
  placeholder rule fed by the shipped schema templates.
- The workflow template's light review lists five kinds of edit: a fix, a
  restructuring, prompt text, tests, check settings.
- `docket/ideas/2026-10-29-records.md` was written before the framework
  split moved each path's review and verification steps into the user's
  workflow file.

## Goals / Non-Goals

**Goals:**

- A record for every piece of light-review work, short enough that
  writing it is cheap and reading it is quick.
- An agent never has to think about whether a name is taken: `dkt` picks
  the slug and says which one it picked.
- `dkt lint` holds records to their shape, as it holds changes to theirs.

**Non-Goals:**

- Tracking: `dkt status`, `dkt sync` and the `Record:` trailer readers
  come in the second change.
- Classifying work by origin (`docket/ideas/2026-10-29-ideas-and-order.md`
  holds that question).
- How a workbench shows records.

## Approaches

The maintainer chose B.

- **A. One change with everything.** Records would be complete when it
  merged, but it would be one pull request of roughly 700 lines of product
  code and 1500 of tests, and any part could hold the rest.
- **B. Two changes (chosen).** This one makes records exist: the file, the
  command, the naming rule for records and changes, lint, the skill text.
  The second makes them tracked: scan reads their content, `dkt sync`
  ticks their tasks, `dkt status` reports them. Each can be accepted on its
  own; between them, records are files that `dkt status` does not list.
- **C. A record as a folder** (`docket/records/<slug>/record.md`), to reuse
  the change folder machinery. Rejected: the idea settled on one file per
  piece of work, and a folder per small fix is a burden on people.

## Decisions

### Naming: `dkt` picks a free slug

The maintainer asked that an agent never retry names. `dkt new` and
`dkt record` take a name; when its slug is used they take the first free
suffix, `-2`, `-3`, ..., and report the slug they created. The suffix form
is the maintainer's stopgap and may be replaced.

Considered with the maintainer:

- **Numbers (`001-`).** Rejected: picking "highest plus one" collides
  between parallel sessions. spec-kit, which does this, has a series of
  duplicate-number issues, including AI assistants passing numbers against
  instructions.
- **A random suffix on every slug** (as reno and Kubernetes `generateName`
  do). Rejected for readability: every slug, command and folder name would
  carry it, to prevent a collision that is rare at this scale and that
  surfaces as a merge conflict anyway.

A record's slug is today's date, a hyphen and the name. Records are never
archived and accumulate, so the date sorts them in the folder, as idea
files are. A change's slug carries no date because OpenSpec adds one when
it archives. Change slugs may not start with a date, so a record slug can
never equal a change slug: each command checks only its own kind, and a
name starting with a date on `dkt lint`'s command line names a record.

The date is `Context.Now` formatted as `2006-01-02` in the location the
time carries. `cmd/dkt` hands in the local time, so a record belongs to the
user's day (my call); tests hand in a fixed time.

When a suffix is taken, the result says so:
- `result.slug` is the created slug;
- `result.requested` is the given one;
- `result.used` is a list of strings, one per place the given slug is used,
  worded as `dkt new`'s refusal names them today ("active in <sources>",
  "archived as <folder> in <sources>"; for a record, "in <sources>"). It is
  empty when no suffix was needed.

The human output adds one line, `<requested> is used: <places>; created
<slug>` (my call). Neither is a finding, so the exit code stays 0.

`dkt new` keeps its lock on `openspec/` and tries each candidate under it.
`dkt record` creates the file with `O_CREATE|O_EXCL`; on "exists" it
tries the next suffix, so two runs in one worktree end with two records.
A candidate is checked against every scanned source before either
command writes.

### The scan lists records

`scan.Scan` already reads every source's tree: worktrees, local and
remote-tracking branches, the default branch. It also lists the `.md` files
directly under `docket/records/` in each of them, by name only, so
`dkt record` can tell a slug is used. The second change reads their
content.
- `Inventory` gains `Records` (slug and the sources holding it) and a
  method that, like `Uses`, returns where a record slug is used (my call).
- A missing `docket/records` is an empty list, not an unreadable source.
- An unreadable source makes `dkt record` exit with 3, as for `dkt new`.

### One template, shared sections

All five kinds share `## What`, `## Why`, `## Verification` and
`## Review` (the maintainer's choice, to keep it simple).

**The template**
- It is one file, embedded from `record/template.md` as
  `workflow/template.md` is.
- It holds the body: one comment on the word limit, the four headings,
  and under each heading one HTML comment.
- The comment names what each kind writes there: for a fix, the symptom
  and its cause under Why; for a refactor, the tests that show behavior did
  not change under Verification.
- `dkt record` writes the front matter itself, then the template, comments
  included. Kind changes nothing else (my call: one file, not five
  variants).

**What the sections say**
- `Verification` and `Review` say what was done. Which steps to do comes
  from the user's workflow file.

**Lint**
- The placeholder rule reports a template comment left in a record, as it
  does in a change. Its source is the embedded record template, passed to
  lint beside the schema templates.
- An HTML comment is not text for the "section filled" rule. A section
  holding only the template comment therefore gets both `placeholder` and
  `record-section`. They are two problems: the hint was left, and nothing
  was written (my call).

**Kinds**
- The five record kinds live in `internal/record`.
- `change.Kinds` loses `fix`. The two lists are separate, and neither
  package imports the other for them (my call).

### Word limits per section

The maintainer asked for limits per section, not per file, and for detail
to live in the code and the pull request. The limit is 60 words for each
of the four sections (my call; the number is easy to change). A run of
letters or digits is a word; each Chinese, Japanese or Korean character is
half a word, so a section holds about 120 such characters. HTML comments
do not count. `## Tasks` has no limit.

### Tasks are optional and checked when present

A large record (a refactor in several steps, a wide prompt rewrite) may
list tasks under `## Tasks`, in the plan's format, as `###` headings inside
that section. Only `##` headings count for the section rule.

**Lint**
- `dkt lint` applies the plan's task heading rules (`task-heading`), with
  line numbers in the record file.
- It does not apply the `**Scenarios:**` or `**Review:**` rules.
- The heading check in `internal/lint` is shared, not copied: it takes the
  parsed tasks of either a plan or a record (my call).

**Commits**
- A task's commits carry `Record: <slug>` and `Task: <N>`.
- Reading those trailers is the second change's work.

### Lint scope: the records the branch adds

`dkt lint` with no names lints the records the branch adds: committed
since the merge base with the default branch, plus uncommitted. A merged
record is history, and a later, stricter rule should not fail it.

**How the added records are found**
- The added-files query runs for `docket/records/` separately from the
  routed folders, so the routing rule never sees records (my call).
- With no default branch, lint reports that it cannot scope records, as it
  does for routing today.

**Named arguments**
- A name starting with a date is looked up as `docket/records/<name>.md`
  in the worktree, whatever its state.
- A missing one gives `record-missing`, worded like the finding for a
  missing change.

**The result**
- `LintResult` gains `records`, the record slugs linted, beside `changes`
  (my call).

### The kind `fix` leaves `dkt new`

A fix restores a known answer, so it is light-review work and gets a
record; a bug whose right behavior cannot be stated sets an expectation and
is a `change-request`. No archived change used `fix`, and nothing but
`dkt new` checks a change's kind.

### The skill keeps records on the light review path

The `docket` skill's light review section says:
- open the record with `dkt record` before the first commit, and use the
  slug it reports;
- fill the four sections as the work and its review go;
- give every commit for it `Record: <slug>`, plus `Task: <N>` for a
  record's task, in the final paragraph;
- run `dkt lint` before the pull request.

The check and direct merge paths keep no record.

## Risks / Trade-offs

- [An agent writes the name it gave instead of the slug `dkt` reported] →
  the skill says to use the reported slug; the second change can flag
  `Record:` trailers that name no record.
- [Two machines pick the same name before either pushes] → each creates
  the same path, and the second merge has a conflict on it; renaming one
  is manual, and its trailers keep the old name. Rare at this scale.
- [Word counting is approximate for mixed scripts] → the rule is simple
  and documented; the limit is a number in one place.
- [Records accumulate in one folder] → the date prefix keeps them in
  order; a later change can move old ones if the folder grows large.

## Migration Plan

- No project outside this repository uses Docket, and this repository has
  no records or `fix` changes, so nothing migrates.
- The prompt change is verified by a trial task the lead runs, as in
  earlier changes: a scratch project, headless Claude Code on two models,
  a light-review prompt, and a check that `dkt record` runs before the
  first commit and that the commits carry `Record:`.
- Rollback: revert the merge.
