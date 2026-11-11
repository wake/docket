# Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let a project keep issues in another tool and still order, take and report them in Docket, through cards: issue files with `external` and `external-id` and no body.

**Architecture:** A card is an issue: `dkt issue` writes it, `dkt lint` checks it with the issue rules (card branch, `## Dropped`, `unsynced`, duplicate address), `dkt status` reads `dropped` on issues, and `dkt snapshot` reports the new fields. dkt never reaches the other tool; the flow with it is skill text.

**Tech Stack:** Go 1.24, standard library only.

**Spec:** `openspec/changes/add-external-issue-cards/` (proposal.md, design.md, specs/issues, specs/ideas, specs/project-snapshot).

## Global Constraints

- No new dependency; `gofmt -l .` prints nothing, `go vet ./...` and `go test ./...` pass.
- Front matter keys: `external`, `external-id`, `dropped`, `unsynced`; section `## Dropped`.
- `unsynced` steps, in this order: `open`, `taken`, `pr`, `merged`, `dropped`; one-line form `unsynced: [<step>, <step>]`.
- Rule names: `external-taken` (dkt issue), `issue-external`, `issue-unsynced`, `issue-front-matter`, `issue-section` (dkt lint and dkt status).
- JSON fields: `result.external`, `result.external_id` (dkt issue, absent without the flags); `external`, `external_id`, `dropped`, `unsynced` on `result.issues[]` (dkt snapshot).
- A card is an issue whose front matter has the `external` key, whatever its value. `external` is compared exactly as the front matter reads it.
- Everything in the repository is written in English.

## Review Focus

- A hand-written `external-id: #92` without quotes reads as empty, since `#` starts a comment: lint reports `issue-front-matter` naming `external-id` and saying to quote the value (Task 2).
- A card holding only an HTML comment is not content: no finding (Task 2).
- Two addresses that differ only by a trailing `/` are two cards: `dkt issue` writes the second (Task 1).
- `dkt issue --external` while `docket/issues/` holds an issue it cannot read: `unreadable` lists it, exit 3, nothing written (Task 1).
- `unsynced: []` is valid: no lint finding, and the snapshot reports `[]` (Tasks 2 and 4).

## Reuse

| Need | Existing (path:symbol) | Decision | Reason | Evidence |
|---|---|---|---|---|
| Write an issue file | internal/cli/cmd_issue.go:runIssue | extend | add the two flags and the card content | `rg -n "func runIssue" internal/cli` |
| Front matter text | internal/change/metadata.go:FrontMatter | reuse | append the two quoted lines before the closing `---` | `rg -n "func FrontMatter" internal/change` |
| One-line value form | internal/change/metadata.go:ValidLine | reuse | the form of a source | `rg -n "func ValidLine" internal/change` |
| Finding and exit 1 | internal/cli/cmd_new.go:finding | reuse | `external-taken` like `slug-invalid` | `rg -n "func finding" internal/cli` |
| Read a file without following links | internal/safefile/safefile.go:ReadNoFollow | reuse | reading the worktree's issues for the duplicate check | `rg -n "func ReadNoFollow" internal/safefile` |
| Result of dkt issue | internal/cli/cmd_new.go:newResult | extend | two `omitempty` fields | `rg -n "type newResult" internal/cli` |
| Issue lint rules | internal/lint/record.go:checkDatedFront | extend | card fields, `dropped` | `rg -n "func checkDatedFront" internal/lint` |
| Issue section rules | internal/lint/record.go:checkDatedSections | extend | card branch, `## Dropped` | `rg -n "func checkDatedSections" internal/lint` |
| Sections after the required ones | internal/lint/record.go:dated.after | reuse | `Dropped` in `issueKind.after` | `rg -n "after " internal/lint/record.go` |
| Issues in the worktree | internal/lint/record.go:(*state).list | reuse | duplicate address in lint | `rg -n "func \(r \*state\) list" internal/lint` |
| Parse a record or issue | internal/record/parse.go:Parse | reuse | sections, outside text, comments blanked | `rg -n "func Parse" internal/record` |
| One-line list of steps | internal/idea/idea.go:Parse | new | `needs` checks idea paths; a step list needs its own vocabulary, and a shared reader would restructure idea parsing for one caller | `rg -n "NeedsForm" internal/` |
| Issue `dropped` for dkt status | internal/idea/idea.go:ParseItem | extend | read `dropped` as Parse does | `rg -n "func ParseItem" internal/idea` |
| Skip dropped items | internal/idea/idea.go:Next | reuse | already skips `Dropped` | `rg -n "func Next" internal/idea` |
| Issue entry of the snapshot | internal/snapshot/result.go:IssueEntry | extend | four fields | `rg -n "type IssueEntry" internal/snapshot` |
| Issue state | internal/snapshot/backlog.go:Backlog | extend | `dropped` first | `rg -n "e.State = \"in-progress\"" internal/snapshot` |
| Old shipped copies | internal/setup/history.go:Earlier | extend | hashes of the replaced prompt files | `rg -n "var Earlier" internal/setup` |

