# Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `dkt init` installs the `docket` skill in `.claude/skills/docket/` and `.agents/skills/docket/` and keeps a marked Docket block in `AGENTS.md` (reached from `CLAUDE.md`), instead of printing a section to paste.

**Architecture:** The root package embeds the skill (`.agents/skills/docket/`) and cuts the block out of Docket's own `AGENTS.md` at its markers. `internal/setup` builds the list of every file init would write, checks all of them (links, reads, markers) before running `openspec init` or writing anything, then writes. Placing a block in a file's bytes is a pure function in `internal/setup/block.go`. `internal/cli/cmd_init.go` maps results to findings.

**Tech Stack:** Go 1.26, `go.yaml.in/yaml/v3` (tests only, for the skill's frontmatter), OpenSpec CLI 1.14.x (only through `internal/openspec`).

**Spec:** `openspec/changes/add-framework-skill/proposal.md`, `openspec/changes/add-framework-skill/specs/project-setup/spec.md`, `openspec/changes/add-framework-skill/design.md`.

## Global Constraints

- Skill paths: `.claude/skills/docket/` and `.agents/skills/docket/`; the shipped skill is one file, `SKILL.md`, with frontmatter `name: docket` and a `description` of at most 1,024 characters.
- Markers: a line that is exactly `<!-- docket:start -->` or `<!-- docket:end -->` (no other characters on the line, a CR before the LF included). The block is the bytes from the start of the start marker line through the end of the end marker text, without the newline after it.
- A new `AGENTS.md` is the block followed by `\n`. A new `CLAUDE.md` is exactly `@AGENTS.md\n`.
- Appending: when the file is not empty and does not end with `\n`, add `\n`; when it is not empty, add one more `\n` (a blank line); then the block and `\n`. Everything before stays byte for byte.
- `CLAUDE.md` reaches `AGENTS.md` when a line, with surrounding spaces and tabs removed, is `@AGENTS.md` or `@./AGENTS.md`, or when it is a symbolic link whose target, after `filepath.Clean`, is `AGENTS.md`.
- File statuses: `written`, `unchanged`, `differs`, `overwritten`. A block appended to an existing file is `written`; replaced under `--force`, `overwritten`.
- Finding rules: `schema-differs`, `skill-differs`, `agents-block-differs` (for `AGENTS.md` or `CLAUDE.md`), `agents-block-malformed`, `symlink-refused`. A file that cannot be read goes in `unreadable` (exit 3, which wins over findings).
- No write and no `openspec init` until every check passed; a refused run writes nothing.
- JSON `result` has exactly `openspec_initialized` and `files`; `agents_section` is gone and nothing is printed to paste.
- Architecture invariants: `safepath.Check` before every write; writes and reads use `O_NOFOLLOW`; only `internal/openspec` runs OpenSpec.
- Everything in the repository is in English. Every commit for a task carries `Change: add-framework-skill` and `Task: <N>` in one final trailer paragraph; structural and behavior changes go in separate commits.

## Review Focus

- `AGENTS.md` that is empty, or lacks a final newline, gets the block appended with its earlier bytes intact. Test: Task 2.
- A marker line with leading spaces or trailing text is not a marker, so the file gets a block appended rather than reported malformed. Test: Task 2.
- `CLAUDE.md` that mentions `@AGENTS.md` inside a sentence does not count as importing it and gets the block. Test: Task 2.
- A file a user added under `.claude/skills/docket/` is neither listed nor touched. Test: Task 3.
- `--force` on a differing block keeps the text after the end marker byte for byte. Test: Task 2.

## Reuse

| Need | Existing (path:symbol) | Decision | Reason | Evidence |
|---|---|---|---|---|
| Embed shipped files | schema.go:Schema | extend | `Skill()` embeds `.agents/skills/docket` the same way; `go:embed` accepts the dotted path (tried in a scratch module) | `rg -n "go:embed" --type go` |
| Cut Docket's text out of `AGENTS.md` | schema.go:AgentsSection | extend | becomes `Block()`, cut at the markers instead of a heading; `AgentsSection` is removed in Task 4 | `rg -n "AgentsSection" --type go` |
| Write an embedded file without following links, keeping an identical one and reporting a differing one | internal/setup/setup.go:install | extend | split into deciding a status (check phase) and writing (write phase), so every check runs before the first write; schema and skill files both go through it | `rg -n "func install\|MkdirAll\|OpenFile" internal/setup/setup.go` |
| Read without following a link | internal/setup/setup.go:readNoFollow | reuse | the check phase reads every target with it | `rg -n "readNoFollow" internal/setup` |
| Refuse a link on a write path | internal/safepath/safepath.go:Check | reuse | before every write, including `AGENTS.md` and `CLAUDE.md` | `rg -n "func Check" internal/safepath` |
| Init's walk, checks and writes | internal/setup/setup.go:Init | extend | becomes check-everything-then-write over schema, skill and block targets | `rg -n "func Init" internal/setup` |
| Find marker lines and place the block | none | new | nothing finds marker lines with byte offsets; `markdown.Lines` rewrites CR and LF, so offsets into the original bytes are lost. `setup.FindBlock` is the one marker scan: `placeBlock` and `docket.Block()` both use it | `rg -n "func Lines" internal/markdown` |
| Findings, unreadable entries and exit codes | internal/cli/envelope.go:Outcome, Unreadable, ExitCode | reuse | unreadable already wins over findings | `rg -n "func ExitCode" -A 14 internal/cli/envelope.go` |
| Map init's result to output | internal/cli/cmd_init.go:initCommand | extend | new rules, no printed section | `rg -n "schema-differs" internal/cli` |
| Parse the skill's frontmatter in a test | external: go.yaml.in/yaml/v3 | reuse | already a dependency (`go.mod`); no frontmatter parser exists in the repository | `rg -n "yaml/v3" go.mod`, `rg -n -i "frontmatter" --type go` |
| Test repositories and OpenSpec | internal/testutil/repo.go:NewRepo, internal/testutil/openspec.go:OpenSpecCmd, FakeOpenSpec | reuse | as the existing init tests do | `rg -n "func (NewRepo\|OpenSpecCmd\|FakeOpenSpec)" internal/testutil` |
| Whole-tree snapshot for "nothing written" | internal/cli/cmd_init_test.go:snapshot, sameSnapshot | reuse | already used by the symlink tests | `rg -n "func snapshot" internal/cli` |

---

### Task 1: The skill and the block

**Scenarios:** project-setup#skill-copies-agree

**Review:** judgment

**Files:**
- Create: `.agents/skills/docket/SKILL.md`
- Modify: `AGENTS.md` (append the block; the "Change workflow" section stays until Task 5)
- Modify: `schema.go`
- Test: `schema_test.go`

**Interfaces:**
- Produces: `docket.Skill() fs.FS` (files of `.agents/skills/docket`, paths relative to it, e.g. `SKILL.md`); `docket.Block() string` (the block as defined in Global Constraints; panics when `AGENTS.md` has no single well-formed block).

- [ ] **Step 1: Write the failing tests**

`TestSkillFrontmatter`: `SKILL.md` from `Skill()` starts with `---\n`, its frontmatter parses with `yaml.v3`, `name == "docket"`, `description` is non-empty and `len(description) <= 1024`. `TestBlock`: `Block()` starts with `<!-- docket:start -->\n`, ends with `\n<!-- docket:end -->`, and contains each marker once.

- [ ] **Step 2: Run them to see them fail**

Run: `go test -run 'TestSkillFrontmatter|TestBlock' .`
Expected: FAIL (undefined `Skill`, `Block`).

- [ ] **Step 3: Write `.agents/skills/docket/SKILL.md`**

Frontmatter: `name: docket`; `description:` the text under "The skill's text" in design.md, as one folded string. Body: the "Change workflow" section of `AGENTS.md` rewritten around four named paths, in this order:
1. One paragraph: these are the rules for development work in this repository; where another skill says otherwise, these win.
2. `## Triage`: the four paths and what puts work on each, carried over from the section's triage bullets (full review: a sentence in `openspec/specs/` changes, or an edit sets an expectation no spec states; light review: restoring fixes, restructurings, prompt text, tests of shown behavior, check settings; check: a direct-merge edit that touches a code or test file; direct merge: docs for people, comment or whitespace edits, bumps, manifest-only upgrades), heaviest wins, sort by what is touched, unready ideas go to `docket/ideas/`. Then: before starting, state in one line the path taken and the rule that chose it.
3. `## Full review`, `## Light review`, `## Check`, `## Direct merge`: each path's steps, carrying over every rule of the section (worktree, `dkt new` with `--kind`, `--source` and `--title`, where brainstorming and writing-plans write, the Reuse table, `**Review:**` lines, the plan review, subagent-driven development with reviewers only for `judgment` tasks, mechanical checks, implementer reports, trailers, the two pre-merge reviewers, splitting, `dkt sync` and `dkt lint` before the pull request, merge commits, `dkt archive`, no `/opsx:*`, mid-way requirement changes).
4. `## Reviews`: a required review is done by a reviewer on a different model; else by the same model in a fresh context; else the pull request says it had no independent review and asks a person to review it. The author's own reading is not a review.
5. `## Parallel sessions` and `## Prompt paths`, carried over.

Name capabilities, never products ("a reviewer on a different model", "a fresh agent given only the diff"). At most 2,000 words. Every rule of the current section must have a home; nothing from the user's own global instructions is added.

- [ ] **Step 4: Append the block to `AGENTS.md`**

Append, by the appending rule in Global Constraints, the block exactly as shown under "The block's text" in design.md.

- [ ] **Step 5: Implement `Skill()` and `Block()` in `schema.go`**

`//go:embed .agents/skills/docket` and `fs.Sub`, like `Schema()`. `Block()` scans the embedded `AGENTS.md` for marker lines as defined in Global Constraints.

- [ ] **Step 6: Run the tests to see them pass**

Run: `go test .`
Expected: PASS (`TestAgentsSection` still passes; the section is still there).

- [ ] **Step 7: Commit**

```bash
git add .agents/skills/docket/SKILL.md AGENTS.md schema.go schema_test.go
git commit -F <message with Change: add-framework-skill / Task: 1>
```

### Task 2: Placing the block in a file

**Scenarios:** project-setup#block-added-to-an-existing-agents-md, project-setup#agents-md-created, project-setup#edited-block, project-setup#malformed-block, project-setup#claude-md-without-the-import

**Review:** judgment

**Files:**
- Create: `internal/setup/block.go`
- Test: `internal/setup/block_test.go`
- Modify: `schema.go` (structural: `Block()` calls `setup.FindBlock`)

**Interfaces:**
- Produces: `FindBlock(b []byte) (from, to int, found bool, err error)`: `b[from:to]` is the block as defined in Global Constraints; `found` is false with a nil error when there is no marker; `err` is `errMalformed` otherwise. `placeBlock(cur []byte, exists bool, block string, force bool) (next []byte, status FileStatus, err error)`, built on `FindBlock`: `next` is nil when nothing is to be written. `errMalformed` (an `error` whose message says which case: no end marker, end marker first, more than one block). `importsAgents(b []byte) bool`.

- [ ] **Step 1: Write the failing tests**

Table test `TestPlaceBlock`, with `block = "<!-- docket:start -->\nB\n<!-- docket:end -->"`:

| case | cur | exists | force | next | status / err |
|---|---|---|---|---|---|
| missing | nil | false | false | block+"\n" | Written |
| empty file | "" | true | false | block+"\n" | Written |
| no final newline | "# A" | true | false | "# A\n\n"+block+"\n" | Written |
| ends with newline | "# A\n" | true | false | "# A\n\n"+block+"\n" | Written |
| identical | "# A\n\n"+block+"\nmore\n" | true | false | nil | Unchanged |
| differs | "# A\n<!-- docket:start -->\nold\n<!-- docket:end -->\nafter\n" | true | false | nil | Differs |
| differs, force | same | true | true | "# A\n"+block+"\nafter\n" | Overwritten |
| no end | "<!-- docket:start -->\nx\n" | true | false | nil | errMalformed |
| end first | "<!-- docket:end -->\n<!-- docket:start -->\n" | true | false | nil | errMalformed |
| two blocks | block+"\n"+block+"\n" | true | false | nil | errMalformed |
| indented marker | "  <!-- docket:start -->\n" | true | false | "  <!-- docket:start -->\n\n"+block+"\n" | Written |
| CRLF copy of the block | "<!-- docket:start -->\r\nB\r\n<!-- docket:end -->\r\n" | true | false | nil | Differs |

`TestFindBlock`: for `"x\n"+block+"\ny\n"`, `from == 2`, `to == 2+len(block)`, `found`; for `"x\n"`, not found and nil error; `errMalformed` for no end marker, end marker first, and two blocks.

`TestImportsAgents`: true for `"@AGENTS.md\n"`, `"# x\n  @./AGENTS.md\t\n"`; false for `"See @AGENTS.md here\n"`, `"@AGENTS.md.bak\n"`, `""`.

- [ ] **Step 2: Run them to see them fail**

Run: `go test ./internal/setup -run 'TestPlaceBlock|TestImportsAgents'`
Expected: FAIL (undefined).

- [ ] **Step 3: Implement `placeBlock` and `importsAgents` in `internal/setup/block.go`**

Scan lines keeping byte offsets; a marker line is compared after dropping one trailing `\r` (so the CRLF case finds its markers, then differs on bytes).

- [ ] **Step 4: Run them to see them pass**

Run: `go test ./internal/setup`
Expected: PASS

- [ ] **Step 5: Commit**

- [ ] **Step 6: Make `docket.Block()` use `setup.FindBlock`**

A separate structural commit: `Block()` calls `setup.FindBlock([]byte(agentsMD))` and panics on an error or when no block is found, replacing its own scan. Run `go test ./...`; expected PASS with no test changed.

### Task 3: Install the skill, checking everything before writing

**Scenarios:** project-setup#fresh-repository-gets-the-skill, project-setup#edited-skill-file, project-setup#locally-edited-template, project-setup#symlinked-schema-file-refused, project-setup#symlinked-skill-directory-refused

**Review:** judgment

**Files:**
- Modify: `internal/setup/setup.go`, `internal/setup/setup_test.go`
- Modify: `internal/cli/cmd_init.go`, `internal/cli/cmd_init_test.go`

**Interfaces:**
- Consumes: `docket.Skill()` (Task 1).
- Produces: `setup.Assets{Schema, Skill fs.FS}` (Task 4 adds `Block string`); `setup.Init(root string, a Assets, force bool, initOpenSpec func() error) (Result, error)`; `File` gains `Kind FileKind` with `json:"-"` and kinds `KindSchema`, `KindSkill` (Task 4 adds `KindBlock`); `*setup.Refused{Problems []Problem}` returned when a check fails, with `Problem{Path string; Kind ProblemKind; Err error}` and kinds `ProblemSymlink`, `ProblemUnreadable` (Task 4 adds `ProblemMalformed`).

Init order: check `openspec` for a link; list targets (schema files under `openspec/schemas/docket/`, then the skill under `.claude/skills/docket/`, then under `.agents/skills/docket/`); for each, `safepath.Check` (a `SymlinkError` is `ProblemSymlink`, any other error `ProblemUnreadable`) and `readNoFollow` (an error other than not-exist is `ProblemUnreadable`) to decide its status; if any problem, return `Result{Files: []}` and `*Refused` listing all of them; else run `initOpenSpec` when `openspec/` is missing; then write. The decision logic of `install` splits into deciding (check phase) and writing (write phase).

- [ ] **Step 1: Write the failing tests**

setup: `TestInitInstallsSkill` (both `SKILL.md` written, statuses `written`, kinds right; second run all `unchanged`); `TestInitLeavesUserSkillFiles` (`.claude/skills/docket/notes.md` written before init is unchanged and not in `Files`); `TestInitRefusesBeforeWriting` (a link at `.agents/skills/docket` and an unreadable schema file (mode 000, skipped as root): `*Refused` with both problems, no file written, `initOpenSpec` not called).
setup: `TestInitWriteFailureThenRerun` (skipped as root): an empty `.agents/skills/docket/` with mode 0555 passes the checks and fails the write; `Init` returns an error with the schema and `.claude/skills/docket/` files already written and listed; after restoring the mode, a second `Init` writes the `.agents` copy and reports the earlier files `unchanged`.
cli: `TestFreshRepositoryGetsTheSkill` (both files equal `docket.Skill()`, exit 0); `TestEditedSkillFile` (edit `.agents/skills/docket/SKILL.md`, exit 1, one finding `skill-differs` with that `file`, file kept; with `--force` exit 0 and status `overwritten`); `TestSymlinkedSkillDirectoryRefused` (`.claude/skills/docket` linked to an outside directory, `--force --json`: exit 1, `symlink-refused` naming `.claude/skills/docket`, `sameSnapshot` of the repository, outside directory empty); `TestUnreadableSkillFile` (mode 000: exit 3, the file in `unreadable`, nothing written); `TestLinkAndUnreadableTogether` (a link at `.claude/skills/docket` and an unreadable schema file: exit 3, the `symlink-refused` finding and the `unreadable` entry both present, nothing written). Rework `TestPartialFailureKeepsResult`: its regular file in place of a directory is now refused before any write, so it builds the write failure of `TestInitWriteFailureThenRerun` instead and asserts exit 1, a `result` listing the files written before the failure, and that a re-run after restoring the mode exits 0. Keep `TestSymlinkedSchemaFileRefused` passing; its snapshot already shows no skill file appeared.

- [ ] **Step 2: Run them to see them fail**

Run: `go test ./internal/setup ./internal/cli -run 'Skill|Refuses|Unreadable|Symlinked|Together|WriteFailure|PartialFailure'`
Expected: FAIL

- [ ] **Step 3: Implement the check-then-write `Init` and the CLI mapping**

`cmd_init.go`: `setup.Assets{Schema: docket.Schema(), Skill: docket.Skill()}`; `*Refused` problems map to `symlink-refused` findings (message from the link error) and `Unreadable{Source: path, Reason: err}`; `Differs` maps by `Kind` to `schema-differs` or `skill-differs` ("differs from the shipped skill; use --force to overwrite"). The printed section stays until Task 4.

- [ ] **Step 4: Run the package tests**

Run: `go test ./internal/setup ./internal/cli`
Expected: PASS, including every existing init test.

- [ ] **Step 5: Commit**

### Task 4: The Docket block in AGENTS.md and CLAUDE.md

**Scenarios:** project-setup#block-added-to-an-existing-agents-md, project-setup#agents-md-created, project-setup#edited-block, project-setup#malformed-block, project-setup#claude-md-imports-agents-md, project-setup#claude-md-created, project-setup#claude-md-without-the-import, project-setup#other-configuration-unchanged, project-setup#re-run-with-no-differences

**Review:** judgment

**Files:**
- Modify: `internal/setup/setup.go`, `internal/setup/setup_test.go`
- Modify: `internal/cli/cmd_init.go`, `internal/cli/cmd_init_test.go`
- Modify: `schema.go`, `schema_test.go` (remove `AgentsSection`, `agentsHeading`, `TestAgentsSection`)

**Interfaces:**
- Consumes: `placeBlock`, `importsAgents`, `errMalformed` (Task 2); `Assets`, `Refused`, `Problem` (Task 3); `docket.Block()` (Task 1).
- Produces: `Assets.Block string`, `KindBlock`, `ProblemMalformed`; `Result` without `AgentsSection`.

After the skill targets, Init adds `AGENTS.md` (checked with `safepath.Check`, read, `placeBlock`) and then `CLAUDE.md`: `os.Lstat`; a link whose cleaned target is `AGENTS.md` is `unchanged`, any other link is `ProblemSymlink`; a regular file that `importsAgents` is `unchanged`; a missing one is written as `@AGENTS.md\n` (`written`); otherwise `placeBlock`. `errMalformed` is `ProblemMalformed`. Both files are listed in `Files` with `KindBlock`. A missing file is created with `O_EXCL`, an existing one rewritten with `O_TRUNC`, both `O_NOFOLLOW`.

- [ ] **Step 1: Write the failing tests**

cli, each with a committed repository and `--json` where the result is read:
- `TestBlockAddedToExistingAgentsMD`: `AGENTS.md` = `"# agents\n"`; exit 0; it becomes `"# agents\n\n" + docket.Block() + "\n"`.
- `TestAgentsMDCreated`: no `AGENTS.md`; it becomes `docket.Block() + "\n"`.
- `TestEditedBlock`: after a first init, replace the line after the start marker in `AGENTS.md` with `edited`; exit 1, finding `agents-block-differs` with `file` `AGENTS.md`, file unchanged; with `--force`, exit 0, status `overwritten`, block restored, text before and after it unchanged.
- `TestMalformedBlock`, one subtest per case: `AGENTS.md` with a start marker and no end marker, with the end marker first, with two blocks, and `CLAUDE.md` (with `AGENTS.md` absent) with a start marker and no end marker. Each: exit 1, one finding `agents-block-malformed` naming that file, `sameSnapshot` of the repository (no schema, skill or block file written).
- `TestUnreadableAgentsFiles`, skipped as root, one subtest each for `AGENTS.md` and `CLAUDE.md` with mode 000: exit 3, the file in `unreadable`, `sameSnapshot` of the repository.
- `TestClaudeMDLinkedElsewhereRefused`: `CLAUDE.md` a link to `README.md`: exit 1, `symlink-refused` naming `CLAUDE.md`, `sameSnapshot` of the repository, `README.md` unchanged.
- `TestClaudeMDImportsAgentsMD`: `CLAUDE.md` = `"@AGENTS.md\n"`; unchanged, status `unchanged`; `AGENTS.md` holds the block.
- `TestClaudeMDLinkedToAgentsMD`: `CLAUDE.md` a link to `AGENTS.md`, no `AGENTS.md`; link kept, `AGENTS.md` created with the block.
- `TestClaudeMDCreated`: no `CLAUDE.md`; it becomes `"@AGENTS.md\n"`.
- `TestClaudeMDWithoutTheImport`: `CLAUDE.md` = `"# claude\n"`; it becomes `"# claude\n\n" + docket.Block() + "\n"`.
- `TestOtherConfigurationUnchanged` (replaces `TestRoutingSectionPrinted`): `.claude/settings.json`, `.claude/skills/mine/SKILL.md` and `openspec/config.yaml` byte-for-byte unchanged; human output lacks `Add this section`; JSON `result` has no `agents_section` key.
- `TestRerunNoDifferences`: extend to every file in `files` (schema, skill, `AGENTS.md`, `CLAUDE.md`): all `unchanged`, no mtime moved.

- [ ] **Step 2: Run them to see them fail**

Run: `go test ./internal/cli -run 'Block|AgentsMD|AgentsFiles|ClaudeMD|OtherConfiguration|Rerun'`
Expected: FAIL

- [ ] **Step 3: Implement the block targets and the CLI changes**

`cmd_init.go`: `Assets.Block = docket.Block()`; `Differs` with `KindBlock` maps to `agents-block-differs` ("the Docket block differs from the shipped one; use --force to replace it"); `ProblemMalformed` to `agents-block-malformed` (message from `errMalformed`); drop the printed section and `AgentsSection`.

- [ ] **Step 4: Run all tests**

Run: `go vet ./... && go test ./...`
Expected: PASS

- [ ] **Step 5: Commit**

### Task 5: Docket's own repository and docs

**Scenarios:** project-setup#skill-copies-agree

**Review:** checks

**Files:**
- Create: `.claude/skills/docket/SKILL.md` (written by `dkt init`)
- Modify: `AGENTS.md`, `schema_test.go`, `README.md`, `README.zh-TW.md`, `docs/ARCHITECTURE.md`, `docket/ideas/2026-10-29-framework-skill.md`
- Delete: `docket/ideas/2026-10-24-routing-section-fence.md`

- [ ] **Step 1: Write the failing test**

`TestInstalledSkillMatchesShipped` in `schema_test.go`: every file of `Skill()` equals the file at the same path under `.claude/skills/docket/` in the repository.

- [ ] **Step 2: Run it to see it fail**

Run: `go test -run TestInstalledSkillMatchesShipped .`
Expected: FAIL (file missing)

- [ ] **Step 3: Set up Docket's repository with its own `dkt init`**

Run: `go install ./cmd/dkt && dkt init --json` (`openspec/` exists, so no OpenSpec CLI is needed)
Expected: exit 0; `.claude/skills/docket/SKILL.md` `written`; every other file `unchanged`.

- [ ] **Step 4: Edit AGENTS.md**

Delete the "Change workflow (superpowers × OpenSpec)" section. In "Development", the prompt paths become `AGENTS.md`, `CLAUDE.md`, `context` in `openspec/config.yaml`, `openspec/schemas/docket/`, `.claude/skills/docket/` and `.agents/skills/docket/`, and the last sentence says `dkt init` ships `openspec/schemas/docket/`, the `docket` skill and the Docket block of `AGENTS.md` to other projects. The Docket block stays as it is.

- [ ] **Step 5: Edit the docs**

Both READMEs: "What Docket is not" says it never touches `.claude/settings.json` and writes only the marked Docket block in `AGENTS.md` and `CLAUDE.md`, creating either when missing; the workflow section says the `docket` skill carries these rules and `dkt init` installs it; `init` in the command list and in "The commands that write files" names the schema, the skill and the block; "Upgrading" gains this release: run `dkt init`, delete a "Change workflow" section pasted from an earlier release, commit the new files. `docs/ARCHITECTURE.md`: the `schema.go` and `internal/setup` rows name the skill and the block. The framework-skill idea records what this change settled (project skill, block, Reuse table stays in the framework) and keeps the two changes that remain.

- [ ] **Step 6: Run the checks**

Run: `go vet ./... && go test ./... && dkt lint add-framework-skill`
Expected: PASS, `dkt lint: ok`

- [ ] **Step 7: Commit**

### Task 6: Trial of the skill in a set-up project

**Scenarios:** project-setup#skill-loads-for-development-work, project-setup#skill-not-loaded-for-a-question, project-setup#skill-wins-over-superpowers

**Review:** checks

**Files:**
- Modify (only if the trial calls for new wording): `.agents/skills/docket/SKILL.md`, `.claude/skills/docket/SKILL.md`, `AGENTS.md`

The lead runs this task. Transcripts and a results table go in the scratchpad; the summary goes in the pull request.

- [ ] **Step 1: Set up a scratch project**

A new git repository outside this one with a small Go program (`main.go` printing `hello`, a README), committed; then the branch's `dkt init` (with the pinned OpenSpec) and a commit.

- [ ] **Step 2: Isolate from the user's own instructions**

Run `claude -p` with `--setting-sources project,local` and `--plugin-dir` pointing at the installed superpowers, asking it to list the instruction files and skills it was given. Expected: superpowers' skills and `docket` are listed, and nothing from the user's global instructions appears. If isolation cannot be had without editing user files, stop and report.

- [ ] **Step 3: Run the load prompts**

With `--output-format stream-json --verbose --max-turns 3`, once on each of two models (the session's default and Sonnet): should load: "Add a --name flag to hello.", "hello prints a trailing space; fix it.", "Fix the typo in the README."; should not: "What does main.go do?", "Why might go test be slow in this repo?". Expected: a `Skill` call for `docket` in every should-load run, before any brainstorming skill; none in the should-not runs.

- [ ] **Step 4: Follow one feature into its design**

"Add a --name flag to hello. Skip the questions: I approve the design you propose. Write the design down and stop before the plan." with edits and `dkt`/`git` commands allowed and `--max-turns 30`. Expected: the design lands in `openspec/changes/<slug>/`, and nothing is written under `docs/superpowers/`.

- [ ] **Step 5: Adjust and re-run on failure**

When a run fails its expectation, change the description or block wording, copy it to every place it lives, run `go test ./...`, and re-run every prompt. Commit any change.
