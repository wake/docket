# Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `dkt` ships a workflow template and reports the state of each user's materialized workflow (`dkt workflow`); the `docket` skill keeps the framework rules, takes each path's review and verification steps from that file, and runs setup when it is missing.

**Architecture:** The root package embeds `workflow/template.md` as `docket.Template()`. A new package `internal/workflow` locates the file from the environment it is given, reads it, and inspects its bytes (frontmatter hash, four headings) against the template's hash; `internal/cli/cmd_workflow.go` maps that to the envelope. The skill text splits in two: `SKILL.md` keeps triage and framework steps, the template takes the review and verification steps, and a second skill file `setup.md` holds the setup instructions.

**Tech Stack:** Go 1.26, `go.yaml.in/yaml/v3` (frontmatter), `crypto/sha256`, `os/user` (only in `cmd/dkt`).

**Spec:** `openspec/changes/add-workflow/proposal.md`, `openspec/changes/add-workflow/specs/materialized-workflow/spec.md`, `openspec/changes/add-workflow/specs/project-setup/spec.md`, `openspec/changes/add-workflow/design.md`.

## Global Constraints

- Workflow file: `<XDG_CONFIG_HOME>/docket/workflow.md` when `XDG_CONFIG_HOME` is an absolute path, else `<home>/.config/docket/workflow.md`. The home is `HOME` when it is an absolute path, else the result of `Options.UserHome`; when neither gives an absolute path, `dkt workflow` is an environment error (exit 4). When a variable appears more than once in the environment, the last value counts.
- Hash: the first 12 lowercase hexadecimal digits of the SHA-256 of the template's bytes.
- Sections: a line, after `markdown.Lines` and with trailing spaces and tabs removed, equal to `## Full review`, `## Light review`, `## Check` or `## Direct merge`. `missing_sections` names them `Full review`, `Light review`, `Check`, `Direct merge`, in that order.
- Frontmatter: the first line is `---`, a later line is `---` (both after the line rule above), and the YAML between has a non-null scalar `template`, read as the text written (so `template: 012345678901` reads `012345678901`). Anything else means no recorded hash.
- Statuses `ok`, `missing`, `incomplete`, `stale`, decided in that priority: `missing`, then `incomplete`, then `stale`, else `ok`. Only `ok` exits 0; the others exit 1. A file that exists but cannot be read is listed in `unreadable` (exit 3), with `result` null.
- Finding rules: `workflow-missing`, `workflow-incomplete` (one finding per missing section), `workflow-stale`; each has `file` set to the workflow path.
- `dkt workflow --json` `result` has exactly `path`, `status`, `template`, `recorded` (string or null) and `missing_sections` (an array, never null). `dkt workflow template --json` `result` has exactly `template` and `text`.
- `dkt` never writes the workflow file.
- Only `cmd/dkt` reads the process environment and calls `os/user`; everything else gets them through `cli.Options`.
- The template names capabilities, never a product or a model.
- Everything in the repository is in English. Every commit for a task carries `Change: add-workflow` and `Task: <N>` in one final trailer paragraph; structural and behavior changes go in separate commits.

## Review Focus

- A hash that YAML would read as a number (`template: 012345678901`) or that is quoted is read as the text written. Test: Task 2 (`TestInspect`).
- A workflow file edited on Windows (CRLF) or with trailing spaces after a heading still has its sections; `## Checklist` or ` ## Check` does not count as `## Check`. Test: Task 2 (`TestInspect`).
- A symbolic link at the workflow path whose target is gone (a moved dotfiles repository) is unreadable, not missing, so setup does not write over the user's link. Test: Task 2 (`TestRead`, `TestWorkflowBrokenLink`).
- `HOME` empty or relative falls back to the user lookup; with no lookup, exit 4 rather than a path under the working directory. Test: Task 2 (`TestLocate`, `TestWorkflowNoHome`).
- An unclosed frontmatter or invalid YAML gives `stale` with `recorded` null, not an error. Test: Task 2 (`TestInspect`).

## Reuse

