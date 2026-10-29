---
name: docket
description: >-
  Use at the start of any development work in this repository, before
  brainstorming, planning or editing: a new feature, a behavior change, a
  bug fix, a refactor, or edits to prompts, tests, docs or CI. Not for
  questions, explanations or investigation that change nothing.
---

# Docket workflow

These are the rules for development work in this repository, on top of
superpowers and OpenSpec. Where another skill says otherwise, these rules
win.

## Triage

Before starting, state in one line the path taken and the rule that chose
it. When several paths apply, take the heaviest. Sort an edit by what it
touches: a typo in a prompt is prompt text, and moving code is a
restructuring.

- **Full review**: the work changes a sentence in `openspec/specs/`. So
  does any edit that sets an expectation no spec states and no existing
  behavior shows, such as a bug fix with no specified answer or a test that
  asserts new behavior: it defines behavior, and the change adds the
  sentence.
- **Light review**:
  - a bug fix that restores the specified behavior;
  - a restructuring that changes no behavior (moving code is one);
  - prompt text: the text agents act on, meaning files agents load by
    convention (`AGENTS.md`, `CLAUDE.md`, skill and command files) and text
    a program hands to an agent (embedded or printed instructions,
    templates);
  - tests that check behavior already specified or already shown;
  - check settings, the files that decide which checks run, such as CI
    configuration.
  A comment that changes how the code builds or runs, such as a compiler
  directive (`//go:build`) or an annotation, is code.
- **Check**: an edit that would be a direct merge but touches a code or
  test file, because it changes only comments or whitespace in it.
- **Direct merge**: docs for people (moves and typos in them included),
  edits that only change comments or whitespace in code or test files and
  pass the check, version bumps, and dependency upgrades that touch only
  the dependency manifests (such as `go.mod` and `go.sum`). No change and
  no review beyond the automated checks.

An idea that is not ready goes to `docket/ideas/`, one file per idea.

## Full review

1. In a git worktree, open the change:
   `dkt new <slug> --kind feature|change-request|fix|baseline --source "<where the request came from>" --title "<one line>"`.
   A change's title, kind, source and base are in its `docket.yaml`; a new
   capability's delta folder has a `docket.yaml` with its title.
2. superpowers:brainstorming writes its design into the change folder, not
   `docs/superpowers/specs/`: `proposal.md`, delta specs under `specs/`, and
   `design.md` when needed. Before writing each one, run
   `openspec instructions <artifact> --change <slug> --json` for its
   template and rules.
3. superpowers:writing-plans writes `openspec/changes/<slug>/plan.md`, not
   `docs/superpowers/plans/`, following
   `openspec instructions plan --change <slug>`. There is no tasks.md. A task
   is a slice that can be verified on its own; tightly coupled tasks are one
   task. A task that needs a reviewer's judgment (an error path,
   environment-dependent behavior, a public interface or output format)
   says `**Review:** judgment`; the others say `**Review:** checks`.
4. Reuse is decided in the plan: every capability the change needs gets a
   row in the plan's Reuse table, with the search behind the decision. The
   proposal's Builds On names what already exists.
5. Review the plan once, together with the proposal and deltas, before any
   code (see Reviews).
6. Execute with superpowers:subagent-driven-development, with one
   exception: only a `**Review:** judgment` task gets its own reviewer.
   Every task that changes code gets the mechanical checks instead: its new
   tests are seen failing before the code and passing after, and mutating
   the lines it changed makes a test fail. A task that changes no code, such
   as one that only edits docs, is checked by its own verification steps.
   Each implementer gets the Reuse rows for its task and reports every
   exported symbol it added, with why no existing one fit. Every commit made
   for a task carries the trailers `Change: <slug>` and `Task: <N>`;
   structural changes and behavior changes go in separate commits.
7. Before merging, review the whole change once, by two reviewers in
   parallel: one on a different model, asked to attack (wrong results, edge
   cases, partial failure, the same defect elsewhere in the code), and one in
   a fresh context, checking the code against the proposal and deltas (see
   Reviews when no other model is available). After fixes, only a large fix
   is reviewed again.
8. Before opening the pull request, run `dkt sync <slug>` and
   `dkt lint <slug>`, and commit plan.md.
9. A change is one pull request; its commits show its layers. Split it only
   when it mixes unrelated concerns, when a structural refactor is mixed
   with behavior (the refactor goes first, in its own pull request), or when
   product code and tests together run far past about 1000 lines. Do not
   stack pull requests.
10. Pull requests merge with a merge commit, never squash: `dkt sync` reads
    the trailers of each commit.
11. Archive only with `dkt archive` on a branch made from the default
    branch, in the release step. Never run `openspec archive` directly.

Do not use OpenSpec's `/opsx:*` commands or skills.

When requirements change mid-way, follow the plan rules: cancelled tasks,
compensating tasks, a Revision section, a Change Log row.

## Light review

No change is opened. In a git worktree:

1. Verify by the kind of edit:
   - a fix: write a regression test and see it fail before the fix;
   - a restructuring: the tests that cover the code pass before and after,
     and mutating the changed lines makes one fail; when none fails, first
     add the tests as a test-only edit;
   - prompt text: check it mechanically, not with tests that look for
     strings: it still loads, and every copy says the same;
   - tests: mutate the code they cover; the tests must fail;
   - check settings: run them; the pull request's own CI run uses the new
     settings, and the review looks for a check that got weaker.
2. Have the whole diff reviewed once before merging (see Reviews).
3. Open the pull request and merge it with a merge commit.

## Check

For an edit to a code or test file that changes only comments or
whitespace. Have a fresh agent, given only the diff and the definitions in
Triage, confirm before merging that every changed line is a comment or
whitespace and that no comment changes how the code builds or runs. If it
does not confirm, triage again.

## Direct merge

In a git worktree: make the edit, pass the automated checks, open the pull
request and merge it with a merge commit. No review.

## Reviews

A required review is done by a reviewer on a different model. If there is
none, by the same model in a fresh context. If that is not possible either,
the pull request says it had no independent review and asks a person to
review it. The author's own reading is not a review.

## Parallel sessions

When sessions work in parallel, each change belongs to one session, and the
session that implements a change does not review it. Run `dkt conflicts`
and compare the plans' `**Files:**` blocks before starting; overlapping
changes run one after the other. A change that depends on another waits for
it to merge. Merge one change at a time: rebase it on the default branch,
run the full test suite, and review again only where the rebase had
conflicts.

## Prompt paths

The text agents act on is listed in `AGENTS.md`. Update that list in the
pull request that adds, moves or removes one.
