# Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship a `docket/workflow.md` template that `dkt init` writes when missing and never replaces, check its template version and section headings in `dkt init` and `dkt inspect`, and move the skill's customizable steps into it.

**Architecture:** The template is a new shipped file. `internal/setup.Decide` gains a `workflow` kind whose decision is create-when-missing, otherwise keep, with a `WorkflowState` carrying the file's version, the shipped version and the missing headings; `internal/cli` turns that state into findings, as it does for the config kind. Two root-package tests pin each template version to its content and check the skill's section references against the template's headings.

**Tech Stack:** Go 1.24 standard library; the repository's `internal/change`, `internal/markdown`, `internal/setup`, `internal/testutil`.

**Spec:** `openspec/changes/add-project-workflow/proposal.md`, `design.md`, `specs/workflow-file/spec.md`, `specs/project-setup/spec.md`, `specs/setup-inspection/spec.md`.

## Global Constraints

- Everything committed is in English, with no host names, personal paths or names.
- Check with `go vet ./...` and `go test ./...`; both must pass at the end of every task.
- No new dependencies.
- Exact names: path `docket/workflow.md`; shipped file `shipped/workflow/workflow.md`; kind `workflow`; rules `workflow-missing`, `workflow-outdated`, `workflow-template-unknown`, `workflow-section-missing`.
- The required sections are the level-3 headings of the shipped template; a file may have them at any heading level, compared by text after `markdown.Headings` trims it.
- `dkt init` never writes an existing `docket/workflow.md`, with or without `--force`.
- Each commit carries `Change: add-project-workflow/<N>` in its last paragraph.
- The prompt texts in Task 1 and Task 4 are copied byte for byte from the approved drafts; do not reword them.

## Review Focus

- A workflow file saved with CRLF line endings or a byte-order mark must still read its version and headings: Task 2 tests a CRLF copy and a BOM-prefixed copy of the shipped template as `same`.
- Sections the project adds must not be reported: Task 2 tests an extra `### Our own step` as `same`.
- A quoted version, `template: "1"`, is the same version: Task 2 tests it as `same`.
- `docket` existing as a regular file must stop `dkt init` cleanly, not panic or write: Task 2 tests that `Decide` reports a problem for `docket/workflow.md`.
- The skill-to-template check must not pass by matching nothing: Task 4's test fails when it finds no reference.

## Reuse

| Need | Existing (path:symbol) | Decision | Reason | Evidence |
|---|---|---|---|---|
| Per-target decision for init and inspect | internal/setup/setup.go:decide | extend | Add a workflow branch beside the config branch | `rg -n "func decide\|KindConfig" internal/setup/` |
| Extra state on a decision | internal/setup/config.go:ConfigState | extend | Same pattern: `Decision.Workflow *WorkflowState` beside `Decision.Config` | `rg -n "Config \*ConfigState" internal/setup/` |
| Shipped target list | internal/setup/setup.go:targetsOf | extend | Append the workflow target | `rg -n "func targetsOf" internal/setup/` |
| Read the `template` value | internal/change/frontmatter.go:ReadFrontMatter | reuse | Handles quoting, duplicates, missing closing line | `rg -n "func ReadFrontMatter" internal/` |
| Headings outside fences | internal/markdown/markdown.go:Headings, FenceMask, Lines | reuse | Already strips BOM and CRLF and skips fences | `rg -n "func Headings\|func FenceMask\|func Lines" internal/markdown/` |
| Create the file without following links | internal/setup/setup.go:Write | reuse | Create action already goes through safefile.Install | `rg -n "func Write\|safefile.Install" internal/setup/` |
| Inspect findings per decision | internal/cli/cmd_inspect.go:decisionFindings | extend | Add a workflow case | `rg -n "func decisionFindings" internal/cli/` |
| Init findings per decision | internal/cli/cmd_init.go:initFindings | extend | Add a workflow case | `rg -n "func initFindings" internal/cli/` |
| Doctor next step on findings | internal/cli/cmd_init.go:initFinish | reuse | Already adds doctor.md when any finding exists | `rg -n "doctor.md" internal/cli/` |
| Shipped file list test | shipped_test.go:TestShippedFileSet, TestShippedStarts | extend | Add the template | `rg -n "func TestShipped" shipped_test.go` |
| Test repositories and fake OpenSpec | internal/testutil:NewRepo, FakeOpenSpec; internal/cli/cmd_init_test.go:doInit, status, hasFinding | reuse | Existing init and inspect tests use them | `rg -n "func doInit\|func status\|func hasFinding" internal/cli/` |
| Template hash | external: crypto/sha256 | reuse | Standard library | `rg -n "crypto/sha256" internal/` |

