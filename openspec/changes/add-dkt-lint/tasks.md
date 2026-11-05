# Tasks

Each group is one slice an implementer finishes test-first. Scenario names refer to `specs/<capability>/spec.md`. Code marked as ported is read with `git show v0.15.0:<path>` (design.md names each piece). Tests run with `go test ./...`; every git call in a test uses `testutil.Env`, and every OpenSpec call in a unit test uses the fake OpenSpec or a stub.

## Reuse

| Need | Existing (path:symbol) | Decision | Reason | Evidence |
|---|---|---|---|---|
| Markdown lines, fences, headings, comments | `internal/markdown/markdown.go:Lines,FenceMask,Headings,Body,BlankComments` | reuse | Already read plan.md for `dkt sync` | `rg -n "^func " internal/markdown/markdown.go` |
| Code spans | `internal/markdown/markdown.go` | extend | Port `CodeSpans` and `backtickRun` from `v0.15.0` with their tests | `rg -n "CodeSpans" internal` (none), `git show v0.15.0:internal/markdown/markdown.go` |
| Pipe tables | none | new | `internal/markdown/table.go`, a port of `v0.15.0`'s `table.go` with its tests | `rg -n "Table" internal/markdown` (none), `git show v0.15.0:internal/markdown/table.go` |
| Plan structure | `internal/change/plan.go:ParsePlan,Task,Plan` | extend | Grow to `v0.15.0`'s structure, keeping comment-blanked reading | `rg -n "Cancelled\|ParsePlan" internal`, `git show v0.15.0:internal/change/plan.go` |
| Kinds and line form | `internal/change/metadata.go:Kinds,ValidLine` | reuse | The form `dkt new` checks | `rg -n "^func ValidLine\|^var Kinds" internal/change` |
| Schema of a change | `internal/change/metadata.go:SchemaOf` | reuse | Same reading as `dkt sync` | `rg -n "^func SchemaOf" internal/change` |
| Read the front matter | none (`internal/change/metadata.go:FrontMatter` only writes it) | new | Nothing reads it back yet; YAML's escapes are not Go's | `rg -n -i "frontmatter\|front matter" internal --type go` |
| Main spec title | none | new | `v0.15.0`'s `internal/lint/titles.go` read titles from `docket.yaml`; the first line is new | `rg -n "Specification" internal --type go` (none), `git show v0.15.0:internal/lint/titles.go` |
| Lint rules and report | none | new | `internal/lint`, a port of `v0.15.0`'s `lint.go`, `proposal.go`, `placeholder.go`, `plan.go`, `scenarios.go` and `structure.go`, reading deltas through OpenSpec instead of `internal/delta` | `rg --files internal \| rg lint` (none), `git show v0.15.0:internal/lint/lint.go`, `git show v0.15.0:internal/lint/structure.go` |
| Scenario slug | none | new | `lint.Slug`, a port of `v0.15.0`'s `delta.Slug` | `rg -n "func Slug" internal` (none), `git show v0.15.0:internal/delta/slug.go` |
| Run OpenSpec, its errors | `internal/openspec/runner.go:Runner,FailedError` | extend | Add the JSON readers and `FailedError.Messages` | `rg -n "^type\|^func" internal/openspec/runner.go` |
| OpenSpec check in a command | `internal/cli/runner.go:openSpecRunner` | reuse | Maps a missing or old OpenSpec to exit 4 | `rg -n "func openSpecRunner" internal/cli` |
| Fake OpenSpec | `internal/testutil/openspec.go:FakeOpenSpec` | extend | Answer `show` and `validate` from files | `rg -n "^case\|^[a-z-]*)" internal/testutil/openspec.go` |
| Command table | `internal/cli/command.go:Command` | extend | `Args: -1` for any number of positionals | `rg -n "cmd.Args" internal/cli/run.go` |
| Envelope, findings, exit codes | `internal/cli/envelope.go:Outcome,Finding,Unreadable` | reuse | Shared output of every command | `rg -n "^type " internal/cli/envelope.go` |
| Shipped templates | `shipped.go:Shipped` | reuse | Embeds `shipped/schema/templates/*.md` | `rg -n "go:embed" shipped.go` |
| Reads confined to the repository | `external: os.OpenRoot, os.Root.FS (Go 1.24)` | reuse | Standard library | `grep -n "Root) FS" "$(go env GOROOT)/api/go1.24.txt"` |
| Real OpenSpec test | `internal/cli/realopenspec_test.go:TestRealOpenSpec` | extend | Already runs `dkt init` and `dkt inspect` against the real CLI | `rg -n "func TestRealOpenSpec" internal/cli` |

## 1. Code spans and tables

**Review:** checks
**Files:** `internal/markdown/markdown.go`, `internal/markdown/table.go`, `internal/markdown/markdown_test.go`, `internal/markdown/table_test.go`

