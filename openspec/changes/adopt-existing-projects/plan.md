# Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let a project adopt Docket without false failures: report installed files git ignores, main specs without titles and changes of another OpenSpec schema, handle such changes in lint, status, archive, sync and rebase, and teach the doctor to fix each with the user's consent.

**Architecture:** `internal/change` reads a change's declared schema and counts checkboxes; every command keys on that one reading. `internal/gitx` asks `git check-ignore` once for the repository files `internal/setup` reports, and `internal/cli` turns the answer into `install-ignored`. `internal/lint` gains the other-schema branch and exports its main-spec title check and an other-schema listing for `dkt inspect`. The doctor's prompt and a conversion guide carry the consent steps.

**Tech Stack:** Go 1.24 standard library, git, `go.yaml.in/yaml/v3` (already a dependency).

**Spec:** `openspec/changes/adopt-existing-projects/proposal.md`, `design.md`, `specs/{docket-metadata,project-setup,setup-inspection,change-lint,change-status,change-archive,plan-sync}/spec.md`.

## Global Constraints

- go.mod stays at `go 1.24`; no new dependency. Public repository: English only, no hostnames or personal paths.
- No test and no manual run writes the real `~/.claude`, `~/.agents` or `$CLAUDE_CONFIG_DIR`: tests use `testutil.Env`; a manual `dkt init` sets `HOME`, `XDG_DATA_HOME` and `CLAUDE_CONFIG_DIR` to new temporary directories. Never `go install` a branch build.
- The schema `docket` is `change.DocketSchema`; a change whose `.openspec.yaml` has no `schema` key, or an empty one, has that schema.
- Findings and messages, exact:
  - `install-ignored`: `file` is the repository-relative path; message `ignored by <source>:<line>: <pattern>`, as git reports source, line and pattern.
  - `other-schema` in `dkt inspect`: `file` is `openspec/changes/<slug>/.openspec.yaml`; message `made with the <schema> schema; Docket leaves it to that schema's files (<ticked> of <total> steps ticked)`.
  - `other-schema` in `dkt sync` and `dkt rebase`: same `file`; message `made with the <schema> schema; dkt <sync|rebase> works only on changes of the docket schema`.
  - `title-missing` in `dkt inspect`: exactly what `dkt lint` reports for a main spec.
- `dkt lint --json`: `result.other_schema` is a list of `{"slug", "schema"}`, sorted by slug, `[]` when none. Text output: one line per such change, `other schema  <slug>: <schema>`.
- `dkt status --json`: every change has `"schema"`; text output appends ` (<schema>)` to the line of a change of another schema.
- `evidence.md` gains the rows `| Schema | <schema> |` and `| Skipped steps | plan, base |` (`none` for a change of the docket schema).
- Read failures go to each command's existing `unreadable` list: a `schema` that is not a YAML scalar, and a git failure of the ignore check.
- Commits: identity `git -c user.name="Claude Code" -c user.email=wake+claude-code@protype.tw`; message through `-F <file>`; the trailers `Change: adopt-existing-projects`, `Task: <N>` and `Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>` together in the final paragraph. A restructuring and a behavior change are separate commits.
- Each skill's `.claude/skills/` and `.agents/skills/` copies are byte-for-byte identical.

## Review Focus

- A negated rule (`.claude/*` then `!.claude/skills/`) leaves the skills visible to git: no `install-ignored` for them, while `.claude/settings.local.json` stays ignored. Pinned in Task 2.
- `tasks.md` checkboxes inside a fenced code block are not counted. Pinned in Task 1.
- `schema:` with an empty value reads as `docket`; `schema: [a]` is unreadable, not a crash. Pinned in Task 1.
- `dkt lint add-x` naming a change of another schema applies only its new-capability titles, as the unnamed run does. Pinned in Task 3.
- `dkt archive add-x add-y`, one change of each schema, archives both and writes the right evidence rows for each. Pinned in Task 6.

