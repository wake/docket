# Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give every change and every main spec a title kept in Docket's own `docket.yaml`, and move a change's kind, source and base there from `.openspec.yaml`.

**Architecture:** `internal/change` owns the `docket.yaml` format: parsing, writing, the title form, and the choice between `docket.yaml` and the `x-` keys of an older change's `.openspec.yaml` (`ReadMetadata`). Every reader of metadata (`archive`, `status`, `lint`) goes through it. `dkt new` writes `docket.yaml`; `dkt lint` gains a title rule; `dkt archive` gains a spec title step after `openspec archive`; `dkt status` reports the title.

**Tech Stack:** Go 1.26, `go.yaml.in/yaml/v3`, OpenSpec CLI 1.14.x (only through `internal/openspec`).

**Spec:** `openspec/changes/add-titles/proposal.md` and the deltas under `openspec/changes/add-titles/specs/` (docket-metadata, change-creation, change-lint, change-archive, change-status, plan-sync, cli-output, openspec-runtime).

## Global Constraints

- The file is named `docket.yaml`. A change's `docket.yaml` has the keys `title`, `kind`, `source`, `base`, written in that order; a spec's has `title`.
- A title is valid when `strings.TrimSpace(s) != ""` and it contains neither `\n` nor `\r`. An invalid title reads as no title.
- A change with no `docket.yaml` is read from `x-kind`, `x-source`, `x-base` of `.openspec.yaml` and has no title. When `docket.yaml` exists, the `x-` keys are ignored. `created` is always read from `.openspec.yaml`.
- Docket never writes `.openspec.yaml` of a change it creates, and never rewrites an archived change.
- Finding rules: `title-missing`, `metadata-invalid`, `spec-title-failed`. The archive step that sets spec titles is reported as step `spec-title`.
- `dkt new` exits with 2 for a missing, blank or multi-line `--title`, before anything else is checked or created.
- `dkt status --json`: each change has `title`, a string or null; `metadata` keeps exactly `kind`, `source`, `base`, `created`.
- `evidence.md` rows, in order: Date, Docket, OpenSpec, Repository, HEAD, Title, Kind, Source, Base; an empty Title, Kind or Source is `none`.
- Architecture invariants: readers take bytes; only `internal/gitx` runs git; only `internal/openspec` runs OpenSpec; `safepath.Check` before every write.
- Everything in the repository is in English. Every commit for a task carries `Change: add-titles` and `Task: <N>` in one final trailer paragraph; structural and behavior changes go in separate commits.

## Review Focus

- A title with YAML-special or non-ASCII text (`Booking: waitlist #2`, `'quoted'`, `- dash`, `預約候補`) must read back exactly after `dkt new` and after the archive title step. Tests: Task 2 (`CreateDocket`, `SetKey` round trips).
- Setting `title` in a main spec's `docket.yaml` that has CRLF endings or no final newline must keep the other lines byte for byte and still parse. Test: Task 2 (`SetKey` append cases).
- A nested capability path (`booking/waitlist`) gets its title at `openspec/specs/booking/waitlist/docket.yaml`, creating the folder when needed. Test: Task 6.
- A symbolic link at the main spec's `docket.yaml` (or its folder) makes the title step report `spec-title-failed` and write nothing through the link. Test: Task 6.
- `dkt status` reads the title of a change whose first source is a branch, not a worktree. Test: Task 7.

## Reuse

