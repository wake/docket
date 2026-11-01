---
title: A user-level command that sets a project up for Docket
---

Recorded 2026-11-01, from setting up a new project. To be discussed after
the change `install-schema-per-machine`.

## Problem

The `docket`, `docket-setup` and `docket-doctor` skills reach a project
only through `dkt init`, so a project that has never run it has no Docket
command to start from. Today the user runs `dkt init` in a terminal, or
asks the agent to run it.

## Direction, not decided

- A command the user invokes (not loaded by the model on its own),
  installed at user level, for example `~/.claude/skills/docket-init/`,
  so that it exists in every project on the machine.
- Running `dkt init` alone adds little over asking the agent to run it.
  The command earns its place with what needs an agent afterwards: set up
  the workflow file when it is missing, then check the setup as
  `docket-doctor` does.
- It builds on `install-schema-per-machine`, which already has `dkt init`
  write something once per machine.

## From the maintainer, 2026-11-01

- It works like Claude Code's own `/init`: it can be typed in any project.
  Whether the environment supports it is something the command checks.
- It is a command used inside a session, but its steps are not limited to
  an agent: what needs no judgment can also run from the CLI.
- It first checks that superpowers and OpenSpec are installed; without
  them, setting up Docket is of no use.
- In an existing project it surveys the code and builds a code map or
  index. The tools that could help are being collected into a report for
  the maintainer to read first; see `2026-10-31-codebase-decomposition.md`.
- After it, the project is ready to start development, or to start a
  refactoring (`2026-10-31-refactoring.md`).

## The steps, agreed 2026-11-01

1. Check the environment, writing nothing: OpenSpec 1.14.x (`dkt` already
   checks it; `dkt inspect` would report it too), superpowers (only the
   agent can see which skills it has, and `dkt` does not read the user's
   agent configuration, so the command checks it), and a git repository.
   When something is missing, stop and say how to install it; installing
   writes outside the project, so the command does not do it.
2. `dkt init`: the project's files and the per-machine schema.
3. `docket-setup` when the workflow file is missing.
4. In an existing project, survey it: the interface list
   (`2026-10-31-codebase-decomposition.md`, `2026-10-24-add-index.md`),
   the adoption checks (`2026-10-31-existing-project-adoption.md`), and a
   draft of capabilities for the user to confirm, as the start of a
   baseline.
5. Check the setup as `docket-doctor` does.
6. Report what to commit, and that development or a refactoring can
   start.

Split: this idea is the command itself, steps 1, 2, 3, 5 and 6, mostly
joining what exists. Step 4 comes from the three ideas it names, each
done as its own piece of work; the command takes each part up as it ships.

## Decided, 2026-11-01

`dkt init` installs the command at user level, `~/.claude/skills/docket-init/`
for Claude Code (`/docket-init`) and `~/.agents/skills/docket-init/` for
Codex (`$docket-init`), the way it installs the per-machine schema: it
writes a missing copy and prints its path, reports one that differs, and
`--force` overwrites it. Only `dkt init` writes it. A plugin was dropped:
it would still need `dkt` installed on its own (carrying built binaries in
the plugin's git source is not practical), and Codex would need a second
format.

## Found, 2026-11-01

From the documentation of Claude Code 2.1.316 and Codex 0.153.4; nothing
was tried.

- User-level skills live in `~/.claude/skills/` for Claude Code and
  `~/.agents/skills/` for Codex. Codex has no custom `/name`; a skill is
  typed as `$name` or picked from `/skills`.
- In Claude Code a personal skill wins over a project skill of the same
  name, so the user-level entry must not be named `docket`, or it would
  hide each project's own `docket` skill.
- Claude Code picks up skills added to `.claude/skills/` while a session
  runs, but not when that folder did not exist when the session started:
  then the user types `/reload-skills`. Codex detects changes but says to
  restart when one does not show. In a new project the command therefore
  reads the `SKILL.md` files `dkt init` just wrote and follows them, rather
  than handing over to `/docket-setup` by name.
- A Claude Code plugin is installed by the user
  (`claude plugin install <plugin> --marketplace wake/docket`, user scope by
  default), its skills are typed as `/<plugin>:<skill>`, and the files in
  its `bin/` are on the PATH of the agent's shell. Third-party marketplaces
  do not update on their own unless the user turns it on. Codex has plugins
  too (`codex plugin marketplace add owner/repo`); their layout was not
  checked.

## Open questions

- Who installs it: `dkt` at the user's request, as it installs the
  per-machine schema, or a plugin. A user's own agent configuration is
  more personal than OpenSpec's data directory, so writing there may need
  its own consent. Prebuilt binaries (`2026-11-01-prebuilt-binaries.md`)
  change the weighing, since a plugin could bring `dkt` with it.
- Whether one skill folder can serve both agents: whether Codex ignores
  Claude Code's frontmatter keys is not documented.
- Its first install still needs a terminal; it only saves that step in
  later projects.

## Related

- `2026-10-31-existing-project-adoption.md`: how an existing project starts
  using Docket.
- `2026-10-31-global-schema.md`: the per-machine install it builds on.