---

### Task 1: dkt issue writes cards

**Scenarios:** issues#issue-written, issues#missing-issue-source, issues#issue-title-over-the-limit, issues#address-without-its-number, issues#blank-external-number, issues#card-from-an-address, issues#address-already-carded

**Review:** judgment

**Files:**
- Modify: `internal/cli/cmd_issue.go`
- Modify: `internal/cli/cmd_new.go` (newResult)
- Test: `internal/cli/cmd_issue_test.go`

**Interfaces:**
- Produces: `newResult.External string` (`json:"external,omitempty"`) and `newResult.ExternalID string` (`json:"external_id,omitempty"`); the `--external` and `--external-id` flags.

- [ ] **Step 1: Write the failing tests** in `internal/cli/cmd_issue_test.go`, using `doIssue` and `recordRepo` (the run date is 2026-10-30):
  - `TestIssueCard`: `login-loop --title "Login loops" --source https://git.example.com/app/issues/92 --external https://git.example.com/app/issues/92 --external-id "#92"` exits 0; the file is exactly `---\ntitle: "Login loops"\nsource: "https://git.example.com/app/issues/92"\nexternal: "https://git.example.com/app/issues/92"\nexternal-id: "#92"\n---\n`; the JSON has `"external_id":"#92"`. `TestIssueWritten` additionally asserts the raw JSON has no `"external"` key.
  - `TestIssueCardFlags`, table-driven, each exits 2 and writes nothing: only `--external` (message names `--external-id`), only `--external-id` (message names `--external`), `--external-id " "`, `--external` with a line break.
  - `TestIssueCardTaken`: with `docket/issues/2026-10-29-login.md` holding `external: "https://git.example.com/app/issues/92"` in the worktree, the same `--external` exits 1 with finding `external-taken` whose message names `docket/issues/2026-10-29-login.md`, and no new file exists. The same run with `.../issues/92/` exits 0.
  - `TestIssueCardUnreadable`: an issue file in `docket/issues/` with mode 0 (skip when running as root) makes `--external` exit 3 with that path in `unreadable`, nothing written.
- [ ] **Step 2: Run** `go test ./internal/cli -run 'TestIssue'` and see the new tests fail.
- [ ] **Step 3: Implement** in `runIssue(c *Context, name, source, title, external, externalID string) (Outcome, error)`: the flag pair check after the title checks (usage errors, exit 2); before writing a card, read each regular `*.md` directly under `docket/issues/` of the worktree with `safefile.ReadNoFollow`, parse its front matter with `change.ReadFrontMatter`, and report `external-taken` on an exact match or list unreadable files and stop; the card's content is the front matter with the two quoted lines and nothing after it.
- [ ] **Step 4: Run** `go test ./internal/cli` and see it pass.
- [ ] **Step 5: Commit** with the trailer `Change: add-external-issue-cards/1`.

### Task 2: dkt lint checks cards, dropped issues and unsynced steps

**Scenarios:** issues#card-file, issues#issue-without-front-matter, issues#issue-title-too-long, issues#card-without-its-number, issues#dropped-that-is-not-a-boolean, issues#issue-section-empty, issues#record-section-in-an-issue, issues#empty-card, issues#card-with-content, issues#two-cards-for-one-address, issues#dropped-card, issues#dropped-without-a-reason, issues#notes-not-written, issues#issue-not-filed-yet, issues#open-on-a-card, issues#unsynced-as-a-block

**Review:** judgment

**Files:**
- Modify: `internal/issue/issue.go`
- Create: `internal/issue/unsynced_test.go`
- Modify: `internal/lint/record.go`
- Test: `internal/lint/issue_test.go`

