# Design

## Context

Requirements are in `specs/change-lint`, `specs/change-metadata` and `specs/cli-output`. `v0.15.0` (branch `legacy`) had `dkt lint`: `internal/lint/` (`lint.go`, `proposal.go`, `placeholder.go`, `plan.go`, `scenarios.go`, `structure.go`, `titles.go`, `otherschema.go`), `internal/change/plan.go` (the full plan structure), `internal/markdown/` (`CodeSpans` in `markdown.go`, `table.go`) and its own delta parser (`internal/delta/`, with `Slug` in `slug.go`); read them with `git show v0.15.0:<path>`. That version kept the title in `docket.yaml` and parsed deltas itself; this one reads the proposal's front matter and reads deltas through OpenSpec.

What exists now: `change.ParsePlan` gives tasks with steps and the cancelled numbers (only `dkt sync` uses it); `internal/markdown` has `Lines`, `FenceMask`, `Headings`, `Body`, `BlankComments`, `Space`, `Trim`; `openspec.Runner` runs OpenSpec after a version check and returns stdout, with `FailedError` on a non-zero exit; the command table takes an exact number of positionals; `docket.Shipped` embeds the schema templates under `shipped/schema/templates/`.

OpenSpec 1.14.0, checked on 2026-11-06 in a project set up with `dkt init` and a change opened with `dkt new`:
- `openspec show <slug> --type change --json --deltas-only` gives `deltas[]`, each with `spec` (the capability path), `operation` (`ADDED`, `MODIFIED`, `REMOVED`, `RENAMED`), and either `requirements[]{name, text, scenarios[]{name, rawText}}` or, for `RENAMED`, `rename{from, to}`. It fails (exit 1) for a proposal without a Why section, and for a missing change; the message is then on stdout as `{"status":[{"severity","code","message"}]}` and stderr is empty. The proposal's front matter does not disturb it.
- `openspec show <capability> --type spec --json` gives `requirements[]` in the same shape; its `title` is the capability's id, not the name in the first line.
- `openspec validate <slug> --type change --json` prints `items[0].issues[]{level, path, message}` and exits 1 when an issue is an error; `path` is `<capability-path>/spec.md` for a delta and `file` for the change as a whole. It accepts a `docket` change whose proposal has only front matter, and reports a change without deltas (unless `.openspec.yaml` has `skip_specs: true`).
- `dkt new` writes a proposal holding only the front matter.

## Goals / Non-Goals

**Goals:**
- `dkt lint` as the specs say, reading every file through a handle confined to the repository.
- One lint package whose rules take parsed input, so most tests need no OpenSpec process.

**Non-Goals:**
- What the proposal leaves to later changes: routing, the default branch, records, issues, the `**Review:**` and `**Files:**` lines.

## Decisions

### Reading the worktree
`dkt lint` opens the repository root with `os.OpenRoot` and hands `root.FS()` to the lint package: every read, including the Reuse table's path checks, follows links only while they stay inside the repository, so a link to `/etc` is a missing path, not a read outside. The changes in scope are the entries of `openspec/changes/` that are directories (a link is never a change) other than `archive`; a named change that is not one of them is `change-missing`. A change's files are read once, every regular file under its folder, as `v0.15.0` did; a file that cannot be read goes to `unreadable` and the rules that need it are skipped. Alternative: `safepath.Check` and `safefile.ReadNoFollow` per file, as `dkt sync` does; refused because lint reads arbitrary paths named in the Reuse table, and a confined handle covers them all with one rule.

### OpenSpec JSON
`internal/openspec/json.go` adds, on `*Runner`:
- `ShowChange(slug) ([]Delta, error)`, `Delta{Spec, Operation string; Requirements []Requirement; Rename *Rename}`, `Rename{From, To string}`;
- `ShowSpec(capability) ([]Requirement, error)`, `Requirement{Name string; Scenarios []Scenario}`, `Scenario{Name string}`;
- `Validate(slug) ([]Issue, error)`, `Issue{Level, Path, Message string; Line int}` (OpenSpec gives `line` for some issues; 0 when absent).

