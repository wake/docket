# Design

## Context

See proposal.md for why. This change completes the first milestone. It
reads every active change from wherever the `workspace-scan` scan finds it:
a worktree on disk, including uncommitted and untracked files, or a branch
tip in git. Until now only `dkt new` uses the scan, and only for slugs;
nothing reads a change's files from a source other than the current
worktree. `dkt lint` and `dkt archive` each walk the current worktree's
`specs/` folder through an `fs.FS`, with their own copy of the rule that
turns `specs/<capability>/spec.md` into a capability path.

## Goals / Non-Goals

**Goals:**

- The overlap list is computed from what OpenSpec would apply at archive
  time, so it neither misses an overwrite nor reports one that cannot
  happen.
- One way to read a change's files from any scanned source, which later
  commands (`status`, the workbench) can use too.

**Non-Goals:**

- Comparing a change with the main specs. Drift against the main specs is
  the base check in `dkt archive`.
- Checking that a delta is well formed. That is `dkt lint`.

## Decisions

- **Read a change's files from any source.** `scan.ReadFiles(g, src, dir,
  keep)` returns every regular file below `dir` in a source that `keep`
  accepts, keyed by its path relative to `dir`, and lists the paths it
  could not read. Files `keep` rejects are not read, so an unreadable file
  that is not a delta spec never makes the command exit with 3. A worktree is
  walked through `os.Root`, so inside the worktree neither `..` nor a link
  leads out; the worktree's own path is opened as listed, as
  `scan.ReadFile` does. Links inside are skipped. A branch is listed
  with `git ls-tree -r -z` at its commit (blobs with a regular file mode
  only, so links and submodules are skipped) and each file is read with
  `git show`. A folder that does not exist yields no files and no error,
  because a change may have no `specs/` folder yet; a `dir` that exists but
  is not a folder is unreadable. The alternative, an
  `fs.FS` backed by a git tree, would let `lint` and `archive` keep their
  walkers unchanged, but it needs an open-file lifetime for worktrees and
  more code than one function, for no caller that needs it now.
- **One rule for capability paths.** `delta.CapabilityPath(rel)` maps a
  path relative to the change folder that starts with `specs/` and ends
  with `/spec.md` to the part between them, and reports false for every
  other file. `lint` and `archive` each carry a copy of this rule today and
  agree on every path; `conflicts` would be the third copy, so all three
  use the one function instead, with no change in behavior.
- **What a change touches is what OpenSpec applies.** The deltas are read
  with `delta.Parse`. A change touches a requirement name in a capability
  when it adds, modifies or removes a requirement with that name, or
  renames from or to it. Names are compared exactly after trimming, as
  OpenSpec matches them when it merges. `delta.Parse` never fails: text it
  reports as a problem is text OpenSpec does not read either, so it cannot
  replace anyone's requirement. That is why there is no `unparseable`
  finding; a delta that cannot be read at all is an unreadable source
  (exit 3), and its problems are `dkt lint`'s job.
- **Operations are a list.** One change can touch one name twice, for
  example renaming "A" to "B" and modifying "B", which OpenSpec allows.
  Each change in an overlap carries `operations`, in the order `ADDED`,
  `MODIFIED`, `REMOVED`, `RENAMED-FROM`, `RENAMED-TO`, without repeats. A
  single change never overlaps with itself.
- **Archived means archived anywhere.** A slug with an archived use in any
  scanned source is not compared, even when a stale branch still holds its
  folder: that copy is left over from before the release, and slugs are
  never reused, since `dkt new` refuses a used one.
- **Content comes from the scan's first source.** Each change is read from
  `Sources[0]`, the source the `workspace-scan` order puts first, and every
  source is listed in the report. Copies of one change in other sources are
  never compared with each other.
- **An overlap is a finding.** Each overlap is reported in
  `result.overlaps` and as one finding with the rule `requirement-overlap`,
  so the exit code and `ok` follow the shared rules of `cli-output`
  without a special case. A slug that is not active is the finding
  `change-not-active`; an invalid slug is `slug-invalid`, as in `dkt new`.
  `result.changes` lists the slugs that were compared, so "no overlap" can
  be told apart from "nothing found".
- **Read-only.** `conflicts` writes nothing, takes no lock and starts no
  OpenSpec.

## Risks / Trade-offs

- [A worktree is edited while the command reads it] → The report reflects
  whatever was on disk; running it again gives the new state. Nothing is
  written, so nothing is damaged.
- [Two unmerged copies of one change differ] → Only the first source's
  content is compared, as the scan defines; an overlap that only the other
  copy has is not seen. Both sources are listed, which shows the copies
  exist.
- [Many branches with large `specs/` folders] → One `git ls-tree` and one
  `git show` per file for each change read from a branch. Changes hold a
  few delta specs, so this stays small.
