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
depend on superpowers and OpenSpec, and come in two kinds:

- **Required**: Docket depends on these rules; they are marked "(required)"
  below. When the user's instructions conflict with one, for example by
  asking for squash merges, forbidding anything extra in commit messages, or
  putting plans elsewhere, you must stop, name the conflict and ask the user
  what to do.
- **Default**: how reviews and verification are done, under "Default steps"
  at the end. The user's instructions or habits come first; follow the
  defaults where they have none.

When the rules of another development workflow skill differ from these,
these take precedence.

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
    `CLAUDE.md`, skill and command files), and text a program hands to
    coding agents (embedded or printed instructions, templates)
  - check settings: the files that decide which checks run, such as CI
    configuration
- **Checked merge**: an edit to a code or test file that changes only
  comments or whitespace. It needs no review: once the check confirms that
  this is all it changes, merge it directly; otherwise, triage it again.
- **Direct merge**: these edits, with no change opened:
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
- **Order**: `docket/order.md` lists the ideas to work on next, in order. To
  pick up the next piece of work, run `dkt status --json` and do the one
  `result.next` names.

In the worktree of a change or a record, do not touch `docket/order.md` or
`docket/ideas/`. A new idea found during the work, and any edit to the
order, is a direct merge of its own. When part of the work cannot be done
with the rest and has to wait, write that part as a new idea and list the
original idea in its `needs`.

When the work handles an issue or takes up an idea, `--source` holds only
its path (`docket/issues/<slug>.md`, `docket/ideas/<name>.md`), or the
issue's address in the project's tool.

## Commit trailers (required)

Every commit made for a change or a record has one trailer line in the last
paragraph of its message, for dkt to read:

- `Change: <slug>/<N>`: a commit in a change that implements Task N. For
  several tasks, write `Change: <slug>/2,3`.
- `Change: <slug>`: a commit in a change that belongs to no task, such as
  the proposal or the plan.
- `Record: <slug>/<N>` and `Record: <slug>`: the same for a record.

Keep the trailer in one paragraph with any other trailers, such as
`Co-Authored-By`, with no blank line between them: git reads only the last
paragraph.

Merge pull requests with a merge commit or a rebase, never a squash: after a
squash the trailers are no longer in the last paragraph, and dkt cannot read
them.

## Full review

1. In a git worktree, open the change:
   `dkt new <slug> --kind feature|change-request|baseline --source "<where the request came from>" --title "<one line>"`.
   - The kind is `feature` for something new, `change-request` for a change
     to existing behavior, and `baseline` for specifying behavior that is
     already released.
   - `dkt new` creates the change folder and writes the title, kind and
     source into the front matter of `proposal.md`.
   - When the slug is taken, it uses the first free suffix (`-2`, `-3`, ...)
     and reports it; use the slug it reports from then on.
   - Always open a change with `dkt new`, never with `/opsx:propose` or
     `/opsx:new` (required).
2. superpowers:brainstorming writes its design documents into the change
   folder, not `docs/superpowers/specs/` (required).
   - The design documents are `proposal.md`, the delta specs under `specs/`,
     and `design.md` when needed.
   - Before writing each one, run
     `openspec instructions <artifact> --change <slug> --json` for its
     template and rules.
   - Give each new capability a name under `specs:` in the front matter of
     `proposal.md`.
3. superpowers:writing-plans writes the plan to
   `openspec/changes/<slug>/plan.md`, not `docs/superpowers/plans/`
   (required), following the template and rules from
   `openspec instructions plan --change <slug>`. There is no tasks.md.
4. Follow the full review in "Default steps": review the plan, execute it,
   and review the change before merging. Execute with
   superpowers:subagent-driven-development; work started from
   `/opsx:apply` follows the same steps.
5. Before opening the pull request, run `dkt sync <slug>` (which ticks the
   plan from the trailers) and `dkt lint <slug>`, and commit plan.md.
6. A change is one pull request unless the default steps say to split it;
   the order of its commits shows its layers.
7. Archive at release time, with OpenSpec's archive (`/opsx:archive`).
   Following the instructions in `openspec/config.yaml`, it runs Docket's
   checks first and, after archiving, replaces each new spec's title with
   the name given in step 2.

When requirements change mid-way, follow the plan rules: cancelled tasks,
compensating tasks, a Revision section.