**Interfaces:**
- Produces: `issue.Steps = []string{"open", "taken", "pr", "merged", "dropped"}`; `issue.UnsyncedForm = "unsynced: [<step>, <step>]"`; `issue.DroppedSection = "Dropped"`; `func issue.ParseUnsynced(v string) ([]string, error)`, which returns the steps in the order written (an empty slice for `[]`) and an error, naming the form, for a value not in brackets, an unknown step or a repeated one.

- [ ] **Step 1: Write the failing tests.** `internal/issue/unsynced_test.go`: `TestParseUnsynced` with `[taken, pr]` giving `["taken", "pr"]`, `[]` giving `[]`, and errors for `taken`, `[later]`, `[pr, pr]`. In `internal/lint/issue_test.go`, using `lintIss` and `only` with `card := "---\ntitle: T\nsource: s\nexternal: \"https://x/1\"\nexternal-id: \"#1\"\n---\n"`:
  - `TestCardClean`: `card` alone, `card+"\n<!-- note -->\n"`, and `card` with `unsynced: []` each give no finding.
  - `TestCardFrontMatter`: without `external-id` gives only `issue-front-matter` naming `external-id`; `external-id: #1` unquoted gives only `issue-front-matter` whose message says to quote it; `dropped: yes` gives only `issue-front-matter`.
  - `TestCardContent`: `card+"\n## Current\nc\n"` gives only `issue-section` naming `Current` with `other tool` in the message; `card+"\nloose text\n"` gives only `issue-section`.
  - `TestIssueDropped`: a card with `dropped: true` and `## Dropped\nClosed in the tracker as a duplicate.\n` gives none; a local issue with `dropped: true` and the four sections but no `## Dropped` gives only `issue-section` naming `Dropped`; `## Dropped` with only a comment gives `issue-section`; `## Dropped` without `dropped: true` gives `issue-section`.
  - `TestIssueUnsynced`: `unsynced: [taken, pr]` on a card gives none; `unsynced: [open]` on a local issue with the four sections gives none; `unsynced: [open]` on a card, `unsynced: [taken]` on a local issue, and `unsynced: [later]` each give only `issue-unsynced`; `unsynced:` followed by an indented `- taken` gives `issue-front-matter`.
  - `TestCardDuplicate`: with `repo` holding a second card at `docket/issues/2026-11-01-other.md` with the same `external`, linting only `issPath` gives `issue-external` on `issPath` naming the other file; with `https://x/1/` in the other, none.
- [ ] **Step 2: Run** `go test ./internal/issue ./internal/lint -run 'Unsynced|Card|IssueDropped|IssueUnsynced'` and see them fail.
- [ ] **Step 3: Implement** `ParseUnsynced` in `internal/issue/issue.go`; in `internal/lint/record.go` add `Dropped` to `issueKind.after`, and in the issue rules: the card fields and `dropped` in `checkDatedFront` (an empty `external-id` read from a value starting with `#` gets the quoting hint), the card branch in `checkDatedSections` (no required section, only `## Dropped` allowed, text outside it reported), the `dropped`/`## Dropped` pairing, `unsynced` through `ParseUnsynced` with the card rule for `open`, and the duplicate check against the other files of `r.list(issue.Dir)`. Record rules stay unchanged; gate the new rules on `d.prefix == "issue"` or a field of `dated`, whichever keeps `lintDated` readable.
- [ ] **Step 4: Run** `go test ./internal/issue ./internal/lint` and see them pass.
- [ ] **Step 5: Commit** with the trailer `Change: add-external-issue-cards/2`.

### Task 3: dkt status skips dropped issues

**Scenarios:** ideas#order-read, ideas#issue-in-the-order, ideas#card-in-the-order, ideas#first-idea-next, ideas#dropped-idea-skipped, ideas#taken-idea-skipped, ideas#issue-next, ideas#issue-taken, ideas#dropped-issue-skipped, ideas#issue-with-dropped-or-needs, ideas#issue-with-needs, ideas#work-named-by-an-address, ideas#issue-without-a-title, ideas#issue-dropped-that-is-not-a-boolean

**Review:** checks

**Files:**
- Modify: `internal/idea/idea.go`
- Test: `internal/idea/idea_test.go`

**Interfaces:**
- Produces: `ParseItem` returns `Idea{Title, Dropped}` for an issue; `Needs` stays nil.

