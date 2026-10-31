# Proposal

## Why

Every project carries its own copy of the `docket` schema while the rules
`dkt lint` checks live in the binary, so after an upgrade the two disagree
until someone runs `dkt init --force`, and `dkt inspect` cannot tell an
older copy from a newer one. Of the twelve releases after v0.1.0, three
changed the schema, and each cost every project a pull request.

## What Changes

- **BREAKING** `dkt init` installs the schema once per machine, in the
  directory OpenSpec 1.14 reads user schemas from
  (`$XDG_DATA_HOME/openspec/schemas/docket/`, by default
  `~/.local/share/openspec/schemas/docket/`), instead of into the
  project's `openspec/schemas/docket/`. A missing copy is written and its
  path printed; a differing one is reported, and `--force` overwrites it
  together with the project's skills and Docket block.
- `dkt init` removes a project copy that is byte-for-byte one of the
  schemas a Docket release shipped, once the per-machine copy matches the
  shipped one. A project copy equal to none of them is kept, still wins
  over the per-machine one as OpenSpec resolves it, and is reported.
- `dkt init` sets `schema: docket` in `openspec/config.yaml` when the file
  is missing, has no `schema` key, or names OpenSpec's default
  `spec-driven`, changing only that line. Another name is reported and
  left, with or without `--force`.
- `dkt new` and `dkt archive` find the schema as OpenSpec does: the
  project's copy, then the per-machine one. When neither exists they
  create or archive nothing and name `dkt init`; when the one found
  differs from the shipped schema they report it and go on.
- `dkt inspect` reports the per-machine copy, a project copy and the
  `schema` key of `openspec/config.yaml`.
- In this repository the schema's source moves from
  `openspec/schemas/docket/` to `schema/docket/`, beside
  `workflow/template.md` and `record/template.md`, so Docket uses the
  per-machine copy like any other project.

## Non-Goals

- Forks of the schema: naming one in `config.yaml`, recognizing one, or
  `docket-doctor` helping make one (`docket/ideas/2026-10-31-schema-fork.md`).
- Headings in other languages: the headings `dkt lint` reads stay fixed
  English keywords.
- Pinning a schema version per project: each machine uses its own `dkt`'s.
- Writing the per-machine copy from any command other than `dkt init`,
  or without the user running it.
- Deleting files in the per-machine directory that the shipped schema
  does not have.
- Adoption of projects with changes made under another schema
  (`docket/ideas/2026-10-31-existing-project-adoption.md`).

## Builds On

- `project-setup`: `setup.Init` and `setup.Inspect` in
  `internal/setup/setup.go` already decide every file before writing,
  refuse symbolic links through `safepath.Check`, and write atomically
  (`install`). The per-machine copy, the removal and `config.yaml` become
  more targets of the same decide-then-write pass.
- `materialized-workflow`: `workflow.Locate` already finds a per-user file
  from `XDG_CONFIG_HOME` or the home directory, with the environment and
  the user lookup passed in through `cli.Options`; the data directory is
  found the same way from `XDG_DATA_HOME`.
- `setup-inspection`: `inspectStatus` in `internal/cli/cmd_inspect.go`
  names the `schema-missing` and `schema-differs` findings this change
  reuses for the per-machine copy.
- `openspec-runtime`: how Docket finds the OpenSpec CLI; the schema's
  location is added there, since it is where that CLI reads it.
- `docket.Schema()` in `schema.go` embeds the schema; only its source path
  changes.
