# Design

## Context

`internal/scan` lists every change folder in every worktree (from disk) and every local and remote-tracking branch (from the committed tree), with archived folders told apart. It reports folders only; it does not read their files, and a use found on a branch does not record the commit it was read from.

`dkt lint` already hands delta parsing to OpenSpec (`openspec.Runner.ShowChange`, `openspec show <slug> --type change --json --deltas-only`), in line with the rebuild's rule that OpenSpec parses deltas. Two facts, checked with OpenSpec 1.14.3, shape how `dkt conflicts` can do the same:

- `openspec show` refuses a change whose `proposal.md` has no `## Why` section ("Change must have a Why section"), which is exactly the `proposal.md` that `dkt new` writes. A change being proposed would be unreadable.
- `openspec show` reads files from disk, and most copies `dkt conflicts` must compare live only in a branch's tree.

One `openspec show` call takes about 0.5 s with telemetry off.

## Goals / Non-Goals

**Goals:**
- Parse every copy's deltas exactly as OpenSpec would apply them, whatever state the rest of the change is in.
- Read nothing through a symbolic link, as the scan already refuses to.
- Stay quick enough to run before every piece of parallel work.

**Non-Goals:**
- Telling which copy of a change is the newest: every copy counts.
- A Go parser of the delta format.

## Decisions

### Parse a scratch copy with OpenSpec

For each distinct copy, dkt writes the copy's `specs/**/spec.md` files under a temporary directory as `openspec/changes/<slug>/`, adds a fixed `proposal.md` with `## Why` and `## What Changes` sections, and runs `ShowChange` there. The change's own `proposal.md`, `design.md` and plan are never copied, so their state cannot make the copy unreadable.

Alternatives:
- Parsing the delta headers in Go, as the previous version did. Rejected: OpenSpec's reader has rules that are easy to get wrong (section titles in any case, repeated sections, code fences, REMOVED as bullets, FROM/TO pairing), and a mismatch would hide a real overlap.
- Running `openspec show` in each worktree. Rejected: it fails on an unfinished `proposal.md`, and branches have no files on disk.

### Every copy counts, identical copies are parsed once

A change found in several sources touches the union of what its copies touch, and each overlap entry lists only the sources whose copy touches that requirement. Copies are keyed by their delta files' paths and contents; identical copies (a branch and its remote-tracking branch, a worktree with nothing uncommitted under `specs/`) are parsed once and share the result.

Alternative: the first source only, as the previous version did. Rejected: the copy that overlaps may be an uncommitted one in another worktree, which is exactly what parallel work produces.

### Archived anywhere is archived

A slug with an archived use in any source is left out, as in the previous version: `dkt new` never reuses a slug, so an archived use means the work is finished, and a stale branch still holding the active folder must not raise overlaps.

### OpenSpec only when there is something to compare

With fewer than two active changes no overlap is possible, so OpenSpec is neither located nor run. Otherwise a missing or too old OpenSpec is the usual `openspec-missing` or `openspec-unsupported` environment error (exit 4). The `openspec show` calls run four at a time.

### Names folded as OpenSpec folds them

OpenSpec matches requirement names case-sensitively when applying a delta, but takes names that differ only in case or interior whitespace for one requirement misspelled (`foldRequirementName`). Two changes touching such spellings collide at archive time, so dkt compares folded names: trimmed, lower-cased, runs of whitespace as one space.

### Report shape

`result` is `{changes, overlaps}`; each overlap is `{capability, requirement, changes: [{slug, operations, sources}]}`. Operations come from `ShowChange`: ADDED, MODIFIED and REMOVED name the requirement; RENAMED gives RENAMED-FROM for its `from` and RENAMED-TO for its `to`. The text output prints, per overlap, the `requirement-overlap` finding (`<capability> / "<requirement>": add-x (MODIFIED), add-y (RENAMED-FROM)`) and one line per change with its sources; with no overlap it prints `Compared N changes; no requirement is touched by two of them.`

An unreadable copy is listed in `unreadable` as `<source>:openspec/changes/<slug>` with the reason, so the overlaps of the other changes are still reported and the exit code is 3.

## Risks / Trade-offs

- [OpenSpec changes what `show` needs from a proposal] → the fixed proposal has both sections OpenSpec documents; a test against the real OpenSpec (run by CI's `TestRealOpenSpec` job on 1.14.0 and the latest) parses a scratch copy.
- [Many copies make the command slow] → identical copies are parsed once and four calls run at once, so the time grows by about 0.5 s for every four distinct copies.
- [A global OpenSpec store setting could make `show` look elsewhere than the scratch directory] → the scratch directory holds its own `openspec/`, which OpenSpec finds first (`root.source` is `nearest`); the real-OpenSpec test would catch a change.
- [A copy that OpenSpec cannot parse is unreadable, so the command exits with 3 while someone is mid-edit] → the overlaps of the other changes are still reported, and the reason names the copy.