| Need | Existing (path:symbol) | Decision | Reason | Evidence |
|---|---|---|---|---|
| Parse and emit YAML | external: go.yaml.in/yaml/v3 | reuse | already the YAML library of `internal/change` | `rg -n "go.yaml.in/yaml" go.mod internal/` |
| Read the `x-` keys of an older change | internal/change/metadata.go:ParseMetadata | extend | becomes the legacy branch of `ReadMetadata`; unexported once no caller is left (Task 7) | `rg -n "ParseMetadata" internal/` |
| Set one key's value, keeping every other byte | internal/change/metadata.go:SetBase | extend | generalized to `SetKey(b, key, value)`, which also appends a missing key | `rg -n "SetBase" internal/` |
| Write `docket.yaml` when a change is created | internal/change/new.go:AppendMetadata | new | it appends to a file OpenSpec wrote; `docket.yaml` is a new file created exclusively, so `CreateDocket` replaces it and it is deleted | `rg -n "AppendMetadata\|O_EXCL" internal/` |
| Title form check | none | new | the only blank-text check, `indented` in `internal/cli/cmd_archive.go`, formats output and does not reject line breaks | `rg -n "TrimSpace\(.*\) == \"\"" internal/` |
| Refuse a link on a write path | internal/safepath:Check | reuse | required before every write | `dkt index: safepath.Check` |
| Lock, create and clean up in `dkt new` | internal/cli/cmd_new.go:newCommand (cleanup, lockOpenSpec) | reuse | the docket.yaml write takes the place of AppendMetadata inside the same cleanup | `rg -n "cleanup\|lockOpenSpec" internal/cli/cmd_new.go` |
| Per-change lint rules | internal/lint/lint.go:Run | extend | one more rule in the loop, plus a main-spec pass when no slug is named | `rg -n "checkStructure\(r, c" internal/lint/lint.go` |
| Know whether a main spec exists | internal/lint/structure.go:(*run).mainSpec | new | `mainSpec` returns nil both for a missing file and for one with no requirements; the title rule needs `fs.Stat` | `rg -n "func \(r \*run\) mainSpec" -A 12 internal/lint/structure.go` |
| Map a spec path to its capability | internal/delta/capability.go:CapabilityPath | reuse | works on `specs/<cap>/spec.md`, for main specs after trimming `openspec/` | `dkt index: delta.CapabilityPath` |
| Load a change's metadata for archive, sync and rebase | internal/archive/load.go:loadChange | extend | reads `docket.yaml` and `.openspec.yaml`, then `change.ReadMetadata` | `rg -n "ParseMetadata" internal/archive/load.go` |
| Create a file or replace one without following links | internal/archive/sync.go:createFile, writeFile, writeOrRefuse | reuse | the title step creates or replaces a main spec's `docket.yaml`; `createFile` needs its folder to exist | `rg -n "func (createFile\|writeFile\|writeOrRefuse)" internal/archive/` |
| Create a missing capability folder | external: os.MkdirAll | reuse | after `safepath.Check` of the folder; nothing in `internal/archive` creates folders, and `setup.install` is unexported and specific to the schema | `rg -n "MkdirAll" internal/archive internal/setup internal/change` |
| Archive step sequence | internal/archive/run.go:(*archiveRun).change | extend | the title step runs between `r.archive` and the evidence step | `rg -n "evPath :=" internal/archive/run.go` |
| Evidence rows | internal/archive/evidence.go:evidence.render | extend | rows renamed and Title added | `rg -n "x-kind" internal/archive/evidence.go` |
| Move the base | internal/archive/rebase.go:Rebase | extend | writes the file `ReadMetadata` read the base from | `rg -n "SetBase" internal/archive/rebase.go` |
| Read a change's files from any source | internal/scan:ReadFiles | reuse | status adds `docket.yaml` to its keep list | `dkt index: scan.ReadFiles` |
| Status of each change | internal/status/load.go:Load | extend | title and metadata through `ReadMetadata` | `rg -n "ParseMetadata" internal/status/load.go` |
| Fake OpenSpec in tests | internal/testutil/openspec.go:FakeOpenSpec | extend | a switch that makes `new` leave a `docket.yaml`, so the create fails | `rg -n "FAKE_OPENSPEC_" internal/testutil/openspec.go` |
| Test repositories | internal/testutil/repo.go:NewRepo | reuse | as every command test does | `dkt index: testutil.NewRepo` |

---

### Task 1: Titles for the main specs and for this change

**Scenarios:** none

**Review:** checks

**Files:**
- Create: `openspec/specs/change-archive/docket.yaml`, `openspec/specs/change-creation/docket.yaml`, `openspec/specs/change-lint/docket.yaml`, `openspec/specs/change-status/docket.yaml`, `openspec/specs/cli-output/docket.yaml`, `openspec/specs/conflict-detection/docket.yaml`, `openspec/specs/openspec-runtime/docket.yaml`, `openspec/specs/plan-sync/docket.yaml`, `openspec/specs/project-setup/docket.yaml`, `openspec/specs/workspace-scan/docket.yaml`
- Create: `openspec/changes/add-titles/docket.yaml`

This change cannot give the existing specs titles through its deltas: CI lints the main specs with the new code before the change is archived. The titles are written directly. `.openspec.yaml` of this change keeps its `x-` keys until Task 8, so `dkt` from any commit of the branch can read the change.

- [ ] **Step 1: Write the ten main spec titles**

One line each, `title: <title>`: change-archive `Change archiving`, change-creation `Change creation`, change-lint `Change lint`, change-status `Change status`, cli-output `Command output`, conflict-detection `Conflict detection`, openspec-runtime `OpenSpec runtime`, plan-sync `Plan sync`, project-setup `Project setup`, workspace-scan `Workspace scan`.

- [ ] **Step 2: Write this change's docket.yaml**

```yaml
title: Titles for changes and specs
kind: feature
source: docket/ideas/2026-10-29-document-titles.md, item 1 of docket/order.md
base: 5df2ba2f4d187386ada6fc9e556edbb32f9597d7
```