A change made with another schema (its `.openspec.yaml` names a schema
other than `docket`) follows that schema's files and has no `plan.md`; to
convert it, use `docket-doctor`.

## Light review

No change is opened; a record in `docket/records/` says what the work did.
In a git worktree:

1. Before the first commit, open the record:
   `dkt record <name> --kind fix|refactor|test|prompt|checks --source "<where the request came from>" --title "<one line>"`.
   - The kind follows the kind of work: a fix (`fix`), a restructuring
     (`refactor`), tests (`test`), prompt text (`prompt`), check settings
     (`checks`).
   - The record's slug is the one `dkt record` reports (the date, the name,
     and a suffix when the name is taken), not the name you passed.
2. Make the edit and verify it by the light review in "Default steps". Fill
   the record's sections as the work goes, keeping them short,
   replacing the template's guidance comments.
3. Larger work may list tasks under the record's `## Tasks` in the plan's
   format; their commits carry `Record: <slug>/<N>`.
4. When the record lists tasks, run `dkt sync <slug>` and commit the record.
   Run `dkt lint` and fix what it reports, then open the pull request and
   merge it.

## Checked merge

For an edit to a code or test file that changes only comments or
whitespace: follow the checked merge in "Default steps", and when it passes,
go on as a direct merge.

## Direct merge

In a git worktree: make the edit, run the direct merge checks in "Default
steps", open the pull request and merge it.

## Parallel work

When sessions work in parallel, each change belongs to one session, and the
session that implements a change does not review it. Run `dkt conflicts`
and compare the plans' `**Files:**` blocks before starting; overlapping
changes run one after the other. A change that depends on another waits for
it to merge. Merge one change at a time: rebase it on the default branch,
run the full test suite, and review again only where the rebase had
conflicts.

## Default steps

These are the steps for when the user's instructions or habits do not say
how to do one.

### Full review

1. Review the plan once, together with the proposal, the delta specs and
   design.md, before any code (see Reviewers), and fix what the review
   finds.
2. Execute the plan task by task, with a fresh implementer agent for each
   task, test first. Each implementer gets the Reuse rows for its task and
   reports every exported symbol it added, with why no existing one fit.
   Structural changes and behavior changes go in separate commits.
3. Only a task marked `**Review:** judgment` gets its own reviewer, in a
   fresh context. Every other task that changes code gets the mechanical
   checks instead: its new tests are seen failing before the code and
   passing after, and mutating the lines it changed makes a test fail. A
   task that changes no code, such as one that only edits docs, is checked
   by its own verification steps.
4. Before merging, review the whole change once, by two reviewers in
   parallel: one on a different model, asked to attack (wrong results, edge
   cases, partial failure, the same defect elsewhere in the code), and one
   in a fresh context, checking the code against the proposal and the delta
   specs (see Reviewers when no other model is available). After fixes, only
   a large fix is reviewed again.
5. Split the pull request only when it mixes unrelated concerns, when a
   structural refactor is mixed with behavior (the refactor goes first, in
   its own pull request), or when product code and tests together run far
   past about 1000 lines. Do not stack pull requests.

### Light review

1. Verify by the kind of edit:
   - a fix: write a regression test and see it fail before the fix;
   - a restructuring: the tests that cover the code pass before and after,
     and mutating the changed lines makes one fail; when none fails, first
     add the tests as a test-only edit;
   - prompt text: check it mechanically, confirming that it still loads and
     that every copy says the same; do not write tests that only check that
     strings are present;
   - tests: mutate the code they cover; the tests must fail;
   - check settings: run them; the pull request's own CI run uses the new
     settings, and the review looks for a check that got weaker.
2. Have the whole diff reviewed once before merging (see Reviewers).

### Checked merge

Have a fresh agent, given only the diff and the triage's definitions,
confirm before merging that every changed line is a comment or whitespace
and that no comment changes how the code builds or runs. If it does not
confirm, triage again.

### Direct merge

Run only the checks that read what the edit changed: none when it changes
only docs that nothing builds, embeds or tests; the build and the tests
when it changes code, test files or dependency manifests. No review.

### Reviewers

A required review is done by a reviewer on a different model. If there is
none, by the same model in a fresh context. If that is not possible either,
the pull request says it had no independent review and asks a person to
review it. The author's own reading is not a review.
