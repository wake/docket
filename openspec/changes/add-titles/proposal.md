# Proposal

## Why

A change and a main spec have no title a reader can use: the slug is the
only name. OpenSpec ignores the H1 of a main spec, writes
`# <id> Specification` for a new one, and reports the id as its title.
`dkt status`, and the workbench that reads it, need a title for each change
and each spec (`docket/ideas/2026-10-29-document-titles.md`, item 1 of
`docket/order.md`). Docket also keeps its own data, the kind, source and
base of a change, as `x-` keys inside OpenSpec's `.openspec.yaml`; a file
of Docket's own keeps OpenSpec's files as OpenSpec writes them.

## What Changes

- A change keeps its title, kind, source and base in
  `openspec/changes/<slug>/docket.yaml`. `dkt new` requires `--title`,
  writes that file, and no longer adds keys to `.openspec.yaml`.
- A main spec's title is `title:` in `openspec/specs/<capability>/docket.yaml`.
  A change gives a spec its title in a `docket.yaml` beside the delta
  (`openspec/changes/<slug>/specs/<capability>/docket.yaml`); after
  `openspec archive`, `dkt archive` sets the main spec's title from it.
- A change with no `docket.yaml`, made before this change, is still read
  from `x-kind`, `x-source` and `x-base` in `.openspec.yaml`, without a
  title. Archived changes are never rewritten.
- `dkt lint` reports a missing title: on a linted change, on a delta for
  a capability that has no main spec yet, and, when no change is named, on
  every main spec.
- `dkt status` gives each change's title; `evidence.md` records it.
- `dkt rebase` moves `base` in `docket.yaml` (`x-base` in `.openspec.yaml`
  for a change made before).
- The specs that said `x-base` now say the change's base.
- The shipped workflow passes `--title` to `dkt new`; the plan template's
  heading loses its title placeholder, since documents in a change are
  named by role under the change's title.
- The ten main specs get titles.

## Non-Goals

- `fix` stays a kind of `dkt new`; it leaves with the records change.
- No command that moves an older change's metadata into `docket.yaml`;
  lint names the file to write.
- No title for the documents inside a change, and no title read from any
  H1.
- No command to change a title after creation; edit `docket.yaml`.
- No title for a requirement or a scenario.
- No workbench; `dkt status` only gives it the title.

## Builds On

- `change.ParseMetadata` and `change.SetBase` (`internal/change/metadata.go`):
  reading `.openspec.yaml` stays for older changes, and setting one line's
  value is what `dkt rebase` needs in either file.
- `change.AppendMetadata` (`internal/change/new.go`), which `dkt new` used
  to write metadata; writing `docket.yaml` takes its place.
- `safepath.Check` before every write; `dkt new`'s lock and cleanup in
  `internal/cli/cmd_new.go`.
- `lint.Run` and its per-change rule loop (`internal/lint/lint.go`);
  `delta.CapabilityPath` to map a delta file to its capability.
- `archive.Archive`'s steps and its evidence step
  (`internal/archive/run.go`, `internal/archive/evidence.go`);
  `archive.Rebase`.
- `status.Load` (`internal/status/load.go`) and `scan.ReadFiles`.
- `go.yaml.in/yaml/v3`, already a dependency.

## Capabilities

### New Capabilities

- `docket-metadata`: where Docket keeps the data it adds to OpenSpec's
  changes and specs, how it reads changes made before, and what a title is.

### Modified Capabilities

- `change-creation`: `dkt new` requires a title and writes `docket.yaml`
  instead of adding keys to `.openspec.yaml`.
- `change-lint`: missing titles are reported; main specs are checked when
  no change is named.
- `change-archive`: the main spec takes the title from the delta's
  `docket.yaml`; evidence records the title; rebase and the checks read
  the base from where the change keeps it.
- `change-status`: each change gives its title; activity counts from the
  change's base.
- `plan-sync`: counts commits from the change's base.
- `cli-output`, `openspec-runtime`: scenarios that run `dkt new` pass
  `--title`; a failed `openspec new change` writes no `docket.yaml`.

## Impact

- `internal/change`: reading and writing `docket.yaml`, and the choice
  between it and `.openspec.yaml`.