| Need | Existing (path:symbol) | Decision | Reason | Evidence |
|---|---|---|---|---|
| Embed the template | schema.go:Skill | extend | `Template()` embeds `workflow/template.md` as a string, as `agentsMD` is | `rg -n "go:embed" --type go` |
| Command table, flags, envelope, exit codes | internal/cli/run.go:commands | extend | add `workflowCommand`; `Outcome`, `Unreadable`, `ExitCode` already give exit 1 for findings and 3 for unreadable | `rg -n "var commands" internal/cli/run.go`, `rg -n "func ExitCode" -A 14 internal/cli/envelope.go` |
| Hand outside facts to a command | internal/cli/envelope.go:Options | extend | `Options.UserHome` and `Context.UserHome` keep `os/user` in `cmd/dkt`, as the clock is handed in with `Options.Now` | `rg -n "type Options" -A 8 internal/cli/envelope.go` |
| Environment error (exit 4) | internal/cli/envelope.go:EnvError | reuse | no home directory is an unusable environment | `rg -n "EnvError" internal/cli/envelope.go` |
| Read a variable from the given environment, last value winning | none | new | `openspec.envValue` does this but is unexported in the OpenSpec runner; `internal/workflow` must not depend on `internal/openspec` for eight lines | `rg -n "func envValue" internal/` |
| Split a file into lines, dropping a BOM and CRs | internal/markdown/markdown.go:Lines | reuse | the workflow reader splits with it | `rg -n "func Lines" internal/markdown` |
| Find the four section headings | none | new | the spec matches whole lines exactly; `markdown.Headings` also accepts closing hashes and extra spaces, which would read the file more loosely than the spec says | `rg -n "func Headings\|closingHashes" internal/markdown/markdown.go` |
| Parse the frontmatter | external: go.yaml.in/yaml/v3 | reuse | already a module dependency; decoding to `yaml.Node` keeps the scalar's text | `rg -n "yaml" go.mod` |
| Hash the template | external: crypto/sha256 | reuse | standard library | `rg -n "sha256" --type go` (no hit) |
| Install a new skill file in projects | internal/setup/setup.go:Init | reuse | installs every file of `docket.Skill()`, so `setup.md` ships with no installer change | `rg -n "WalkDir" internal/setup/setup.go` |
| Keep Docket's installed skill equal to the shipped one | schema_test.go:TestInstalledSkillMatchesShipped | reuse | walks every file of `Skill()`, so it covers `setup.md` | `rg -n "func TestInstalledSkillMatchesShipped" schema_test.go` |
| Test repositories and environments | internal/testutil/repo.go:NewRepo | reuse | `testutil.Env` gives a temporary `HOME` and no `XDG_CONFIG_HOME` | `rg -n "func (NewRepo\|Env)" internal/testutil/repo.go` |
| Run the skill headless in a scratch project | external: claude | reuse | as the v0.6.0 trial did, with `--setting-sources project,local` | `claude --help` |

---

### Task 1: The workflow template

**Scenarios:** materialized-workflow#template-names-capabilities

**Review:** checks

**Files:**
- Create: `workflow/template.md`
- Modify: `schema.go`
- Test: `schema_test.go`

**Interfaces:**
- Produces: `docket.Template() string`, the bytes of `workflow/template.md`.

- [ ] **Step 1: Write the failing tests**

`TestTemplateHeadings`: each of `## Full review`, `## Light review`, `## Check`, `## Direct merge` and `## Reviews` is a whole line of `Template()`, once. `TestTemplateNamesNoProduct`: `strings.ToLower(Template())` contains none of `claude`, `opus`, `sonnet`, `haiku`, `fable`, `gpt`, `codex`, `gemini`, `openai`, `anthropic`, `superpowers`, `copilot`.

- [ ] **Step 2: Run them to see them fail**

Run: `go test -run 'TestTemplate' .`
Expected: FAIL (undefined `Template`).

- [ ] **Step 3: Write `workflow/template.md`**

Exactly this text. It is today's per-path review and verification steps from `SKILL.md`, worded in capabilities; Task 3 removes them from the skill.

````markdown
# Docket workflow steps

The review and verification steps for each path the `docket` skill
triages work into. The skill holds the rest: the triage, the change
folder, the plan's format, commit trailers and merging.

## Full review

1. Review the plan once, together with the proposal, the delta specs and
   design.md, before any code (see Reviews), and fix what the review finds.
2. Execute the plan task by task, with a fresh implementer agent for each
   task, test first. Each implementer gets the Reuse rows for its task and
   reports every exported symbol it added, with why no existing one fit.
   Structural changes and behavior changes go in separate commits.
3. Only a task marked `**Review:** judgment` gets its own reviewer, in a
   fresh context. Every task that changes code gets the mechanical checks
   instead: its new tests are seen failing before the code and passing
   after, and mutating the lines it changed makes a test fail. A task that
   changes no code, such as one that only edits docs, is checked by its own
   verification steps.
