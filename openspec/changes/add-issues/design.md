# Design

## Context

Records already do what a file issue needs, one step later in the life of a
piece of work: `dkt record` writes a dated file from an embedded template
under a folder no link may redirect, never reuses a slug any scanned source
holds, and leaves nothing behind when the write fails; `dkt lint` checks
its front matter, sections, word counts and leftover template text; the
scan finds records in every worktree and unmerged branch. Docket links to
no outside system, so whatever goes to an outside issue tool goes through
the agent.

## Goals / Non-Goals

**Goals:**

- One issue structure, the template's four sections, whether the issue
  lives in an outside tool or in a file.
- File issues written, checked and listed with the same guarantees as
  records, by the same code.
- An issue's state derived from the work that names it, so no worktree
  edits a shared issue file to mark progress.

**Non-Goals:**

- Any link from `dkt` to an outside issue tool.
- A user-level or per-project setting that `dkt` reads.

## Decisions

- **Where the tool is named.** The project's `AGENTS.md`, in a `## Issues`
  section outside Docket's block, names the outside tool and how to file
  there. The tracker belongs to the project, its contributors share it, and
  agents read `AGENTS.md` already. A settings file under `docket/` was
  considered and dropped: `dkt` never uses the tool, so it would read the
  file for nothing. The user's workflow file was dropped too: it is one per
  user, and projects differ.
- **Filing in an outside tool needs the user's agreement.** Filing
  publishes text outside the repository. The skill has the agent show the
  issue built from `dkt issue template` and file it once the user agrees.
  A file issue is written inside the worktree as part of the work and
  needs no extra step.
- **Matching a handler.** A record or change handles a file issue when its
  `source` is exactly the issue's path. A free-text source that mentions the
  path does not count: an exact value keeps the rule checkable, and
  `issue-source` catches a path that names no issue.
- **State.** `in-progress` wins over `handled`: one issue may need several
  records, and it is handled only when none is still open. A change counts
  whether active or archived, since a merged change stays active on the
  default branch until it is archived at release. `dropped: true` and
  `idea` are the only states people write, because they are decisions, not
  facts `dkt` can see.
- **An issue that needs design.** It gets `idea: <path>` naming the idea it
  became. The idea itself does not change: ideas have no `source` field,
  and giving them one belongs to `2026-10-29-ideas-and-order.md`.
- **Dated names.** Issue slugs look like record slugs, and an issue and the
  record that fixes it may share one. `dkt lint <slug>`, `dkt status <slug>`
  and `dkt sync <slug>` keep reading a dated name as a record; issues are
  linted only when no name is given, and `dkt status` lists them only then.
- **`template` is not a name.** `dkt issue template` prints the template,
  as `dkt workflow template` does; an issue cannot be named `template`.
- **Reuse.** The record code that writes and checks files (`Create`, the
  front-matter parser, `Words`, `ValidateName`, `Slug`) takes the folder
  and the section list as parameters instead of having them fixed, and
  issues call it. No second copy of the write path, so its guarantees hold
  for both.

## Risks / Trade-offs

- An agent may write a free-text source such as `fixes
  docket/issues/2026-11-02-x.md`; the issue then stays `pending`. The
  skill's wording and the `Record names its issue` scenario keep sources to
  the bare path, and `dkt status` shows the issue still pending, which is
  visible.
- Issues in an outside tool are invisible to `dkt status`; that is the cost
  of linking to no outside system.
- Sharing the record code widens the change to `records` code paths; the
  existing record tests guard them.
