# Follow-ups from `add-init-and-new`

Status: small items found while reviewing `add-init-and-new` and deferred.
Recorded 2026-10-24. Pick them up in the change that touches the same code
(most CLI output items fit `add-lint`, which adds more finding output), or
fix them directly when they change no spec sentence.

## CLI output and arguments

- In text mode an error is printed on standard error and again as a finding
  on standard output.
- `jsonRequested` (`internal/cli/run.go`), used only after a parse failure or
  for an unknown command, also matches `--json` given as another flag's value
  (`--default-branch --json --bogus`) and a bare `json` positional, so such a
  failing call can answer in JSON.
- `dkt --bogus probe --json` reports the command as `""`, and `repo` is null
  after a bad flag even inside a repository.
- An internal error together with unreadable sources exits 1, not 3.
- Messages end in a dangling `: ` when git or OpenSpec wrote nothing to
  standard error (`gitx.ExitError`, `openspec.FailedError`).
- The routing section is cut out of AGENTS.md by heading, so a `## ` line
  inside a code fence would end it early.

## Behavior to revisit

- A worktree whose HEAD the default branch contains, with uncommitted edits to
  a change, is shadowed by an unmerged branch holding the same slug: that is
  the precedence order in the `workspace-scan` spec. Revisit when `status`
  shows where content comes from.
- On disk, a change folder that is a symbolic link is not listed
  (`DirEntry.IsDir` is false for it). Decide whether it should be reported.
- `gitx.Toplevel` does not model filesystem boundaries
  (`GIT_DISCOVERY_ACROSS_FILESYSTEM`): a `.git` above a mount point keeps
  git's own error instead of "not inside a git work tree". Both exit 4.
- `dkt new --source "   "` is accepted; only an empty `--source` is refused.
- Schema files and `.openspec.yaml` are rewritten in place, not written to a
  temporary file and renamed.
- `change.SetBase` drops a trailing comment on the `x-base:` line.

## Performance

- The scan starts several git processes per ref (`merge-base --is-ancestor`
  and two `ls-tree`); `git for-each-ref --no-merged` could drop merged refs in
  one call. `Toplevel` runs git twice.
- `openspec.Runner` does not cache a failed version check, and its cache is
  not safe for concurrent use.

## Tests and CI

- `testutil.InstallSchema` repeats the walk-and-write loop of `setup.Init`; a
  thin wrapper over `setup.Init` would keep test fixtures identical to what
  `dkt init` installs.
- `TestOriginHEADNotASource` cannot fail on its own, because `origin/HEAD`
  points at the default tip; the guard is tested in `internal/gitx`.
- CI runs on both `push` and `pull_request`, so a pull request branch runs
  twice, and it tests `go-version: stable` only, not the `go 1.24` floor in
  go.mod.