4. Before merging, review the whole change once, by two reviewers in
   parallel: one on a different model, asked to attack (wrong results, edge
   cases, partial failure, the same defect elsewhere in the code), and one
   in a fresh context, checking the code against the proposal and the delta
   specs (see Reviews when no other model is available). After fixes, only
   a large fix is reviewed again.
5. Split the pull request only when it mixes unrelated concerns, when a
   structural refactor is mixed with behavior (the refactor goes first, in
   its own pull request), or when product code and tests together run far
   past about 1000 lines. Do not stack pull requests.

## Light review

1. Verify by the kind of edit:
   - a fix: write a regression test and see it fail before the fix;
   - a restructuring: the tests that cover the code pass before and after,
     and mutating the changed lines makes one fail; when none fails, first
     add the tests as a test-only edit;
   - prompt text: check it mechanically, not with tests that look for
     strings: it still loads, and every copy says the same;
   - tests: mutate the code they cover; the tests must fail;
   - check settings: run them; the pull request's own CI run uses the new
     settings, and the review looks for a check that got weaker.
2. Have the whole diff reviewed once before merging (see Reviews).

## Check

Have a fresh agent, given only the diff and the triage's definitions,
confirm before merging that every changed line is a comment or whitespace
and that no comment changes how the code builds or runs. If it does not
confirm, triage again.

## Direct merge

Pass the automated checks. No review.

## Reviews

A required review is done by a reviewer on a different model. If there is
none, by the same model in a fresh context. If that is not possible either,
the pull request says it had no independent review and asks a person to
review it. The author's own reading is not a review.
````

- [ ] **Step 4: Implement `Template()` in `schema.go`**

`//go:embed workflow/template.md` into an unexported string; `Template()` returns it. Update the package comment to name the workflow template.

- [ ] **Step 5: Run the tests to see them pass**

Run: `go test .`
Expected: PASS

- [ ] **Step 6: Commit**

### Task 2: Reading the workflow file and `dkt workflow`

**Scenarios:** materialized-workflow#workflow-in-the-config-home, materialized-workflow#workflow-under-xdg-config-home, materialized-workflow#xdg-config-home-ignored-when-relative, materialized-workflow#template-printed, materialized-workflow#no-workflow, materialized-workflow#complete-workflow, materialized-workflow#a-path-missing, materialized-workflow#made-from-an-older-template, materialized-workflow#no-frontmatter, materialized-workflow#unreadable-workflow

**Review:** judgment

**Files:**
- Create: `internal/workflow/workflow.go`, `internal/workflow/workflow_test.go`
- Create: `internal/cli/cmd_workflow.go`, `internal/cli/cmd_workflow_test.go`
- Modify: `internal/cli/run.go` (command table; `Context.UserHome` from `Options`), `internal/cli/envelope.go` (`Options.UserHome`, `Context.UserHome`), `cmd/dkt/main.go`