- Searched for an existing reader of OpenSpec's data directory or of
  `config.yaml`'s `schema` key: none (`rg -n 'XDG_DATA_HOME|config\.yaml'
  --type go` finds only test fixtures).

## Capabilities

### New Capabilities

### Modified Capabilities
- `openspec-runtime`: where Docket finds the `docket` schema.
- `project-setup`: `dkt init` writes the per-machine copy, removes shipped
  project copies and sets the `schema` key; the files it writes change.
- `setup-inspection`: `dkt inspect` reports the per-machine copy, project
  copies and the `schema` key, and writes nothing outside the repository
  either.
- `change-creation`: `dkt new` finds the schema in either place and
  reports one that differs.
- `change-archive`: `dkt archive` requires the schema before any step and
  reports one that differs.

## Impact

- `internal/setup`, `internal/cli` (`init`, `inspect`, `new`, `archive`),
  `cmd/dkt` (passing the environment it already passes), `schema.go`
  (embed path), and the test helper that installs the schema for tests
  that run a real OpenSpec.
- Prompt text: the `docket-doctor` skill's list of findings; `AGENTS.md`'s
  list of prompt paths.
- Docs: README (install, upgrade note, what `init` writes),
  `docs/ARCHITECTURE.md`.
- Projects: after upgrading, running `dkt init` once per machine and once
  per project; teammates on an older `dkt` re-create the project copy
  until they upgrade. CI that runs `openspec` on a change runs `dkt init`
  first.

## Implementation Contract

### Observable Behavior

- On a machine without the per-machine copy, `dkt init` in any project
  writes it and lists each file with its absolute path.
- After upgrading `dkt`, `dkt new` and `dkt inspect` say the per-machine
  copy differs and that `dkt init --force` overwrites it.
- In a project set up by an older `dkt`, `dkt init` removes
  `openspec/schemas/docket/` when it is an unedited copy, and sets
  `schema: docket` in `openspec/config.yaml`; the user commits both.
- An edited project copy keeps working as today, and `dkt init` and
  `dkt inspect` say so.

### Interfaces

- Per-machine directory: `$XDG_DATA_HOME/openspec/schemas/docket/` when
  `XDG_DATA_HOME` is set and not empty, else
  `<home>/.local/share/openspec/schemas/docket/`.
- `dkt init`: `result.files[]` lists per-machine files with absolute
  paths and the existing statuses; project copy files removed get the new
  status `removed`; `openspec/config.yaml` is listed as `written`,
  `unchanged` or `differs`.
- New findings: `schema-edited` (a project copy equal to no shipped
  schema), `schema-project-copy` (a project copy of a shipped schema not
  yet removed), `config-schema-default` (`dkt inspect` only: the key is
  missing or `spec-driven`), `config-schema-other` (another name).
  `schema-missing` and `schema-differs` also name per-machine files.
- `dkt new` and `dkt archive`: `result.schema` is
  `{"path": "<absolute directory>", "matches": <bool>}`, the copy OpenSpec
  reads and whether it is byte-for-byte the shipped schema. `dkt new`
  gives it with the change it created; a later failure has no result, as
  today. `dkt archive` gives it in every result once it found the schema,
  and `null` before.
- `dkt inspect`: `result.files[]` gains the per-machine files (kind
  `schema`, absolute path) and `openspec/config.yaml` (kind `config`).

### Failure Modes

- `XDG_DATA_HOME` set to a relative path, or no home directory: `dkt init`
  exits with 4 naming the reason and writes nothing; `dkt new` and
  `dkt archive` do the same only when the project has no copy of the schema
  (with one they do not need the per-machine directory); `dkt inspect` reports `schema-missing` with the reason.
- A symbolic link in the repository, or at or below the per-machine
  `openspec/schemas/docket/`: `dkt init` changes nothing and names the
  link. Links above it, such as `XDG_DATA_HOME` itself, are followed.
- Per-machine copy differs and `--force` is absent: the project copy is
  not removed and is reported as `schema-project-copy`.
- `openspec/config.yaml` whose `schema` key is not one plain top-level
  line: `dkt init` writes and removes nothing, lists the file as
  unreadable and exits with 3.
- Neither copy exists: `dkt new` and `dkt archive` exit with 1 with
  `schema-missing` and run no OpenSpec command.

### Acceptance Criteria

- `openspec-runtime#schema-from-the-data-directory`
- `openspec-runtime#default-under-home`
- `openspec-runtime#project-copy-first`
- `openspec-runtime#relative-data-directory`
- `project-setup#fresh-repository`
- `project-setup#fresh-machine`
- `project-setup#linked-data-directory-followed`
- `project-setup#locally-edited-template`
- `project-setup#machine-copy-forced`
- `project-setup#other-configuration-unchanged`
- `project-setup#symlinked-schema-file-refused`
- `project-setup#shipped-project-copy-removed`
- `project-setup#edited-project-copy-kept`
- `project-setup#folder-without-a-schema`
- `project-setup#removal-waits-for-the-machine-copy`
- `project-setup#default-schema-replaced`
- `project-setup#config-file-created`
- `project-setup#other-schema-left`
- `project-setup#unreadable-schema-key`
- `setup-inspection#nothing-written`
- `setup-inspection#before-init`
- `setup-inspection#machine-copy-differs`
- `setup-inspection#no-home-directory`
- `setup-inspection#project-copy-reported`
- `setup-inspection#default-schema-key-reported`
- `change-creation#schema-not-installed`
- `change-creation#schema-from-the-machine`
- `change-creation#schema-differs`
- `change-archive#schema-not-installed`
- `change-archive#archive-with-a-differing-schema`

## Change Log

| Date | Source | Change | Impact |
|---|---|---|---|