Each decodes only the fields named, so OpenSpec may add others. `Validate` takes stdout as its answer whatever the exit status, when stdout decodes as an object with `items`; otherwise the `FailedError` stands. On any `FailedError`, the messages of a `status[]` array on stdout are kept in a new `FailedError.Messages []string`, and `Error()` adds them after the standard error, so `openspec-failed` names what OpenSpec said even when it said it on stdout. Output that does not decode is an error naming the command. The fake OpenSpec answers `show` and `validate` by printing the file `$FAKE_OPENSPEC_JSON/<args joined with _>.json` (for example `show_add-x_--type_change_--json_--deltas-only.json`) and exiting with the number in the file beside it named `.code`, if any; a missing file is exit 1 with a `status` message, as OpenSpec does for a missing change.

### The lint package
`internal/lint` holds the rules. `Run(in Input) (Report, error)`, with `Input{Root fs.FS; Templates fs.FS; Slugs []string; OpenSpec func() (OpenSpec, error)}` and `OpenSpec` an interface of the three methods above, which `*openspec.Runner` satisfies and tests stub. `Run` first settles the scope and each change's schema; when a `docket` change is in scope it calls `in.OpenSpec` once, before any rule runs, and returns its error unchanged (the command turns it into `openspec-missing` or `openspec-unsupported`, exit 4, reporting nothing else). A change of another schema needs no OpenSpec. `Report{Changes []Linted{Slug, Schema}; Specs int; Findings []Finding; Unreadable []Unreadable}`, findings sorted by change, file, line, rule and message as `v0.15.0` did. The command maps them onto `cli.Finding` and `cli.Unreadable` one to one.

Per `docket` change, in order: front matter and new capabilities, proposal sections, placeholders, then (with a plan.md) the plan rules, then the delta rules. Per change of another schema: new capabilities only. Then, with no change named, main spec titles.

### Front matter and titles
`internal/change/frontmatter.go`: `ReadFrontMatter(b []byte) (Front, bool, error)`, `Front{Fields map[string]Field; Specs map[string]string; SpecsLine int}` (not `FrontMatter`, which names the writer), `Field{Value string; Line int}`; the bool is false when the file has no front matter (its first line, after a BOM, is not `---`). Within it, a line at the first column is `key: value` or `key:` (an empty value reads as the empty string, for lint to judge; `specs:` takes no value); an indented line belongs to `specs:` and is `<capability-path>: <name>`; blank lines and lines starting with `#` are skipped. A value is plain (trimmed, a ` #` comment dropped), single-quoted (`''` is `'`) or double-quoted, unescaped by YAML's escape set (`\0 \a \b \t \n \v \f \r \e \  \" \/ \\ \N \_ \L \P \xHH \uHHHH \UHHHHHHHH`); anything else, a key given twice, or no closing `---` is an error. `strconv.Unquote` is not used: Go's escapes are not YAML's (`\/`, `\N`, `\e`, octal).

Rules: no front matter gives `title-missing` and one `metadata-invalid` per missing `kind` and `source`; an error gives one `metadata-invalid` naming the line and nothing else from the front matter; otherwise `title-missing` when `title` is absent or fails `ValidLine`, and `metadata-invalid` for a `kind` not in `Kinds` or a `source` that is absent or fails `ValidLine`, each with the key's line.

New capabilities: each `specs/<capability-path>/spec.md` file of the change (the path may have several segments) whose `openspec/specs/<capability-path>/spec.md` does not exist needs a name under `specs:`. A file there is the change's delta for that capability, by OpenSpec's layout, even when OpenSpec finds no requirement in it (which `validate` reports); reading the files rather than `show` keeps the rule working for every schema and when `show` fails.

`change.SpecTitle(firstLine, capability string) (string, bool)`: the first line matching `^# (.+) Specification$`, its name trimmed; no match, or a name equal to the capability's path, is no title. Main specs are every `openspec/specs/**/spec.md`, the capability being the folder's path under `openspec/specs/`; `Report.Specs` counts them.

### Document rules
Proposal sections: level-two headings, compared case-insensitively, from the comment-blanked lines: Why, What Changes, Non-Goals, Capabilities, Impact; Non-Goals must hold a line that is not blank after comments are blanked. A missing `proposal.md` is one `proposal-section` naming the file.

Placeholders: ported from `v0.15.0`'s `placeholder.go`: every HTML comment and every `<[A-Za-z][^<>\n]*>` outside a comment in `Templates`' `templates/*.md` is known text; in every `.md` file of the change, a match outside fences and code spans whose text is known is `placeholder`, with its line. `markdown.CodeSpans(line) [][2]int` is ported from `v0.15.0` with its tests. The record and issue templates are left out (later changes).

