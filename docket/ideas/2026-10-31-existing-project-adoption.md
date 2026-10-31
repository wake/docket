---
title: Adopting Docket in an existing project
---

Recorded 2026-10-31, from a read-only trial of dkt v0.9.1 (OpenSpec 1.14.3)
on a copy of the Purdex repository: a project with its own rules, an
OpenSpec folder kept out of git, and a change made with OpenSpec's
`spec-driven` schema. Nothing below is decided.

## Found

1. **Installed files that git ignores.** Purdex's `.gitignore` has
   `.claude/` and its `.git/info/exclude` has `openspec/`. `dkt init` wrote
   the skills and the schema and reported ok, and so did `dkt inspect`.
   None of the files could be committed, so a new worktree or clone has no
   Docket.
2. **Changes made with another schema.** For a `spec-driven` change
   (`tasks.md`, no `plan.md`, no `docket.yaml`):
   - `dkt lint` reports six findings under Docket's rules: a missing title,
     four proposal sections and a capability's `docket.yaml`. The title
     finding says that a change made by an older dkt moves its fields,
     which does not apply, and nothing says that the change uses another
     schema.
   - `dkt status` shows it as `design 0/0 tasks`, since it reads only
     `plan.md`.

   An archived change of that schema causes no finding.
3. **Project rules that disagree with the workflow.** Purdex's AGENTS.md
   states its own process (two review rounds, a version bump after
   merging). `dkt init` appends the Docket block without noticing.
   `docket-doctor` finds such conflicts, but the adoption steps do not say
   to run it.
4. **OpenSpec content outside git.** Purdex's `openspec/` (a `spec-driven`
   config and one archived change) was never committed. Whether to commit,
   rewrite or drop it is the project's step, not `dkt`'s.

## Leads

- `dkt init` and `dkt inspect` check each path they install with
  `git check-ignore` and report an ignored one, naming the rule that would
  let it in (for `.claude/`: `.claude/*` and `!.claude/skills/`).
- `dkt lint` and `dkt status` read the `schema` in `.openspec.yaml`. A
  change of another schema gets one finding that says so, and Docket's
  rules skip it; `dkt status` names its schema.
- The README's adoption steps name `docket-doctor` after `dkt init`, and
  say what to do with OpenSpec content already there.

## Related

- `2026-10-31-global-schema.md`: the shipped schema leaves the project and
  is installed once per machine, so found item 1 then concerns the skills
  and the Docket block only. This idea builds on it.
- `2026-10-31-scan-tree-batch.md`: the speed of the scan in a large
  repository, from the same trial.
- `2026-10-31-codebase-decomposition.md`: describing the code that already
  exists.

## Open questions

- Whether `dkt status` lists a change of another schema at all, and whether
  `dkt archive` refuses one.
- Whether `dkt init` writes the ignore exception itself or only reports it.
