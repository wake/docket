# Design

## Context

Requirements are in `specs/openspec-runtime`, `specs/project-setup`, `specs/setup-inspection` and `specs/cli-output`. docket-init (`skills/docket-init/SKILL.md`) runs `dkt init --json --tools <coding agent>`, then `dkt inspect --json`.

`v0.15.0` (branch `legacy`) had both commands, with a per-machine schema, an `AGENTS.md` block, user-level skills and a workflow file; none of these remain. This change ports only the parts that still apply, read with `git show v0.15.0:<path>`.

Checked on 2026-11-05 with OpenSpec 1.7.0: `openspec init --tools claude` in an empty repository writes `openspec/config.yaml` (the line `schema: spec-driven` followed only by comments), six `.claude/skills/openspec-*/SKILL.md` and six `.claude/commands/opsx/*.md`. With the draft `docket` schema in `openspec/schemas/docket/`, `openspec new change`, `openspec instructions plan` and `openspec status` work.

## Goals / Non-Goals

**Goals:**
- `dkt init` and `dkt inspect` as the specs say, deciding each file's status in one place, so the two cannot disagree.
- Unit tests that run against a fake OpenSpec, and one integration test that runs against the real OpenSpec in CI, at 1.7.0 and at the latest release.

**Non-Goals:**
- What the proposal leaves out: the trailer, title and other-schema checks of `dkt inspect`, checking for superpowers, docket-doctor.
- Recording the hashes of shipped files at release time; the release step does that.
- Windows: the tests use a shell script for the fake OpenSpec and run on Linux and macOS, as CI does.

## Decisions

### Shipped files
They live under `shipped/` at the repository root and are embedded by the root package `docket` (`shipped.go`, `//go:embed shipped`), as `v0.15.0` embedded `schema/`:

| Source | Written to, or used as |
|---|---|
| `shipped/schema/schema.yaml`, `shipped/schema/templates/{proposal,spec,design,plan}.md` | `openspec/schemas/docket/` |
| `shipped/skill/SKILL.md` | `.claude/skills/docket/SKILL.md` and `.agents/skills/docket/SKILL.md` |
| `shipped/config/operations.yaml` | the `operations` section of `openspec/config.yaml` |
| `shipped/next-steps/{commit,reload,doctor}.md` | the entries of `result.next_steps`, trimmed |

Each file is its edited English text, except the schema, which is its draft; its review is deferred. `AGENTS.md`'s list of prompt files gains `shipped/`.

### Earlier releases
`internal/setup` holds a table from each target (the `operations` section counts as one) to the SHA-256 of the files earlier releases, v0.16.0 and later, shipped there. It is empty in this change; each release adds the hashes of what it ships. Tests replace it through a package variable.

### One decision table
`internal/setup` reads every target and decides, per target, the status `dkt inspect` reports (`same`, `missing`, `outdated`, `differs`) and the action `dkt init` takes (none, create, update, overwrite with `--force`, keep). It also collects refusals: a symbolic link on a path, an unreadable file, an unreadable `config.yaml` entry. `dkt inspect` reports the table. `dkt init` stops on any refusal before running anything, runs `openspec init` when needed, decides again (`config.yaml` may exist now), writes, and then checks ignores.

It extends `v0.15.0`'s `decideAll`, `decide`, `targetsOf`, `problemForTarget` and `write` in `internal/setup/setup.go`, which already decide missing, same and differs, overwrite with force, and collect symbolic links and unreadable files before writing: this change drops their per-machine folders, `AGENTS.md` block, user skills and project copy, and adds `outdated`. Ported as they are: `safepath.Check` (the walk that refuses symbolic links), and `install` and `readNoFollow` (new files created 0644 and directories 0755, both subject to the umask as with git; an existing file keeps its mode; never through a link).

### config.yaml
Edited line by line; the standard library has no YAML, and comments must survive.
- The `schema` line: `v0.15.0`'s `schemaKey` and `NameDocket` without `crossCheck`, which needs `go.yaml.in/yaml/v3`. A top-level `schema:` with a plain or quoted value and an optional comment; only the value's bytes are replaced, so the comment and the line ending stay. `schema:` with no value on its line, or two `schema:` lines, is unreadable.
- What `crossCheck` caught, now refused line by line: a line at the first column, not blank and not a comment, that starts with `{`, `[`, `"`, `'`, `?`, `%`, `---` or is `...`, or reads `schema` or `operations` followed by spaces before its colon, makes the file unreadable; so does a leading byte order mark.
- A CRLF file's section is compared as its LF text; the section dkt writes uses LF lines.
- The `operations` section: the top-level `operations:` line and every following line that is blank or starts with whitespace; it ends at the first line starting with anything else, a comment at the first column included. It is compared with the shipped section after trailing blank lines are trimmed. `operations:` with a value on its own line, or two such lines, is unreadable. The section is appended after one blank line, unless the file already ends with one.
- `--force` touches neither part when it holds another schema or other guidance.

