# Design

## Context

See proposal.md for why, and `add-init-and-new`'s design.md for the
milestone and for the choice of approach A: Docket reads the structure of
deltas itself, following OpenSpec 1.14's rules, and OpenSpec stays the only
thing that validates and merges specs.

The readers built here are used three ways: by `dkt lint` on worktree files,
by `dkt conflicts` on blobs read from other branches, and by `dkt archive`,
which compares them with OpenSpec's own reading before archiving.

## Goals / Non-Goals

**Goals:**

- Readers that take bytes, not paths, so the same code reads a worktree file
  and a git blob.
- A delta reader that agrees with OpenSpec 1.14.3 on every file OpenSpec
  accepts, and that reports, rather than guesses at, everything else.

**Non-Goals:**

- A general Markdown parser. OpenSpec's reader is line-based, and Docket
  matches it line for line; a CommonMark library would disagree with it on
  exactly the edge cases that matter.

## Decisions

- **Port the delta rules from OpenSpec's source, not from its docs.** The
  rules below come from `src/core/parsers/requirement-blocks.ts` and
  `code-fence.ts` in 1.14.3:
  - Strip a leading BOM; turn CRLF and lone CR into LF.
  - Lines inside fenced code blocks are ignored. A fence opens with three or
    more backticks or tildes after optional indentation, and closes with a
    line of at least as many of the same character and nothing else.
  - Level-two sections match `^##\s+(.+)$`; their titles compare without
    regard to case (`## ADDED Requirements`, `## Purpose`).
  - Requirement headers match `^###\s*Requirement:\s*(.+)\s*$` without
    regard to the case of "Requirement". The name drops trailing `#`
    characters and surrounding spaces, and compares with case.
  - Scenarios are `#### Scenario: <name>` inside a requirement block.
  - RENAMED pairs are a FROM line followed by a TO line with no second FROM
    in between; anything else is unpaired.
  - OpenSpec's own categories for what it will not apply (orphaned
    requirements, unapplied content, unpaired renames, skipped headers)
    become `unparseable` findings.
- **Readers for proposal.md and plan.md are Docket's own.** Both read
  headings by level and a few marked lines (`**Scenarios:**`,
  `**Compensates:**`) and pipe tables. Section titles compare without regard
  to case. Tables follow the GitHub pipe syntax: a header row, a delimiter
  row, then rows; escaped `\|` stays in its cell.
- **Placeholder rule from the shipped templates.** The set of placeholder
  strings is computed from the embedded schema templates at run time, so it
  follows the templates when they change instead of being a second list.
- **Contract test.** `testdata/deltas/` holds example changes: ordinary
  deltas, every operation, fenced headers, CRLF and BOM files, odd spacing
  and case, and cases adapted from OpenSpec's own tests (MIT licensed;
  credited in the directory's README). A test reads each one with Docket and
  with `openspec show <change> --type change --json --deltas-only`, and
  compares capability, operation, requirement names and scenario names in
  order. It runs when `DOCKET_TEST_OPENSPEC` is set, as for the other tests
  that use the real CLI, which the CI job does with OpenSpec 1.14.3; it is
  skipped locally otherwise.
- **Routing check uses git.** Committed additions come from
  `git diff --name-status --diff-filter=A <merge-base> HEAD`, uncommitted
  ones from `git status --porcelain --untracked-files=all`, both limited to
  `docs/superpowers/specs/` and `docs/superpowers/plans/`.

## Risks / Trade-offs

- [OpenSpec 1.15 changes the delta format, for example with scenario-level
  operations] → The version gate stops Docket from running against it; the
  contract test is the first thing run when the pinned version moves.
- [The placeholder rule flags a real use of a template phrase] → It only
  matches strings copied verbatim from the templates, and never inside code;
  the finding names the template the string came from.
- [Scenario slugs collide across scripts or punctuation, such as "Join (when
  full)" and "Join when full"] → Reported by `scenario-duplicate`; the
  author renames one.
