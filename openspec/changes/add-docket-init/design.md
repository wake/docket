# Design

## Context

`dkt init` already writes one thing outside the project: the `docket` schema,
to `<data home>/openspec/schemas/docket/`. In `internal/setup` a file to
install is a `target` whose `base` is an absolute directory (the schemas
directory) and whose `rel` starts with the folder `dkt init` owns
(`docket/...`). `safepath.Check(base, rel)` refuses a link at or below that
folder and follows any above it; `Inspect` reports the same targets without
writing. The CLI resolves the directory from the environment
(`setup.SchemaDir`, `userdirs`), and `setup` reads no environment itself.

The skills that reach projects are embedded from `.agents/skills/` and
written to both project skill roots. `docket-doctor` already runs only when
invoked, in both agents, through its frontmatter and `agents/openai.yaml`.

`dkt inspect` resolves the default branch first and exits with 4 when there
is none, so in a repository with no commit it reports nothing; `dkt init`
and `dkt workflow` work there.

## Goals / Non-Goals

**Goals:**

- The user-level command is installed, inspected and refused on links by
  the code that does it for the schema, not by a second mechanism.
- The command adds no rule of its own where a shipped skill already has one:
  it follows `docket-setup` and `docket-doctor`.

**Non-Goals:**

- No new `dkt` command, flag or environment variable of Docket's own.
- Other commands that need a default branch keep exiting with 4 without one.

## Decisions

1. **Source folder `user-skills/docket-init/`, apart from `.agents/skills/`.**
   `.agents/skills/` in this repository holds what `dkt init` writes into
   projects; a user-level skill there would also load as this repository's
   project skill and blur that line. It is embedded on its own and is a
   prompt path. Alternative: `.agents/skills/docket-init/`, rejected for
   the reasons above.

2. **Destinations as schema targets.** Each file is a target with `base`
   the agent's skills directory (`<claude>/skills`, `<home>/.agents/skills`)
   and `rel` `docket-init/<file>`. A link at or below `docket-init` is
   refused; a linked `~/.claude`, `~/.claude/skills` or `~/.agents` is
   followed, as dotfile managers link them. The CLI resolves both
   directories and passes them to `setup` as it passes the schema
   directory. Alternative: refuse every link on the way, rejected because
   it breaks linked dotfiles for no gain; the folder `dkt init` owns is
   still protected.

3. **`CLAUDE_CONFIG_DIR` is honored, `CODEX_HOME` is not.** Claude Code's
   documentation says `CLAUDE_CONFIG_DIR` overrides its configuration
   directory, `~/.claude`, where personal skills live. Codex's
   documentation names only `$HOME/.agents/skills` for user skills. A
   relative `CLAUDE_CONFIG_DIR` is an error, as a relative `XDG_DATA_HOME`
   is in `userdirs`.

4. **Kind `user-skill`, findings `user-skill-missing` and
   `user-skill-differs`.** A user-level copy is fixed once per machine and
   seen by every project, so the doctor and the user must tell it from a
   project skill. The messages follow the `skill-*` ones.

5. **Both copies always.** `dkt init` does not look for Claude Code or
   Codex; it writes both copies, as it writes both project skill roots.
   Looking for an agent would mean reading its configuration, which `dkt`
   does not do.

6. **`dkt inspect` without a default branch.** Inspect alone treats
   `scan.ErrNoDefaultBranch` as a skipped trailer check: `result.changes`
   is empty, `result.default_branch` is null, and the human output has a
   `skipped  trailers: <reason>` line. It is not a finding: a new
   repository is not a broken setup. `result.default_branch` names the
   branch otherwise, so a caller can tell an empty list from a skipped one.

7. **The command follows the skills from their files.** Claude Code does
   not load a `.claude/skills/` folder created after the session started
   until `/reload-skills`; Codex may need a restart. The command therefore
   reads `docket-setup` and `docket-doctor` from the `SKILL.md` files
   `dkt init` just wrote and follows them, and tells the user at the end how
   to load the skills.

8. **Checks in the command, in order.** git first (`dkt inspect` needs a
   work tree), then that `dkt` runs, then OpenSpec through
   `dkt inspect --json` (`result.openspec`), then superpowers, which only
   the agent can see in its own skill list. Install instructions name
   commands from each project's own documentation: superpowers'
   `/plugin install superpowers@claude-plugins-official` in Claude Code
   and `/plugins` in Codex, OpenSpec 1.14.x through npm or
   `DOCKET_OPENSPEC`, and the README's install line for `dkt`.

## Risks / Trade-offs

- [Personal skills might not move with `CLAUDE_CONFIG_DIR`] → the
  documentation says the whole configuration directory moves; `dkt init`
  and `dkt inspect` print the absolute paths they use, so a wrong guess
  shows.
- [`dkt init` writes into the user's agent directories] → only `dkt init`,
  run by the user, writes there, only inside `docket-init/`, and lists
  every file; this was decided with the maintainer (idea, 2026-11-01).
- [Codex and Claude Code read each other's keys] → `docket-doctor` already
  ships the same frontmatter and `agents/openai.yaml` pair.
- [An environment with `XDG_DATA_HOME` but no home directory] → `dkt init`
  now exits with 4 there, where it used to install the schema; the message
  names the missing home directory.
- [The command and the doctor drift apart] → the command follows the
  doctor's file and repeats none of its checks.

## Migration Plan

Nothing to migrate. After the release, a plain `dkt init` in any project
installs the command; until then `dkt inspect` reports `user-skill-missing`.

## Open Questions

- Whether Codex lists a skill added under `~/.agents/skills/` without a
  restart: checked by hand after the release; the command's last step
  already says to restart when it does not show.