- [ ] **Step 3: Check that OpenSpec accepts the files**

Run: `openspec validate --specs --strict` and `openspec validate add-titles --strict` (OpenSpec 1.14.3)
Expected: every spec and the change are valid.

- [ ] **Step 4: Commit**

### Task 2: The docket.yaml format and metadata reading in internal/change

**Scenarios:** docket-metadata#metadata-read-from-docket-yaml, docket-metadata#change-made-before-docket-yaml, docket-metadata#both-files-present, docket-metadata#title-beside-a-main-spec, docket-metadata#title-on-two-lines

**Review:** judgment

**Files:**
- Create: `internal/change/docket.go`, `internal/change/docket_test.go`
- Modify: `internal/change/metadata.go`, `internal/change/metadata_test.go`, `internal/archive/rebase.go` (the `SetBase` call only)

**Interfaces:**
- Produces:
  - `const DocketFile = "docket.yaml"`, `const OpenSpecFile = ".openspec.yaml"`
  - `type Docket struct { Title, Kind, Source, Base string }` with yaml tags `title`, `kind,omitempty`, `source,omitempty`, `base,omitempty`
  - `func ValidTitle(s string) bool`
  - `func ParseDocket(b []byte) (Docket, error)`: fails when `b` is not a YAML mapping or a key does not decode as a string; an invalid title reads as `""`.
  - `func (d Docket) Marshal() ([]byte, error)`: keys in struct order, empty ones left out.
  - `func CreateDocket(path string, d Docket) error`: creates `path` with `O_WRONLY|O_CREATE|O_EXCL|O_NOFOLLOW`, mode 0644, writes `d.Marshal()`, and fails when the file does not parse back to `d`.
  - `type Metadata struct { Title, Kind, Source, Base, Created, File string }` (the yaml tags move to an unexported struct used to read `.openspec.yaml`; `Schema` goes if nothing reads it)
  - `func ReadMetadata(files map[string][]byte) (Metadata, error)`: `files` holds a change folder's files by relative path. With `docket.yaml` present: Title, Kind, Source, Base from it, `File` = `docket.yaml`. Otherwise from `.openspec.yaml`'s `x-` keys, `File` = `.openspec.yaml`, no title. `Created` from `.openspec.yaml` when it parses, else `""`, without failing. Neither file: error `docket.yaml is missing`, `File` = `docket.yaml`. On error, `File` names the file at fault.
  - `func SetKey(b []byte, key, value string) ([]byte, error)`: replaces the value on the one line that starts with `key:`; with no such line, appends `key: value` as a new line (adding a line ending first when `b` does not end with one, `\r\n` when `b` uses CRLF). The value is written as `yaml.Marshal` writes the scalar. Fails when more than one line starts with `key:`, or when the result does not parse to a mapping where `key` equals `value`.
  - `SetBase` is removed; `ParseMetadata` stays exported until Task 7.

- [ ] **Step 1: Structural commit — `SetBase` becomes `SetKey`**

Rename `SetBase(b, base)` to `SetKey(b, key, value)` with the same behavior for `key` = `x-base` (still failing when the line is missing), update `internal/archive/rebase.go` to call `change.SetKey(b, "x-base", nb)`, and keep every existing test green. Commit as a refactor.

- [ ] **Step 2: Write the failing tests**

- `TestReadMetadataFromDocket`: files `docket.yaml` = `title: Booking waitlist\nkind: feature\nsource: issue 12\nbase: <sha>\n` and `.openspec.yaml` = `schema: docket\ncreated: 2026-01-02\n` → `Metadata{Title: "Booking waitlist", Kind: "feature", Source: "issue 12", Base: sha, Created: "2026-01-02", File: "docket.yaml"}`.
- `TestReadMetadataLegacy`: only `.openspec.yaml` with `x-kind`, `x-source`, `x-base` → those values, `Title == ""`, `File == ".openspec.yaml"`.
- `TestReadMetadataBothFiles`: `docket.yaml` base `aaa…`, `.openspec.yaml` `x-base: bbb…` → `Base` is `aaa…`.
- `TestReadMetadataErrors`: `docket.yaml` = `- a\n` → error, `File == "docket.yaml"`; no files → error, `File == "docket.yaml"`; only `.openspec.yaml` = `schema: [\n` → error, `File == ".openspec.yaml"`.
- `TestParseDocketTitleForm`: a block scalar `title: |\n  a\n  b\n` (decodes to `"a\nb\n"`) → `Title == ""`; a double-quoted `title: "a\rb"` with the YAML escape `\r` → `""`; `title: "   "` → `""`; `title: 預約候補` → `"預約候補"`; `title: [a]` → error.
- `TestValidTitle`: `""`, `"   "`, `"a\nb"`, `"a\rb"` → false; `"Booking waitlist"`, `"預約候補"` → true.
- `TestCreateDocketRoundTrip`: for titles `Booking: waitlist #2`, `'quoted'`, `- dash`, `預約候補`, `CreateDocket` then `ParseDocket` gives the same `Docket`; the file starts with `title:` and has the keys in order title, kind, source, base; a second `CreateDocket` on the same path fails with `fs.ErrExist`; a symbolic link at the path is not followed.
- `TestSetKeyAppend`: `SetKey([]byte("a: 1"), "title", "T")` → `"a: 1\ntitle: T\n"`; `SetKey([]byte("a: 1\r\n"), "title", "T")` → `"a: 1\r\ntitle: T\r\n"`; `SetKey([]byte("title: Old\nb: 2\n"), "title", "Booking: x")` → `"title: 'Booking: x'\nb: 2\n"`; two `title:` lines → error.

