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
win. Each path's review and verification steps are not here: they come from
the user's workflow file.

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
  the dependency manifests (such as `go.mod` and `go.sum`). No change is
  opened.

An idea that is not ready goes to `docket/ideas/`, one file per idea.

Leave `docket/order.md` and the files under `docket/ideas/` unchanged in the worktree of a change or a record. An idea you find during the work, and any edit to the order, is a direct merge of its own. When work takes an idea, `--source` is the idea's path alone (`docket/ideas/<file>.md`). When that work leaves a part for later, write the part as a new idea whose `needs` lists the first idea's path, as a direct merge of its own. To take the next piece of work, run `dkt status --json` and take the idea of the line `result.next` names; do not read the checkboxes or the first lines of `docket/order.md`.

Work whose expected behavior is already settled, by a spec sentence, the docs or behavior that shipped, and that is not done now is an issue, not an idea: a bug found outside the work at hand, an optimization, a missing test, a request from someone else. When the project's `AGENTS.md` has a `## Issues` section, write the issue in the tool it names, with the sections `dkt issue template` prints; show it to the user and file it only once they agree. Otherwise run `dkt issue <name> --title "<one line>" --source "<where it was found>"` and fill its sections. When you cannot point to where the expected behavior is settled, it is an idea.

## The workflow file

Right after triage, before any other step of the work, run
`dkt workflow --json`. It reports the user's workflow file
(`result.path`) and its status (`result.status`):

- `ok`: on each path below, do the steps under that path's heading in the
  file, and under `## Reviews` where they point to it. On these steps the
  file wins over the user's other instructions.
- `missing` or `incomplete`: run the `docket-setup` skill first; then start
  the work.
- `stale`: follow the file as for `ok`. Tell the user once in this session
  that it was made from an older template, what the current template
  (`dkt workflow template`) does differently, and that the `docket-setup`
  skill can update it.
- anything else, such as the file being unreadable: tell the user what
  `dkt workflow` reported and ask how to go on. Do not write the file.

## Full review

1. In a git worktree, open the change:
   `dkt new <slug> --kind feature|change-request|baseline --source "<where the request came from>" --title "<one line>"`.
   A change's title, kind, source and base are in its `docket.yaml`; a new
   capability's delta folder has a `docket.yaml` with its title. When the
   slug is taken, `dkt new` creates the change under the first free suffix
   (`-2`, `-3`, ...) and reports it; use the slug it reports. When the work handles an issue, `--source` is the issue alone: the path of a file issue (`docket/issues/<slug>.md`) or its address in the project's tool. When it takes an idea, `--source` is the idea's path alone.
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
5. Review the plan, execute it and review the change before merging by
   the steps under `## Full review` in the workflow file. Every commit made
   for a task carries the trailers `Change: <slug>` and `Task: <N>`. Put
   them in the message's last paragraph, with any other trailers such as
   `Co-Authored-By` and no blank line between them: git reads trailers
   from the last paragraph only.
   A change whose `.openspec.yaml` names another schema follows that
   schema's files, gets no `plan.md` unless converted (see `docket-doctor`),
   and is still archived with `dkt archive`.
6. Before opening the pull request, run `dkt sync <slug>` and
   `dkt lint <slug>`, and commit plan.md.
7. A change is one pull request unless the workflow file says to split
   it; its commits show its layers. Pull requests merge with a merge
   commit, never squash: `dkt sync` reads the trailers of each commit.
8. Archive only with `dkt archive` on a branch made from the default
   branch, in the release step. Never run `openspec archive` directly.

Do not use OpenSpec's `/opsx:*` commands or skills.

When requirements change mid-way, follow the plan rules: cancelled tasks,
compensating tasks, a Revision section, a Change Log row.

## Light review

No change is opened; a record in `docket/records/` says what the work
did. In a git worktree:

1. Before the first commit, open the record:
   `dkt record <name> --kind fix|refactor|test|prompt|checks --source "<where the request came from>" --title "<one line>"`.
   The kind follows triage: a fix (`fix`), a restructuring
   (`refactor`), tests (`test`), prompt text (`prompt`), check settings
   (`checks`). The record's slug is its file name without `.md`: the
   date, the name, and a suffix when the name is taken. Use that slug,
   as `dkt record` reports it, never the bare name you passed. When the work handles an issue, `--source` is the issue alone: the path of a file issue (`docket/issues/<slug>.md`) or its address in the project's tool. When it takes an idea, `--source` is the idea's path alone.
2. Make the edit following the steps under `## Light review` in the
   workflow file. Fill the record's sections as the work and its review
   go, in a few sentences each, replacing the comments.
3. Every commit for the work carries the trailer `Record: <slug>`.
   Larger work may list tasks under `## Tasks` in the plan's format; a
   commit for one also carries `Task: <N>`. Put them in the message's
   last paragraph, with any other trailers such as `Co-Authored-By` and
   no blank line between them: git reads trailers from the last
   paragraph only.
4. When the record lists tasks, run `dkt sync <slug>` and commit the
   record. Run `dkt lint` and fix what it reports. Then open the pull
   request and merge it with a merge commit.

## Check

For an edit to a code or test file that changes only comments or
whitespace: do the steps under `## Check` in the workflow file, and when
they pass, go on as a direct merge.

## Direct merge

In a git worktree: make the edit, do the steps under `## Direct merge` in
the workflow file, open the pull request and merge it with a merge commit.

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
