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

## Open questions

- Who installs it: `dkt init` as part of its per-machine step, a separate
  command, a plugin, or the user by hand. A user's own agent
  configuration is more personal than OpenSpec's data directory, so
  writing there may need its own consent.
- Where Codex looks for user-level skills, and whether one copy can serve
  both agents.
- Its first install still needs `dkt init` from a terminal; it only saves
  that step in later projects.

## Related

- `2026-10-31-existing-project-adoption.md`: how an existing project starts
  using Docket.
- `2026-10-31-global-schema.md`: the per-machine install it builds on.
