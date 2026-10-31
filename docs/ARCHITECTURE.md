# Architecture

Docket (`dkt`) is a Go CLI. Read this before writing a proposal or a plan.

## Codemap

| Path | Role |
|---|---|
| `schema.go` | Root package `docket`: embeds the OpenSpec schema (`schema/docket`), the `docket`, `docket-setup` and `docket-doctor` skills, the Docket block of `AGENTS.md`, the workflow template (`workflow/template.md`) and the record template (`record/template.md`) |
| `cmd/dkt` | Entry point; the only place that reads the process environment and working directory |
| `internal/cli` | Command table, argument handling, text and JSON output, exit codes |
| `internal/gitx` | Every git call: refs, worktrees, merge-base, file contents at a revision |
| `internal/openspec` | Locates, version-checks and runs the OpenSpec CLI |
| `internal/scan` | Finds changes across worktrees and branches; resolves the default branch |
| `internal/setup` | Installs the schema once per machine, the skills and the Docket block (`dkt init`), removes a project copy of a shipped schema, and decides what it would do without writing (`dkt inspect`) |
| `internal/change` | Change-folder rules: slug validation, `docket.yaml` metadata, plan steps and ticking |
| `internal/archive` | Closes changes: `dkt sync` (plans and records), `dkt archive`, `dkt rebase`, the base check, the cross-check of Docket's reading against OpenSpec's, the trailer check of `dkt inspect`, and `evidence.md` |
| `internal/markdown` | Line, heading and table readers that skip code fences and comments |
| `internal/delta` | Reads delta specs and main specs; makes scenario slugs |
| `internal/status` | Summarizes each active change and each record: phase, task states from commit trailers, progress, last activity |
| `internal/conflicts` | Finds requirements that more than one active change touches |
| `internal/record` | Records: kinds, name and slug, the template rendering, creating a file under a free name, parsing and word counting |
| `internal/lint` | Runs the lint rules over change folders through an `fs.FS` |
| `internal/userdirs` | Finds the home and the data directory (`XDG_DATA_HOME`) from values `cmd/dkt` passes in |
| `internal/safepath` | Refuses a write path that passes through a symbolic link |
| `internal/workflow` | Locates the user's workflow file and reports its state against the shipped template; never writes it |
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
| Record title, kind, source and base | `change.CreateDocket` |
| Install the schema | `setup.Init` |
| Find the home directory | `userdirs.Home` |
| Find the data directory | `userdirs.DataHome` |
| Find the per-machine schema directory | `setup.SchemaDir` |
| Find the schema as OpenSpec does | `setup.FindSchema` |
| Hash a schema tree or a directory | `setup.TreeHash`, `setup.HashDir` |
| Tell whether a tree is a shipped schema | `setup.IsShipped` |
| Read or set `schema:` in `openspec/config.yaml` | `setup.SchemaKey` |
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
| Locate the user's workflow file | `workflow.Locate` |
| Read the workflow file | `workflow.Read` |
| Report a workflow file's state | `workflow.Inspect` |
| Decide what init would install, without writing | `setup.Inspect` |
| Check the trailers of the changes on the default branch | `archive.Trailers` |
| Hash the template | `workflow.Hash` |
| Pick a free slug | `change.Suffixed` with `scan.Inventory.Uses` or `RecordSources` |
| Where a record slug is used | `scan.Inventory.RecordSources` |
| Count a record section's words | `record.Words` |
| Load records with their state, merge commit and tag | `status.LoadRecords` |
| Match trailers to tasks for a change or a record | `change.MatchTasks` |
| Commits on HEAD that carry a record's trailers | `record.CommitsFor` |
| Files under a directory that each first-parent commit added | `Git.Additions` |
| List tags; the first tag that holds each record | `Git.Tags`; `scan.RecordTags` |
| Tick a record's tasks from commit trailers | `archive.SyncRecord` |
| Check task headings in any file | `lint.checkTaskHeadings` |
| Report a result | `cli.Outcome`; map to an exit code with `cli.ExitCode` |
| Build a test repository | `testutil.NewRepo` |
| Stand in for OpenSpec | `testutil.FakeOpenSpec` |

## Invariants

- Only `internal/openspec` starts OpenSpec.
- Only `internal/gitx` starts git (test helpers excepted).
- Nothing below `cmd/dkt` reads the process environment; it is passed in through `cli.Options`, the clock (`Options.Now`) and the user lookup (`Options.UserHome`) included (test helpers excepted).
- Readers take bytes, not paths, so that content can come from any revision.
- Nothing writes through a symbolic link in the repository: check the path with `safepath.Check` first.