- [x] 1.1 Port the tests of `CodeSpans` and `FindTable` from `v0.15.0` (`internal/markdown/markdown_test.go`, `table_test.go`); verify they fail
- [x] 1.2 Port `CodeSpans`, `backtickRun`, `Table`, `Row`, `FindTable`, `splitRow` and `cell` (design.md, Document rules and Plan rules); verify every test passes

## 2. Plan structure

**Review:** checks
**Files:** `internal/change/plan.go`, `internal/change/plan_test.go`

- [x] 2.1 Write failing tests for `ParsePlan`: a task's `**Scenarios:**` line (its line, a comma-separated list with backticks dropped, `none` giving no items, an empty value giving one empty item); `**Compensates:** Task 3` (number, line, OK) and a malformed one (not OK); `## Task 3: x`, `#### Task 4: y`, `### Task 0: z` and `### Task 2 - x` as bad headings with level and line (none of them a task), and `### Tasks left` not a bad heading; Cancelled bullets `- Task 5: Name - reason` (OK, number, name, reason) and `- Task 5: Name` (not OK, number 5); the Reuse section's line and table, and no table when the section has none; a heading or bullet inside an HTML comment or a fence counts for nothing; the existing tests still pass with `Cancelled` holding structs; verify they fail
- [x] 2.2 Implement them (design.md, Plan rules); verify every test passes

## 3. Front matter and titles

**Review:** judgment (error paths)
**Files:** `internal/change/frontmatter.go`, `internal/change/frontmatter_test.go`

- [ ] 3.1 Write failing tests for every scenario of change-metadata's Front matter read and Main spec title, plus: what `FrontMatter` writes reads back to the same title, kind and source (`Booking: waitlist "v2"`, a backslash, a tab); a single-quoted value with `''`; a plain value with a ` # comment`; the YAML escapes `\/`, `\e`, `\N`, `é`, `\U0001F600`; an unknown escape, a key given twice, a top-level line that is not `key: value`, and an indented line outside `specs:` are errors; no front matter (first line `# Proposal`) reports false; a BOM before `---`; each field's line; `SpecTitle` for `# booking Specification` with capability `booking` (no title), `# identity/user-auth Specification` with capability `identity/user-auth` (no title), `# Booking Specification` (`Booking`) and `# Booking` (no title); verify they fail
- [ ] 3.2 Implement `ReadFrontMatter` and `SpecTitle` (design.md, Front matter and titles); verify every test passes

## 4. OpenSpec JSON

**Review:** judgment (environment-dependent, error paths)
**Files:** `internal/openspec/json.go`, `internal/openspec/json_test.go`, `internal/openspec/runner.go`, `internal/testutil/openspec.go`

- [ ] 4.1 Extend the fake OpenSpec to answer `show` and `validate` from `$FAKE_OPENSPEC_JSON` (design.md, OpenSpec JSON). Write failing tests, using the JSON OpenSpec 1.14.0 printed (design.md, Context) as fixtures: `ShowChange` gives the ADDED, MODIFIED and REMOVED requirements with their scenario names and the RENAMED pair; `ShowSpec` gives requirements and scenario names; `Validate` gives the issues (level, path, message, and `line` when OpenSpec gives one, else 0) when OpenSpec exits 1 with `items` on stdout, and none for a valid change; a `show` that exits 1 with a `status` message on stdout gives a `FailedError` whose `Messages` and `Error()` hold that message; stdout that does not decode gives an error naming the command; an unsupported version gives `UnsupportedError` and runs no `show`; unknown fields are ignored; verify they fail
- [ ] 4.2 Implement them; verify every test passes

## 5. Lint: scope and document rules

**Review:** judgment (error paths)
**Files:** `internal/lint/lint.go`, `internal/lint/document.go`, `internal/lint/lint_test.go`, `internal/lint/document_test.go`