---

## Revision 1 (2026-11-06): HTML comments, docket-doctor, two end-to-end tests

Asked for by the pre-merge review in this session (the attacker on a different model, and the fresh spec check): a heading inside an HTML comment counted as a section; the skill sends agents to docket-doctor, which this repository does not have yet; `workflow-template-unknown` had no end-to-end test for `template: one` in `dkt inspect` or for `dkt init`. Task 6 compensates Task 2 for the first.

### Task 1: Ship the workflow template and pin its version

**Scenarios:** workflow-file#shipped-template-versioned

**Review:** checks

**Files:**
- Create: `shipped/workflow/workflow.md`
- Modify: `shipped_test.go`

**Interfaces:**
- Produces: the embedded file `workflow/workflow.md` in the `shipped/` tree, starting with `---\ntemplate: 1\n---\n`.

- [x] **Step 1: Write the failing tests** in `shipped_test.go`:
  - add `"shipped/workflow/workflow.md"` to the `want` list of `TestShippedFileSet`, keeping it sorted;
  - add `"shipped/workflow/workflow.md": "---\ntemplate: "` to `TestShippedStarts`;
  - add `var workflowTemplates = map[int]string{}` (version to lowercase hex SHA-256 of the template as shipped at that version) and `TestWorkflowTemplatePinned`: read the template, take the version from the front matter with `(?m)\A---\n(?:.*\n)*?template: (\d+)\n`, require it to be a positive integer equal to the largest key of `workflowTemplates`, and require the file's SHA-256 to equal `workflowTemplates[version]`; on a mismatch fail with "shipped/workflow/workflow.md changed: raise its template version and add the new hash to workflowTemplates".
- [x] **Step 2: Run** `go test . -run 'TestShipped|TestWorkflowTemplate'`; expect FAIL: the file does not exist.
- [x] **Step 3: Create `shipped/workflow/workflow.md`** as a byte-for-byte copy of the approved draft, the file `docs/pages/prompts/tpl-workflow.en.md` in the main worktree (the lead gives its absolute path), and put its SHA-256 in `workflowTemplates` under key 1.
- [x] **Step 4: Run** `go test .` and `go vet ./...`; expect PASS.
- [x] **Step 5: Commit** `shipped/workflow/workflow.md` and `shipped_test.go`.

### Task 2: dkt init writes the workflow file when missing and keeps it otherwise

**Scenarios:** workflow-file#fresh-repository-gets-the-workflow-file, workflow-file#edited-workflow-kept, workflow-file#version-that-is-not-a-number, workflow-file#section-at-another-heading-level, workflow-file#heading-only-inside-a-code-block, workflow-file#lower-template-version, project-setup#fresh-repository-gets-the-files, project-setup#re-run-with-nothing-changed, project-setup#file-from-an-earlier-release, project-setup#edited-skill-kept, project-setup#edited-skill-forced, project-setup#files-listed, project-setup#nothing-to-do, setup-inspection#statuses-listed

**Review:** judgment

**Files:**
- Create: `internal/setup/workflow.go`
- Create: `internal/setup/workflow_test.go`
- Modify: `internal/setup/setup.go`
- Modify: `internal/setup/setup_test.go`
- Modify: `internal/cli/cmd_init_test.go`

**Interfaces:**
- Consumes: the shipped `workflow/workflow.md` from Task 1.
- Produces, in package `setup`:
  - `const KindWorkflow Kind = "workflow"`
  - `type WorkflowState struct { Version int; Shipped int; Missing []string }`: `Version` is the file's template version, 0 when it has none; `Shipped` is the shipped template's; `Missing` lists, in template order, the template's level-3 headings the file lacks, and is nil unless `Version == Shipped`.
  - `Decision.Workflow *WorkflowState`, set for the workflow kind when the file exists, nil otherwise.
  - Decision for `docket/workflow.md`, the last target: missing → `StatusMissing`, `ActionCreate`; otherwise `ActionKeep` whatever `force` is, with `StatusOutdated` when `0 < Version < Shipped`, `StatusDiffers` when `Version == 0`, `Version > Shipped` or `Missing` is not empty, and `StatusSame` otherwise.
  - `Decide` returns an error when the shipped template has no template version.

