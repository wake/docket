# Proposal

## Why

Projects that plan with superpowers and record behavior in OpenSpec have no
tool that enforces the rules joining the two; the rules live in prompts, and
agents skip them. Docket replaces those rules with checks a program can run.
This first change gives Docket its command-line foundation and the two
commands that start the workflow: preparing a repository and opening a change.
Everything later (`lint`, `conflicts`, `sync`, `archive`) builds on it.

## What Changes

- A `dkt` binary whose commands share one set of conventions: `--json`
  output, the repository and commit in every result, fixed exit codes, and
  sources that could not be read reported as unreadable, never as empty.
- Locating the OpenSpec CLI and refusing versions outside the tested range.
- A scan of every place a change can live: worktrees, local and
  remote-tracking branches, and the default branch with its archive.
- `dkt init`: installs the `docket` OpenSpec schema into a repository.
- `dkt new`: opens a change with the `docket` schema and records its kind,
  its source and the commit it started from.

## Non-Goals

- No `lint`, `conflicts`, `sync`, `archive` or `rebase`. They come in the
  changes `add-lint`, `add-conflicts` and `add-archive`.
- Docket never edits AGENTS.md, CLAUDE.md, anything under `.claude/`, or
  `openspec/config.yaml`.
- No network access: Docket never fetches, pushes or calls a remote service.
- No state of its own: no database, and nothing written under `.git/`.
- No configuration file. The conventions (schema name, kinds, trailers) are
  fixed in this milestone.
- No Windows support in this milestone; macOS and Linux only.

## Builds On

- Nothing in this repository: it has no code yet (searched with `git ls-files`).
- Reused instead of written: the OpenSpec CLI for `init`, `new change` and
  its change metadata, where `x-` keys are an official extension point; the
  git CLI for worktrees, refs and trees (`worktree list --porcelain`,
  `for-each-ref`, `merge-base`, `ls-tree`, `cat-file --batch`).
- The `docket` schema already in `openspec/schemas/docket/`, forked from
  OpenSpec's `spec-driven`, is what `dkt init` ships.

## Capabilities

### New Capabilities

- `cli-output`: what every command prints and returns: JSON output, the
  result envelope, repository identity, exit codes, unreadable sources.
- `openspec-runtime`: how Docket finds the OpenSpec CLI and which versions
  it accepts.
- `workspace-scan`: where Docket looks for changes, how it identifies them,
  and how it reports what it could not read.
- `project-setup`: installing the `docket` schema with `dkt init`.
- `change-creation`: opening a change with `dkt new`.

### Modified Capabilities

None.

## Impact

- New Go module with the `dkt` binary, `docs/ARCHITECTURE.md`, and a CI
  workflow running `go vet`, `go test` and a jscpd check for new duplicated
  code.
- Runtime dependencies: `git`; the OpenSpec CLI 1.14.x for `dkt init` and
  `dkt new`.
- The schema in `openspec/schemas/docket/` becomes the copy Docket ships:
  this repository uses the same files that `dkt init` installs elsewhere.

## Implementation Contract

### Observable Behavior

- `dkt init` in a git repository leaves `openspec/schemas/docket/` identical
  to the shipped schema, running `openspec init --tools none` first when
  `openspec/` is missing, and prints the AGENTS.md routing section for the
  user to add.
- `dkt new <slug> --kind <kind> --source <text>` creates
  `openspec/changes/<slug>/` with the `docket` schema and `x-kind`,
  `x-source` and `x-base` in its `.openspec.yaml`, or creates nothing and
  says why.
- Every command prints human-readable text, or with `--json` exactly one
  JSON object.

### Interfaces

- `dkt init [--force] [--json]`
- `dkt new <slug> --kind feature|change-request|fix|baseline --source <text> [--json]`
- Flags on every command: `--json`, `--default-branch <name>`.
- Environment: `DOCKET_OPENSPEC` names the OpenSpec command.
- Exit codes: 0 clean, 1 problems found or request refused, 2 usage error,
  3 a needed source could not be read, 4 unusable environment.
