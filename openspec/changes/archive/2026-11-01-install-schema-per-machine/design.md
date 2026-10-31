# Design

## Context

See proposal.md for why. What shapes the approach:

- OpenSpec 1.14.3 resolves a schema from the project's `openspec/schemas/`,
  then its data directory (`getGlobalDataDir`: `XDG_DATA_HOME` when set
  and not empty, used as given even when relative; else
  `~/.local/share/openspec`, or `LOCALAPPDATA` on Windows), then its
  package. `openspec schema which --all --json` reports each schema's
  source and what it shadows, but no command prints the data directory
  when nothing is installed there.
- When the schema cannot be resolved, `openspec archive` does not fail: its
  task count swallows the error and falls back to a top-level `tasks.md`
  (`resolveTrackedTasksGlob` in `dist/utils/task-progress.js`). A Docket
  change has none, so the count silently becomes zero.
- `dkt` builds only for Unix-like systems (`syscall.O_NOFOLLOW`,
  `syscall.Flock`), so the Windows branch does not apply.
- `setup.Init` decides every target before it writes and refuses on any
  problem; `install` writes through a temporary file and a rename.
- Tests run `dkt` with `HOME` set to a temporary directory
  (`testutil.Env`), and the real OpenSpec CLI in CI gets the same
  environment, so both resolve the same temporary data directory.
- Four distinct schema trees have shipped, from v0.1.0 to v0.9.2: the
  trees at v0.1.0, v0.4.0, v0.5.0 and v0.9.2.

## Goals / Non-Goals

**Goals:**

- One rule for where the schema lives, in one place (`internal/setup`),
  used by `init`, `inspect`, `new` and `archive`.
- A migration that removes only what can be restored from a Docket
  release, and never leaves a half-removed copy where OpenSpec reads it.

**Non-Goals:**

- Asking OpenSpec where its data directory is.
- Supporting `DOCKET_OPENSPEC` wrappers that change `XDG_DATA_HOME` or
  `HOME` for OpenSpec only.

## Decisions

### Approaches considered (decided with the user)

1. **Keep a copy per project** (today). Rejected: the schema and the lint
   that checks it drift apart after every upgrade, and each schema change
   costs every project a pull request.
2. **Per machine, written by whichever command finds it out of date.**
   Rejected by the user: nothing may write without the user running a
   command for it or agreeing.
3. **Per machine, written by a separate host-level command.** Rejected by
   the user: one command keeps adoption to one step.
4. **Per machine, written by `dkt init`** (chosen). `--force` overwrites
   the per-machine copy together with the project's skills and block,
   which is what an upgrade wants anyway.

Also decided with the user: forks are left for later, and the headings
`dkt lint` reads stay fixed English keywords, as OpenSpec's delta markers
are.

### Locating the data directory

`dkt` mirrors `getGlobalDataDir` for Unix: `$XDG_DATA_HOME/openspec` when
`XDG_DATA_HOME` is not empty, else `<home>/.local/share/openspec`, with
the home directory found as `workflow.Locate` finds it (`HOME` when
absolute, else the user lookup). The environment comes through
`cli.Options`, as it does for the workflow file.

A relative `XDG_DATA_HOME` is refused with exit 4 rather than followed.
OpenSpec resolves it against its working directory, which is the
repository root when `dkt` starts it but whatever directory an agent is in
when it runs `openspec` itself, so no single directory is the right one.
`workflow.Locate` ignores a relative `XDG_CONFIG_HOME` instead; following
that here would write where OpenSpec does not read.

Alternative: ask OpenSpec. Rejected: no command reports the directory
before a schema is installed there, `schema which` is marked experimental,
and it would add an OpenSpec process to `dkt inspect`.

### Telling a shipped copy from an edited one

A schema tree is identified by a SHA-256 over its files, sorted by path,
each as its slash-separated path, a NUL, its length and its bytes. The
hashes of the four shipped trees are listed in `internal/setup`. A test
asserts that the hash of the embedded schema is in the list and prints it
when it is not, so every edit of `schema/docket/` adds its hash in the
same pull request. The list may then hold trees that were never released;
that only lets `dkt init` remove a copy of one of them, which a release
of Docket's own repository could have produced.

