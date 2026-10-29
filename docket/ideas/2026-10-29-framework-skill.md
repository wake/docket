---
title: Ship the framework as a skill, with workflow templates
---

Recorded 2026-10-29. The points below were decided with the maintainer; the
change that takes this idea turns them into a proposal and deltas.

## Problem

`dkt init` ships its guidance in the "Change workflow" section of
`AGENTS.md` and in the instructions of `openspec/schemas/docket/schema.yaml`.
Agents load `AGENTS.md` in every session, so they triage every task,
questions and investigations included. And the guidance describes one way of
working: a user who already has tools and habits of their own has no place
to bring them in.

## Decided

- Two layers:
  - The **framework**, shipped by Docket: the triage of work into four paths
    (full review, light review, check, direct merge) and the kinds of work
    behind them, what `dkt` reads or checks, and a **template** for each
    path. Templates are written in capabilities, such as "cross-model
    review" or "a fresh agent", never a product name; the text shipped today
    is their starting point.
  - The **user's workflow**: the tools and habits a user already has, kept
    where the user keeps them, for example their global agent instructions.
    Docket does not copy it into the repository.
- The **materialized workflow** joins the two: for each path, the template
  brought as close as the user's workflow and tools allow. It is neither the
  template verbatim nor the user's workflow verbatim. It lives at the user
  level, never in the project repository, in one of two modes chosen at
  setup:
  - reference: written into the user's own global workflow, which Docket
    refers to, so a change there applies everywhere;
  - fixed: a workflow used by Docket only.
- Setup: when `dkt init` finds no materialized workflow, the agent checks
  the environment, reads the templates, guides the user, and shows the
  result before writing it. `dkt` itself only reports facts and checks
  structure: whether the materialized workflow exists, and which template
  version it was made from, so a changed template shows it is out of date.
- Review fallback in the templates: a cross-model review; else a subagent
  review (same model, fresh context); else the work is recorded as having no
  independent review and a person reviews the pull request. A self-review is
  not a review step.
- Lookup order: a project template (optional, committed, for a team) before
  Docket's template; a materialized workflow is always at the user level.
  The first version builds only Docket's template and the user-level
  materialized workflow.
- A `docket doctor` skill checks the setup, among other things the review
  workflow: whether the materialized workflow covers every path, which
  review fallback applies in this environment, and conflicts with the
  framework, such as squash merges, which drop the trailers `dkt sync`
  reads.
- The framework ships as a skill that `dkt init` installs. The triage moves
  out of always-loaded instructions into it, so it applies only to work on
  a Docket project. Agents that do not read skills are not supported:
  `AGENTS.md` keeps no pointer or summary for them.
- Docket supports a repository only after `dkt init`, with OpenSpec and
  superpowers installed. Docket's own repository is set up like any other.
- Settled by the change `add-framework-skill`: the framework skill is
  installed in the project (`.claude/skills/docket/` and
  `.agents/skills/docket/`, committed); a marked Docket block in `AGENTS.md`
  points to it and lets it win over superpowers defaults, and `CLAUDE.md`
  reaches the block; the Reuse table stays in the framework.

## Open questions

- Where the materialized workflow lives for each agent: Claude Code reads
  user skills from `~/.claude/skills`, Codex from `~/.agents/skills`.
- The checks `docket doctor` runs, and which of them `dkt` can do without
  judgment.
- Queued ideas written before this split, to update when each is taken up:
  `2026-10-29-records.md` keeps its mechanism and kinds, without its review
  sections; `2026-10-29-commit-gate.md` keeps its checks that work lands in
  the right container; `2026-10-28-lint-review-and-files.md` becomes a check
  of fields a project declares, or is dropped.

## Status

The change `add-framework-skill` took the first part (the skill and the
block). Two later changes remain: the materialized workflow, and
`docket doctor`.

## Notes

- Prior art for the setup: spec-kit's constitution command (the agent asks
  or infers, then writes one file) and Claude Code's interactive `/init`
  (it shows a proposal before writing). No tool found both detects an
  existing workflow and proposes one from the user's habits.
- Generated context files can lower agents' success and raise cost, and long
  instruction files are followed less: keep the materialized workflow short
  and have the user review it.
- A git worktree does not carry git-ignored files, so a materialized
  workflow kept ignored in the project would be missing in every new
  worktree.
- The skill's description says when to use it, not what it says, or the
  agent follows the summary instead; the description and `when_to_use` are
  cut at 1,536 characters together. After compaction only the first 5,000
  tokens of a loaded skill are kept. Check loading with prompts that should
  and should not load it.