## Reuse

| Need | Existing (path:symbol) | Decision | Reason | Evidence |
|---|---|---|---|---|
| Read `.openspec.yaml` keys | internal/change/metadata.go:parseMetadata | extend | add the `schema` key beside `created` and the `x-` keys, through a `ReadSchema` that does not need `docket.yaml` | `rg -n "openspecFile" internal/change` |
| Recognize a checkbox line | internal/change/tick.go:stepRe | reuse | the plan's step pattern; `tasks.md` uses the same `- [ ]` form | `rg -n "stepRe" internal/change` |
| Skip fenced code | internal/markdown/markdown.go:FenceMask | reuse | as `ParsePlan` does | `rg -n "FenceMask" internal/change` |
| Run git with input | internal/gitx/git.go:outputIn | reuse | feeds the paths on stdin; exit 1 means none ignored, as `runOptional` treats it | `rg -n "func (g Git) outputIn" internal/gitx` |
| Repository files init installs or keeps | internal/setup/setup.go:Result | reuse | `Result.Files` and `Report.Files` already list them, repository-relative unless per-machine | `rg -n "type File struct" internal/setup` |
| Map setup output to findings | internal/cli/cmd_init.go:initCommand | extend | add the ignore check after a successful write | `rg -n "res.Files" internal/cli/cmd_init.go` |
| Inspect findings | internal/cli/cmd_inspect.go:inspectCommand | extend | add the ignore check, main-spec titles and other-schema changes | `rg -n "inspectStatus" internal/cli` |
| Main-spec title rule | internal/lint/titles.go:checkSpecTitles | extend | exported as `SpecTitles` for inspect, one rule in one place | `rg -n "func checkSpecTitles" internal/lint` |
| Delta title rule | internal/lint/titles.go:checkTitles | extend | split so a change of another schema runs only the delta part | `rg -n "func checkTitles" internal/lint` |
| Per-change lint loop | internal/lint/lint.go:Run | extend | branch on schema after `load` | `rg -n "checkProposal\(r, c\)" internal/lint` |
| Lint result | internal/cli/cmd_lint.go:LintResult | extend | `OtherSchema` field | `rg -n "type LintResult" internal/cli` |
| Status per change | internal/status/load.go:Load | extend | read `tasks.md`, set `Schema`, summarize checkboxes | `rg -n "Summarize\(plan, named\)" internal/status` |
| Phase thresholds | internal/status/status.go:Summarize | extend | a `SummarizeChecklist` beside it with the same phase names | `rg -n "func Summarize" internal/status` |
| Archive per change | internal/archive/run.go:change | extend | skip the plan and base steps for another schema | `rg -n "func (r \*archiveRun) change" internal/archive` |
| Change loader | internal/archive/load.go:loadChange | extend | read the schema first; do not require a base or plan for another schema | `rg -n "has no %s" internal/archive` |
| Evidence | internal/archive/evidence.go:evidence | extend | two rows | `rg -n "type evidence struct" internal/archive` |
| Sync and rebase entry | internal/archive/sync.go:Sync | extend | refuse another schema after loading | `rg -n "func Sync|func Rebase" internal/archive` |
| Rebase entry | internal/archive/rebase.go:Rebase | extend | refuse another schema after loading | `rg -n "func Rebase" internal/archive` |
| Changes with a given schema in tests | internal/testutil/openspec.go:FakeOpenSpec | reuse | its `new` writes `schema: <arg>`; tests may also write `.openspec.yaml` directly | `rg -n "schema: %s" internal/testutil` |
| Doctor steps | .agents/skills/docket-doctor/SKILL.md | extend | three new problem kinds and the conversion guide beside it | `ls .agents/skills/docket-doctor` |

---

### Task 1: Read a change's schema and count checkboxes

**Scenarios:** docket-metadata#schema-read, docket-metadata#schema-missing

**Review:** checks

