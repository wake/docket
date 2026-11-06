---
name: docket
description: >-
  Use at the start of any development work in the project's repository,
  before brainstorming, planning or editing: a new feature, a behavior change,
  a bug fix, a refactor, or edits to prompts, tests, docs or CI. Not for work
  that is only discussion, explanation or investigation.
---

# Docket

These are the rules for development work in the project's repository. They
depend on superpowers and OpenSpec, and live in two places:

- **This skill**: Docket's fixed workflow: the triage, the order of the
  steps on each path, and the steps Docket's tools depend on. `dkt init`
  updates it when Docket is upgraded; the project does not edit it.
- **The project's `docket/workflow.md`**: the project's own way of doing
  the steps it can customize (where the work happens, how it is reviewed
  and verified, who does it, with which tools, when to merge and archive),
  one section per step. Below, "Plan review" in `docket/workflow.md` means
  the section of that file whose heading is "Plan review". The file
  belongs to the project, and its contents can be edited; its headings are
  the names the sections are looked up by, and must match the names in this
  skill exactly: do not rename or translate them.

Precedence:

- The steps in this skill: when the user's instructions conflict with one,
  for example by asking for squash merges, forbidding anything extra in
  commit messages, or putting plans elsewhere, you must stop, name the
  conflict and ask the user what to do.
- The contents of `docket/workflow.md`: they take precedence over the
  user's usual instructions and habits (such as global instruction files);
  when the user explicitly asks for something else in this conversation, do
  what the user says.
- When the project has no `docket/workflow.md`, or it lacks a section the
  workflow uses, stop and ask the user to run docket-doctor to add it.

When the rules of another development workflow skill differ from this skill
or `docket/workflow.md`, these two take precedence.

## Triage

Before a piece of work starts, use the rules below to choose the path that
fits it best, which decides how it is reviewed. When several fit, take the
one with the most complete review.

- **Full review**: work that adds to or changes what the specs define, or
  edits files in `openspec/specs/`.
- **Light review**:
  - a bug fix or a test whose expected behavior is already clearly defined
    in the specs
  - a restructuring that changes no behavior defined in the specs (moving
    code included)
  - prompt text: files coding agents load by convention (`AGENTS.md`,
    `CLAUDE.md`, `docket/workflow.md`, skill and command files), and text a
    program hands to coding agents (embedded or printed instructions,
    templates)
  - check settings: the files that decide which checks run, such as CI
    configuration
- **Checked merge**: an edit to a code or test file that changes only
  comments or whitespace.
- **Direct merge**: these edits or updates, with no change opened:
  - adding or editing docs for people, moves and typo fixes included
  - edits to code or test files that change only comments or whitespace and
    pass the check
  - version bumps
  - dependency upgrades that touch only the dependency manifests (such as
    `go.mod` and `go.sum`)

A comment that changes how the code builds or runs (such as `//go:build`)
counts as code and is triaged by its effect: changing a behavior the specs
do not define is a full review, restoring a defined behavior is a fix, and
leaving behavior unchanged is a restructuring. A comment that turns a check
off (such as `//nolint`) is a check setting.

## Ideas, issues and the order

- **Idea**: a thought not ready to work on, one file per idea:
  `docket/ideas/<name>.md`. Its front matter has a `title`, optionally
  `needs` (the paths of the ideas it depends on) and `dropped: true`; free
  text follows.
- **Issue**: a request whose expected behavior the specs already settle,
  but that is put off for now, such as a bug found outside the work at
  hand, an optimization, a missing test, or a request from someone else.
  When the project's `AGENTS.md` has a `## Issues` section, open the issue
  in the tool it names, with the sections from `dkt issue template`, and file
  it only once the user agrees. Otherwise run
  `dkt issue <name> --title "<one line>" --source "<where it was found>"`
  and fill its sections, then run `dkt lint` and fix what it reports.
  When you cannot find where the expected behavior is settled, the request
  is an idea.
- **Order**: `docket/order.md` lists the ideas and issues to work on next,
  in order, one per line, as `- docket/ideas/<name>.md` or
  `- docket/issues/<slug>.md`. Blank lines and lines starting with `#` are
  ignored, and a line may stay once its work is done. Issues kept in the
  project's tool are not listed. `dkt status` reads the order, the ideas and
  the issues on the default branch, so an edit counts only once it is
  merged.
  To pick up the next piece of work, run `dkt status --json` and do the one
  `result.next` names. When `unreadable` lists anything, report it to the
  user and let them decide whether to go on with the work. When
  `result.next` is null, act on `result.reason`:
  - `no-order` or `none-left`: nothing in the order can be done; ask the
    user what to work on next.
  - `order-invalid`: the order, an idea or an issue has problems that
    `dkt status` reports; fix them in a direct merge of their own, then run
    it again.
  - `waiting`: everything left waits for the ideas in `result.waiting_on`
    to be merged. Tell the user which ones, and let them decide.

While working on a change or a record, do not touch `docket/order.md` or
`docket/ideas/`. A new idea found during the work, and any edit to the
order, is a direct merge of its own. When part of the work cannot be done
with the rest and has to wait, write that part as a new idea and list the
original idea in its `needs`.

