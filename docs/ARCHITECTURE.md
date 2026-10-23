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
| `internal/change` | Change-folder rules: slug validation, `.openspec.yaml` metadata |
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
| Record kind, source and base | `change.AppendMetadata` |
| Install the schema | `setup.Init` |
| Report a result | `cli.Outcome`; map to an exit code with `cli.ExitCode` |
| Build a test repository | `testutil.NewRepo` |
| Stand in for OpenSpec | `testutil.FakeOpenSpec` |

## Invariants

- Only `internal/openspec` starts OpenSpec.
- Only `internal/gitx` starts git (test helpers excepted).
- Nothing below `cmd/dkt` reads the process environment; it is passed in through `cli.Options` (test helpers excepted).
- Readers take bytes, not paths, so that content can come from any revision.
