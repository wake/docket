# Design

## Context

See `proposal.md` - Why. Today every command resolves the repository in `runCommand` (`internal/cli/run.go`) before it runs, and fails with `no-repository` outside one; only `--version` identifies the repository on a best-effort basis. Nothing in dkt writes outside the repository, and several specs say a command "writes nothing" (`setup-inspection`, `domain-tree`, `plan-sync --check`).

## Goals / Non-Goals

**Goals:**

- `dkt projects ls --json` is the whole list a reader needs, ignored entries included, without running anything else.
- A project shows up the first time any dkt command runs in it, at the cost of reading two small files.
- No command's output or exit code depends on the index.

**Non-Goals:**

- The proposal's non-goals. At design level, also: a file format version, and keeping keys a later dkt might add to an entry.

## Decisions

**A package of its own, `internal/projects`.** It holds the index (load, save, find, sort), the settings, the two file locations from the environment, and the project path of a directory. `dkt projects` and the registration in `runCommand` both call it, so both resolve a project the same way. It takes the environment as `[]string`, as `cli.Options.Env` is, so tests point it at temporary directories without touching the process environment.

**Registration runs after the command, in `runCommand`.** Running after lets `dkt init` register the project it has just set up. It runs whatever the command's result, because the project exists either way, and skips `dkt projects` so that `rm` and `ignore` mean exactly what they say. `--version`, `--help` and usage errors never reach that point.

**The fast path costs no git run.** Registration first looks for an entry whose path is `repo.root`, which the command already resolved. Only when there is none and `docket/workflow.md` exists does it list the worktrees to find the main one, compare again, and write. A project already registered therefore costs reading the settings and the index, nothing more.

**The main worktree is the first entry of `git worktree list`.** `gitx.Worktrees` already returns it first. For a bare repository that entry is the bare folder; it is still a stable name for the repository, so it is used as is rather than adding a case. `docket/workflow.md` is looked for in the worktree the command ran in, since that is where a user setting up Docket on a branch has it.

**A failed registration is a warning, not a finding.** Adding a finding would turn `dkt status` into exit 1 on a machine whose index is broken, and a JSON consumer would see a problem that is not the project's. One line on standard error is enough for a person, and the next command tries again.

**`dkt projects` needs no repository.** `Command` gets a field `NoRepo`; with it, `runCommand` identifies the repository as `--version` does (any failure gives `repo` null) and still runs the command. `ls`, `rm` and `ignore` work from anywhere; `add` without a path then reports `not-a-repository`.

**The file is written whole, through `safefile.Install`.** A temporary file beside the index is renamed over it, so a failed write leaves the old index. The mode is `0600`: origins can be URLs holding credentials, as `repo.origin` already shows. Entries are sorted by path on write, so the file diffs cleanly and `ls` needs no extra sort. The index is read with `safefile.ReadNoFollow`, so a link at the index itself is unreadable rather than followed.

**A broken index is never written over.** `dkt projects` lists it as unreadable and exits with 3; registration warns and stops. Rewriting it from an empty index would lose every entry.

**Times are UTC, to the second, RFC 3339.** `Context.Now` gives them, so tests fix the clock.

**Text output.** `add`, `rm` and `ignore` print one line `<state> <path>`. `ls` prints one line per entry, the path followed by `missing` and `ignored` when they apply, and nothing for an empty index.

## Risks / Trade-offs

- [Two dkt processes write the index at the same time] → Each reads, changes and replaces the whole file, so one change can be lost; the file itself is never torn. A lost automatic registration is redone by the next command; a lost `rm` or `ignore` needs the person to run it again. A lock file was not added: the window is a few milliseconds around a write that happens once per project.
- [An origin holding a token is copied into the index] → The file is `0600`, and the same URL is already printed in every envelope's `repo.origin`. Stripping credentials is left to the tool that serves the list.
- [Every command now reads two more files] → Both are small; the fast path runs no git. A broken file costs one warning line per command until it is fixed.
- [A stored origin goes stale when the remote changes] → Out of scope; `rm` then `add` refreshes it.
- [Tests of other commands now write an index] → `testutil.Env` gives each test its own `HOME`, so they write into a temporary folder.

## Migration Plan

None. A machine without the files gets them on the first command; deleting `projects.json` empties the index, and `autoRegister: false` stops new entries.