### Files `openspec init` created
`dkt init` lists the paths of `git status --porcelain=v1 -z --untracked-files=all --no-renames` before and after running `openspec init`, with `v0.15.0`'s `gitx` `status`, `changedPaths` and `splitNUL`, comparing parsed paths, not raw records. Each path in the second listing and not in the first is reported with kind `openspec`: `created` when its record is untracked or added, `updated` otherwise (a tracked file `openspec init` changed). A path dkt then writes keeps dkt's kind, with status `created` when it did not exist before the run. A path that already had changes before `openspec init` is not reported, since its listing cannot tell. Files git ignores do not show up; the ignore check covers only Docket's own files.

### Ignore check
`v0.15.0`'s `gitx.Ignored`, which runs `git check-ignore -v -z --stdin`. Without `--no-index`, check-ignore never reports a tracked file, which is what the specs ask. `dkt init` runs it after writing; `dkt inspect` runs it on every target, whether or not the file exists.

### OpenSpec runner
`v0.15.0`'s `internal/openspec` `Runner`, with two changes:
- The 1.14.x range becomes a minimum: `--version` must print `MAJOR.MINOR.PATCH`, compared number by number with 1.7.0. Any other output, and a non-zero exit of `--version`, is unsupported; only a program that cannot be found or started is missing.
- Every child gets `OPENSPEC_TELEMETRY=0`, appended last to its environment so it overrides an earlier value.

Its errors map to findings: missing to `openspec-missing`, unsupported to `openspec-unsupported` (exit 4 from `dkt init`; a finding and exit 1 from `dkt inspect`), a failed call to `openspec-failed` (exit 1).

### Commands and help
`internal/cli` gets a command table: each command has a name, its own `flag.FlagSet` (which also accepts `--json`), and a run function. `dkt <command> --help` prints `usage: dkt init [--tools <coding agent>] [--force] [--json]` and the flag defaults; `dkt --help` lists the commands. A command that needs a repository resolves the root first and returns an `EnvError` with `no-repository` or `git-missing`.

### Output
- `dkt init`: `result` is `{"openspec_init": "<value>" | null, "files": [{"path", "kind", "status"}], "next_steps": ["…"]}`. As text: one line per file, `<status>  <path>`, then `Next steps:` and the numbered steps.
- `dkt inspect`: `result` is `{"openspec": {"version": "<v>" | null, "minimum": "1.7.0", "supported": <bool>}, "files": [{"path", "kind", "status"}]}`. As text: one line for OpenSpec, then the files.
- Findings and unreadable sources use the existing envelope and writers.

### Tests
- Unit tests use the fake OpenSpec ported from `v0.15.0`'s `testutil.FakeOpenSpec`: a shell script passed as `DOCKET_OPENSPEC` that logs its arguments and `OPENSPEC_TELEMETRY`. Extended: `init --tools <value>` writes a `config.yaml` like 1.7.0's and one `.claude/skills/openspec-explore/SKILL.md`, and fails when the value is `nosuch`.
- One integration test, `TestRealOpenSpec`, skipped unless `DOCKET_TEST_OPENSPEC=1`: in a new repository it runs `dkt init --tools claude` with the real CLI from `DOCKET_OPENSPEC`, then `openspec new change add-x`, `openspec instructions <artifact> --change add-x` for every artifact, and `openspec status --change add-x`.
- CI gains a job `openspec` on Ubuntu with Go stable and Node, for `openspec: ['1.7.0', 'latest']`, running `go test -run TestRealOpenSpec ./...` with `DOCKET_TEST_OPENSPEC=1` and `DOCKET_OPENSPEC="npx -y @fission-ai/openspec@<version>"`.

## Risks / Trade-offs

- [Line-based editing refuses valid YAML forms, such as a flow mapping or a value on the next line] → the file is listed as unreadable, nothing is written, and the user can rewrite the entry by hand.
- [The `latest` job fails when a new OpenSpec release breaks something, on a pull request that did not cause it] → intended: it is how Docket learns of the break. Pin the job to the last good version if it blocks unrelated work.
- [The schema's instructions ship as an unreviewed draft] → its review comes before the first release; replacing the file is a light review.
- [A file another process creates while `openspec init` runs is reported as created by it] → accepted.
