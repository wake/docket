# Architecture

Docket (`dkt`) is a Go CLI. Read this before writing a proposal or a plan.

## Codemap

| Path | Role |
|---|---|
| `schema.go` | Root package `docket`: embeds the OpenSpec schema and the `AGENTS.md` workflow section |
| `cmd/dkt` | Entry point; the only place that reads the process environment and working directory |
| `internal/cli` | Command table, argument handling, text and JSON output, exit codes |
| `internal/gitx` | Every git call: refs, worktrees, merge-base, file contents at a revision |
| `internal/openspec` | Locates, version-checks and runs the OpenSpec CLI |
| `internal/scan` | Finds changes across worktrees and branches; resolves the default branch |
| `internal/setup` | Installs the schema (`dkt init`) |
| `internal/change` | Change-folder rules: slug validation, `docket.yaml` metadata, plan steps and ticking |
| `internal/archive` | Closes changes: `dkt sync`, `dkt archive`, `dkt rebase`, the base check, the cross-check of Docket's reading against OpenSpec's, and `evidence.md` |
| `internal/markdown` | Line, heading and table readers that skip code fences and comments |
| `internal/delta` | Reads delta specs and main specs; makes scenario slugs |
| `internal/status` | Summarizes each active change: phase, task states from commit trailers, progress, last activity |
| `internal/conflicts` | Finds requirements that more than one active change touches |
| `internal/lint` | Runs the lint rules over change folders through an `fs.FS` |
| `internal/safepath` | Refuses a write path that passes through a symbolic link |
| `internal/testutil` | Test-only helpers: temporary repositories, a fake OpenSpec |

## Capability index

| Job | Use |
|---|---|
| Run git | `gitx.Git.Run` (and its typed helpers, e.g. `Git.Show`, `Git.Worktrees`) |
| Find changes everywhere | `scan.Scan`, then `Inventory.Uses` |
| Resolve the default branch | `scan.ResolveDefaultBranch` |
| Read a file from a scanned source | `scan.ReadFile` |
| Start OpenSpec | `openspec.Runner.Run` (build with `openspec.New`) |
| Validate a slug | `change.ValidateSlug` |
| Record kind, source and base | `change.CreateDocket` |
| Install the schema | `setup.Init` |
| Check a path before writing to it | `safepath.Check` |
| Read Markdown lines, headings and tables | `markdown.Lines`, `markdown.Headings`, `markdown.FindTable` |
| Blank out HTML comments outside fenced code | `markdown.BlankComments` |
| Read a delta or a main spec | `delta.Parse`, `delta.ParseSpec` |
| Make a scenario slug | `delta.Slug` |
| Read proposal.md or plan.md | `change.ParseProposal`, `change.ParsePlan` |
| Lint change folders | `lint.Run` |
| Read change metadata (`docket.yaml`, or `.openspec.yaml` of an older change) | `change.ReadMetadata`, `change.ParseDocket` |
| Set one recorded key, such as the base | `change.SetKey` |
| Tick plan steps | `change.Tick` |
| List commits between two revisions | `Git.Commits` |
| List uncommitted paths | `Git.Uncommitted` |
| Read a file at a revision if it exists | `Git.ShowIfExists` |
| Tick a plan from commit trailers | `archive.Sync` |
| Move a change's recorded base | `archive.Rebase` |
| Check and archive changes, then record evidence | `archive.Archive` |
| Compare Docket's reading of deltas with OpenSpec's | `archive.CrossCheck` |
| Files a branch adds | `Git.AddedFiles`, `Git.UncommittedAdded` |
| List the files under a directory at a revision | `Git.ListFiles` |
| Read many files from a scanned source | `scan.ReadFiles` |
| Map a delta spec path to its capability | `delta.CapabilityPath` |
| Load every active change's deltas | `conflicts.Load` |
| Find requirements touched by more than one change | `conflicts.Find` |
| Keep the overlaps that involve one change | `conflicts.Involving` |
| Load every active change's status | `status.Load` |
| Work out a change's phase, tasks and progress | `status.Summarize` |
| Match commits to plan tasks by trailer | `change.NamedTasks` |
| List uncommitted paths under given paths | `Git.UncommittedUnder` |
| List commits with their times, limited to paths | `Git.Commits` (`Commit.Time`, `paths`) |
| Report a result | `cli.Outcome`; map to an exit code with `cli.ExitCode` |
| Build a test repository | `testutil.NewRepo` |
| Stand in for OpenSpec | `testutil.FakeOpenSpec` |

## Invariants

- Only `internal/openspec` starts OpenSpec.
- Only `internal/gitx` starts git (test helpers excepted).
- Nothing below `cmd/dkt` reads the process environment; it is passed in through `cli.Options`, the clock (`Options.Now`) included (test helpers excepted).
- Readers take bytes, not paths, so that content can come from any revision.
- Nothing writes through a symbolic link in the repository: check the path with `safepath.Check` first.