- [x] **Step 1: Write the failing tests** in `internal/setup/workflow_test.go`, using `shipped(t)`, `put`, `get`, `find` and `decideT` from `setup_test.go`. Build each file from the shipped template's text; for a different shipped version, wrap `shipped(t)` in an `fstest.MapFS` copy whose `workflow/workflow.md` has `template: 2`.
  - `TestWorkflowCreatedWhenMissing`: empty root → `Kind == KindWorkflow`, `StatusMissing`, `ActionCreate`; after `Write`, the file equals the shipped bytes and `docket/` was created.
  - `TestWorkflowKeptWithForce`: the template with every section's text replaced, `force` true → `StatusSame`, `ActionKeep`, `Workflow == &WorkflowState{1, 1, nil}`; after `Write`, the bytes are unchanged.
  - `TestWorkflowVersionNotANumber` (`template: one`), `TestWorkflowNoFrontMatter` (no front matter, no headings), `TestWorkflowUnclosedFrontMatter` (`---\ntemplate: 1\n` and no closing line) → `Version == 0`, `StatusDiffers`, `Missing == nil`.
  - `TestWorkflowQuotedVersion` (`template: "1"`), `TestWorkflowCRLF` (the template with `\r\n` line endings), `TestWorkflowBOM` (the template with a leading UTF-8 byte-order mark), `TestWorkflowExtraSection` (the template plus `### Our own step`), `TestWorkflowHeadingAnyLevel` (`### Plan review` changed to `## Plan review`) → `StatusSame`.
  - `TestWorkflowSectionMissing` (the `### Plan review` line removed) → `StatusDiffers`, `Missing == []string{"Plan review"}`; `TestWorkflowHeadingInFence` (the line moved inside a ```` ``` ```` fence) → the same.
  - `TestWorkflowHigherVersion` (`template: 2`, shipped 1) → `StatusDiffers`, `Missing == nil`; `TestWorkflowLowerVersion` (file 1, shipped 2) → `StatusOutdated`, `Missing == nil`.
  - `TestWorkflowDocketIsAFile`: `docket` is a regular file → `Decide` returns no decision for `docket/workflow.md` and one problem naming it, no panic.
  - `TestWorkflowShippedWithoutVersion`: a shipped copy whose template has no front matter → `Decide` returns an error.
- [x] **Step 2: Run** `go test ./internal/setup/ -run Workflow`; expect FAIL to compile (no `KindWorkflow`).
- [x] **Step 3: Implement** `targetsOf` appending `{rel: "docket/workflow.md", kind: KindWorkflow}` with the shipped bytes, and `decideWorkflow(d Decision, got, want []byte, shipped int) Decision` in `internal/setup/workflow.go`, called from `decide` after the read, the way `decideConfig` is. Read versions with `change.ReadFrontMatter` and `strconv.Atoi` on `Fields["template"].Value` (positive values only); read headings with `markdown.Headings(lines, markdown.FenceMask(lines))` over `markdown.Lines`. Parse the shipped version once in `targetsOf`.
- [x] **Step 4: Update** the tests in `internal/setup/setup_test.go` that list every target: `TestDecideEmptyRootIsAllMissingInOrder` ends with `docket/workflow.md`, and `TestDecideSameAfterWrite` expects the workflow decision to be `StatusSame` with `ActionKeep` while every other target is `ActionNone`. Run `go test ./internal/setup/`; expect PASS. Then `go test ./...`: update the tests in `internal/cli/cmd_init_test.go` that the new target changes, to what the delta specs say: `TestInitFresh` also finds `docket/workflow.md` with kind `workflow`, status `created` and the shipped bytes; `TestInitRerunSame` expects every file `same` except `docket/workflow.md`, `kept`, with no finding and exit 0. Add `TestInitWorkflowKeptWithForce`: a customized current file and `dkt init --force` → unchanged bytes, `kept`, no finding naming it, exit 0. The existing tests for an earlier release's file and for an edited skill, kept and forced, must pass unchanged: the rules for existing files still apply to every target but `openspec/config.yaml` and `docket/workflow.md`.
- [x] **Step 5: Run** `go vet ./...` and `go test ./...`; expect PASS.
- [x] **Step 6: Commit** the five files.

### Task 3: Report the workflow file's state in dkt init and dkt inspect

**Scenarios:** workflow-file#customized-but-current, workflow-file#missing-before-init, workflow-file#outdated-workflow-reported-by-init, workflow-file#higher-template-version, workflow-file#section-removed, workflow-file#sections-not-checked-without-a-version

**Review:** judgment

**Files:**
- Modify: `internal/cli/cmd_inspect.go`
- Modify: `internal/cli/cmd_init.go`
- Modify: `internal/cli/setup.go`
- Modify: `internal/cli/cmd_inspect_test.go`
- Modify: `internal/cli/cmd_init_test.go`

**Interfaces:**
- Consumes: `setup.KindWorkflow`, `setup.WorkflowState`, `Decision.Workflow` from Task 2.
- Produces: `shippedFS` in `internal/cli/setup.go` becomes a package variable, `var shippedFS = func() fs.FS { ... }`, with the comment that tests may replace it and must restore it, as `setup.Earlier` says; its callers are unchanged.
- Produces: `workflowFindings(d setup.Decision) []Finding` in `cmd_inspect.go`, used by both commands. Each finding has `File: "docket/workflow.md"` and these messages:
  - `workflow-outdated`: `docket/workflow.md follows template <V>; this release ships template <S>: run docket-doctor to bring in the changes`
  - `workflow-template-unknown`: `docket/workflow.md has no template version this release knows; it ships template <S>`
  - `workflow-section-missing`, one per heading: `docket/workflow.md has no "<heading>" section; the template has it`
  - `workflow-missing` (inspect only): `docket/workflow.md is missing; dkt init writes it`

- [x] **Step 1: Write the failing tests.**
  - `cmd_inspect_test.go`: `TestInspectWorkflowMissing` (before init: `workflow-missing` names the file, status `missing`, exit 1); `TestInspectWorkflowCustomized` (current version, every section reworded: `same`, no finding names it); `TestInspectWorkflowHigherVersion` (`template: 99`: `differs`, `workflow-template-unknown`); `TestInspectWorkflowNoFrontMatter` (no front matter or headings: `workflow-template-unknown`, no `workflow-section-missing`).
  - `TestWorkflowFindings`, a table test calling `workflowFindings` with constructed decisions: `{Status: outdated, Workflow: {1, 2, nil}}` → one `workflow-outdated` whose message holds `template 1` and `template 2`; `{differs, {0, 1, nil}}` → one `workflow-template-unknown`; `{differs, {1, 1, ["Plan review", "Reviewers"]}}` → two `workflow-section-missing`, in that order; `{same, {1, 1, nil}}` → none.
  - `cmd_init_test.go`: `TestInitWorkflowSectionMissing` (the shipped template without `### Plan review`, then `dkt init --json`: unchanged and `kept`, exactly one `workflow-section-missing` naming the file with "Plan review" in its message, exit 1, last `next_steps` entry equal to `shipped/next-steps/doctor.md` trimmed); `TestInitWorkflowOutdated`, end to end: set the repository up with `dkt init`, then replace `shippedFS` (restored in `t.Cleanup`) with an `fstest.MapFS` copy of the shipped tree whose `workflow/workflow.md` says `template: 2`, and run `dkt init --json`: `docket/workflow.md` is unchanged and `kept`, `workflow-outdated` names it with `template 1` and `template 2` in its message, exit 1, and the last `next_steps` entry is the doctor step. `TestInspectWorkflowOutdated`, the same setup with `dkt inspect --json`: the file is `outdated`.
- [x] **Step 2: Run** `go test ./internal/cli/ -run Workflow`; expect FAIL.
- [x] **Step 3: Implement** a `setup.KindWorkflow` case in `decisionFindings` (`workflow-missing` when `StatusMissing`, else `workflowFindings`) and in `initFindings` (`workflowFindings`). `result.files` already carries `string(d.Status)` and the init status from `actionStatus`.
- [x] **Step 4: Run** `go vet ./...` and `go test ./...`; expect PASS.
- [x] **Step 5: Commit** the five files.

### Task 4: Replace the docket skill and docket-init with the approved texts

**Scenarios:** none

**Review:** checks

**Files:**
- Modify: `shipped/skill/SKILL.md`
- Modify: `skills/docket-init/SKILL.md`
- Modify: `shipped_test.go`

- [x] **Step 1: Write the failing test** `TestSkillSectionsInTemplate` in `shipped_test.go`: collapse each run of whitespace in `shipped/skill/SKILL.md` to one space, collect every name matched by `"([^"]+)"(?: and "([^"]+)")? in ` + "`docket/workflow.md`" (both groups), and require at least one name and each name to be a `### ` heading of `shipped/workflow/workflow.md`.
- [x] **Step 2: Run** `go test . -run TestSkillSectionsInTemplate`; expect FAIL: no reference found.
- [x] **Step 3: Replace** `shipped/skill/SKILL.md` with `docs/pages/prompts/docket-skill.en.md` and `skills/docket-init/SKILL.md` with `docs/pages/prompts/docket-init.en.md`, both byte for byte from the main worktree (the lead gives the absolute paths).
- [x] **Step 4: Run** `go test ./...` and `go vet ./...`; expect PASS. Check mechanically that each file is byte for byte its draft (`cmp`), that both still start with their front matter (`name: docket`, `name: docket-init`), and that `shipped/skill/SKILL.md` no longer contains "Default steps" or "(required)". docket-init's behavior is prompt text and gets no test of its own.
- [x] **Step 5: Commit** the three files.

### Task 5: Use docket/workflow.md in this repository

**Scenarios:** none

**Review:** checks

**Files:**
- Create: `docket/workflow.md`
- Modify: `.claude/skills/docket/SKILL.md`
- Modify: `.agents/skills/docket/SKILL.md`
- Modify: `AGENTS.md`

- [x] **Step 1: Run** `go run ./cmd/dkt init --force --json` at the worktree root. Expect `docket/workflow.md` `created`, both `SKILL.md` copies `overwritten`, every other file `same`, and no finding.
- [x] **Step 2: Edit** `docket/workflow.md`: under `### When to archive`, replace the text with "Until this repository has a release process, archive each change in its own pull request, as its last commit." Leave every heading and the front matter as they are.
- [x] **Step 3: Edit** `AGENTS.md`: remove the bullet starting "Review, verify and merge by the docket skill's "Default steps"" and the bullet starting "Until the release process exists, archive each change"; add `docket/workflow.md` to the list under "Prompt files", after `shipped/`.
- [x] **Step 4: Run** `go run ./cmd/dkt inspect --json`; expect `docket/workflow.md` listed as `same` and no `workflow-` finding. Run `go test ./...`.
- [x] **Step 5: Commit** the four files.

### Task 6: Fixes from the pre-merge review

**Compensates:** Task 2

**Scenarios:** workflow-file#heading-only-inside-an-html-comment, workflow-file#version-that-is-not-a-number

**Review:** checks

**Files:**
- Modify: `internal/setup/workflow.go`
- Modify: `internal/setup/workflow_test.go`
- Modify: `internal/cli/cmd_inspect_test.go`
- Modify: `internal/cli/cmd_init_test.go`
- Modify: `AGENTS.md`

- [x] **Step 1: Write the tests.** `internal/setup/workflow_test.go`: `TestWorkflowHeadingInComment`, the template with the `### Plan review` line moved inside `<!--` and `-->` lines → `StatusDiffers`, `Missing == []string{"Plan review"}`. `internal/cli/cmd_inspect_test.go`: `TestInspectWorkflowNotANumber`, after `dkt init`, the file with `template: one` → `differs` and `workflow-template-unknown` naming it. `internal/cli/cmd_init_test.go`: `TestInitWorkflowTemplateUnknown`, after `dkt init`, the file with `template: 99`, then `dkt init --json` → unchanged and `kept`, `workflow-template-unknown` naming it, exit 1.
- [x] **Step 2: Run** `go test ./internal/setup/ -run TestWorkflowHeadingInComment`; expect FAIL (`Missing` is nil). The two cli tests cover code that already exists and pass from the start.
- [x] **Step 3: Implement** in `headingTexts`: blank HTML comments with `markdown.BlankComments(lines, mask)` before `markdown.Headings`, `mask` being `markdown.FenceMask` of the original lines.
- [x] **Step 4: Edit** `AGENTS.md`: the bullet on what does not exist yet also names docket-doctor, with a sub-bullet: when the docket skill, docket-init or `dkt init` says to run docket-doctor, tell the user what was reported instead, and copy a missing section of `docket/workflow.md` back from `shipped/workflow/workflow.md`.
- [x] **Step 5: Run** `go vet ./...` and `go test ./...`; expect PASS. Commit the five files.
