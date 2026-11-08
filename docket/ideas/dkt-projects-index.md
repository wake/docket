---
title: A local index of the projects on a machine
---

dkt itself keeps track of which projects on a machine use Docket. A tool that works across the projects on a machine, such as a viewer that serves them through a local server, passes this list on and needs no setup of its own to know them.

## In scope

- `dkt projects add [path]`: accepts only a repository that has `docket/workflow.md`; a worktree is replaced by its main repository; a path already in the index is reported as already there. An explicit `add` also lifts an ignore.
- `dkt projects ls [--json]`: lists every entry; a path that no longer exists is marked `missing`, not removed.
- `dkt projects rm <path>`: only removes the entry.
- `dkt projects ignore <path>`: excludes the project, so it is no longer registered on its own.
- Automatic registration: every dkt command checks the current project and adds it when it is neither in the index nor ignored; it writes only when the entry is missing.
- The index, `~/.local/state/docket/projects.json` (`$XDG_STATE_HOME`): each entry's path, origin, time added and ignore.
- The settings, `~/.config/docket/config.json` (`$XDG_CONFIG_HOME`): for now only `{"projects": {"autoRegister": true}}`; when the file does not exist, the defaults apply.

## Out of scope

- Stamps and pruning: caching and change detection belong to the tool that reads the index; a path that is gone is only marked, and `rm` removes it. A path on an external disk is also gone for a while before the disk is mounted.
- A command for the settings: edit the file by hand when needed.
