# Design

## Context

See proposal.md for why, and `add-init-and-new`'s design.md for the
milestone. This change can be built in parallel with `add-conflicts`; both
use the scan from `add-init-and-new` and the readers from `add-lint`, and
they touch different capabilities.

The workflow it serves: a member opens a pull request after `dkt sync`; the
pull request merges into the default branch with a merge commit; later, in
the release step, someone runs `dkt archive` on a branch made from the
default branch tip and commits the result together with the version bump.

## Goals / Non-Goals

**Goals:**

- An archive that never lets one change silently replace another's edit to
  a requirement.
- Every refusal says which check failed and what to do next.

**Non-Goals:**

- Merging drifted requirements automatically. OpenSpec's own
  parallel-merge plan proposes base fingerprints and a rebase flow; if
  OpenSpec ships that, Docket's base check should give way to it.

## Decisions

- **Trailers through git.** Commits come from `git log <x-base>..HEAD` with
  a format of `%H`, `%s` and `%(trailers:key=Change,valueonly)` and
  `%(trailers:key=Task,valueonly)`, so trailer parsing (last paragraph, key
  case, folding) is git's. Only
  commits whose `Change` values include the slug count. Range `x-base..HEAD`
  keeps an old, unrelated use of the same slug out, and slugs are never
  reused anyway. A `Task` value that is not the number of a `### Task`
  heading is a finding, so `sync --check`, and with it the plan check, fails
  on it.
- **One reader for `.openspec.yaml`.** `add-lint` left it for its first
  user. It reads `x-base`, `x-kind` and `x-source`; `rebase` rewrites only
  the `x-base` line and reads the file back to check the value.
- **Ticking edits bytes in place.** `sync` finds each `### Task N` section
  with the plan reader and rewrites only the bracket characters of its
  `- [ ]` steps, keeping line endings and everything else as read.
- **Base check by requirement block.** The main spec at `x-base` comes from
  `git show <x-base>:openspec/specs/<capability>/spec.md`; the current side
  comes from the working tree, not HEAD, so a change archived earlier in the
  same run counts as a change underneath the next one. (The tree is clean
  when the run starts, so for the first change the two are the same.) Both
  are read with the `add-lint` reader. Names match exactly, as OpenSpec
  matches them when it merges, and a block, from its header line to the
  next requirement or `## ` line, is compared after line-ending
  normalization and trimming the whitespace around it, as OpenSpec
  compares blocks (`normalizeBlockRaw`), so any edit inside it counts. A
  capability file that does not exist at `x-base` has no blocks. A requirement
  a change renames is looked up by its FROM name.
- **A clean tree means `git status` lists nothing.** Untracked files count,
  since they would end up in the commit of the archive; ignored files do
  not.
- **Cross-check before the irreversible step.** `openspec archive` rewrites
  specs and moves the folder, so the comparison with
  `openspec show --json --deltas-only` happens just before it. A mismatch
  points to a reader bug or an OpenSpec format change; either way it is
  safer not to archive.
- **Archived folder from OpenSpec's answer.** `openspec archive <slug> -y
  --json` reports where it moved the change (`archive.archivedAs` and
  `archive.path`); Docket writes `evidence.md` there rather than predicting
  the dated name, and takes the date from that name. The repository is
  recorded by its `origin` URL, never by its local path, because
  `evidence.md` is committed.
- **Stop at first failure, keep earlier results.** Each archived change is
  a separate, already-complete result in the working tree; undoing them
  would need a rollback that `git restore` already provides. The report
  lists which changes were archived before the failure.
- **Rebase only moves the base.** It cannot know whether the author
  reconciled the delta with the newer spec; it lists every touched
  requirement whose text moved between the two bases, so the author and
  reviewers can check. It edits only the `x-base` line, keeping the rest of
  `.openspec.yaml` byte for byte.

## Risks / Trade-offs

- [A pull request merged with squash loses per-commit trailers, so `sync`
  and the plan check find no task commits] → AGENTS.md and the README state
  that merges must use merge commits; the plan check then fails loudly
  instead of archiving a plan that looks unfinished.
- [A task finished without any commit, such as a manual verification] → Its
  steps are ticked by hand; the plan check accepts ticks from either source.
- [Release branches with uncommitted bump edits] → `archive` requires a
  clean tree; run it before editing the version, then commit both together.