**Files:**
- Modify: `internal/change/metadata.go`, `internal/change/tick.go`
- Test: `internal/change/metadata_test.go`, `internal/change/tick_test.go`

**Interfaces:**
- Produces: `const DocketSchema = "docket"`; `func ReadSchema(files map[string][]byte) (string, error)` — the `schema` of `files[OpenSpecFile]`; `DocketSchema` when the file or the key is missing or the value is empty; an error when the file is not a YAML mapping or the value is not a scalar. `func Checkboxes(b []byte) (ticked, total int)` — every line `stepRe` matches outside fenced code.

- [ ] **Step 1: Write the failing tests.** `TestReadSchema` table: `schema: spec-driven\ncreated: 2026-01-02\n` → `spec-driven`; `created: 2026-01-02\n` → `docket`; `schema:\n` → `docket`; no `.openspec.yaml` → `docket`; `schema: [a]\n` → error; `- a\n` → error. `TestCheckboxes`: `"- [x] 1.1 a\n- [ ] 1.2 b\n  * [X] c\n"` → 2, 3; a checkbox inside a ```` ``` ```` fence is not counted; `"no boxes\n"` → 0, 0.
- [ ] **Step 2: Run** `go test ./internal/change -run 'TestReadSchema|TestCheckboxes'` — expect FAIL (undefined).
- [ ] **Step 3: Implement** both in the named files; `ReadSchema` decodes the same document node `parseMetadata` checks.
- [ ] **Step 4: Run** `go test ./internal/change` — expect PASS.
- [ ] **Step 5: Commit** (`Task: 1`).

### Task 2: Report installed files that git ignores

**Scenarios:** project-setup#ignored-skill-reported, project-setup#tracked-file-not-reported, project-setup#nothing-ignored, setup-inspection#ignored-file-inspected

**Review:** judgment

**Files:**
- Create: `internal/gitx/ignore.go`, `internal/gitx/ignore_test.go`
- Create: `internal/cli/ignored.go`
- Modify: `internal/cli/cmd_init.go`, `internal/cli/cmd_inspect.go`
- Test: `internal/cli/cmd_init_test.go`, `internal/cli/cmd_inspect_test.go`

**Interfaces:**
- Produces: `type Ignore struct { Path, Source, Pattern string; Line int }`; `func (g Git) Ignored(paths []string) ([]Ignore, error)` — one `git check-ignore -v -z --stdin` with the paths NUL-separated on stdin; exit 1 is an empty answer; an entry whose pattern starts with `!` is dropped (a negated match means not ignored); tracked paths are not reported (git's default).
- Produces (cli): `func ignoredFindings(c *Context, files []setup.File) ([]Finding, []Unreadable)` — checks every `File` whose `Path` is not absolute and whose `Status` is not `Removed`.

- [ ] **Step 1: Write the failing gitx tests** in a `testutil.NewRepo` with a commit: `.gitignore` holding `.claude/*`, `!.claude/skills/`, `openspec/`, `AGENTS.md`, with `AGENTS.md` force-added. `Ignored` of `.claude/skills/docket/SKILL.md`, `.claude/settings.local.json`, `openspec/config.yaml` (not on disk), `AGENTS.md` returns exactly `.claude/settings.local.json` (`.gitignore`, 1, `.claude/*`) and `openspec/config.yaml` (`.gitignore`, 3, `openspec/`). No paths ignored → empty slice, nil error.
- [ ] **Step 2: Run** `go test ./internal/gitx -run TestIgnored` — expect FAIL.
- [ ] **Step 3: Implement** `Ignored`.
- [ ] **Step 4: Write the failing cli tests.**
  - `TestInitIgnoredSkill`: committed repo with `.gitignore` line 3 `.claude/`; `dkt init` exits 1; every `.claude/skills/...` file exists; one `install-ignored` finding per file with message `ignored by .gitignore:3: .claude/`; `.gitignore` byte-for-byte unchanged.
  - `TestInitTrackedNotReported`: `.gitignore` holds `AGENTS.md` and `AGENTS.md` is committed → no `install-ignored` names it.
  - `TestInitNegatedRuleNotReported`: `.claude/*` and `!.claude/skills/` → no `install-ignored`.
  - `TestInspectIgnoredFile`: `.git/info/exclude` with `openspec/` on line 7 and no `openspec/` folder → `install-ignored` naming `openspec/config.yaml`, message `ignored by .git/info/exclude:7: openspec/`; the exclude file unchanged.
  - Existing clean-setup tests (`TestInspectCleanSetup`, `TestRerunNoDifferences`) still pass: nothing is ignored there.
- [ ] **Step 5: Run** `go test ./internal/cli -run 'TestInitIgnored|TestInitTracked|TestInitNegated|TestInspectIgnored'` — expect FAIL.
- [ ] **Step 6: Implement** `ignoredFindings` in `internal/cli/ignored.go`, call it in `dkt init` after a successful `setup.Init` (not on refusal), and in `dkt inspect` on `rep.Files`; a git error lists the paths' source `git check-ignore` as unreadable.
- [ ] **Step 7: Run** `go vet ./... && go test ./...` — expect PASS. Commit (`Task: 2`).

### Task 3: lint a change of another schema

**Scenarios:** change-lint#change-of-another-schema, change-lint#new-capability-of-another-schema, change-lint#change-of-another-schema-has-no-title, change-lint#change-without-title, change-lint#new-capability-without-title, change-lint#existing-capability-needs-no-delta-title, change-lint#main-spec-without-title, change-lint#named-change-skips-main-specs, change-lint#broken-docket-yaml

**Review:** judgment

**Files:**
- Modify: `internal/lint/lint.go`, `internal/lint/titles.go`, `internal/cli/cmd_lint.go`
- Test: `internal/lint/titles_test.go`, `internal/lint/lint_test.go`, `internal/cli/cmd_lint_test.go`

**Interfaces:**
- Consumes: `change.ReadSchema`, `change.DocketSchema`, `change.Checkboxes` (Task 1).
- Produces: `type OtherSchema struct { Slug, Schema string; Ticked, Total int }`; `Report.OtherSchema []OtherSchema` (sorted by slug); `func SpecTitles(repo fs.FS) ([]Finding, []Unreadable)` — the main-spec title check `Run` does when no change is named; `func OtherSchemas(repo fs.FS) ([]OtherSchema, []Unreadable)` — every active change under `openspec/changes/` of another schema, with the checkboxes of its `tasks.md`.

- [ ] **Step 1: Restructure (no behavior change).** Split `checkTitles` into the change-title part and `checkDeltaTitles(r *run, c *loaded, ds []capDelta)`; add `SpecTitles` wrapping `checkSpecTitles`. Run `go test ./internal/lint` — PASS. Commit (`Task: 3`, `refactor(lint): split the title checks`).
- [ ] **Step 2: Write the failing tests.**
  - `TestOtherSchemaChange` (lint package): `add-x` with `.openspec.yaml` `schema: spec-driven`, a `proposal.md` lacking Non-Goals, a `tasks.md`, no `docket.yaml`, a delta for an existing main spec `booking` (which has its title) → no finding names `add-x`; `Report.OtherSchema` is `[{add-x spec-driven ...}]`; `add-x` is in `Report.Changes`.
  - `TestOtherSchemaNewCapability`: same with a delta for `waitlist` and no main spec → exactly one finding, `title-missing` for `waitlist`; with a `specs/waitlist/docket.yaml` holding a YAML list → `metadata-invalid` instead.
  - `TestOtherSchemaNamed`: `Run` with `Slugs: ["add-x"]` gives the same findings as the unnamed run for `add-x`.
  - `TestOtherSchemaUnreadable`: `schema: [a]` → the `.openspec.yaml` is in `Unreadable`, and the change gets no other finding.
  - `TestOtherSchemas`: two changes, one `docket`, one `spec-driven` with 1 of 3 boxes ticked → one entry, Ticked 1, Total 3; the `archive` folder is not listed.
  - cli `TestLintOtherSchemaJSON`: `dkt lint --json` → `result.other_schema` is `[{"slug":"add-x","schema":"spec-driven"}]`, exit 0; text output has the line `other schema  add-x: spec-driven`; with no such change `result.other_schema` is `[]`.
- [ ] **Step 3: Run** `go test ./internal/lint ./internal/cli -run 'OtherSchema'` — expect FAIL.
- [ ] **Step 4: Implement.** In `Run`, after `load`, read the schema; on an error list `.openspec.yaml` as unreadable and skip the change's rules; for another schema record it and run only `deltas` and `checkDeltaTitles`. `OtherSchemas` uses the same reading. `LintResult.OtherSchema` maps to `{slug, schema}`.
- [ ] **Step 5: Run** `go vet ./... && go test ./...` — expect PASS. Commit (`Task: 3`).

### Task 4: dkt inspect reports main-spec titles and changes of another schema

**Scenarios:** setup-inspection#main-spec-title-inspected, setup-inspection#change-of-another-schema-inspected

**Review:** judgment

**Files:**
- Modify: `internal/cli/cmd_inspect.go`
- Test: `internal/cli/cmd_inspect_test.go`

**Interfaces:**
- Consumes: `lint.SpecTitles`, `lint.OtherSchemas` (Task 3).

- [ ] **Step 1: Write the failing tests.**
  - `TestInspectMainSpecTitle`: after init, add `openspec/specs/booking/spec.md` with no `docket.yaml` → `title-missing` naming `openspec/specs/booking/docket.yaml`, message equal to `dkt lint`'s for it; exit 1.
  - `TestInspectOtherSchema`: `openspec/changes/add-x/.openspec.yaml` `schema: spec-driven`, `tasks.md` with four unticked boxes → `other-schema`, `file` `openspec/changes/add-x/.openspec.yaml`, message `made with the spec-driven schema; Docket leaves it to that schema's files (0 of 4 steps ticked)`.
  - `TestInspectCleanSetup` and `TestInspectNothingWritten` still pass (their main specs, if any, have titles; add none).
- [ ] **Step 2: Run** `go test ./internal/cli -run 'TestInspectMainSpecTitle|TestInspectOtherSchema'` — expect FAIL.
- [ ] **Step 3: Implement** in `inspectCommand`, reading the worktree through `os.OpenRoot(c.Repo.Root)` as `dkt lint` does; unreadable entries join `out.Unreadable`.
- [ ] **Step 4: Run** `go test ./internal/cli` — expect PASS. Commit (`Task: 4`).

### Task 5: dkt status shows the schema

**Scenarios:** change-status#change-of-another-schema-shown, change-status#no-tasks-file, change-status#docket-change

**Review:** judgment

**Files:**
- Modify: `internal/status/load.go`, `internal/status/status.go`, `internal/cli/cmd_status.go`
- Test: `internal/status/status_test.go`, `internal/status/load_test.go`, `internal/cli/cmd_status_test.go`

**Interfaces:**
- Consumes: `change.ReadSchema`, `change.Checkboxes` (Task 1).
- Produces: `Change.Schema string` (`json:"schema"`); `func SummarizeChecklist(tasks []byte, ok bool) (phase string, progress Progress)` — `ok` false or no box → `Design`; none ticked → `Ready`; all → `Done`; else `Implementing`; `StepsTicked`/`StepsTotal` the box counts, tasks 0 of 0.

- [ ] **Step 1: Write the failing tests.** `TestSummarizeChecklist` table: no file → design 0/0; `"- [ ] a\n"` → ready 0/1; one of three ticked → implementing 1/3; all ticked → done. `TestLoadOtherSchema`: a change with `schema: spec-driven`, `tasks.md` one of three ticked, no `plan.md` → `Schema` `spec-driven`, phase `implementing`, steps 1/3, tasks `[]`, tasks 0/0. `TestLoadDocketSchema`: a change made with `schema: docket` → `Schema` `docket`, phase from its plan. cli `TestStatusOtherSchema`: JSON has `"schema":"spec-driven"`; the text line ends with ` (spec-driven)`; a `docket` change's line has no suffix.
- [ ] **Step 2: Run** `go test ./internal/status ./internal/cli -run 'Checklist|OtherSchema|DocketSchema'` — expect FAIL.
- [ ] **Step 3: Implement.** `Load` keeps `tasks.md`, sets `Schema` (an unreadable schema adds an unreadable entry for `.openspec.yaml` and leaves `Schema` `docket`), and uses `SummarizeChecklist` for another schema instead of the plan.
- [ ] **Step 4: Run** `go vet ./... && go test ./...` — expect PASS. Commit (`Task: 5`).

### Task 6: archive, sync and rebase a change of another schema

**Scenarios:** change-archive#change-of-another-schema-archived, change-archive#new-capability-without-a-title-stops-the-archive, change-archive#rebase-refused-for-another-schema, change-archive#evidence-written, plan-sync#sync-refused-for-another-schema

**Review:** judgment

**Files:**
- Modify: `internal/archive/load.go`, `internal/archive/run.go`, `internal/archive/evidence.go`, `internal/archive/sync.go`, `internal/archive/rebase.go`
- Test: `internal/archive/run_test.go`, `internal/archive/evidence_test.go`, `internal/archive/sync_test.go`, `internal/archive/rebase_test.go`, and the matching `internal/cli/cmd_archive_test.go`, `cmd_sync_test.go`, `cmd_rebase_test.go` where the CLI tests live

**Interfaces:**
- Consumes: `change.ReadSchema`, `change.DocketSchema` (Task 1); lint's other-schema behavior (Task 3) through `Input.Lint`.
- Produces: `loadedChange.Schema string`; `evidence.Schema string` and `evidence.Skipped []string`.

- [ ] **Step 1: Write the failing tests.**
  - `TestArchiveOtherSchema`: `add-x` with `schema: spec-driven`, `tasks.md`, no `plan.md` or `docket.yaml`, merged → archived, exit 0; its `evidence.md` has `| Schema | spec-driven |` and `| Skipped steps | plan, base |`.
  - `TestArchiveOtherSchemaUntitledCapability`: same with a new capability `waitlist` and no title beside its delta → stops at `lint` with `title-missing`, and the fake OpenSpec log has no `archive` call.
  - `TestArchiveBothSchemas`: `dkt archive add-x add-y`, `add-x` of the docket schema, `add-y` of another → both archived; `add-x`'s evidence says `| Schema | docket |` and `| Skipped steps | none |`.
  - `TestSyncOtherSchema` and `TestRebaseOtherSchema`: exit 1, one `other-schema` finding with the Global Constraints' message and `file`, no file changed (snapshot the change folder); `dkt sync --check` the same.
  - Existing evidence tests updated for the two new rows.
- [ ] **Step 2: Run** `go test ./internal/archive ./internal/cli -run 'OtherSchema|BothSchemas'` — expect FAIL.
- [ ] **Step 3: Implement.** `loadChange` reads the schema with the metadata (an error is unreadable); for another schema it skips the base requirement, the base-is-commit check and `plan.md`. `archiveRun.change` skips `computeSync`/plan findings and `checkBase` for it and fills `Skipped`. `Sync` and `Rebase` return the `other-schema` finding right after loading.
- [ ] **Step 4: Run** `go vet ./... && go test ./...` — expect PASS. Commit (`Task: 6`).

### Task 7: The doctor's adoption steps and the conversion guide

**Scenarios:** setup-inspection#ignore-rule-changed-with-consent, setup-inspection#titles-written-with-consent, setup-inspection#unstarted-change-converted, setup-inspection#started-change-left

**Review:** judgment

**Files:**
- Modify: `.agents/skills/docket-doctor/SKILL.md`, `.claude/skills/docket-doctor/SKILL.md`
- Create: `.agents/skills/docket-doctor/convert.md`, `.claude/skills/docket-doctor/convert.md`
- Modify: `.agents/skills/docket/SKILL.md`, `.claude/skills/docket/SKILL.md`

- [ ] **Step 1: Doctor `SKILL.md`** (both copies identical). Step 1 lists the new findings. Step 4 gets three entries, in the doctor's style:
  - `install-ignored`: group the findings by rule; before proposing, list the files the fix would let git see, including files under that directory ignored today (`git ls-files --others --ignored --exclude-standard -- <dir>`); propose the narrowest rule (for `.claude/`: `.claude/*` and `!.claude/skills/`); say whether the ignore file is shared (`.gitignore`) or only on this machine (`.git/info/exclude`, `core.excludesFile`); edit only after the user agrees. For `openspec/` kept out of git, point to the README's adoption steps.
  - `title-missing`: read each spec's Purpose, draft a title (the title form `dkt lint` checks), list all of them, write every `docket.yaml` only after the user agrees, then run `dkt lint`.
  - `other-schema`: with 0 steps ticked, propose converting it by `convert.md` in this skill's folder; otherwise propose no conversion, say to finish it with its own files and archive it with `dkt archive`, and list the capabilities it adds that have no title beside their delta (offer to draft them as for `title-missing`).
- [ ] **Step 2: `convert.md`** (both copies identical), steps in order: work in the change's own worktree; set `schema: docket` in `.openspec.yaml` first; write `docket.yaml` with `title`, `kind` and `source` confirmed with the user and `base` the merge base of HEAD and the default branch; run `openspec instructions proposal --change <slug> --json` and add the sections the proposal lacks; give each capability it adds a `docket.yaml` title beside its delta; write `plan.md` from `tasks.md` with superpowers:writing-plans following `openspec instructions plan --change <slug>`, then delete `tasks.md`; run `dkt lint <slug>` until it reports nothing; the change then continues at its plan stage, where the workflow's plan review comes next. Nothing is committed without the user.
- [ ] **Step 3: `docket` skill** (both copies identical): one rule under Full review — a change whose `.openspec.yaml` names another schema follows that schema's files, gets no `plan.md` unless converted (see `docket-doctor`), and is still archived with `dkt archive`.
- [ ] **Step 4: Verify.** `diff -r .agents/skills/docket-doctor .claude/skills/docket-doctor` and the same for `docket` print nothing; `go test .` passes (`TestInstalledSkillMatchesShipped`, frontmatter tests); each edited file reads through as an agent would follow it. Commit (`Task: 7`).

### Task 8: README and architecture notes

**Scenarios:** none

**Review:** checks

**Files:**
- Modify: `README.md`, `README.zh-TW.md`, `docs/ARCHITECTURE.md`

- [ ] **Step 1: README (both languages):** a section on adopting Docket in an existing project: run `/docket-init` (or `dkt init` then `docket-doctor`); fix ignored install files; give main specs titles; changes of another schema are left to their own files and archived with `dkt archive`, and one not started can be converted; OpenSpec content kept out of git is the project's to commit, rewrite or drop before relying on it.
- [ ] **Step 2: ARCHITECTURE.md:** the capability table adds `change.ReadSchema`, `change.Checkboxes`, `gitx.Git.Ignored`, `lint.SpecTitles`, `lint.OtherSchemas`, `status.SummarizeChecklist`.
- [ ] **Step 3: Verify** each file reads through and the zh-TW section says what the English one does. Commit (`Task: 8`).