- [ ] **Step 3: Run the tests to see them fail**

Run: `go test ./internal/change/`
Expected: FAIL (undefined: `ReadMetadata`, `ParseDocket`, `CreateDocket`, …; `SetKey` append cases fail).

- [ ] **Step 4: Implement the Interfaces above in `internal/change/docket.go` and `internal/change/metadata.go`**

- [ ] **Step 5: Run the tests to see them pass**

Run: `go test ./internal/change/ ./internal/archive/`
Expected: PASS.

- [ ] **Step 6: Mutate the changed lines and commit**

### Task 3: dkt new requires a title and writes docket.yaml

**Scenarios:** change-creation#missing-title, change-creation#blank-title, change-creation#change-created, change-creation#date-prefix-rejected, change-creation#uppercase-rejected, change-creation#reserved-name-rejected, change-creation#slug-active-in-another-worktree, change-creation#slug-archived, change-creation#unreadable-worktree-blocks-creation, change-creation#concurrent-runs-keep-the-first, change-creation#schema-not-installed, change-creation#symlinked-changes-folder-refused, cli-output#json-on-success, cli-output#json-on-failure, openspec-runtime#unsupported-version, openspec-runtime#cli-missing, openspec-runtime#openspec-command-fails

**Review:** judgment

**Files:**
- Modify: `internal/cli/cmd_new.go`, `internal/cli/cmd_new_test.go`, `internal/change/new.go`, `internal/change/new_test.go`, `internal/testutil/openspec.go`
- Modify when they run `dkt new`: other `internal/cli/*_test.go`

**Interfaces:**
- Consumes: `change.ValidTitle`, `change.Docket`, `change.CreateDocket`, `change.DocketFile` (Task 2)
- Produces: `change.NewResult` gains `Title string \`json:"title"\`` after `Slug`; `change.AppendMetadata` is deleted.

- [ ] **Step 1: Write the failing tests**

