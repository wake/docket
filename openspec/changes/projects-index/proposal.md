---
title: "A local index of the projects on a machine"
kind: feature
source: "docket/ideas/dkt-projects-index.md"
specs:
  project-index: Project Index
---

# Proposal

## Why

Nothing on a machine knows which repositories use Docket. A tool that works across them, such as a viewer serving a machine's projects through a local server, would need its own list and setup. dkt already runs in every such project, so it can keep the list itself and hand it out.

## What Changes

- New command `dkt projects` with four subcommands: `add [<path>]`, `ls`, `rm <path>` and `ignore <path>`.
- An index file in the user's state directory (`$XDG_STATE_HOME/docket/projects.json`, else `~/.local/state/docket/projects.json`) holding each project's path, origin, time added, and whether it is ignored.
- Automatic registration: after any other command runs in a Docket project that is neither in the index nor ignored, dkt adds it. It writes only when the entry is missing.
- A settings file in the user's configuration directory (`$XDG_CONFIG_HOME/docket/config.json`, else `~/.config/docket/config.json`) whose only setting, `projects.autoRegister`, turns automatic registration off. Without the file the default, on, applies.
- `cli-output` says that the index is the one place outside the repository a command writes to.

## Non-Goals

- Stamps, caching or change detection: the tool that reads the index does that.
- Pruning: a path that is gone is marked `missing` and stays until `rm` removes it, since a path on an unmounted disk comes back.
- A command to read or edit the settings: the file is edited by hand.
- Refreshing a stored origin when a project's remote changes.
- Locking the index against two dkt processes writing it at once (see `design.md`).
- Registering from `dkt --version`, `--help` or a usage error, and from `dkt projects` itself.
- A viewer or server: `dkt projects ls --json` is the data one reads.

## Capabilities

### New Capabilities
- `project-index`: the index of Docket projects on a machine, the `dkt projects` command, the settings file, and automatic registration.

### Modified Capabilities
- `cli-output`: the index is the only place outside the repository that a command writes to, so "writes nothing" elsewhere in the specs means nothing in the repository.

## Impact

- New: `internal/projects` (index, settings, project path), `internal/cli/cmd_projects.go`.
- Changed: `internal/cli/command.go` (the command list, a command that needs no repository), `internal/cli/run.go` (automatic registration after a command), `README.md`.
- Every command now reads the settings and the index once, and writes the index only when it adds the project. A failure there is a warning on standard error and changes neither the output nor the exit code.
- No new dependency; no network; OpenSpec is not needed.