- JSON envelope: `tool`, `version`, `command`, `repo`, `ok`, `findings`,
  `unreadable`, `result`.
- `.openspec.yaml` extension keys: `x-kind`, `x-source`, `x-base`.

### Failure Modes

- Outside a git repository, or `git` missing: exit 4.
- OpenSpec CLI missing, or its version outside 1.14.x: exit 4, naming the
  version found, the range accepted, and `DOCKET_OPENSPEC`.
- A listed worktree that cannot be read: reported as unreadable; `dkt new`
  then creates nothing and exits 3, since it cannot show the slug is unused.
- Invalid, date-prefixed or already used slug: exit 1, naming every place
  the slug is used.
- `docket` schema missing when running `dkt new`: exit 1, pointing to
  `dkt init`.
- A schema file that differs from the shipped one, without `--force`: left
  as it is and reported; exit 1.
- `openspec` itself fails: its exit code and error output are reported, and
  no step that depends on it runs.

### Acceptance Criteria

- JSON output on success and failure: `cli-output#json-on-success`,
  `cli-output#json-on-failure`, `cli-output#envelope-fields-present`.
- Repository identity: `cli-output#detached-head`, `cli-output#no-origin-remote`.
- Exit codes: `cli-output#usage-error`, `cli-output#not-a-git-repository`,
  `cli-output#unreadable-source-with-other-findings`.
- OpenSpec CLI: `openspec-runtime#override-with-npx`,
  `openspec-runtime#default-from-path`, `openspec-runtime#unsupported-version`,
  `openspec-runtime#cli-missing`, `openspec-runtime#openspec-command-fails`.
- Scan: `workspace-scan#change-only-in-another-worktree`,
  `workspace-scan#change-only-on-a-remote-tracking-branch`,
  `workspace-scan#merged-branch-skipped`, `workspace-scan#origin-head-set`,
  `workspace-scan#no-default-branch`, `workspace-scan#merged-change-also-on-a-branch`,
  `workspace-scan#worktree-with-unmerged-work-comes-first`,
  `workspace-scan#archived-slug-found`, `workspace-scan#worktree-on-an-unmounted-volume`,
  `workspace-scan#scan-offline`.
- Setup: `project-setup#fresh-repository`, `project-setup#existing-openspec-project`,
  `project-setup#locally-edited-template`, `project-setup#re-run-with-no-differences`,
  `project-setup#routing-section-printed`.
- New change: `change-creation#date-prefix-rejected`,
  `change-creation#uppercase-rejected`, `change-creation#reserved-name-rejected`,
  `change-creation#slug-active-in-another-worktree`,
  `change-creation#slug-archived`, `change-creation#unreadable-worktree-blocks-creation`,
  `change-creation#missing-source`, `change-creation#unknown-kind`,
  `change-creation#change-created`, `change-creation#schema-not-installed`.

## Change Log

| Date | Source | Change | Impact |
|---|---|---|---|
| 2026-10-24 | self: Task 5 implementer's report | Content precedence ranks a worktree by whether the default branch already contains its HEAD; the main checkout no longer shadows an unmerged branch. New scenario `workspace-scan#worktree-with-unmerged-work-comes-first` | Task 5 gets a fix round; no other task affected |
| 2026-10-24 | self: Task 7 task review | The slug `archive` is reserved: it names OpenSpec's archive folder, and a failed `dkt new archive` could have removed it. New scenario `change-creation#reserved-name-rejected` | Task 7 gets a fix round; no other task affected |
| 2026-10-24 | review: codex review of PR #5 | `dkt init` and `dkt new` refuse to write through a symbolic link under `openspec/`, and runs of `dkt new` in one worktree take turns, so a failing run cannot remove a folder another run created. New scenarios `project-setup#symlinked-schema-file-refused`, `change-creation#symlinked-changes-folder-refused`, `change-creation#concurrent-runs-keep-the-first` | Tasks 6 and 7 get a fix round; no other task affected |