A project copy counts as shipped only when its file set and every byte
match one listed tree. An extra file, such as a user's own template,
makes it edited. A copy made by a newer `dkt` is unknown to an older one
and is kept as edited, so mixed versions never delete each other's copy.

`result.schema.matches` in `new` and `archive` compares the copy found
with the embedded tree by the same hash.

### Order in `dkt init`

1. Decide every target and check every path, as today: per-machine files,
   skills, blocks and the project copy. Any problem refuses the whole run.
2. Run `openspec init --tools none` when `openspec/` is missing.
3. Decide `openspec/config.yaml` now, because step 2 may have written it.
4. Write the per-machine files, the project files and `config.yaml`.
5. Remove the project copy last, and only when every per-machine file is
   `unchanged`, `written` or `overwritten`.

Removal first renames `openspec/schemas/docket` to
`openspec/.docket-removed-<pid>`, which OpenSpec does not read, and then
removes that tree. A failure after the rename leaves only that folder,
and the error names it for the user to delete; no later run deletes a
folder by its name alone, since its content was not checked. Removing
file by file instead could leave a partial copy that OpenSpec would
still resolve and that would then count as edited.

A folder `openspec/schemas/docket/` without `schema.yaml` is not a copy:
OpenSpec does not read it as a schema, so `dkt init` leaves it and
reports nothing.

### The `schema` key of `config.yaml`

`dkt` reads it line by line, and cross-checks the result with the YAML parser
the module already has: a file that does not parse, is not a mapping, or has
a top-level `schema` key the line reader did not find (quoted, spaced, in a
flow mapping) is unreadable. A top-level key is a
line starting `schema:` at column 0; lines inside a block scalar are
indented, so none of them can be one. Its value must
be on the same line: a bare word, or one in single or double quotes,
optionally followed by a comment. No such line means no key. More than one
such line, or a `schema:` line with nothing after it, is unreadable.
Replacing keeps the line ending and any trailing comment.

### Symbolic links outside the repository

In the repository, links stay refused everywhere, as today. In the data
directory, links at or below `openspec/schemas/docket` are refused,
because that is the folder `dkt` owns. Links above it are the user's own
layout, for example a data directory on another disk, and are followed.
A repository cannot plant a link outside itself, so the threat that the
repository rule guards against does not arise there.

### Reporting

Per-machine files reuse `schema-missing` and `schema-differs` with
absolute paths, and the human lines print those paths, which is how
`dkt init` "prints the path". `dkt inspect` gains the file kind `config`.
`new` and `archive` report a differing copy only in `result.schema` and a
human line, not as a finding, so their exit code still means whether the
change was created or archived.

### This repository

The schema's source moves to `schema/docket/` with `git mv`, the embed in
`schema.go` follows, and `openspec/schemas/docket/` is gone, so this
repository uses the per-machine copy like any project. The move is not
split into its own pull request: without the per-machine copy it would
leave this repository's own `openspec` commands without a schema.
Tests keep `testutil.InstallSchema` for a project copy and gain
`testutil.InstallDataSchema` for a per-machine one, under the test's
temporary data directory.

## Risks / Trade-offs

- [A teammate on an older `dkt` runs `dkt new`, is told the schema is not
  installed, and `dkt init` writes the project copy back] → The README
  upgrade note says that everyone on a project upgrades; a copy written
  back is a shipped tree, so the next `dkt init` on a newer `dkt` removes
  it again.
- [A `DOCKET_OPENSPEC` wrapper gives OpenSpec another `XDG_DATA_HOME`, as
  a sandbox does] → Not supported; run `dkt` with the same variable. The
  README says so next to `DOCKET_OPENSPEC`.
- [An edit to the schema forgets its hash] → The test fails in that pull
  request.
- [CI that runs `openspec` on a change now needs the schema installed] →
  It runs `dkt init` first; this repository's CI already runs OpenSpec
  only through `go test`, whose tests install it.
- [A project copy left in place by a differing per-machine copy keeps the
  project on the old rules] → Reported as `schema-project-copy` until
  `dkt init --force`.

## Migration Plan

1. Release with an upgrade note: after upgrading, run `dkt init` in each
   project (the first run also installs the per-machine copy), review that
   it removed `openspec/schemas/docket/` and set `schema: docket`, and
   commit both.
2. Rollback: an older `dkt init` writes the project copy back, and the
   per-machine copy is ignored by OpenSpec while a project copy exists.
