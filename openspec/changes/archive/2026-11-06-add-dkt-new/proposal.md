# Proposal

## Why

The `docket` skill opens every full-review change with `dkt new <slug> --kind … --source … --title …`, and that command does not exist yet. It is the first of the three commands (`dkt new`, `dkt sync`, `dkt lint`) this repository needs before it can switch from OpenSpec's built-in workflow to Docket's own.

## What Changes

- Add `dkt new <slug> --kind feature|change-request|baseline --source <text> --title <text>`:
  - It refuses a slug that starts with a date or is `archive`; OpenSpec checks the rest of the slug's form.
  - It never reuses a slug: when the slug is used in any worktree or branch, active or archived, it creates the change under the first free `<slug>-2`, `<slug>-3`, … and reports where the slug was used.
  - It needs the project's `docket` schema; when `openspec/schemas/docket/schema.yaml` is missing it creates nothing and points to `dkt init`. Whether the schema matches the shipped one is `dkt inspect`'s job.
  - It runs `openspec new change <slug> --schema docket`, then writes `proposal.md` holding only the front matter: `title`, `kind` and `source`.
  - It refuses symbolic links on the change's path and keeps two runs in one worktree from taking the same slug.
- Add the workspace scan `dkt new` uses: the change slugs, active and archived, in every worktree (uncommitted and untracked files included), local branch and remote-tracking branch, without contacting a remote. Later commands (`dkt status`, `dkt conflicts`) extend it.
- Define the change metadata in `proposal.md`'s front matter: the keys and the form of a title.
- cli-output: `dkt new` joins the commands that need a repository.

## Non-Goals

- No `base` is stored: it is derived from the first commit that carries the change's trailer, when a later command needs it.
- No default branch is resolved: `dkt new` reads every branch, so it needs none. Resolving it, and choosing which source's content wins when a slug appears in several, come with `dkt status` and `dkt conflicts`.
- No check of the schema's content and no `result.schema`; `dkt inspect` reports that.
- `dkt new` does not write the body of `proposal.md`; the agent writes it from `openspec instructions proposal`.
- Records, issues and ideas in the scan come with `dkt record` and `dkt issue`.

## Capabilities

### New Capabilities
- `change-creation`: how `dkt new` opens a change: the slug rules, never reusing a slug, the required metadata, and what it writes.
- `workspace-scan`: where Docket looks for changes and how it reports a place it cannot read.
- `change-metadata`: a change's title, kind and source in the front matter of `proposal.md`, and the form of a title.

### Modified Capabilities
- `cli-output`: `dkt new` needs a repository.

## Impact

- Adds `dkt new`, the workspace scan in `internal/`, and their tests.
- Dependencies: still only the Go standard library, the system `git` and the OpenSpec CLI.
