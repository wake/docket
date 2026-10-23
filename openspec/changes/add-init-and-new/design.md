# Design

## Context

See proposal.md for why. The repository has no code yet. It has OpenSpec
initialized with `--tools none`, the `docket` schema in
`openspec/schemas/docket/`, and four changes that together make milestone 1:

| Order | Change | Commands | Depends on |
|---|---|---|---|
| 1 | `add-init-and-new` (this one) | `init`, `new`; shared CLI, scan, OpenSpec runner | nothing |
| 2 | `add-lint` | `lint`; readers for proposal, deltas, plan | 1 |
| 3 | `add-archive` | `sync`, `archive`, `rebase` | 1, 2 |
| later | `add-conflicts` | `conflicts` | 1, 2 |

`add-archive` comes third so that every change, these included, can be
closed through `dkt archive`. `add-conflicts` follows the status feed and
the web workbench in the queue kept in `docs/ideas/2026-10-24-web-workbench.md`;
it touches different capabilities from those and can run in a parallel
worktree. This change was opened with `openspec new change` and is linted by
hand, because `dkt` does not exist yet; from `add-lint` on, changes are
opened with `dkt new`.

Constraints that hold for the whole milestone:

- Public project: everything in the repository is in English, and nothing
  refers to a particular person's machines, hosts or customers.
- Docket keeps no state: every result can be recomputed from the repository
  and git. It never calls a language model and never judges content.
- Docket never edits AGENTS.md, CLAUDE.md, `.claude/` or
  `openspec/config.yaml`, and never touches the network.
- OpenSpec is pinned to 1.14.x. Docket does not reimplement OpenSpec's spec
  merging; only `openspec archive` writes `openspec/specs/`.
- Pull requests are merged with merge commits, so every commit keeps its
  `Change:` and `Task:` trailers.

## Goals / Non-Goals

**Goals:**

- A foundation the later commands share without reshaping it: output
  envelope, exit codes, git access, scan, OpenSpec runner.
- `dkt init` and `dkt new` usable on this repository as soon as the change
  merges, so `add-lint` can be opened with `dkt new`.

**Non-Goals:**

- Reading the content of proposals, deltas or plans. The readers arrive
  with `add-lint`; this change's scan only needs change and archive folder
  names.
- Windows. Paths and process handling are written for macOS and Linux.
- Release packaging (binaries, Homebrew). `go install` is enough for now.

## Approaches

How Docket reads the structure of deltas (which capability, which
operation, which requirement, which scenarios) decides most of the
milestone's design. It was settled for the whole milestone now, because the
scan built here has to support it.

- **A. Docket reads the structure itself; OpenSpec validates and archives
  (chosen).** A small line-based reader in Go, written to OpenSpec 1.14's
  own parsing rules, reading files from worktrees and blobs straight from
  git objects. Milliseconds per change, no checkout needed for branches,
  and `lint` and `conflicts` need no Node.js. The cost is following
  OpenSpec's format by hand.
- **B. Ask `openspec show --json --deltas-only` for everything.** Always
  agrees with OpenSpec, but measured at about 0.5 s per change on 1.14.3,
  needs branches exported to a temporary directory first, needs Node.js for
  every check, and still leaves plan.md and proposal.md for Docket to parse,
  because OpenSpec knows nothing of tasks or contracts.
- **C. OpenSpec for worktrees, Docket for git objects.** Dropped: two
  readers of the same files can disagree.

A was chosen because `conflicts` runs often across many parallel worktrees,
and plan.md and proposal.md need a Docket reader anyway. Its risk is
contained in `add-lint` and `add-archive`: the reader follows OpenSpec's
source and test cases, a contract test compares it with
`openspec show --json` on a shared set of examples, anything it cannot
classify is an error rather than a silent skip, and `dkt archive` reads
every delta both ways and refuses to archive on any disagreement.

## Decisions

- **Module and layout.** Module path `github.com/wake/docket`, so
  `go install github.com/wake/docket/cmd/dkt@latest` works. A package at the
  module root holds only the embedded schema; the binary is `cmd/dkt`; all
  other code is under `internal/`. Package boundaries inside `internal/` are
  left to the plan.