- `internal/cli/cmd_new.go`: `--title`, and the file it writes.
- `internal/lint`: a title rule; main specs read when no change is named.
- `internal/archive`: metadata loading, rebase, evidence, and a step after
  `openspec archive`.
- `internal/status`: the title, and metadata from either file.
- `AGENTS.md`, both READMEs, the plan template, `docs/ARCHITECTURE.md`.
- Ten new `openspec/specs/<capability>/docket.yaml` files.
- Projects whose active changes were made by an older `dkt` see
  `title-missing` until they write `docket.yaml` for them; main specs
  without one fail `dkt lint` run without names.

## Implementation Contract

### Observable Behavior

- `dkt new add-x --kind feature --source "issue 12" --title "Booking waitlist"`
  creates the change as before, with `docket.yaml` holding the title, kind,
  source and base, and `.openspec.yaml` exactly as OpenSpec wrote it.
- `dkt lint` reports `title-missing` for a change, a new capability's
  delta, or a main spec without a title.
- `dkt archive` gives a new main spec the title from its delta's
  `docket.yaml`.
- `dkt status` shows each change's title.

### Interfaces

- `dkt new <slug> --kind <kind> --source <text> --title <text>`; JSON
  `result` gains `title`.
- `openspec/changes/<slug>/docket.yaml`: a YAML mapping with `title`,
  `kind`, `source`, `base` (full SHA).
- `openspec/specs/<capability>/docket.yaml` and
  `openspec/changes/<slug>/specs/<capability>/docket.yaml`: a YAML mapping
  with `title`.
- `dkt status --json`: each change gains `title`, a string or null;
  `metadata` keeps `kind`, `source`, `base`, `created`.
- `evidence.md`: rows Title, Kind, Source and Base replace `x-kind`,
  `x-source` and `x-base`.
- Finding rules: `title-missing`, `metadata-invalid` (lint, for a
  `docket.yaml` that is not a YAML mapping), `spec-title-failed`
  (archive).

### Failure Modes

- `--title` missing, empty, only whitespace or more than one line: usage
  error, exit 2, nothing created.
- Writing `docket.yaml` fails after OpenSpec created the folder: the
  folder is removed, as when writing metadata failed before; a link on the
  path: nothing written, exit 1.
- `docket.yaml` that is not a YAML mapping: lint reports
  `metadata-invalid`; status gives no metadata and no title for the
  change, as for a broken `.openspec.yaml`; archive refuses the change.
- A change with both files: `docket.yaml` wins and the `x-` keys are
  ignored.
- Setting a main spec's title fails after `openspec archive` succeeded:
  `spec-title-failed`, exit 1, the run stops, and what was archived stays
  in the working tree uncommitted.
- A main spec's `docket.yaml` that cannot be read: listed as unreadable,
  exit 3.

### Acceptance Criteria

- `change-creation#change-created`, `change-creation#missing-title`,
  `change-creation#blank-title`
- `docket-metadata#metadata-read-from-docket-yaml`,
  `docket-metadata#change-made-before-docket-yaml`,
  `docket-metadata#both-files-present`,
  `docket-metadata#title-on-two-lines`
- `change-lint#change-without-title`,
  `change-lint#new-capability-without-title`,
  `change-lint#existing-capability-needs-no-delta-title`,
  `change-lint#main-spec-without-title`,
  `change-lint#named-change-skips-main-specs`,
  `change-lint#broken-docket-yaml`
- `change-archive#title-carried-to-a-new-spec`,
  `change-archive#title-replaced-on-an-existing-spec`,
  `change-archive#evidence-written`,
  `change-archive#rebase-after-merging-the-default-branch`,
  `change-archive#rebase-a-change-made-before-docket-yaml`
- `change-status#title-shown`, `change-status#change-without-title`
- `plan-sync#commits-counted-from-the-base`

## Change Log

| Date | Source | Change | Impact |
|---|---|---|---|
| 2026-10-29 | Pre-merge spec check | change-archive "Spec titles carried" says that a file the title step cannot read is listed as unreadable with exit 3, as Failure Modes already did | Wording only; the code and tests already behave this way |
