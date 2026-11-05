# Design

## Context

Requirements are in `specs/change-creation`, `specs/workspace-scan`, `specs/change-metadata` and `specs/cli-output`. `v0.15.0` (branch `legacy`) had `dkt new` (`internal/cli/cmd_new.go`, `internal/change/new.go`) and a workspace scan (`internal/scan/`) that also resolved the default branch, ordered sources and found records and issues; read them with `git show v0.15.0:<path>`. This change ports only what `dkt new` needs.

Checked on 2026-11-06 with OpenSpec 1.14.3, in a project set up by `dkt init`:
- `openspec new change <slug>` creates only `openspec/changes/<slug>/.openspec.yaml` (`schema: docket`, `created: <date>`); it refuses an uppercase name and a folder that exists, and accepts a name that starts with a date.
- A `proposal.md` holding only front matter makes `openspec status` count the proposal as done; `openspec instructions proposal` still prints the template, whose instruction says `dkt new` writes `title`, `kind` and `source`.
- `git ls-tree -z -d <ref> -- openspec/changes/ openspec/changes/archive/` prints the active change folders and the archived ones in one call (the `archive` folder itself is replaced by its entries), and prints nothing, with exit 0, when the path does not exist.

## Goals / Non-Goals

**Goals:**
- `dkt new` as the specs say, with a scan that later commands extend instead of replace.

**Non-Goals:**
- What the proposal leaves out: the default branch, content precedence between sources, records and issues in the scan, `base`, checking the schema's content.

## Decisions

### Positional arguments
`dkt new` is the first command with a positional argument. `Command` gains `Args int`, the number of positionals it takes (0 for `init` and `inspect`). Parsing alternates: parse flags, take the next non-flag argument as a positional, parse again, until no arguments are left, so `dkt new add-x --kind feature` and `dkt new --kind feature add-x` both work. Fewer or more positionals than `Args` is a usage error (exit 2), as an unknown flag is.

### Order of steps
1. Flags and the slug present, `--kind` known, `--title` and `--source` of the right form (exit 2).
2. The slug rules: date prefix or `archive` (`slug-invalid`, exit 1).
3. `safepath.Check` on `openspec/changes/<slug>/proposal.md` for the requested slug (`symlink-refused`, exit 1), before anything runs or reads through `openspec/changes`.
4. `openspec/schemas/docket/schema.yaml` exists as a regular file (`schema-missing`, exit 1).
5. OpenSpec located and its version checked (`openspec-missing`, `openspec-unsupported`, exit 4).
6. Take the lock (below), then scan (unreadable, exit 3).
7. Choose the slug: the requested one, then `-2`, `-3`, …, skipping any the scan found and any name that exists in this worktree's `openspec/changes/` (the scan reads this worktree too; the second check covers an entry that is not a directory, such as a stray file).
8. `safepath.Check` on the chosen slug's `proposal.md` (`symlink-refused`), then `openspec new change <slug> --schema docket`.
9. `safepath.Check` again: a link that appeared while OpenSpec ran leaves everything as it is (`symlink-refused`). Then write `proposal.md`.

Nothing runs before step 5 and nothing is written before step 8. The schema is passed with `--schema docket` so that a project whose `config.yaml` names another schema still gets a Docket change.

### Lock
`v0.15.0`'s `lockOpenSpec`: an exclusive `flock` on the `openspec/` directory, held from before the scan until `proposal.md` is written. It serializes runs in one worktree; runs in different worktrees see each other's folders through the scan, and a race between them in the same instant is accepted (Risks).

### OpenSpec failure
Under the lock, and with no folder of that name before step 7, whatever is under `openspec/changes/<slug>/` after a failed `openspec new change` was made by this run; it is removed, unless a link is now on its path, in which case it is left (and the finding says so). Ported from `v0.15.0`'s `cleanup`.

### Workspace scan
A new package `internal/scan`, a reduced port of `v0.15.0`'s:
- Worktrees: `v0.15.0`'s `gitx.Worktrees` (`internal/gitx/worktree.go`), ported as is: it parses `git worktree list --porcelain -z` entry by entry, falling back to the line form for git before 2.36; a `bare` entry is skipped. For each worktree, `safepath.Check` on `openspec/changes/archive` first: a symbolic link on that path makes the worktree unreadable, naming the link, because what it points to is not in that worktree. Then `os.ReadDir` of `<worktree>/openspec/changes` and of its `archive/`; an entry counts when it is a directory (`DirEntry.IsDir`, which does not follow a link). A missing `openspec/changes` holds no use; a missing worktree directory, or any other read error, makes that worktree unreadable. The reason for a missing directory names `git worktree prune`.
- Branches: `git for-each-ref --format=%(refname)%09%(symref)%09%(objectname) refs/heads refs/remotes`, skipping symbolic refs. Refs that point to the same commit are read once, and each of them is a source of what that commit holds. Each commit is read with `git ls-tree -z -d <commit> -- openspec/changes/ openspec/changes/archive/` (without `--name-only`): each record is `<mode> <type> <object>\t<path>`, and only records of type `tree` count, so a file, a link or a submodule at a change's path is not a use. An error makes the refs that point to that commit unreadable.
- A use is `{Slug, Archived, Path, Source}`, with the folder's path relative to the repository root and the source as the worktree's path or the ref's full name. Archived folder names must start with `YYYY-MM-DD-`; others are ignored.
- The scan reads git's state as it is: no fetch, no network.

### Result
`used` groups the uses of the requested slug by folder path, each with the sources that hold it, in the order the scan found them (worktrees first, then refs in `for-each-ref` order), because one archived folder usually appears on many branches. The text output prints one line per folder, naming at most three sources and how many more.

### Front matter
`internal/change` holds the kinds, the title-and-source form check (one line: no `\n` or `\r`; valid UTF-8; not only whitespace; the value is kept as given, not trimmed) and the front matter writer. `title` and `source` are written with Go's `strconv.Quote`: for valid UTF-8 its escapes (`\"`, `\\`, `\t`, `\xXX`, `\uXXXX`, `\UXXXXXXXX`, …) are all valid YAML double-quoted escapes. `proposal.md` is created with `O_CREATE|O_EXCL|O_NOFOLLOW`, mode 0644 subject to the umask. Reading the front matter comes with `dkt lint`.

### Tests
- Unit tests run `cli.Run` against the fake OpenSpec. The fake gains `new change <slug> [--schema <s>]`: it refuses an uppercase name and an existing folder as OpenSpec does, and otherwise writes `.openspec.yaml`; a variable makes it create the folder and then fail.
- Worktrees in tests come from `git worktree add`; a missing one is a worktree whose directory the test removed.
- `TestRealOpenSpec` opens its change with `dkt new` instead of `openspec new change`, and checks the front matter.

## Risks / Trade-offs

- [Two runs in different worktrees choose the same free slug in the same instant] → accepted: each sees the other only once its folder exists; the second branch's PR then shows a duplicate slug, which `dkt lint` can report later.
- [A repository with many branches costs one `git ls-tree` per distinct commit] → accepted; reading refs at the same commit once keeps the common case small.
- [A deleted worktree that git still lists blocks `dkt new` until `git worktree prune`] → intended: a place Docket cannot read is never taken for empty; the reason says how to clear it.
- [`openspec status` counts a proposal holding only front matter as done] → accepted: the `docket` skill has the agent write the proposal right after `dkt new`, and `dkt lint` reports missing sections.
