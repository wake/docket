---
title: Ship the framework as a skill, and no workflow
---

Recorded 2026-10-29. The points below were decided with the maintainer; the
change that takes this idea turns them into a proposal and deltas.

## Problem

`dkt init` ships two kinds of guidance mixed together, in the "Change
workflow" section of `AGENTS.md` and in the instructions of
`openspec/schemas/docket/schema.yaml`:

- the framework: the files and fields `dkt` reads or checks, and how to use
  the commands;
- a development workflow: how work is triaged, when a change is opened,
  which reviews run when and on which model, mutation checks, pull request
  size.

The workflow is one maintainer's practice. It assumes tools another project
may not have, such as a second model for reviews, so it does not belong in
what Docket ships. And `AGENTS.md` is loaded in every session, so agents
apply the workflow to every task, questions and investigations included.

## Decided

- Docket ships the framework only. It ships no workflow, not even an
  optional default: each project writes its own.
- The boundary: what `dkt` reads or checks is framework; what only an agent
  reads, to decide who reviews what and when, is workflow.
  - Framework: the change folder and `docket.yaml`; `proposal.md`, the delta
    specs and `plan.md` with its task headings, `**Scenarios:**` lines and
    checkboxes; the `Change:` and `Task:` trailers, and merge commits
    because `dkt sync` reads them; the `## Reuse` table while `dkt lint`
    checks it; the commands themselves.
  - Workflow: triage into paths, `**Review:** judgment|checks` and which
    tasks get a reviewer, tests seen failing first and mutation checks,
    the plan review and the reviews before merging, a different model,
    pull request size, parallel sessions.
- The framework ships as a skill that `dkt init` installs. Agents that do
  not read skills are not supported: the shipped `AGENTS.md` keeps no
  pointer or summary for them.
- Docket supports a repository only after `dkt init`, with OpenSpec and
  superpowers installed.
- The workflow leaves what is shipped: the triage, review and pull request
  rules of the "Change workflow" section, and in `schema.yaml` the plan
  instruction's `**Review:**` rule and plan review and the apply
  instruction's execution and review rules.

## Open questions

- Where the skill is installed (the project's `.claude/skills/`, or a
  plugin), how `dkt init --force` updates it, and whether `dkt init` checks
  that superpowers is installed.
- Where a project's own workflow plugs in: its own skill, its `AGENTS.md`,
  or `rules` in `openspec/config.yaml` for extra plan fields.
- Whether the `## Reuse` table and the `**Files:**` block are framework:
  deciding reuse in the plan is a workflow choice, and nothing reads
  `**Files:**` yet.
- Where Docket's own workflow lives once the shipped section holds the
  framework only.
- Queued ideas that build the workflow into `dkt`, each to be kept,
  reworked or dropped: `2026-10-29-records.md` (kinds named after the light
  review path), `2026-10-29-commit-gate.md` (the triage and direct-merge
  commit types), `2026-10-28-lint-review-and-files.md` (the `**Review:**`
  line).

## Notes for the skill

- The description is what an agent sees before it loads the skill: it says
  when to use the skill, not what the skill says, or the agent follows the
  summary instead. The description and `when_to_use` together are cut at
  1,536 characters.
- After compaction only the first 5,000 tokens of each loaded skill are
  kept, so what an agent needs on every turn goes first and details go in
  referenced files.
- Check loading with prompts that should and should not load the skill,
  each run a few times in a fresh session.