When the work handles an issue or takes up an idea, `--source` holds only
its path (`docket/issues/<slug>.md`, `docket/ideas/<name>.md`), or the
issue's address in the project's tool.

## Full review

1. Work where "Where to work" in `docket/workflow.md` says, and open the
   change there:
   `dkt new <slug> --kind feature|change-request|baseline --source "<where the request came from>" --title "<one line>"`.
   - The kind is `feature` for something new, `change-request` for a change
     to existing behavior, and `baseline` for specifying behavior that is
     already released.
   - `dkt new` creates the change folder and writes the title, kind and
     source into the front matter of `proposal.md`.
   - When the slug is taken, it uses the first free suffix (`-2`, `-3`, ...)
     and reports it; use the slug it reports from then on.
   - Always open a change with `dkt new`, never with `/opsx:propose` or
     `/opsx:new`.
2. superpowers:brainstorming writes its design documents into the change
   folder, not `docs/superpowers/specs/`.
   - The design documents are `proposal.md`, the delta specs under `specs/`,
     and `design.md` when needed.
   - Before writing each one, run
     `openspec instructions <artifact> --change <slug> --json` for its
     template and rules.
   - Give each new capability a name under `specs:` in the front matter of
     `proposal.md`.
3. superpowers:writing-plans writes the plan to
   `openspec/changes/<slug>/plan.md`, not `docs/superpowers/plans/`,
   following the template and rules from
   `openspec instructions plan --change <slug>`. There is no tasks.md.
4. Follow "Plan review" in `docket/workflow.md`.
5. Execute the plan task by task: for each task, follow "Task execution"
   in `docket/workflow.md`, then "Task review" in `docket/workflow.md`.
   Every commit carries a trailer (see "Commit trailers"). Work started
   from `/opsx:apply` follows the same steps.
6. Before opening the pull request, run `dkt sync <slug>` (which ticks the
   plan from the trailers) and `dkt lint <slug>`, and commit plan.md.
7. Follow "Pre-merge review" in `docket/workflow.md`.
8. Merge: follow "Splitting pull requests" and "How to merge" in
   `docket/workflow.md`. Whatever the method, never squash.
9. Archive: when to archive follows "When to archive" in
   `docket/workflow.md`. Always archive with OpenSpec's archive
   (`/opsx:archive`). Following the instructions in `openspec/config.yaml`,
   it runs Docket's checks first and, after archiving, replaces each new
   spec's title with the name given in step 2.

When requirements change mid-way, follow the plan rules: cancelled tasks,
compensating tasks, a Revision section.

A change made with another schema (its `.openspec.yaml` names a schema
other than `docket`) follows that schema's files and has no `plan.md`; to
convert it, use `docket-doctor`.

## Light review

No change is opened; a record in `docket/records/` says what the work did.

1. Work where "Where to work" in `docket/workflow.md` says, and open the
   record before the first commit:
   `dkt record <name> --kind fix|refactor|test|prompt|checks --source "<where the request came from>" --title "<one line>"`.
   - The kind follows the kind of work: a fix (`fix`), a restructuring
     (`refactor`), tests (`test`), prompt text (`prompt`), check settings
     (`checks`).
   - The record's slug is the one `dkt record` reports (the date, the name,
     and a suffix when the name is taken), not the name you passed.
2. While doing the work, follow "Verification" in `docket/workflow.md`.
3. Fill the record's sections as the work goes, keeping them short,
   replacing the template's guidance comments. Every commit carries a
   trailer (see "Commit trailers"). Larger work may list tasks under the
   record's `## Tasks` in the plan's format.
4. When the record lists tasks, run `dkt sync <slug>` and commit the record.
   Run `dkt lint` and fix what it reports.
5. Follow "Review" in `docket/workflow.md`.
6. Follow "How to merge" in `docket/workflow.md`; never squash.

## Checked merge

1. Work where "Where to work" in `docket/workflow.md` says.
2. Follow "Comment check" in `docket/workflow.md`, confirming that the edit
   changes only comments or whitespace. If that cannot be confirmed, triage
   it again.
3. Once confirmed, do steps 2 and 3 of "Direct merge".

## Direct merge

1. Work where "Where to work" in `docket/workflow.md` says.
2. Follow "Checks" in `docket/workflow.md`.
3. Follow "How to merge" in `docket/workflow.md`.

## Commit trailers

Every commit made for a change or a record has one trailer line in the last
paragraph of its message, for dkt to read:

- `Change: <slug>/<N>`: a commit in a change that implements Task N. For
  several tasks, write `Change: <slug>/2,3`.
- `Change: <slug>`: a commit in a change that belongs to no task, such as
  the proposal or the plan.
- `Record: <slug>/<N>` and `Record: <slug>`: the same for a record.

Keep the trailer in one paragraph with any other trailers, such as
`Co-Authored-By`, with no blank line between them: git reads only the last
paragraph. After a squash the trailers are no longer in the last paragraph
and dkt cannot read them, so never merge with a squash.

## Parallel work

Before starting, run `dkt conflicts` and compare the plans' `**Files:**`
blocks; overlapping changes run one after the other. A change that depends
on another waits for it to merge. For the rest, follow "Parallel work" in
`docket/workflow.md`.
