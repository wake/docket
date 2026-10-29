# Design

## Context

See proposal.md for why. Today `setup.Init` walks the embedded schema,
checks every target with `safepath.Check` before writing anything, and
writes each file with `O_NOFOLLOW`, leaving an identical file alone and a
differing one in place unless `--force`. `docket.AgentsSection` cuts the
"Change workflow" section out of the embedded `AGENTS.md` by heading, and
`dkt init` prints it.

Two findings shape the approach. Mainstream tools whose skill only matters
in projects that adopted them (OpenSpec 1.x, spec-kit, beads) write it into
the project from their `init` and expect it to be committed, one copy per
agent directory: Claude Code reads `.claude/skills/`, Codex and others read
`.agents/skills/`, and neither reads the other's. Tools that write into
`AGENTS.md` (beads, Nx, Convex, spec-kit, Next.js) keep a short block
between HTML comment markers, create the file when missing, append the
block when absent, and never touch text outside the markers.

## Goals / Non-Goals

**Goals:**

- One source for the skill and one for the block, embedded in `dkt`; the
  copies in Docket's own repository are what `dkt init` installs there.
- Every check (links, markers, reads) runs before the first write.

**Non-Goals:**

- Line-ending conversion: a block is compared byte for byte.
- Validating the skill's prose beyond loading and agreeing copies.

## Decisions

### Install in the project, one copy per agent directory

`dkt init` writes `.claude/skills/docket/SKILL.md` and
`.agents/skills/docket/SKILL.md`, meant to be committed, so every worktree
and every teammate has the skill at the version the project's schema came
from. Alternatives: a plugin (loads in every project and drifts from the
project's schema version; `dkt init` cannot install it), a user-level skill
(same drift, and `dkt` would write outside the repository), and one copy
with a symlink (init never writes through a link, and links in git are
fragile on some systems).

### Sources and embedding

The skill's source is `.agents/skills/docket/` in Docket's repository,
embedded like the schema; a test checks that `.claude/skills/docket/`
holds the same files. The block's source is the block in Docket's own
`AGENTS.md`, embedded as today and cut at the markers instead of a
heading, which retires the routing-section-fence idea. If `go:embed`
refuses a path that starts with a dot, the source moves to a plain
directory and the same test keeps both installed copies equal to it.

### Check everything, then write

Init first builds the list of writes: schema files, skill files, the
`AGENTS.md` block, and `CLAUDE.md` when it needs one. For each it checks
links, reads the current file, and for `AGENTS.md` and `CLAUDE.md` finds
the markers. A link, a malformed block or an unreadable file stops the run
before any write. Only then does it write. A missing file is created with
no-follow, exclusive writes as today; an existing file, which may now be a
user's `AGENTS.md`, is written to a temporary file in its directory with
its mode and renamed over it, so a failed write never leaves it
half-written (this takes up the atomic-writes idea for init). A failure
during writing can leave earlier files written; a re-run finishes, since
identical files are left alone.

### Block placement and comparison

A marker is a line that is exactly `<!-- docket:start -->` or
`<!-- docket:end -->`. The block is compared byte for byte from the start
marker through the end marker. Appending adds a blank line before the
block, and a newline first when the file lacks a final one; `--force`
replaces only the marker lines and what lies between them. A new
`AGENTS.md` holds the block and a final newline.

### CLAUDE.md

`CLAUDE.md` reaches `AGENTS.md` when one of its lines, trimmed, is
`@AGENTS.md` or `@./AGENTS.md`, or when it is a symbolic link whose target
is `AGENTS.md` or `./AGENTS.md`. Any other link is refused. A missing
`CLAUDE.md` is created holding `@AGENTS.md` and a final newline. Otherwise
it gets the block by the same rules as `AGENTS.md`.

### The block's text

```markdown
<!-- docket:start -->
## Docket

At the start of any development work in this repository (a new feature, a
behavior change, a bug fix, a refactor, or edits to prompts, tests, docs or
CI), use the `docket` skill before brainstorming, planning or editing.

Where the `docket` skill and another skill disagree, follow the `docket`
skill. In particular: brainstorming writes its design into the change
folder, not `docs/superpowers/specs/`; plans go to
`openspec/changes/<slug>/plan.md`, not `docs/superpowers/plans/`; only
tasks marked `**Review:** judgment` get their own reviewer; and OpenSpec's
`/opsx:*` commands are not used. This block does not override the user's
own instructions.
<!-- docket:end -->
```

Naming each replaced behavior, not only "follow the skill", is deliberate:
in Vercel's evaluation, explicit wording raised how often agents used a
skill, and small wording changes moved the result a lot.

### The skill's text

Frontmatter: `name: docket`, and a description that says when to use it
and nothing about its rules:

```
Use at the start of any development work in this repository, before
brainstorming, planning or editing: a new feature, a behavior change, a
bug fix, a refactor, or edits to prompts, tests, docs or CI. Not for
questions, explanations or investigation that change nothing.
```

The body is the current "Change workflow" section rewritten around the
four paths: the triage first (heaviest path wins, one line stating the
path and the rule that chose it), then full review, light review, check
and direct merge, each with its steps; the review fallback; the
superpowers defaults it replaces; parallel sessions; requirements that
change mid-way. Review steps name capabilities ("a reviewer on a different
model", "a fresh agent given only the diff"), never products. Claude Code
keeps only the first 5,000 tokens of a loaded skill after compaction, so
the body stays well under that, the triage first.

### Validating the trigger

The scenarios on loading are checked by trial, not by `go test`: in a
scratch repository set up by the built `dkt`, with superpowers installed,
Claude Code runs headless with prompts that should load the skill (add a
feature, fix a bug, edit the README) and prompts that should not (what a
function does, why CI is slow), and the transcript shows whether the
`docket` skill was loaded and whether it came before brainstorming. One
feature prompt is followed until the design is written, to see where it
lands. The results go in the pull request. Codex gets the mechanical
checks only.

## Risks / Trade-offs

- [The agent does not load the skill] → the block points to it, the
  description names the work that should load it, and the trial runs
  before release; wording is adjusted until it passes.
- [superpowers still wins a conflict] → the block names each replaced
  behavior, and the trial follows one feature into brainstorming.
- [Projects that pasted the old section get the rules twice] → the README's
  Upgrading section says to delete it.
- [A CRLF checkout makes the block always differ] → not handled; Docket's
  CI runs on macOS and Linux.
- [The user's global instructions still triage every task] → outside
  Docket; the user removes that after this release.

## Migration Plan

Upgrade `dkt`, run `dkt init`, delete any "Change workflow" section pasted
from an earlier release, and commit `.claude/skills/docket/`,
`.agents/skills/docket/`, `AGENTS.md` and `CLAUDE.md`. Rolling back is
deleting those files and the block.
