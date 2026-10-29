# Proposal

## Why

Docket's workflow rules live in the "Change workflow" section of
`AGENTS.md`, which `dkt init` prints for the user to paste. Agents load
`AGENTS.md` in every session, so they triage every task, questions and
investigations included; and users ignore a section they are asked to
paste. The rules belong in a skill that an agent loads when development
work starts (`docket/ideas/2026-10-29-framework-skill.md`, item 2 of
`docket/order.md`). This is the first of three changes taken from that
idea; the materialized workflow and `docket doctor` follow.

A skill alone loses one thing the section had. Some of Docket's rules
replace superpowers defaults, such as where brainstorming writes its design
and which tasks get their own reviewer. superpowers ranks instructions in
`AGENTS.md` and `CLAUDE.md` above skills, so those replacements hold today;
moved into a skill, they would be one skill against another. A short block
in `AGENTS.md` keeps them on top.

## What Changes

- `dkt init` installs a `docket` skill at `.claude/skills/docket/` (Claude
  Code) and `.agents/skills/docket/` (Codex and other agents that read
  `.agents/skills`), meant to be committed. Existing files are handled as
  schema files are: identical ones are left alone, differing ones are
  reported and replaced only with `--force`.
- The skill holds the triage into four paths (full review, light review,
  check, direct merge) and each path's steps, written in capabilities
  ("cross-model review", "a fresh agent") rather than products, with a
  review fallback. Its description says only when to use it: at the start
  of development work, not for questions or investigation.
- `dkt init` writes a marked block into `AGENTS.md`: use the `docket` skill
  at the start of development work, and follow it where it disagrees with
  another skill, naming the superpowers behaviors it replaces. The block
  does not override the user's own instructions. `AGENTS.md` is created
  when missing; text outside the markers is never touched.
- `CLAUDE.md` must reach the block, since Claude Code reads `CLAUDE.md`:
  one that imports `@AGENTS.md` is left alone, a missing one is created
  holding `@AGENTS.md`, and any other gets the same block.
- `dkt init` no longer prints a section to paste, and its JSON result
  loses `agents_section`.
- Docket's own repository is set up by `dkt init` like any other: its
  `AGENTS.md` loses the "Change workflow" section and gains the block, and
  the skill is committed under both paths.

## Non-Goals

- No materialized workflow, no versioned templates, no reading of the
  user's own workflow; that is the next change.
- No `docket doctor` skill; that is the change after.
- No agent selector: both skill paths are always written, since Docket
  supports only agents that read one of them.
- No command that removes the skill or the block; the markers make the
  block easy to delete by hand.
- No detection of a "Change workflow" section pasted from an earlier
  release; the README's Upgrading section says to delete it.
- No plugin or user-level install, and no support for agents that do not
  read skills.

## Builds On

- `setup.Init` and `setup.install` (`internal/setup/setup.go`): the
  identical / differing / `--force` handling and the no-follow writes
  extend to the skill files.
- `safepath.Check` (`internal/safepath`) before every write.
- `docket.Schema` and `docket.AgentsSection` (`schema.go`): the embedding
  pattern for shipped files; `AgentsSection` and its routing section go.
- `initCommand` (`internal/cli/cmd_init.go`): findings and human output.
- The `project-setup` capability, whose requirements this change modifies.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `project-setup`: `dkt init` installs the skill and writes the `AGENTS.md`
  block (and reaches it from `CLAUDE.md`); "User configuration untouched"
  gives way to "Files init writes", which drops the section to paste and
  extends the no-symlink rule to every path init writes.

## Impact

- `internal/setup`, `internal/cli/cmd_init.go`, `schema.go`.
- New shipped prompt text: `.claude/skills/docket/SKILL.md`,
  `.agents/skills/docket/SKILL.md` and the `AGENTS.md` block, each listed
  in `AGENTS.md`'s prompt paths.
- `AGENTS.md` loses the "Change workflow" section; both READMEs (what
  Docket does not do, the workflow, Upgrading), `docs/ARCHITECTURE.md`.
- Scripts that read `agents_section` from `dkt init --json` lose it.
- `docket/ideas/2026-10-24-routing-section-fence.md` no longer applies and
  is removed; `docket/ideas/2026-10-29-framework-skill.md` keeps the two
  changes that remain.

## Implementation Contract

### Observable Behavior

- In a repository without the skill, `dkt init` writes both `SKILL.md`
  files and the `AGENTS.md` block, creates `CLAUDE.md` with `@AGENTS.md`
  when it is missing, lists every file with its status, and exits with 0.
- Run again with nothing changed, it writes nothing and exits with 0.
- After an upgrade, a skill file or block that differs from the shipped
  one is reported and kept, with exit 1; `--force` replaces it.
- In a project set up this way, an agent asked to add a feature loads the
  `docket` skill before brainstorming; asked what some code does, it does
  not load it.

### Interfaces

- `dkt init [--force]`: flags unchanged.
- JSON `result`: `openspec_initialized` and `files` (each `path` and
  `status`: `written`, `unchanged`, `differs` or `overwritten`), now
  listing the skill files, `AGENTS.md` and `CLAUDE.md`; `agents_section`
  is removed.
- Findings: `schema-differs` (unchanged), `skill-differs`,
  `agents-block-differs`, `agents-block-malformed`, `symlink-refused`
  (now for any path init would write).
- The block is the lines from `<!-- docket:start -->` to
  `<!-- docket:end -->`, each marker on its own line.

### Failure Modes

- Any path init would write passes through a symbolic link (other than a
  `CLAUDE.md` that links to `AGENTS.md`, which init does not write): it
  writes nothing, names the link, and exits with 1.
- `AGENTS.md` or `CLAUDE.md` has a start marker without an end marker, an
  end marker first, or more than one block: init writes nothing, reports
  `agents-block-malformed` for the file, and exits with 1.
- A file init needs cannot be read: it is listed as unreadable, nothing is
  written, and the exit code is 3.
- A write fails partway: files already written stay; re-running `dkt init`
  finishes the rest, since identical files are left alone.

### Acceptance Criteria

- `project-setup#fresh-repository-gets-the-skill`
- `project-setup#re-run-with-no-differences`
- `project-setup#locally-edited-template`
- `project-setup#edited-skill-file`
- `project-setup#skill-loads-for-development-work`
- `project-setup#skill-not-loaded-for-a-question`
- `project-setup#skill-copies-agree`
- `project-setup#block-added-to-an-existing-agents-md`
- `project-setup#agents-md-created`
- `project-setup#edited-block`
- `project-setup#skill-wins-over-superpowers`
- `project-setup#malformed-block`
- `project-setup#claude-md-imports-agents-md`
- `project-setup#claude-md-created`
- `project-setup#claude-md-without-the-import`
- `project-setup#other-configuration-unchanged`
- `project-setup#symlinked-schema-file-refused`
- `project-setup#symlinked-skill-directory-refused`

## Change Log

| Date | Source | Change | Impact |
|---|---|---|---|
