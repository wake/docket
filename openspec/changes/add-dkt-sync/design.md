# Design

## Context

Requirements are in `specs/plan-sync`, `specs/change-metadata` and `specs/cli-output`. `v0.15.0` (branch `legacy`) had `dkt sync` (`internal/cli/cmd_sync.go`, `internal/archive/sync.go`), plan parsing (`internal/change/plan.go`, `tick.go`), Markdown line reading (`internal/markdown/markdown.go`) and trailer reading (`internal/gitx/log.go`); read them with `git show v0.15.0:<path>`. That version read two trailers (`Change: <slug>` and `Task: N`) from the commits after a recorded base; this one reads one (`Change: <slug>/<N>`) from the whole history of HEAD.

The shipped prompts already describe the behaviour: the plan template and schema (`Change: <change-name>/<N>`, steps as `- [ ]` outside code blocks, Cancelled Tasks, a task finished without commits ticked by hand), the `docket` skill (run `dkt sync <slug>` before the pull request) and the archive guidance (`dkt sync <change> --check`: stop if a task has no commit or a task with commits is not ticked).

## Goals / Non-Goals

**Goals:**
- `dkt sync` as the specs say, with plan parsing that `dkt lint` extends instead of replaces.

**Non-Goals:**
- What the proposal leaves out: records, structure checks, a lock.

## Decisions

### Shared file helpers
`internal/setup`'s `readNoFollow` and `install` move unchanged to a new package `internal/safefile` as `ReadNoFollow(path)` and `Install(dst, data, perm, keepMode, rel)`, with the `rename` hook that a setup test replaces (exported as `Rename`); `internal/setup` calls them. This is a restructuring with no change in behaviour, in its own commit before any `dkt sync` code; `internal/setup`'s tests cover `Install` through `Write`, and a new `internal/safefile` test covers `ReadNoFollow` refusing a link at the file itself, which `Decide` never reaches because `safepath.Check` refuses it first. `dkt sync` writes plan.md with `Install(…, mode of the existing file, true, …)` after `safepath.Check`, so a link on the path is refused and the file keeps its mode.

### Plan parsing
`internal/markdown` ports, from `v0.15.0`'s `internal/markdown/markdown.go`, only what ticking needs: `Lines` (drops a leading BOM, turns CRLF and lone CR into line ends), `FenceMask`, `Headings`, `Body` and `BlankComments`, with their tests. `internal/change/plan.go` ports `ParsePlan` reduced to `Plan{Tasks []Task; Cancelled []int}`, with `Task{Number int; Name string; Line int; Steps []Step}` and `Step{Line int; Done bool}`; a step is a checkbox line of the task's body outside fences and comments. `internal/change/tick.go` ports `Tick` and `lineStarts` as they are: `Tick` changes the one byte of each named line's box, so line endings, a BOM and every other byte stay. A plan with two `### Task N` headings of the same number has both; every section with a named number is ticked (`dkt lint` reports the duplicate later).

### Trailers
`gitx.ChangeTrailers()` runs `git log -z --reverse --format=%H%x00%(trailers:key=Change,valueonly,unfold) HEAD` and returns, oldest first, each commit's SHA and its `Change` values (one per line of the field, trimmed, empty ones dropped). The format has no trailing `%x00`: with `-z` git puts one NUL between commits, so the output is the SHA and the values field alternating, NUL-separated, and a commit with no trailer has an empty values field. Every commit reachable from HEAD is read, both sides of a merge included. Git matches the key case-insensitively and reads only the last paragraph, as the spec asks. An unborn HEAD (`Repo.Head` nil) has no commits and runs no `git log`.

`change.TaskRefs(value, slug string) (tasks []int, ok, mine bool)`: `mine` is false unless the value is `slug` or starts with `slug + "/"`; for `slug` alone, no tasks and `ok`; after the `/`, the list is split on commas, each part trimmed, and each must match `^[1-9][0-9]*$`, else `ok` is false (`trailer-malformed`).

### Change schema
`change.SchemaOf(b []byte) string` reads `.openspec.yaml` line by line: the first line at the first column of the form `schema:` followed by a value, plain or in single or double quotes, with an optional comment; the value, unquoted. No such line, or a missing file, gives `docket`. An unreadable file is listed as unreadable.

### Command
`syncCommand` (`Args: 1`, flag `--check`), in `internal/cli/cmd_sync.go`:
1. `safepath.Check` on `openspec/changes/<slug>/plan.md`: a link anywhere on that path gives `symlink-refused` (exit 1), with or without `--check`, so a link is reported the same way in both modes. Then `openspec/changes/<slug>` must be a directory (`change-missing`); `.openspec.yaml`'s schema must be `docket` (`other-schema`); `plan.md` must exist (`plan-missing`) and be readable with `safefile.ReadNoFollow` (unreadable, exit 3).
2. Read the trailers; for each commit and value: skip values that are not `mine`; `trailer-malformed` for a bad list; `trailer-unknown-task` for a number with no `### Task` heading (a cancelled number included). Findings name the commit's short SHA and the value; their `File` is the plan's path.
3. For each task, its commits and its unticked steps. Without `--check`, tick and write when any step is due (`symlink-refused` when a link is on the path). With `--check`, findings `task-unticked` (commits and unticked steps) and `task-no-commit` (no commit and an unticked step), each with the task heading's line.
4. Result and text as the spec says: `Task <N>: <done>/<steps> steps, <k> commits` per task, with `, ticked <t>` when the run ticked some.

## Risks / Trade-offs

- [The whole history of HEAD is read on every run] → accepted: one `git log` call; a repository with a very long history pays for it once per run.
- [A squashed pull request loses its trailers] → the `docket` skill and the schema forbid squash merges; after one, `--check` reports the tasks as having no commit.
- [A task finished without commits and not ticked by hand blocks the archive] → intended: the plan rules say to tick it by hand.
