# Design

## Context

Requirements are in `specs/default-branch`, `specs/change-lint` and `specs/setup-inspection`. `v0.15.0` (branch `legacy`) had all three checks: `internal/scan/defaultbranch.go` (`ResolveDefaultBranch`, with a `--default-branch` flag this change does not bring back), `internal/lint/routing.go` (`checkRouting`, `RoutedDirs`) and the trailer check in `internal/archive/trailers.go` (`Trailers`), which read a recorded base and two trailers; read them with `git show v0.15.0:<path>`. This version records no base and reads one trailer, `Change: <slug>` or `Change: <slug>/<N>`, from the whole history of a commit, as `dkt sync` does.

What exists now: `gitx.Git` runs git with an isolated environment (`output`, `outputIn` with stdin, `Run`, `runOptional` where exit 1 means no answer); `ChangeTrailers()` reads the trailers of HEAD's history; `TreeDirs(commit, paths...)` lists directories in a commit's tree; `change.SchemaOf`, `change.ParsePlan`, `change.TaskRefs`; `internal/lint` with the main spec title check inside `Run`; `dkt lint` and `dkt inspect` in `internal/cli`.

## Goals / Non-Goals

**Goals:**
- One way to find the default branch, used by both commands.
- The trailer check reads git objects only, so it sees what was merged, not the worktree.

**Non-Goals:**
- A flag or setting for the default branch (see the proposal).

## Decisions

### Git calls
In `internal/gitx`:
- `ChangeTrailers(rev string) ([]Trailers, error)`: the existing function with the revision as a parameter instead of `HEAD`; `dkt sync` passes `"HEAD"`. This is a restructuring with no change in behaviour, in its own commit before anything else.
- `DefaultBranch() (DefaultBranch, bool, error)`, `DefaultBranch{Name, Commit string}`: as specs/default-branch says. The target of `origin/HEAD` comes from `git for-each-ref --format=%(symref) refs/remotes/origin/HEAD` (empty output when the ref does not exist or is not symbolic, never an error for that); which branches exist, and their commits, come from the existing `Refs()` (one `for-each-ref` over `refs/heads` and `refs/remotes`). `false` with a nil error means there is no default branch.
- `MergeBase(a, b string) (string, bool, error)`: `git merge-base a b` through `runOptional` (exit 1 means none). Both arguments must be commits: `git merge-base` exits 128, not 1, for a name that does not resolve, so callers pass commit ids, and `dkt lint` checks first that HEAD has a commit (`Repo.Head` is nil while HEAD is unborn), treating an unborn HEAD as no merge base.
- `ReadBlobs(commit string, paths []string) (map[string][]byte, error)`: one `git cat-file --batch` call with `<commit>:<path>` per line on stdin; a `missing` answer (one line, no body) or an object that is not a blob (a header, then a body of the given size and a newline, which the parser consumes and drops) leaves the path out of the map. Alternative: `git show` per file, refused because a missing path is exit 128, which cannot be told apart from a real failure without reading git's message.
- `AddedSince(base string, dirs ...string) ([]string, error)`: the union, sorted and without duplicates, of `git diff --name-only -z --no-renames --diff-filter=A <base> -- <dirs>` (committed or staged since `base`, still in the worktree) and `git ls-files -z --others --exclude-standard -- <dirs>` (untracked and not ignored).

### Routing
`dkt lint` resolves the default branch and the merge base of HEAD's commit and its tip before linting (an unborn HEAD has none); with both, it passes `AddedSince(mergeBase, "docs/superpowers/specs", "docs/superpowers/plans")` as a new `lint.Input.Added []string`, and nil otherwise. `lint.Run` reports `routing` for each path in `Added`, with no change and no line, message: the file was written to superpowers' folder and belongs in the change folder (proposal.md, specs/, design.md or plan.md). The result's `default_branch` is the name when the rule ran and null otherwise; the text output adds `No default branch found; the routing rule was skipped.` when it did not run because there is no default branch, and `No merge base with <name>; the routing rule was skipped.` when there is no merge base.

### Main spec titles shared
`lint.SpecTitles(root fs.FS) ([]Finding, []Unreadable, int)` is the body of the present `checkSpecTitles`, returning instead of appending; `Run` appends what it returns. This is a restructuring in its own commit. `dkt inspect` opens the repository with `os.OpenRoot` and calls it, mapping findings and unreadable entries as `dkt lint` does.

### Trailer check
In `internal/cli/cmd_inspect.go`, after the present checks: resolve the default branch; without one, `result.default_branch` is null and the text output says `No default branch found; the trailer check was skipped.` With one:
1. `TreeDirs(tip, "openspec/changes/")` lists the folders; skip `archive`.
2. `ReadBlobs(tip, …)` reads each folder's `.openspec.yaml` and `plan.md` in one call.
3. A folder whose `SchemaOf` is not `docket`, or without a plan.md, or whose `ParsePlan` has no task, is not checked.
4. `ChangeTrailers(tip)` once; a change is carried when some value `v` gives `mine` from `change.TaskRefs(v, slug)` (well-formed or not: a malformed list still names the change).
5. Each change not carried is `trailer-missing`, file `openspec/changes/<slug>/plan.md`, change `<slug>`, message saying no commit on `<name>` carries a `Change:` trailer for it, which is what a squash merge leaves, and that `dkt sync` cannot tick its plan.

Git errors other than "no answer" are returned as errors, as the existing commands do.

## Risks / Trade-offs

- [A project whose default branch is neither `origin/HEAD`'s target nor `main` nor `master` gets no routing or trailer check] → the result says so (`default_branch` null, a text line); a later change can add a setting.
- [`origin/HEAD` is only set by `git clone` or `git remote set-head`] → without it, `main` and `master` are tried.
- [The trailer check reads the remote-tracking tip, which may be behind until the next fetch] → it reports what is known locally; a fetch updates it.
- [A change merged by squash and then given a later commit with its trailer passes] → accepted: the check asks whether dkt can find any commit for the change, not whether every commit kept its trailer; `dkt sync --check` reports the tasks left without commits.
- [A folder on a linted path replaced by a link between listing and reading] → as before, reads go through the confined handle.
