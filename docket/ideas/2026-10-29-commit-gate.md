---
title: A pull request gate on each commit's path (dkt check)
needs:
  - docket/ideas/2026-10-29-records.md
---

Recorded 2026-10-29. The points below were decided with the maintainer; the
change that takes this idea turns them into a proposal and deltas.

## Problem

The triage decides how much process a piece of work gets, but nothing checks
that a commit followed it: a prompt edit labelled as docs, or a
restructuring labelled as a chore, merges with no review.

## Decided

- `dkt check` looks at every non-merge commit on the branch since it left
  the default branch, and runs in CI as a required check.
- `docket/config.yaml` lists which paths hold code, tests, prompts and check
  settings (CI configuration and `docket/config.yaml` itself). With main
  specs, change folders and records, these paths are protected.
- A commit that touches a protected path carries a trailer that matches:
  `Change: <slug>` for a main spec (only a change edits one) and for that
  change's folder; `Record: <slug>` for that record; `Change:` or `Record:`
  for code, tests, prompts and check settings. The change or record must
  exist.
- A commit that touches no protected path passes when its prefix is one of
  the direct-merge types: `docs`, `style`, `build`, `chore`, `release`.
- In code and test files a `docs:` or `style:` commit passes without a
  trailer: whether an edit changed only comments or whitespace is not
  decided from the content, since that needs every language's syntax and a
  comment can change the build. The triage has a fresh agent confirm it
  before merging; `dkt check` calls no agent and cannot see whether that
  happened.
- Every other commit fails, named with what it lacks.
- With no `docket/config.yaml`, or no code paths in it, `dkt check` fails
  instead of passing everything. `dkt init` writes a template.
- A release commit that archives changes carries the `Change:` trailer of
  each archived change.
- Before the spec is settled, the rules are run over Docket's own history,
  and every commit they would fail is listed with the reason.

## Open questions

- A task commit whose trailers sit outside the last paragraph has no trailer
  git can read, so the gate fails it when it touches a protected path.
  Whether that settles `docket/ideas/2026-10-24-stray-trailers.md`.