### Plan rules
`change.ParsePlan` grows to `v0.15.0`'s structure, keeping this version's reading from comment-blanked lines: `Task` gains `Scenarios []string`, `ScenariosLine int`, `Compensates []Ref`; `Plan` gains `BadHeadings []BadHeading`, `ReuseLine int`, `Reuse *markdown.Table`, and `Cancelled` becomes `[]Cancelled{Number int; Name, Reason string; Line int; OK bool}` (nothing outside `internal/change` reads it yet). `markdown.FindTable` and its types are ported from `v0.15.0`'s `table.go`. A bad heading is any heading whose text matches `^Task\s+\d` and is not a level-three `Task N: <name>` with N positive, so `### Task 2 - x` is one. The rules port `v0.15.0`'s `plan.go` as they are: task headings and numbers (tasks and cancelled bullets together, the later line named, the message listing every line), the Reuse table (header, at least one row, Decision, the path before the first `:` checked with `fs.Stat` on `Root` unless `external:`, Reason for `new`, Evidence), cancelled bullets, and `**Compensates:**` lines.

### Delta rules
With the deltas from `ShowChange` and, for each capability they touch whose main spec exists, the requirements from `ShowSpec` (once per capability per run):
- `scenario-ref` and `scenario-unclaimed`, ported from `v0.15.0`'s `scenarios.go`: the known slugs of a capability are its main spec's and its ADDED and MODIFIED scenarios'; every ADDED and MODIFIED scenario must be claimed.
- `scenario-duplicate`, from `checkDuplicates`: the main spec's requirements, less those REMOVED, those MODIFIED, and those RENAMED to a name that is also MODIFIED; then the delta's ADDED and MODIFIED scenarios in order; every scenario whose slug is already placed is reported on its own place, naming the earlier one. Unlike `v0.15.0`, two main spec scenarios that already clash are reported too: the spec asks that no two scenarios share a slug once the delta is applied, and a clash makes a `**Scenarios:**` reference ambiguous whoever wrote it.
- `Slug(name)` is `v0.15.0`'s `delta.Slug`, in `internal/lint`.
- OpenSpec gives no line numbers. A finding about a scenario names its file and finds the line itself: the first `#### Scenario: <name>` heading after the `### Requirement: <requirement>` heading it belongs to (fences and comments masked); no such heading gives line 0. This only places the finding; what the deltas hold comes from OpenSpec.
- `openspec-invalid`: each issue of level `ERROR` from `Validate`, its message prefixed with OpenSpec's `path`; the file is `openspec/changes/<slug>/specs/<path>` when `path` ends in `/spec.md`, with OpenSpec's line, else none. Warnings are not reported.
- `openspec-failed`: `ShowChange`, `ShowSpec` or `Validate` failing gives one finding with the error, and the rules needing that answer are skipped for the change (or, for `ShowSpec`, for that capability).

### Command
`lintCommand` in `internal/cli/cmd_lint.go`, `Usage: "lint [<slug>...] [--json]"`, `Args: -1`. `Command.Args` documents `-1` as any number, and `runCommand` skips the "unexpected argument" check for it. `runLint` refuses a name that is empty, holds `/` or `\`, or is `.` or `..` as a usage error (exit 2) before reading anything; drops repeated names; opens the root; passes `fs.Sub(docket.Shipped, "shipped/schema")` as `Templates` and `openSpecRunner` as `OpenSpec`. The result is `{"changes": [{"slug", "schema"}], "specs": N}` (`changes` never null). Text output is one line before the findings, `Linted <n> changes and <m> main spec titles.`, and the findings as every command prints them.

## Risks / Trade-offs

- [Every lint of a `docket` change starts OpenSpec two or more times (`show`, `validate`, one `show` per touched main spec); through `npx` that is seconds] → accepted: lint runs before a pull request and before archiving, not in a loop.
- [A scenario's line is found by a text search, so a delta OpenSpec reads differently from that search gets line 0] → the finding still names the file and the scenario.
- [Placeholders come from the templates this dkt ships, not the ones installed in the project] → as in `v0.15.0`; a project on an older schema may see text it never had flagged, and `dkt init` updates the templates.
- [This repository's main specs start with `# <capability> Specification`, so `dkt lint` reports `title-missing` for each] → intended; they get titles when the repository switches to Docket's workflow.
- [A folder or file on a linted path replaced by a link between listing and reading] → the confined handle refuses a link out of the repository at read time; a link inside it is read, which is no more than lint could read anyway.