**Interfaces:**
- Consumes: `docket.Template()` (Task 1).
- Produces, in `internal/workflow`:
  - `Locate(env []string, userHome func() (string, error)) (string, error)`; `ErrNoHome` (an `error` var), returned wrapped or bare so that `errors.Is(err, ErrNoHome)` holds when no absolute home is found (a nil `userHome`, an error from it, or an empty or relative result).
  - `Hash(text string) string`.
  - `Read(path string) (b []byte, found bool, err error)`: a path that does not exist gives `found` false and a nil error; a symbolic link whose target does not exist, a directory, or any other read failure gives an error.
  - `type Status string` with `OK = "ok"`, `Missing = "missing"`, `Incomplete = "incomplete"`, `Stale = "stale"`; `var Sections = []string{"Full review", "Light review", "Check", "Direct merge"}`.
  - `type Report struct { Status Status; Recorded *string; MissingSections []string }`; `Inspect(b []byte, shipped string) Report` never leaves `MissingSections` nil, and its status is never `Missing` (a missing file is the caller's to report).
- Produces, in `internal/cli`: `Options.UserHome func() (string, error)` and `Context.UserHome` (nil means no lookup); `WorkflowResult{Path string "path"; Status workflow.Status "status"; Template string "template"; Recorded *string "recorded"; MissingSections []string "missing_sections"}`; `TemplateResult{Template string "template"; Text string "text"}`.

- [ ] **Step 1: Write the failing unit tests**

In `internal/workflow/workflow_test.go`, with `four = "## Full review\n## Light review\n## Check\n## Direct merge\n"` and `fm(h) = "---\ntemplate: " + h + "\n---\n"`:

`TestInspect`, shipped `abcdefabcdef`:

| case | bytes | status | recorded | missing sections |
|---|---|---|---|---|
| complete | `fm("abcdefabcdef")+four` | ok | `abcdefabcdef` | `[]` |
| quoted | `fm("\"abcdefabcdef\"")+four` | ok | `abcdefabcdef` | `[]` |
| CRLF, trailing blanks | `"---\r\ntemplate: abcdefabcdef\r\n---\r\n## Full review  \r\n## Light review\t\r\n## Check\r\n## Direct merge\r\n"` | ok | `abcdefabcdef` | `[]` |
| BOM | `"﻿"+fm("abcdefabcdef")+four` | ok | `abcdefabcdef` | `[]` |
| digits | `fm("012345678901")+four` | stale | `012345678901` | `[]` |
| older hash | `fm("0123456789ab")+four` | stale | `0123456789ab` | `[]` |
| older hash, no Check | `fm("0123456789ab")` + four without `## Check` | incomplete | `0123456789ab` | `["Check"]` |
| no headings | `fm("abcdefabcdef")+"# x\n"` | incomplete | `abcdefabcdef` | all four, in order |
| near misses | `fm("abcdefabcdef")` + four with `## Check` replaced by `## Checklist\n ## Check` | incomplete | `abcdefabcdef` | `["Check"]` |
| no frontmatter | `four` | stale | nil | `[]` |
| unclosed | `"---\ntemplate: abcdefabcdef\n"+four` | stale | nil | `[]` |
| invalid YAML | `fm("[")+four` | stale | nil | `[]` |
| null | `fm("~")+four` | stale | nil | `[]` |
| no key | `"---\nother: x\n---\n"+four` | stale | nil | `[]` |
| empty | `""` | incomplete | nil | all four |

`TestLocate`: `HOME=/h` → `/h/.config/docket/workflow.md`; plus `XDG_CONFIG_HOME=/x` → `/x/docket/workflow.md`; `XDG_CONFIG_HOME=/x/` → `/x/docket/workflow.md`; `XDG_CONFIG_HOME=rel` and `XDG_CONFIG_HOME=` → `/h/.config/docket/workflow.md`; `HOME=/a` then `HOME=/b` → `/b/.config/docket/workflow.md`; `HOME=rel` with a lookup returning `/u` → `/u/.config/docket/workflow.md`; no `HOME` with a lookup returning `/u` → `/u/...`; no `HOME` and a nil lookup, a lookup returning an error, `""` or `rel` → `errors.Is(err, ErrNoHome)`. (Build expected paths with `filepath.Join`.)

`TestHash`: `Hash("") == "e3b0c44298fc"`; `len(Hash("x")) == 12`.

`TestRead`: a missing path → `found` false, nil error; a file → its bytes, `found` true; a link to a file → the file's bytes; a link to a missing file → non-nil error; a directory → non-nil error.

- [ ] **Step 2: Run them to see them fail**

Run: `go test ./internal/workflow`
Expected: FAIL (package does not build).

- [ ] **Step 3: Implement `internal/workflow/workflow.go`**

Package comment: locates a user's workflow file, reads it, and reports its state against the shipped template; it never writes the file. `Locate` reads only the `env` it is given (an unexported last-value lookup). `Read` uses `os.ReadFile`; on `fs.ErrNotExist` it calls `os.Lstat`, and an existing entry (a dangling link) is an error saying the link's target does not exist. `Inspect` splits with `markdown.Lines`, reads the frontmatter by the rule in Global Constraints (decode into `struct{ Template yaml.Node }`, take `Value` of a scalar whose tag is not `!!null`), checks `Sections` by the line rule, and orders the status as in Global Constraints.

- [ ] **Step 4: Run them to see them pass**

Run: `go test ./internal/workflow`
Expected: PASS

- [ ] **Step 5: Commit**

- [ ] **Step 6: Write the failing command tests**

In `internal/cli/cmd_workflow_test.go`, a helper runs `Run` with `Options{Env: env, UserHome: home, ...}` in a `testutil.NewRepo` directory, where `env` is `testutil.Env(t)` with extra variables appended. `hash := hex of sha256(docket.Template())[:12]`, computed in the test with `crypto/sha256`. Each test reads `result` from `--json` unless it says text.

- `TestWorkflowPathInConfigHome`: `HOME=<temp h>`: `result.path == h/.config/docket/workflow.md`.
- `TestWorkflowPathUnderXDG`: `XDG_CONFIG_HOME=<temp x>`: `result.path == x/docket/workflow.md`.
- `TestWorkflowXDGRelativeIgnored`: `XDG_CONFIG_HOME=rel`, `HOME=<temp h>`: `result.path == h/.config/docket/workflow.md`.
- `TestWorkflowTemplatePrinted`: `workflow template --json`: exit 0, `result.text == docket.Template()`, `result.template == hash`, `result` has exactly two keys. Text mode: exit 0, stdout starts with `docket.Template()`.
- `TestWorkflowMissing`: no file: exit 1, `status` `missing`, `recorded` null, `missing_sections` `[]`, `template == hash`, findings exactly one `workflow-missing` with `file == result.path`; afterwards the config directory still does not exist.
- `TestWorkflowComplete`: file `fm(hash)+four`: exit 0, `status` `ok`, `recorded == hash`, findings `[]`. Text mode: stdout ends with `dkt workflow: ok\n`.
- `TestWorkflowSectionMissing`: `fm("0123456789ab")` + four without `## Check`: exit 1, `status` `incomplete`, `missing_sections == ["Check"]`, findings exactly one `workflow-incomplete`.
- `TestWorkflowStale`: `fm("0123456789ab")+four`: exit 1, `status` `stale`, `recorded == "0123456789ab"`, `template == hash`, findings exactly one `workflow-stale`.
- `TestWorkflowNoFrontmatter`: `four`: `status` `stale`, `recorded` null.
- `TestWorkflowUnreadable`, skipped as root: mode 000: exit 3, `unreadable` has one entry with `source == path`, `result` null. Subtest: a directory at the path, exit 3 (not skipped as root).
- `TestWorkflowBrokenLink`: the path is a link to a missing file: exit 3, the path in `unreadable`.
- `TestWorkflowThroughLink`: the path is a link to `fm(hash)+four` elsewhere: exit 0, `status` `ok`.
- `TestWorkflowNoHome`: `env` without `HOME` and `XDG_CONFIG_HOME`, `UserHome` nil: exit 4, one finding with rule `environment`. With `UserHome` returning a temp `u`: `result.path == u/.config/docket/workflow.md`.
- `TestWorkflowUsage`: `workflow other` and `workflow template extra`: exit 2.

- [ ] **Step 7: Run them to see them fail**

Run: `go test ./internal/cli -run TestWorkflow`
Expected: FAIL (unknown command `workflow`).

- [ ] **Step 8: Implement the command**

`envelope.go`: `Options.UserHome func() (string, error)` (comment: the operating system's lookup of the user's home directory; nil means none) and `Context.UserHome`; `run.go` copies it into the context and adds `workflowCommand` to `commands`. `cmd/dkt/main.go` sets `UserHome` to a function returning `user.Current()`'s `HomeDir`.

`cmd_workflow.go`: `Name: "workflow"`, `Usage: "workflow [template]"`. No argument: report the file; `template` alone: print the template; anything else: `UsageError{Msg: "workflow takes no argument except template"}`.

- Report: `Locate(c.Env, c.UserHome)`; `ErrNoHome` becomes `EnvError` with its message. `Read`; an error gives `Outcome{Unreadable: [{Source: path, Reason: err.Error()}]}` and nothing else. Not found gives status `missing`; otherwise `Inspect(b, Hash(docket.Template()))`. `MissingSections` is `[]string{}` unless incomplete. Human line: `fmt.Sprintf("%-10s %s", status, path)`. Findings, each with `File: path`:
  - `workflow-missing`: `no workflow file; the docket skill runs setup to write it`
  - `workflow-incomplete`, one per missing section: `no "## <name>" section; the docket skill runs setup to add it`
  - `workflow-stale`: `made from template <recorded>; the shipped template is <hash>`, or with no recorded hash, `no template hash recorded; the shipped template is <hash>`
- Template: `result` is `TemplateResult{Template: Hash(text), Text: text}`; the human lines are the template's lines (the final newline dropped before splitting).

- [ ] **Step 9: Run all tests**

Run: `go vet ./... && go test ./...`
Expected: PASS

- [ ] **Step 10: Commit**

### Task 3: The skill split, setup and the block

**Scenarios:** project-setup#steps-from-the-workflow, project-setup#setup-before-the-first-work, project-setup#stale-workflow-kept, project-setup#workflow-wins-over-user-instructions

**Review:** checks

**Files:**
- Modify: `.agents/skills/docket/SKILL.md`, `.claude/skills/docket/SKILL.md`
- Create: `.agents/skills/docket/setup.md`, `.claude/skills/docket/setup.md`
- Modify: `AGENTS.md` (the Docket block only)

**Interfaces:**
- Consumes: `dkt workflow` and `dkt workflow template` (Task 2), the template's headings (Task 1).

- [ ] **Step 1: Rewrite `.agents/skills/docket/SKILL.md`**

Keep the frontmatter, `## Triage`, `## Parallel sessions` and `## Prompt paths` byte for byte. Replace the rest as follows.

The opening paragraph gains a last sentence: `Each path's review and verification steps are not here: they come from the user's workflow file.`

After `## Triage`, insert:

```markdown
## The workflow file

Right after triage, before any other step of the work, run
`dkt workflow --json`. It reports the user's workflow file
(`result.path`) and its status (`result.status`):

- `ok`: on each path below, do the steps under that path's heading in the
  file, and under `## Reviews` where they point to it. On these steps the
  file wins over the user's other instructions.
- `missing` or `incomplete`: read `setup.md` in this skill's directory and
  run setup first; then start the work.
- `stale`: keep following the file. Tell the user once in this session
  that it was made from an older template, what the current template
  (`dkt workflow template`) does differently, and that setup can update it.
- anything else, such as the file being unreadable: tell the user what
  `dkt workflow` reported and ask how to go on. Do not write the file.
```

`## Full review` keeps steps 1 to 4, the `/opsx:*` sentence and the mid-way paragraph as they are; steps 5 to 11 become:

```markdown
5. Review the plan, execute it and review the change before merging by
   the steps under `## Full review` in the workflow file. Every commit made
   for a task carries the trailers `Change: <slug>` and `Task: <N>`.
6. Before opening the pull request, run `dkt sync <slug>` and
   `dkt lint <slug>`, and commit plan.md.
7. A change is one pull request unless the workflow file says to split
   it; its commits show its layers. Pull requests merge with a merge
   commit, never squash: `dkt sync` reads the trailers of each commit.
8. Archive only with `dkt archive` on a branch made from the default
   branch, in the release step. Never run `openspec archive` directly.
```

`## Light review`, `## Check` and `## Direct merge` become:

```markdown
## Light review

No change is opened. In a git worktree, make the edit following the steps
under `## Light review` in the workflow file; then open the pull request
and merge it with a merge commit.

## Check

For an edit to a code or test file that changes only comments or
whitespace: do the steps under `## Check` in the workflow file, and when
they pass, go on as a direct merge.

## Direct merge

In a git worktree: make the edit, do the steps under `## Direct merge` in
the workflow file, open the pull request and merge it with a merge commit.
```

`## Reviews` is removed (it is in the template).

- [ ] **Step 2: Check that every rule of the old skill has a home**

Compare the old `SKILL.md` (`git show HEAD:.agents/skills/docket/SKILL.md`) with the new one and `workflow/template.md`, rule by rule: old step 5 → template Full review 1; old step 6 → template Full review 2 and 3, trailers → skill step 5; old step 7 → template Full review 4; old step 9 → skill step 7 and template Full review 5; old steps 8, 10, 11 → skill steps 6, 7, 8; old Light review 1 and 2 → template Light review; old Check → template Check; old Direct merge's checks and "No review" → template Direct merge; old Reviews → template Reviews. Expected: no rule dropped, none in both places except the path names.

- [ ] **Step 3: Write `.agents/skills/docket/setup.md`**

Exactly this text:

````markdown
# Workflow setup

Run setup when `dkt workflow` reports `missing` or `incomplete` at the start
of development work, before that work, or when the user asks to update a
`stale` workflow. It writes the user's workflow file: Docket's template,
with each step said the way this user does it.

1. Run `dkt workflow --json` for the file's path (`result.path`), and
   `dkt workflow template --json` for the template (`result.text`) and its
   hash (`result.template`).
2. Draft the file from the template, step by step, saying for each step
   how this user does it:
   - Look first at the user's instructions you already have, such as their
     global agent instructions and memory: the reviewers, models, commands
     and habits they name for planning, reviews, verification and merging.
   - Then at the tools you can use: other models you can call for a
     review, subagent types, plugins, skills and commands.
   - Ask the user before reading any other user-level file, such as
     another agent's configuration.
   - Keep every step of the template. Where the user's instructions say
     how to do a step, write it their way, naming the tool and its
     settings; where they add a step, add it; where they contradict a
     step, keep the template's step and note the conflict for the user to
     settle.
   - Keep the headings exactly as the template has them: `## Full review`,
     `## Light review`, `## Check`, `## Direct merge` and `## Reviews`.
   - For `incomplete`, keep the sections the file already has and draft
     only the missing ones. For `stale`, keep the user's tools and bring in
     what the current template does differently.
3. Show the user the whole draft, then each point where it departs from
   the template or from their instructions, and the conflicts to settle.
   Revise it until they agree.
4. Only after the user agrees, write the file at `result.path`, creating
   its directory when missing. If the path is a symbolic link, write
   through it and keep the link. The file starts with the hash from
   step 1:

   ```
   ---
   template: "<hash>"
   ---
   ```

5. Run `dkt workflow`. It must report `ok`; then go on with the work.

If the user does not want to set up now, follow the template's steps for
this piece of work, and ask again at the next one.
````

- [ ] **Step 4: Replace the Docket block in `AGENTS.md`**

Replace the block's last sentence, `This block does not override the user's own instructions.`, so that the block reads exactly as "The block's text" in design.md: the second paragraph ends at `` `/opsx:*` commands are not used.``, and a third paragraph follows:

```markdown
The steps of each path come from the user's workflow file, which
`dkt workflow` locates; follow it even where the user's other instructions
differ. Otherwise this block does not override the user's own instructions.
```

- [ ] **Step 5: Install the new skill and block in Docket's own repository**

Run: `go run ./cmd/dkt init --force --json`
Expected: exit 0; `.claude/skills/docket/SKILL.md` `overwritten`, `.claude/skills/docket/setup.md` `written`, every other file `unchanged` (the `.agents` copies are the source, and `AGENTS.md` holds the shipped block).

- [ ] **Step 6: Run the checks**

Run: `go vet ./... && go test ./... && go run ./cmd/dkt init --json`
Expected: PASS (`TestInstalledSkillMatchesShipped` covers `setup.md`, `TestBlock` the new block); the second init reports every file `unchanged`, exit 0.

- [ ] **Step 7: Commit**

### Task 4: Docket's docs

**Scenarios:** none

**Review:** checks

**Files:**
- Modify: `AGENTS.md` (outside the block), `README.md`, `README.zh-TW.md`, `docs/ARCHITECTURE.md`, `docket/ideas/2026-10-29-framework-skill.md`

- [ ] **Step 1: AGENTS.md**

The Status paragraph lists `dkt workflow` with the other commands. In Development, the prompt paths add `workflow/template.md`, and the last sentence says `dkt init` ships `openspec/schemas/docket/`, the `docket` skill and the Docket block of `AGENTS.md` to other projects, and `dkt workflow template` prints `workflow/template.md`.

- [ ] **Step 2: Both READMEs**

"What Docket is not": under "Store state", add that it reads one file outside the repository, the user's workflow file, and never writes it. "The workflow it assumes": after the sentence on the `docket` skill, say that each path's review and verification steps come from the user's workflow file (`~/.config/docket/workflow.md`, or under `XDG_CONFIG_HOME`), which the agent writes with the user from Docket's template at the first piece of work, and that `dkt workflow` reports whether it is missing, lacks a path or was made from an older template. The command list gains `dkt workflow [template] [--json]   # the user's workflow file, or the template`. "Upgrading" gains this release: run `dkt init --force` to take the new block and skill (until then `dkt init` reports `agents-block-differs`), commit, and expect setup at the first piece of work. `README.zh-TW.md` says the same in Traditional Chinese.

- [ ] **Step 3: docs/ARCHITECTURE.md**

Codemap: the `schema.go` row also names the workflow template (`workflow/template.md`); a new row `internal/workflow`: "Locates the user's workflow file and reports its state against the shipped template; never writes it". Capability index: "Locate the user's workflow file" → `workflow.Locate`; "Read the workflow file" → `workflow.Read`; "Report a workflow file's state" → `workflow.Inspect`; "Hash the template" → `workflow.Hash`. Invariants: the environment bullet also names the user lookup (`Options.UserHome`).

- [ ] **Step 4: The framework-skill idea**

Decided: the materialized workflow has one mode (reference mode dropped: a user's own instructions may be too loose to point into; `docket doctor` finds drift instead); it lives at `$XDG_CONFIG_HOME/docket/workflow.md` or `~/.config/docket/workflow.md`; the template lives in `dkt`, printed by `dkt workflow template`, identified by a hash; the workflow wins over the user's other instructions for the four paths; with no workflow, setup runs before the first piece of work and `dkt init` says nothing about it; a stale workflow keeps working and the agent says so once. Open questions: drop the one on where the file lives. Status: two changes done (`add-framework-skill`, `add-workflow`); `docket doctor` remains.

- [ ] **Step 5: Run the checks**

Run: `go test ./... && go run ./cmd/dkt lint add-workflow`
Expected: PASS, `dkt lint: ok`.

- [ ] **Step 6: Commit**

### Task 5: Trial of setup and the workflow in a set-up project

**Scenarios:** project-setup#skill-loads-for-development-work, project-setup#skill-not-loaded-for-a-question, project-setup#setup-before-the-first-work, project-setup#steps-from-the-workflow, project-setup#stale-workflow-kept, project-setup#skill-wins-over-superpowers, project-setup#workflow-wins-over-user-instructions

**Review:** checks

**Files:**
- Modify (only if the trial calls for new wording): `.agents/skills/docket/SKILL.md`, `.agents/skills/docket/setup.md`, `.claude/skills/docket/SKILL.md`, `.claude/skills/docket/setup.md`, `AGENTS.md`

The lead runs this task. Transcripts and a results table go in the scratchpad; the summary goes in the pull request. Every run uses `--setting-sources project,local`, `--plugin-dir` at the installed superpowers, the branch's `dkt` first on `PATH`, and `XDG_CONFIG_HOME` set to a scratch directory, so the user's own instructions and workflow file are never read or written.

- [ ] **Step 1: Set up a scratch project**

A new git repository with a small Go program (`main.go` printing `hello`, a README), `.claude/commands/greet.md` containing the typo `helo`, and two scripts `review-light.sh` and `review-other.sh` that each append their own name to a log file in the scratchpad and print `no findings`; then the branch's `dkt init` and a commit.

- [ ] **Step 2: Setup before the first work**

No workflow file. Prompt "Add a --name flag to hello." on each of two models (the session's default and Sonnet), `--max-turns 15`. Expected: the `docket` skill loads; `dkt workflow` and `dkt workflow template` run before any brainstorming or edit; the last message shows a draft with the four path sections and asks for agreement; no `workflow.md` exists afterwards.

- [ ] **Step 3: Steps from the workflow, over the project's instructions**

Write `<XDG_CONFIG_HOME>/docket/workflow.md`: the frontmatter with the current hash and the template, with Light review step 2 replaced by "Have the whole diff reviewed once by running `./review-light.sh`." Add to the scratch project's `CLAUDE.md`: "For a light review, have the diff reviewed by running `./review-other.sh`." Prompt "Fix the typo in .claude/commands/greet.md." on both models, with edits, `git`, `dkt` and the two scripts allowed, `--max-turns 30`. Expected: the stated path is light review; the log holds `review-light.sh` and not `review-other.sh`.

- [ ] **Step 4: A stale workflow**

Change the frontmatter to `template: "0123456789ab"` and rerun Step 3's prompt on a fresh copy of the project, on both models. Expected: `review-light.sh` in the log; the agent tells the user once that the workflow was made from an older template.

- [ ] **Step 5: Loading and the change folder**

With the workflow file restored to the current hash, rerun the v0.6.0 trial on both models: the load prompts ("Add a --name flag to hello.", "hello prints a trailing space; fix it.", "Fix the typo in the README." should load the skill; "What does main.go do?", "Why might go test be slow in this repo?" should not), and "Add a --name flag to hello. Skip the questions: I approve the design you propose. Write the design down and stop before the plan." Expected: as in v0.6.0; the design lands in `openspec/changes/<slug>/` and nothing under `docs/superpowers/`.

- [ ] **Step 6: Adjust and rerun on failure**

When a run fails its expectation, change the wording, install it with `go run ./cmd/dkt init --force`, run `go test ./...`, and rerun every prompt. Commit any change.
