---
title: Install the shipped schema once per machine
---

Recorded 2026-10-31, from the discussion of adopting Docket in an existing
project. Decided: Docket follows OpenSpec's layout. The shipped schema is
not copied into projects; a project names it, and a project that wants
other rules forks it under a new name. Also decided: nothing writes the
per-machine copy unless the user runs a command for it or agrees to it;
other commands only report it missing or out of date.

## Today

- `dkt init` copies the schema into every project's
  `openspec/schemas/docket/`. `dkt` never reads its content; OpenSpec does,
  when an agent runs `openspec instructions` or `openspec status` and when
  `dkt new` and `dkt archive` call it. The rules `dkt lint` checks are in
  the binary.
- `openspec init` leaves `openspec/config.yaml` at `schema: spec-driven`,
  and `dkt init` does not change it. Each change names `docket` in its
  `.openspec.yaml`, because `dkt new` passes `--schema docket`.

## Why

- One set of rules is split in two: the schema in the project says what
  the agent writes, and the lint in the binary checks it. After upgrading
  `dkt`, a project's copy stays old until someone runs `dkt init --force`.
  Only `dkt inspect` notices; `lint` and `status` do not.
- `dkt inspect` compares the copy with its own and cannot tell an older
  copy from a newer one. On a machine with an older `dkt`, the advice to
  run `dkt init --force` overwrites a newer committed copy.
- Every schema change costs every project a pull request, reviewed as
  prompt text. Of the eleven releases after v0.1.0, three changed the
  schema.

## How OpenSpec does it

- Its built-in `spec-driven` schema stays in its package; a project only
  names it in `config.yaml`.
- It looks for a schema in the project's `openspec/schemas/`, then in
  `$XDG_DATA_HOME/openspec/schemas/` (by default
  `~/.local/share/openspec/schemas/`), then in its package.
- `openspec schema fork <source> [name]` copies a schema into the project
  under a new name, `<source>-custom` by default.
- Its standalone binary keeps a copy of its built-in schemas in the same
  data directory.

Tried with OpenSpec 1.14.3 in a sandbox:

- With the schema only in the user directory, `openspec schema which
  docket` reports `Source: user`, and `status` and `instructions` work.
  Without it: `Unknown schema 'docket'. Available: spec-driven`.
- `openspec schema fork docket` reads it from there and writes
  `openspec/schemas/docket-custom/`: the name changes and two description
  lines are rewrapped; the instructions and the templates are unchanged.
  The fork does not record what it was forked from.

## Proposed shape

To settle in the change:

1. `dkt` owns `$XDG_DATA_HOME/openspec/schemas/docket/` and keeps it equal
   to the schema it embeds.
2. Only a command the user runs to install it writes it (decided). `dkt
   new`, `dkt archive` and `dkt inspect` report it when it is missing or
   differs, and name that command. Writing it from `dkt new` whenever it
   is out of date was considered and dropped: nothing may write without
   the user's action or consent. Nobody edits it in place: other rules
   are a fork.
3. `dkt init` sets `schema: docket` in `openspec/config.yaml`. That also
   marks the project as one that uses Docket.
4. A project with a fork names it in `config.yaml`, and `dkt new` uses the
   schema `config.yaml` names.
5. A project's `openspec/schemas/docket/` wins over the user directory, so
   the copies made so far go:
   - one equal to a shipped version is removed by `dkt init`, which needs
     the hashes of every shipped version;
   - one equal to none counts as edited in place, and `docket-doctor`
     helps fork it under a new name;
   - Docket's own repository keeps its copy, which is the source.

Accepted costs:

- Each machine uses the schema of its own `dkt`; a project does not pin a
  version.
- CI that runs `openspec` on a Docket change installs the schema first.
  CI that runs only `dkt lint` needs nothing, since lint does not call
  OpenSpec.

## Related

- `2026-10-31-existing-project-adoption.md` builds on this: its `init`
  and `inspect` checks and the lint that reads a change's `schema`.

## Open questions

- How `dkt` tells that a fork is Docket's, since the fork does not say
  what it came from.
- What `dkt init` does when `config.yaml` already names another schema,
  as in a project with `spec-driven` changes.
- Whether the record and workflow templates, already in the binary, get a
  project override too.