- `TestNewMissingTitle`: `dkt new add-x --kind feature --source x` → exit 2, finding rule `usage`, no `openspec/changes/add-x/`, and the fake OpenSpec log has no `new` call.
- `TestNewBlankTitle`: `--title "   "`, `--title "a\nb"` and `--title "a\rb"` (Go strings with a real line feed and carriage return) → exit 2, nothing created.
- `TestNewWritesDocketYAML`: `dkt new add-x --kind feature --source "issue 12" --title "Booking waitlist"` → `ParseDocket` of `openspec/changes/add-x/docket.yaml` is `{Booking waitlist, feature, issue 12, <merge base>}`; `.openspec.yaml` is byte for byte `schema: docket\ncreated: 2026-01-02\n`, as the fake wrote it; the JSON `result.title` is `Booking waitlist`.
- `TestNewDocketYAMLFails`: with `FAKE_OPENSPEC_DOCKET_YAML=1` (the fake's `new` also writes `docket.yaml`), → finding `metadata-failed`, exit 1, `openspec/changes/add-x/` removed. Every `CreateDocket` error takes this one branch, so this test covers cleanup for write failures too.
- Existing `dkt new` tests: add `--title X` to every invocation that is not about the title, so each still checks what it did (slug rules, slug reuse, unreadable worktree, concurrency, schema missing, symlink refusal, JSON output, unsupported version, OpenSpec failure). The OpenSpec failure test asserts no `docket.yaml` exists.

- [ ] **Step 2: Run the tests to see them fail**

Run: `go test ./internal/cli/ -run 'TestNew'`
Expected: FAIL (`--title` is an unknown flag; no `docket.yaml`).

- [ ] **Step 3: Implement**

In `newCommand`: define `--title`; after the `--source` check, `!change.ValidTitle(title)` (empty included) is `UsageError{Msg: "new needs --title: one line of text that is not blank"}`; the usage line becomes `new <slug> --kind <…> --source <text> --title <text>`. Check `openspec/changes/<slug>/docket.yaml` with `safepath.Check` where `.openspec.yaml` was checked, before and after OpenSpec runs, and replace `AppendMetadata` with `CreateDocket(…, change.Docket{Title: title, Kind: kind, Source: source, Base: base})` inside the same cleanup. Delete `AppendMetadata` and its tests, and the fake's `FAKE_OPENSPEC_BAD_YAML` and `FAKE_OPENSPEC_NO_EOL` when no test uses them; add `FAKE_OPENSPEC_DOCKET_YAML`.

- [ ] **Step 4: Run the tests to see them pass**

Run: `go test ./internal/cli/ ./internal/change/ ./internal/testutil/`
Expected: PASS.

- [ ] **Step 5: Mutate the changed lines and commit**

### Task 4: dkt lint reports missing titles

**Scenarios:** change-lint#change-without-title, change-lint#new-capability-without-title, change-lint#existing-capability-needs-no-delta-title, change-lint#main-spec-without-title, change-lint#named-change-skips-main-specs, change-lint#broken-docket-yaml, change-lint#lint-every-active-change, change-lint#named-change-missing, change-lint#no-openspec-installed

**Review:** judgment

**Files:**
- Create: `internal/lint/titles.go`, `internal/lint/titles_test.go`
- Modify: `internal/lint/lint.go`
- Modify where a test expects a clean report: `internal/lint/*_test.go`, `internal/cli/cmd_lint_test.go`, `internal/cli/cmd_archive_test.go`, `internal/cli/cmd_status_test.go`, `internal/cli/run_test.go`

**Interfaces:**
- Consumes: `change.ParseDocket`, `change.DocketFile` (Task 2), `delta.CapabilityPath`
- Produces: `func checkTitles(r *run, c *loaded, ds []capDelta)` and `func checkSpecTitles(r *run)`, unexported.

Rules (findings' `File` is repository-relative):
- The change: `docket.yaml` absent and not unreadable → `title-missing` on `openspec/changes/<slug>/docket.yaml`, message saying the change has no title and that a change made by an older dkt moves `x-kind`, `x-source` and `x-base` there as `kind`, `source` and `base`. Present: a `ParseDocket` error → `metadata-invalid` on it; a `""` title → `title-missing` on it.
- Each delta: a `docket.yaml` beside it is checked the same way. Absent, and `fs.Stat` of `openspec/specs/<cap>/spec.md` says it does not exist → `title-missing` on `openspec/changes/<slug>/specs/<cap>/docket.yaml`, naming the capability. Any other `Stat` error → unreadable.
- Main specs, only when `in.Slugs` is empty: walk `openspec/specs`; for every regular `spec.md` that `CapabilityPath` maps, check the `docket.yaml` beside it as above, with no change on the finding. A missing `openspec/specs` is no finding; a walk error, or a `docket.yaml` that exists but cannot be read, is unreadable and gets no `title-missing`.

- [ ] **Step 1: Write the failing tests**

One test per scenario in `titles_test.go`, on `fstest.MapFS` repositories (`repoWith`):
- `TestTitleMissingOnChange`: `add-x` with a proposal and no `docket.yaml`, `runOn(repo, "add-x")` → one `title-missing` with `File == "openspec/changes/add-x/docket.yaml"`.
- `TestTitleMissingOnNewCapability`: delta `specs/booking/spec.md`, no main spec, no delta `docket.yaml` → `title-missing` on `openspec/changes/add-x/specs/booking/docket.yaml`.
- `TestNoDeltaTitleForExistingCapability`: same with `openspec/specs/booking/spec.md` present → no `title-missing` for booking.
- `TestTitleMissingOnMainSpec`: `openspec/specs/booking/spec.md` without `docket.yaml`, `runOn(repo)` → `title-missing` on `openspec/specs/booking/docket.yaml` with `Change == ""`.
- `TestNamedChangeSkipsMainSpecs`: same main spec, `add-x` with a title, `runOn(repo, "add-x")` → no finding on `openspec/specs/`.
- `TestBrokenDocketYAML`: `openspec/changes/add-x/docket.yaml` = `- a\n` → `metadata-invalid` on that file; a delta `docket.yaml` = `- a\n` → `metadata-invalid` on it; `openspec/specs/booking/docket.yaml` = `- a\n` with `runOn(repo)` → `metadata-invalid` on it, and no `title-missing` for it.
- `TestUnreadableMainSpecTitle` (on `osRepo`, as `TestUnreadableFile` does): `openspec/specs/booking/docket.yaml` with mode 0 → listed in `Unreadable`, no `title-missing` for booking; `openspec/specs/booking/` with mode 0 → `openspec/specs/booking` listed in `Unreadable`.
- `TestBlankTitle`: `title: "  "` → `title-missing`.
- Fixtures in other lint and command tests that assert a clean report gain a `docket.yaml` with a title, so they keep testing what they tested (`lint-every-active-change`, `named-change-missing` and `no-openspec-installed` are the existing tests, kept green).

- [ ] **Step 2: Run the tests to see them fail**

Run: `go test ./internal/lint/ -run 'Title|DocketYAML|MainSpec'`
Expected: FAIL (no `title-missing` or `metadata-invalid` findings).

- [ ] **Step 3: Implement the rules in `internal/lint/titles.go` and call them from `Run`**

`checkTitles` runs for every loaded change after `checkStructure`; `checkSpecTitles` runs once, after the loop, when `len(in.Slugs) == 0`.

- [ ] **Step 4: Run the tests to see them pass**

Run: `go test ./internal/lint/ ./internal/cli/` and `go run ./cmd/dkt lint --default-branch main` in this worktree
Expected: PASS, and `dkt lint: ok` (the titles of Task 1 satisfy the rule).

- [ ] **Step 5: Mutate the changed lines and commit**

### Task 5: Archive, rebase and sync read the base where the change keeps it

**Scenarios:** change-archive#unticked-task, change-archive#squashed-pull-request, change-archive#requirement-changed-since-base, change-archive#two-changes-in-one-run-touch-one-requirement, change-archive#other-requirement-in-the-same-capability, change-archive#added-name-now-taken, change-archive#evidence-written, change-archive#rebase-after-merging-the-default-branch, change-archive#rebase-a-change-made-before-docket-yaml, change-archive#nothing-to-move, plan-sync#task-with-commits-ticked, plan-sync#task-without-commits-left-alone, plan-sync#two-tasks-in-one-commit, plan-sync#commits-counted-from-the-base

**Review:** judgment

**Files:**
- Modify: `internal/archive/load.go`, `internal/archive/rebase.go`, `internal/archive/evidence.go`, `internal/archive/base.go`, `internal/archive/run.go`, `internal/cli/cmd_rebase.go`, `internal/cli/cmd_archive.go`
- Test: `internal/archive/rebase_test.go`, `internal/archive/sync_test.go`, `internal/archive/evidence_test.go`, `internal/archive/run_test.go`, `internal/archive/base_test.go`, `internal/cli/cmd_rebase_test.go`, `internal/cli/cmd_archive_test.go`

**Interfaces:**
- Consumes: `change.ReadMetadata`, `change.SetKey`, `change.Metadata` (Task 2)
- Produces: `loadedChange.Meta` is a `change.Metadata` with `File` set; nothing new is exported.

Behavior:
- `loadChange` reads `docket.yaml` and `.openspec.yaml` (absent is fine, other read errors are unreadable) into a map and calls `ReadMetadata`. An error is `metadata-invalid` on `m.File`; an empty base is `metadata-invalid` saying `<file> has no base` (`base`, or `x-base` for `.openspec.yaml`); `base-missing` names `m.File`.
- `Rebase` sets `base` in `docket.yaml`, or `x-base` in `.openspec.yaml` when that is the file the metadata came from, with `SetKey`, and writes it with `writeOrRefuse`. It never creates `docket.yaml`.
- `evidence.render` writes the rows of the Global Constraints; the Checks line says `none changed since the base`.
- Messages and human output that say `x-base` say `the base` (`base.go`, `run.go` no-task-commits, `cmd_archive.go` drift header `at the base:`, `cmd_rebase.go` `base: <old> -> <new>` and `base is already <sha>`).

- [ ] **Step 1: Write the failing tests**

- `TestRebaseDocketYAML`: change with `docket.yaml` (`title`, `kind`, `source`, `base: <old>`) and a `.openspec.yaml` without `x-base`; after the default branch merge, `Rebase` → `docket.yaml` equals the old bytes with only the base value replaced; `.openspec.yaml` unchanged.
- `TestRebaseLegacy`: change with only `.openspec.yaml` `x-base` → `x-base` moved, no `docket.yaml` created.
- `TestSyncCountsFromDocketBase`: base recorded in `docket.yaml`; a commit with `Change: add-x`, `Task: 1` reachable from that base is not counted; one after it is.
- `TestEvidenceRows`: a change whose `docket.yaml` has title `Booking waitlist` → `evidence.md` has `| Title | Booking waitlist |`, `| Kind | feature |`, `| Source | … |`, `| Base | <sha> |` in that order after `| HEAD |`; a legacy change → `| Title | none |`.
- `TestLoadDocketYAMLInvalid`: `docket.yaml` = `- a\n` → `dkt archive` stops at step `plan` with `metadata-invalid` on `openspec/changes/add-x/docket.yaml`.
- Existing archive, base check and sync tests keep their `.openspec.yaml` fixtures (the legacy path) and stay green; update the assertions on the renamed messages.

- [ ] **Step 2: Run the tests to see them fail**

Run: `go test ./internal/archive/ ./internal/cli/ -run 'Rebase|Sync|Evidence|Archive'`
Expected: FAIL.

- [ ] **Step 3: Implement the behavior above**

- [ ] **Step 4: Run the tests to see them pass**

Run: `go test ./internal/archive/ ./internal/cli/`
Expected: PASS.

- [ ] **Step 5: Mutate the changed lines and commit**

### Task 6: dkt archive carries spec titles to the main specs

**Scenarios:** change-archive#title-carried-to-a-new-spec, change-archive#title-replaced-on-an-existing-spec, change-archive#second-of-two-fails

**Review:** judgment

**Files:**
- Create: `internal/archive/titles.go`, `internal/archive/titles_test.go`
- Modify: `internal/archive/run.go`

**Interfaces:**
- Consumes: `change.ParseDocket`, `change.Docket.Marshal`, `change.SetKey` (Task 2); `createFile`, `writeFile`, `safepath.Check`
- Produces: `func (r *archiveRun) specTitles(c *loadedChange, folder string) ([]Finding, []Unreadable, error)`, unexported.

Behavior: for each capability of `sortedCaps(c)`, read `<folder>/specs/<cap>/docket.yaml` (the archived folder). Absent: skip. Otherwise the target is `openspec/specs/<cap>/docket.yaml`: `safepath.Check` it; when its folder is missing, `os.MkdirAll` it (the folder path passed `safepath.Check`); when the target does not exist, `createFile` with `Docket{Title: t}.Marshal()`; when it does, read it, `SetKey(b, "title", t)` and `writeFile`.
- A delta `docket.yaml` or a target that exists but cannot be read is listed as unreadable (exit 3), as the proposal's failure modes say.
- A delta `docket.yaml` that does not parse or has no valid title, a link, `SetKey` failing, or a failure creating the folder or writing the file is `spec-title-failed` on the target, naming the capability and the reason (exit 1).
- The step stops at the first capability with either. In `change`, the step runs after `r.archive`; when it stops, the change is listed in `Archived` with `Evidence: ""`, the run stops at step `spec-title`, no `evidence.md` is written, and the archived folder stays in the working tree.

- [ ] **Step 1: Write the failing tests**

- `TestTitleCarriedToNewSpec`: delta for new capability `waitlist` with `docket.yaml` `title: Waitlist`; after `Archive` → `openspec/specs/waitlist/docket.yaml` is `title: Waitlist\n`, then `evidence.md` exists.
- `TestTitleReplacedOnExistingSpec`: `openspec/specs/booking/docket.yaml` = `title: Bookings\nowner: x\n`, delta `docket.yaml` `title: Booking` → file is `title: Booking\nowner: x\n`.
- `TestNoDeltaTitleLeavesSpecAlone`: no delta `docket.yaml` → main spec `docket.yaml` bytes unchanged, none created.
- `TestTitleNestedCapability`: capability `booking/waitlist` → `openspec/specs/booking/waitlist/docket.yaml` created with its folder.
Every failure test below asserts the same outcome, through one helper: `Stopped == {Change: "add-x", Step: "spec-title"}`, `Archived[0].Folder` exists, `Archived[0].Evidence == ""`, no `evidence.md` in it, and the target's bytes as before the run. These also pin the order: evidence comes after the title step.
- `TestTitleTargetIsLink`: `openspec/specs/booking/docket.yaml` is a symbolic link to a file outside the repository → `spec-title-failed`, the outside file unchanged.
- `TestTitleDeltaInvalid`: delta `docket.yaml` = `- a\n`, and another = `title: "  "` → `spec-title-failed`.
- `TestTitleWriteFails`: existing target and `writeTemp` replaced by a failing one (as `rebase_test.go` does) → `spec-title-failed`.
- `TestTitleTargetUnreadable`: existing target with mode 0 (skipped as root, like the lint test) → `Unreadable` names `openspec/specs/booking/docket.yaml`, no finding.
- The existing two-change test still stops `add-y` at `lint` with `add-x` archived and its evidence written.

- [ ] **Step 2: Run the tests to see them fail**

Run: `go test ./internal/archive/ -run 'Title'`
Expected: FAIL.

- [ ] **Step 3: Implement `specTitles` and call it from `change`**

- [ ] **Step 4: Run the tests to see them pass**

Run: `go test ./internal/archive/ ./internal/cli/`
Expected: PASS.

- [ ] **Step 5: Mutate the changed lines and commit**

### Task 7: dkt status gives each change's title

**Scenarios:** change-status#title-shown, change-status#change-without-title, change-status#idle-change, change-status#uncommitted-edits, change-status#no-commit-yet

**Review:** judgment

**Files:**
- Modify: `internal/status/load.go`, `internal/status/load_test.go`, `internal/cli/cmd_status.go`, `internal/cli/cmd_status_test.go`, `internal/change/metadata.go`, `internal/change/metadata_test.go`

**Interfaces:**
- Consumes: `change.ReadMetadata` (Task 2)
- Produces: `status.Change` gains `Title *string \`json:"title"\`` right after `Slug`; `change.ParseMetadata` becomes unexported (no caller outside `internal/change` is left).

Behavior: `keep` adds `docket.yaml`; `Load` calls `ReadMetadata(files)`. An error leaves `Metadata` and `Title` nil (as a broken `.openspec.yaml` does now); otherwise `Metadata` is `{Kind, Source, Base, Created}` and `Title` points to the title when it is not `""`. The base for commits and activity is `m.Base`. The human line is `<slug>  "<title>"  <phase>  …` (`%q`), and without a title `<slug>  <phase>  …` as now.

- [ ] **Step 1: Write the failing tests**

- `TestLoadTitle`: worktree change with `docket.yaml` `title: Booking waitlist` → `*ch.Title == "Booking waitlist"`, `Metadata.Base` from `docket.yaml`.
- `TestLoadTitleFromBranch`: the change exists only on `refs/remotes/origin/add-y` with a `docket.yaml` → its title is read.
- `TestLoadNoTitle`: legacy change → `ch.Title == nil`, metadata from the `x-` keys; JSON has `"title":null`.
- `TestLoadActivityFromDocketBase`: base in `docket.yaml` → last activity counts from it (the idle-days case of `change-status#idle-change`).
- `TestStatusLineTitle`: `dkt status` human output for a titled change contains `add-x  "Booking waitlist"  `; exit 0.
- Existing activity and uncommitted tests stay green.

- [ ] **Step 2: Run the tests to see them fail**

Run: `go test ./internal/status/ ./internal/cli/ -run 'Title|Status'`
Expected: FAIL.

- [ ] **Step 3: Implement the behavior above**

- [ ] **Step 4: Run the tests to see them pass**

Run: `go test ./...`
Expected: PASS.

- [ ] **Step 5: Mutate the changed lines and commit**

### Task 8: Docs, shipped prompts and this change's metadata

**Scenarios:** none

**Review:** checks

**Files:**
- Modify: `AGENTS.md`, `README.md`, `README.zh-TW.md`, `docs/ARCHITECTURE.md`, `openspec/schemas/docket/templates/plan.md`, `openspec/schemas/docket/schema.yaml`, `openspec/changes/add-titles/.openspec.yaml`

- [ ] **Step 1: Edit the docs and prompts**

- `AGENTS.md` (the shipped "Change workflow" section): the `dkt new` line gains `--title "<title>"`, and one sentence says that a change's title, kind, source and base are in its `docket.yaml`, and that a new capability's delta has a `docket.yaml` with its title.
- `README.md` and `README.zh-TW.md`: the `dkt new` usage line gains `--title`; `rebase` writes one line of `docket.yaml`; one paragraph on upgrading: changes made by an older dkt still work, `dkt lint` asks for their `docket.yaml`, and main specs need one each.
- `docs/ARCHITECTURE.md`: `internal/change` row says `docket.yaml` metadata; the capability index replaces `change.AppendMetadata`, `change.ParseMetadata` and `change.SetBase` with `change.CreateDocket`, `change.ReadMetadata`, `change.ParseDocket` and `change.SetKey`.
- `templates/plan.md`: the heading is `# Implementation Plan`.
- `schema.yaml` plan instruction: after "Keep the writing-plans header and task structure", say the heading is `# Implementation Plan`, since the change's title names the work.
- `.openspec.yaml` of this change: remove the `x-kind`, `x-source` and `x-base` lines; `docket.yaml` holds them.

- [ ] **Step 2: Check them mechanically**

Run: `openspec instructions plan --change add-titles` (OpenSpec 1.14.3), `go test ./...`, `go run ./cmd/dkt lint --default-branch main`, `go run ./cmd/dkt status add-titles --json`, and `rg -n "x-base|x-kind|x-source|AppendMetadata|SetBase" AGENTS.md README.md README.zh-TW.md docs/ARCHITECTURE.md openspec/schemas/`
Expected: the instructions print the new heading; tests pass; lint is ok; status gives the title `Titles for changes and specs` and the base from `docket.yaml`; `rg` finds only sentences about changes made by an older dkt.

- [ ] **Step 3: Commit**
