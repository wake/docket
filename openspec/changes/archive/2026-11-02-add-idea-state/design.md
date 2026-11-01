# Design

## Context

See proposal.md for why. The relevant state today:

- `internal/status/issues.go:LoadIssues` finds an issue's handlers: every
  record and every change copy (active or archived), read from the default
  branch tip when the tip holds it and from its first source otherwise,
  whose trimmed source equals the issue's path. It reads every record and
  every change's `docket.yaml` once per run.
- Nothing reads `docket/order.md` or the idea files; the only code that
  knows `docket/ideas/` is the issue lint's `idea` check.
- Sources written so far: of the nine finished order lines, four have a note
  after the path (`docket/ideas/2026-10-31-global-schema.md (order 5)`,
  `docket/ideas/2026-10-24-scan-git-calls.md, with measurements ...`).
- The ideas in this repository have `title`, and three have `needs:`; none
  has `to:` or `dropped:`.

## Goals / Non-Goals

**Goals:**

- One writer for ideas and the order: no change or record worktree edits
  them, and nothing in `dkt` writes them.
- The order's finished lines read as done without touching archived
  changes or merged records.
- One rule for what a source names, for issues and ideas alike.

**Non-Goals:** see proposal.md.

## Decisions

1. **Ideas and the order are read from the default branch tip only.** They
   change only by the maintainer's decisions, which take effect on the
   default branch; reading branches would bring back the question of which
   copy wins. Handlers are still found on every source, as for issues, so a
   change on a branch makes its idea `in-progress`. Lint reads the
   worktree's copies, as it reads everything else.
2. **Done means every handler is on the tip**, the rule issues use for
   `handled`. An idea whose first piece of work merged reads `done` even if
   a second is planned; the skill's rule "one idea is one piece of work"
   keeps that from happening, by writing the further part as a new idea that
   `needs:` the first.
3. **The path a source names is its first field.** The text up to the first
   whitespace, without one trailing comma. It accepts every source written
   so far and rejects nothing that matched before. The skill still asks for
   the path alone. Rejected: an exact match only (four finished lines would
   read pending) and editing old sources (archived changes are history).
4. **The checkbox goes.** A done mark written by hand and a state derived
   from sources would disagree sooner or later. The cost: `docket/order.md`
   read on its own no longer shows what is done; `dkt status` does, and
   later the workbench.
5. **`result.next` is the first line whose idea is pending or in progress.**
   That keeps the meaning of "the first unticked line": a line in progress
   is still the next piece of work. Lines whose idea is missing or dropped
   are lint errors and are skipped.
6. **Lint checks the order and every idea whenever no name is given**, as it
   checks the main spec titles. In this repository ideas and the order are
   pushed to the default branch without a pull request, so the next lint
   anywhere is what catches a mistake.
7. **One handler pass for issues and ideas.** Status reads each record and
   change once and matches its source against both the issue paths and the
   idea paths.
8. **This change edits `docket/order.md` and `AGENTS.md` itself.** The rule
   that a change's worktree leaves the order alone starts with this change;
   the new format and the lint that reads it have to land together, or the
   default branch fails lint between the two.

## Risks / Trade-offs

- A source that starts with an idea path but means something else (for
  example `docket/ideas/x.md was rejected, see y`) counts as a handler. The
  skill's rule to pass the path alone keeps new sources clean.
- An idea split across several changes before this rule (the framework
  skill's three changes) reads done only once all are merged, which is
  already true for every such idea in this repository.
- Ideas edited in a worktree do not show in `dkt status` until pushed.