- **Go, standard library first.** Commands are dispatched by hand with the
  `flag` package; no CLI framework. One dependency, `go.yaml.in/yaml/v3`, for
  `.openspec.yaml`. `go.mod` declares Go 1.24 so that recent toolchains
  build it. *Alternative:* cobra; rejected because seven commands do not
  need it and every dependency is one more thing to keep current.
- **Git through the `git` binary, not a library.** `git` is required anyway,
  and calling it keeps behavior identical to the user's own git: worktrees,
  `safe.directory`, alternates, configured remotes. Reading many blobs goes
  through one `git cat-file --batch` process per source. *Alternative:*
  go-git; rejected for differing from git on worktrees and config.
- **The shipped schema is this repository's own schema.** A Go package at
  the module root embeds `openspec/schemas/docket/` with `go:embed`, so the
  files `dkt init` installs elsewhere are the same files this repository
  uses, with no copy to drift.
- **Result envelope built once.** Each command returns a result value plus
  findings and unreadable sources; one writer turns that into JSON or text
  and picks the exit code (3 over 1 when both apply). Usage errors (2) and
  environment errors (4) stop before any work.
- **OpenSpec runner.** Resolves `DOCKET_OPENSPEC` or `openspec`, checks the
  version once per process before first use, runs with the repository root
  as working directory, and captures standard error for reporting. The
  environment is passed through unchanged.
- **Scan.** `git worktree list --porcelain` for worktrees (a `prunable`
  entry or a missing directory is unreadable);
  `git for-each-ref refs/heads refs/remotes` with
  `git merge-base --is-ancestor <tip> <default-tip>` to skip merged
  branches; `git ls-tree` on `openspec/changes/` for each branch source; the
  filesystem for each worktree. The scan returns slugs, their sources, their
  archived uses, and unreadable sources. Content readers plug into it in
  later changes.
- **`x-base` written by Docket.** `openspec new change` writes
  `.openspec.yaml`; Docket then appends the three `x-` keys. OpenSpec 1.14
  writes that file only when it creates a change, and reads `x-` keys as
  extension metadata without warning (`src/utils/change-metadata.ts`).
  Docket never rewrites the keys OpenSpec wrote.
- **Architecture map written with the code.** This change adds
  `docs/ARCHITECTURE.md`: a one-page codemap of `cmd/` and `internal/`, a
  short index of shared capabilities ("run git" → its package and function,
  "read a change folder from a worktree or a git tree" → …), and the
  invariants, for example "only the OpenSpec runner starts `openspec`" and
  "readers take bytes, never paths". Later changes read it when they write
  Builds On and the Reuse table, and update it when they add a module or
  move a boundary. AGENTS.md points to it in one line.
- **Duplicate detection in CI.** CI runs jscpd against the default branch
  (`--baseline-from-ref origin/main --fail-on-new-clones 0`), so a pull
  request fails on new duplicated code without being blocked by any that
  already exists. CI has Node.js for the OpenSpec integration job anyway.
- **Tests.** Go tests build real git repositories in temporary directories:
  a bare remote, clones, worktrees, unmerged and merged branches. A fake
  OpenSpec (a test helper binary) stands in for unit tests. An integration
  job in CI runs the same scenarios against the real CLI through
  `DOCKET_OPENSPEC="npx -y @fission-ai/openspec@1.14.3"`.

## Risks / Trade-offs

- [OpenSpec 1.15 changes `new change` output or metadata handling] → The
  version gate refuses untested versions; the CI integration job is the
  place a version bump is tried first.
- [`origin/HEAD` is often unset in clones made by tools] → Falls back to
  `main`, then `master`, and `--default-branch` overrides.
- [Remote-tracking refs are stale] → Reported as read "as of last fetch";
  Docket never fetches on its own. Users fetch before planning.
- [A worktree on an unmounted disk blocks `dkt new`] → Intended: an unknown
  source could already use the slug. The message names the worktree and
  suggests `git worktree prune` when the worktree is really gone.

## Open Questions

None.