- [ ] 5.1 Write failing tests, on a temporary directory read through `os.OpenRoot` and a stub OpenSpec, for every scenario of change-lint's What is linted, Front matter checked, New capabilities named, Proposal sections, Leftover template text and Main spec titles, plus: the `archive` folder and a link under `openspec/changes/` are not changes, and naming either gives `change-missing`; a name given twice is linted once; an unreadable file goes to `Unreadable` and its rules are skipped (skip when running as root); a change of another schema gets only `title-missing` for its new capabilities and `in.OpenSpec` is never called; `in.OpenSpec` failing returns its error and no report, before any rule runs; a nested capability path `identity/user-auth` is a new capability when its main spec is missing; a missing proposal.md is one `proposal-section`; no front matter gives `title-missing` and `metadata-invalid` for `kind` and `source`; a front matter error gives one `metadata-invalid` and no `title-missing`; with a readable front matter, each of: no `title`, `title: "   "` (`title-missing`), no `kind`, no `source`, `source: "a\nb"` (`metadata-invalid` naming the key, on the key's line when it has one); a `specs/<capability>/spec.md` with no delta section still needs a name (a file there is the delta, by OpenSpec's layout); an angle-bracket placeholder from a template (`<capability-path>`) outside a code span is `placeholder`, one inside a fenced block is not, an HTML comment that is not in a template is not, and a `.yaml` file of the change is not scanned; findings come sorted by change, file, line, rule; `Report.Specs` counts the main specs, 0 when a change is named; verify they fail
- [ ] 5.2 Implement `Run`, the scope, the front matter, new capability, proposal, placeholder and main spec title rules (design.md, Reading the worktree, The lint package, Front matter and titles, Document rules); verify every test passes

## 6. Lint: plan rules

**Review:** checks
**Files:** `internal/lint/plan.go`, `internal/lint/plan_test.go`

- [ ] 6.1 Write failing tests for every scenario of change-lint's Plan rules only with a plan, Task headings, Reuse table and Cancelled and compensating tasks, plus: a Reuse table with the wrong header; a Decision outside reuse, extend and new; a `new` row without Reason; a row without Evidence; Existing in backticks; a `reuse` row whose path is a link out of the repository counts as missing; `**Compensates:** Task 9` with no Task 9; a malformed Compensates line; a task number used by a task and a cancelled bullet; verify they fail
- [ ] 6.2 Port the plan rules (design.md, Plan rules); verify every test passes

## 7. Lint: delta rules

**Review:** judgment (reads OpenSpec's answers, error paths)
**Files:** `internal/lint/deltas.go`, `internal/lint/deltas_test.go`, `internal/lint/slug.go`, `internal/lint/slug_test.go`

- [ ] 7.1 Write failing tests, with a stub OpenSpec, for every scenario of change-lint's Scenario references, Scenario slugs and Deltas read through OpenSpec, plus: a change without plan.md gets no `scenario-ref` or `scenario-unclaimed` but still `scenario-duplicate` and `openspec-invalid`; a task without a `**Scenarios:**` line and a malformed item give `scenario-ref`; a reference to a main spec scenario the delta does not touch is accepted; a REMOVED requirement frees its scenarios' slugs; a RENAMED requirement keeps its scenarios (no duplicate with themselves, and a duplicate still reported when one of them clashes), and RENAMED plus MODIFIED under the new name replaces them; two main spec scenarios the delta does not touch that share a slug are reported, naming both; a MODIFIED requirement replaces the main one's scenarios; a finding about a scenario names the delta file and the heading's line, and line 0 when the heading cannot be found; `ShowSpec` failing for one capability gives `openspec-failed` and still checks the others; `Validate` warnings give nothing and an error whose path is `booking/spec.md` names `openspec/changes/<slug>/specs/booking/spec.md` and OpenSpec's line when it gives one; `Slug` of `Join twice — rejected`, `Ünïcode 名前 2`, `--x--`; verify they fail
- [ ] 7.2 Implement them (design.md, Delta rules); verify every test passes

## 8. dkt lint

**Review:** judgment (output format, public interface)
**Files:** `internal/cli/cmd_lint.go`, `internal/cli/cmd_lint_test.go`, `internal/cli/command.go`, `internal/cli/run.go`, `internal/cli/command_test.go`

- [ ] 8.1 Write failing tests for change-lint's Lint result and cli-output's Lint outside a repository, plus: `Args: -1` takes zero, one and three positionals while `dkt sync a b` is still a usage error; `dkt lint ""`, `dkt lint a/b`, `dkt lint 'a\b'`, `dkt lint .`, `dkt lint ..` (the inputs of `TestSyncPathInPlaceOfSlug`) are usage errors (exit 2) that read nothing; a `docket` change with no OpenSpec on `PATH` gives `openspec-missing` and exit 4 with no other finding; a repository with only main specs needs no OpenSpec; an unreadable file gives exit 3; the text output's first line `Linted 1 changes and 0 main spec titles.` followed by the findings; the JSON `result` with `changes` an empty array when nothing is linted; verify they fail
- [ ] 8.2 Implement `dkt lint` (design.md, Command); verify every test passes

## 9. Integration

**Review:** checks
**Files:** `internal/cli/realopenspec_test.go`

- [ ] 9.1 Extend `TestRealOpenSpec`: after `dkt init`, `dkt new add-x`, write a complete proposal (front matter kept, the five sections, `specs:` naming the new capability), a delta with one requirement and scenario, and a plan.md that breaks no plan rule and whose task claims that scenario; `dkt lint add-x --json` exits 0 with `result.changes` `[{"slug":"add-x","schema":"docket"}]`; with the claim removed it reports `scenario-unclaimed`; with the scenario removed it reports `openspec-invalid`. Run it locally with `DOCKET_OPENSPEC="npx -y @fission-ai/openspec@1.14.0"` and with `@latest`; verify both pass, and that the CI job passes on the pull request
- [ ] 9.2 Run `gofmt -l .`, `go vet ./...` and `go test ./...`; verify they are clean

## Workflow follow-up

- Archive the change with OpenSpec as the last commit of the pull request, after the reviews.