- [ ] **Step 1: Write the failing tests** in `internal/idea/idea_test.go`: `TestParseItemIssueDropped` (`dropped: true` gives `Dropped` true; `dropped: yes` gives an error mentioning `true or false`; `needs: [docket/ideas/a.md]` is ignored); `TestNextDroppedIssue` (an order of a dropped issue then an idea gives the idea); `TestNextAddressDoesNotTake` (a work map keyed by `https://git.example.com/app/issues/92` leaves the card at `docket/issues/2026-11-12-login-loop.md` next); `TestCheckCard` (through `Check`, an order naming a card with title `Login loops` holds it in `items` with that title and no problem).
- [ ] **Step 2: Run** `go test ./internal/idea` and see the dropped tests fail.
- [ ] **Step 3: Implement** the `dropped` read in `ParseItem`, with the same values and error as `Parse`, and update the comment on `Idea`.
- [ ] **Step 4: Run** `go test ./internal/idea ./internal/cli -run 'Status|Next|Parse|Check'` and see them pass.
- [ ] **Step 5: Commit** with the trailer `Change: add-external-issue-cards/3`.

### Task 4: dkt snapshot reports card fields and dropped issues

**Scenarios:** project-snapshot#issue-with-a-handler, project-snapshot#issue-read-from-its-first-location, project-snapshot#card-reported, project-snapshot#local-issue-reported, project-snapshot#open-issue, project-snapshot#one-handler-merged-of-two, project-snapshot#every-handler-merged, project-snapshot#dropped-issue-with-a-handler

**Review:** judgment

**Files:**
- Modify: `internal/snapshot/result.go`
- Modify: `internal/snapshot/backlog.go`
- Test: `internal/snapshot/backlog_test.go`

**Interfaces:**
- Consumes: `issue.ParseUnsynced` (Task 2).
- Produces: `IssueEntry.External *string` (`json:"external"`), `IssueEntry.ExternalID *string` (`json:"external_id"`), `IssueEntry.Dropped bool` (`json:"dropped"`), `IssueEntry.Unsynced []string` (`json:"unsynced"`, never null).

- [ ] **Step 1: Write the failing tests** in `internal/snapshot/backlog_test.go`, with `issueIn` and a card file `---\ntitle: Fix X\nexternal: "https://git.example.com/app/issues/92"\nexternal-id: "#92"\nunsynced: [taken]\n---\n`: `TestCardEntry` (those values, `Dropped` false, `Unsynced` `["taken"]`); `TestLocalIssueEntry` (`issueFile` gives nil `External` and `ExternalID`, `Unsynced` `[]`, and JSON `"unsynced":[]`); `TestDroppedIssueState` (`dropped: true` with an unmerged handler gives `State` `dropped` and keeps the handler; `dropped: yes` gives `Dropped` false and `State` `open`); `TestUnsyncedBadForm` (`unsynced: later` gives `[]`). `TestIssueStates` keeps passing.
- [ ] **Step 2: Run** `go test ./internal/snapshot -run 'Issue|Card|Unsynced'` and see the new tests fail.
- [ ] **Step 3: Implement** the four fields in `IssueEntry` and their reading in `Backlog` from the first location's front matter; `dropped` set `State` to `dropped` before the handler states.
- [ ] **Step 4: Run** `go test ./internal/snapshot ./internal/cli -run 'Snapshot|Issue'` and see them pass.
- [ ] **Step 5: Commit** with the trailer `Change: add-external-issue-cards/4`.

### Task 5: The skill and the proposal template describe cards

**Scenarios:** none

**Review:** judgment

**Files:**
- Modify: `shipped/skill/SKILL.md`
- Modify: `shipped/schema/templates/proposal.md`
- Modify: `.claude/skills/docket/SKILL.md`, `.agents/skills/docket/SKILL.md`, `openspec/schemas/docket/templates/proposal.md` (by `dkt init --force`)
- Modify: `internal/setup/history.go`

- [ ] **Step 1: Draft in Traditional Chinese** the new text of "Ideas, issues and the order" in `shipped/skill/SKILL.md` (the five cases, making a card, the notes and `unsynced`, `--source` holding the card's path) and the `source:` line of the proposal template, as design.md "The flow, in the skill" says; put the draft on the prompt page as a pending prompt for the maintainer to edit.
- [ ] **Step 2: Wait** for the maintainer to save the edited draft, then translate it into English in the two shipped files; nothing else of the files changes.
- [ ] **Step 3: Run** `dkt init --force` in the worktree, and `go test ./internal/setup -run TestEarlierHoldsShippedFiles`; add the hashes it asks for to `internal/setup/history.go`, and run `go test ./...` to pass.
- [ ] **Step 4: Commit** with the trailer `Change: add-external-issue-cards/5`.
