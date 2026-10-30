# Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Records exist: `dkt record` writes `docket/records/<date>-<name>.md` from a shipped template, `dkt lint` holds records to their shape, the `docket` skill keeps one on the light review path, and `dkt new` takes a free `-2` suffix instead of refusing a used slug.

**Architecture:** A new `internal/record` package holds the record's rules: kinds, name and slug, the template rendering, the file creation that skips taken names, parsing and word counting. `scan.Scan` also lists the record slugs of every source by name, so `dkt record` can pick a free one the way `dkt new` now does. `internal/lint` lints records with the plan's task heading check and the placeholder rule, both reworked to take any file; `dkt lint` feeds it the records the branch adds.

**Tech Stack:** Go 1.26, `go.yaml.in/yaml/v3` (front matter, through `change.Docket`), git; headless Claude Code for the trial.

**Spec:** `openspec/changes/add-records/proposal.md`, `openspec/changes/add-records/specs/records/spec.md`, `openspec/changes/add-records/specs/change-creation/spec.md`, `openspec/changes/add-records/specs/change-lint/spec.md`, `openspec/changes/add-records/design.md`.

## Global Constraints

- A record is `docket/records/<YYYY-MM-DD>-<name>[-N].md`. Only files directly under `docket/records/` whose name starts with a date and ends in `.md` are records; anything else there is ignored.
- Record kinds are exactly `fix`, `refactor`, `test`, `prompt`, `checks`. Change kinds are exactly `feature`, `change-request`, `baseline`. The two lists are separate.
- Record front matter: `title` (`change.ValidTitle`), `kind`, `source` (not blank). Sections: `## What`, `## Why`, `## Verification`, `## Review`, in that order, then optionally `## Tasks`. No other level-two heading.
- Word limit: 60 words for each of the four sections, `## Tasks` unlimited. HTML comments do not count. A run of letters or digits is one word; each character of the Han, Hiragana, Katakana or Hangul scripts is half a word and ends a run.
- A used slug takes the first free `<slug>-2`, `<slug>-3`, ... A slug, suffix included, is at most 200 characters; past that the command reports `slug-invalid` and writes nothing.
- `dkt record` result keys: exactly `slug`, `path` (repository-relative), `title`, `kind`, `source`, `requested`, `used`. `dkt new` result gains `requested` and `used`. `used` is always an array, empty when no suffix was needed.
- When a suffix was taken, the human output has the line `<requested> is used: <places joined by "; ">; created <slug>`, and the exit code stays 0.
- Lint rules: `record-front-matter`, `record-kind`, `record-section`, `record-too-long`, `record-missing`, plus `placeholder` and `task-heading` on records. A record finding has `file` set to the record's path and no `change`. `dkt lint`'s result gains `records`, always an array.
- Exit codes for `dkt record`: 0 written; 1 `slug-invalid` or `symlink-refused`; 2 a missing or invalid argument or flag; 3 a source that cannot be read; 4 no git repository or no default branch.
- The date of a record is `Context.Now` in its own location, formatted `2006-01-02`; `dkt record` never calls `time.Now`.
- Only `internal/gitx` starts git (test helpers excepted). Nothing below `cmd/dkt` reads the process environment.
- Commands other than `new`, `record` and `lint` keep their output, except that a `docket/records` folder that cannot be listed is now unreadable for every command that scans.
- Everything in the repository is in English.
- TDD, one item per round. A task's "Write the failing tests" step is its test list: take one test, see it fail, write the code that makes it pass, then take the next.
- Commits:
  - every commit for a task carries `Change: add-records` and `Task: <N>` in one final trailer paragraph (write the message to a file and use `git commit -F`);
  - structural and behavior changes go in separate commits;
  - author `Claude Code <wake+claude-code@protype.tw>`, passed with `git -c user.name=... -c user.email=...`.

## Review Focus

- A name whose suffixed slug passes 200 characters: `slug-invalid`, nothing created, for both commands. Test: Task 1 (`TestNewSuffixTooLong`), Task 3 (`TestCreateTooLong`, `TestRecordNameTooLong`).
- Files in `docket/records/` that are not records (`README.md`, `notes.txt`, a subfolder's files): not listed by the scan, not linted when the branch adds them. Test: Task 2 (`TestScanListsRecords`), Task 4 (`TestLintIgnoresNonRecords`).
- Front matter lines that look like Markdown (a YAML comment `# Task 1 x`, a title starting with `#`): never headings, tasks or sections. Test: Task 4 (`TestRecordFrontMatterIsNotMarkdown`).
- `dkt lint` named with a change and a record together lints both; named with only records lints no change and no spec titles. Test: Task 4 (`TestLintMixedNames`).
- A record the branch added and then deleted, uncommitted: not linted, and no error. Test: Task 4 (`TestLintDeletedRecordSkipped`).

## Reuse

| Need | Existing (path:symbol) | Decision | Reason | Evidence |
|---|---|---|---|---|
| Slug rules, length limit, date prefix | internal/change/new.go:ValidateSlug | extend | export `MaxSlug` and `StartsWithDate`, add `Suffixed`; `dkt record` checks its name with the same rules | `rg -n -e "func ValidateSlug" -e "maxSlug" -e "datePrefix" internal/change/new.go` |
| Kinds `dkt new` accepts | internal/change/new.go:Kinds | extend | drop `fix` | `rg -n "var Kinds" internal/` |
| Where a change slug is used | internal/scan/scan.go:Inventory | reuse | `Inventory.Uses` gives the active change and archived folders for `dkt new`'s suffix loop | `rg -n -F "func (inv Inventory) Uses" internal/scan/scan.go` |
| Read every worktree, branch and the default branch | internal/scan/scan.go:Scan | extend | list record names beside the change folders, in the same pass | `rg -n -F -e "func readDisk" -e "func readTree" -e "func (f *found) add" internal/scan/scan.go` |
| Where a record slug is used | internal/scan/scan.go:Inventory | extend | `Records` and `RecordSources`, like `Changes` and `Uses` | `rg -n -A6 "^type Inventory struct" internal/scan/scan.go` |
| List a commit's files under a folder | internal/gitx/git.go:ListFiles | reuse | recursive, so the scan keeps names directly under `docket/records` | `rg -n -F "func (g Git) ListFiles" internal/gitx/git.go` |
| Resolve the default branch and scan, exit 4 when none | internal/cli/cmd_new.go:scanDefaultBranch | reuse | as `dkt new` does | `rg -n "func scanDefaultBranch" internal/cli/` |
| Report unreadable sources | internal/cli/cmd_conflicts.go:scanUnreadable | reuse | maps `scan.Unreadable` to the envelope | `rg -n "func scanUnreadable" internal/cli/` |
| Name sources in a message | internal/cli/cmd_new.go:sourceNames | reuse | for the `used` places | `rg -n "func sourceNames" internal/cli/` |
| Refuse writing through a link | internal/safepath/safepath.go:Check | reuse | with `refuse` for `symlink-refused` | `rg -n -e "func Check" -e "func refuse" internal/safepath/ internal/cli/` |
| Create a file, never overwriting or following a link | internal/change/docket.go:CreateDocket | new | `record.Create` tries candidate names with `O_CREATE\|O_EXCL\|O_NOFOLLOW`, moves on when one exists and stops at a link; `CreateDocket` writes one fixed path and reads it back as YAML | `rg -n "O_EXCL" internal/` |
| Write and read the front matter | internal/change/docket.go:Docket | reuse | `Docket.Marshal` writes title, kind and source (base is empty and left out); `ParseDocket` reads a mapping and blanks an invalid title | `rg -n -F -e "func ParseDocket" -e "func (d Docket) Marshal" internal/change/docket.go` |
| Title rule | internal/change/docket.go:ValidTitle | reuse | same rule as `dkt new` | `rg -n "func ValidTitle" internal/change/` |
| Split front matter from the body | internal/workflow/workflow.go:recorded | new | `recorded` reads one key and returns nil on any error; a record needs the YAML text, its end line and the reason it is missing | `rg -n "func recorded" internal/workflow/` |
| The clock | internal/cli/run.go:RunWith | reuse | `Context.Now`, set from `Options.Now` | `rg -n "Now" internal/cli/run.go` |
| Embed a shipped template | schema.go:Template | extend | add `RecordTemplate` for `record/template.md` | `rg -n -e "go:embed" -e "func Template" schema.go` |
| Command table | internal/cli/run.go:commands | extend | add `recordCommand` | `rg -n "var commands" internal/cli/run.go` |
| Headings, section bodies, comments, fences | internal/markdown/markdown.go:Headings | reuse | with `Lines`, `FenceMask`, `Body`, `BlankComments` | `rg -n -e "func Headings" -e "func Body" -e "func BlankComments" -e "func FenceMask" internal/markdown/` |
| Task headings and steps | internal/change/plan.go:ParsePlan | reuse | parses `### Task N:` headings and task-like ones anywhere in a file | `rg -n "func ParsePlan" internal/change/` |
| Task heading rule | internal/lint/plan.go:checkPlan | extend | extract `checkTaskHeadings(r, c, file, p)` for plans and records | `rg -n -e "BadHeadings" -e "func checkTaskNumbers" internal/lint/plan.go` |
| Placeholder rule | internal/lint/placeholder.go:checkPlaceholders | extend | extract the per-file check; read the record template beside the schema templates | `rg -n -F -e "func checkPlaceholders" -e "func (r *run) readPlaceholders" internal/lint/placeholder.go` |
| Which names lint looks at | internal/lint/lint.go:Run | extend | split named arguments into changes and records; `Report.Records` | `rg -n -F -e "func (r *run) scope" -e "func (r *run) missing" -e "type Report struct" internal/lint/lint.go` |
| Files the branch adds | internal/cli/cmd_lint.go:addedFiles | extend | take the folders as arguments; called for the routed folders and for `docket/records` | `rg -n -e "func addedFiles" -e "RoutedDirs" internal/cli/cmd_lint.go` |
| Committed and uncommitted additions | internal/gitx/added.go:AddedFiles | reuse | with `UncommittedAdded` | `rg -n -F -e "func (g Git) AddedFiles" -e "func (g Git) UncommittedAdded" internal/gitx/added.go` |
| Count words in a section | none | new | `record.Words`: nothing in the repository counts words | `rg -n -i -e "words" -e "unicode\.Han" internal/` |
| Record kinds | internal/change/new.go:Kinds | new | `record.Kinds`, separate from `change.Kinds` by design | `rg -n "var Kinds" internal/` |
| Test repositories, worktrees, fake OpenSpec | internal/testutil/repo.go:NewRepo | reuse | with `testutil.FakeOpenSpec`, and in package `cli` `cleanRepo`, `newRepo`, `runNew`, `runLint`, `decode`, `hasRule`, `noChange` | `rg -n -e "^func cleanRepo" -e "^func newRepo" -e "^func runNew" -e "^func runLint" -e "^func decode" -e "^func hasRule" -e "^func noChange" internal/cli/` |
| The two skill copies match and ship | schema_test.go:TestInstalledSkillMatchesShipped | reuse | the mechanical check of the skill text | `rg -n "func TestInstalledSkillMatchesShipped" schema_test.go` |
| Run the skill headless in a scratch project | external: claude | reuse | as in the add-doctor trial | `claude --help` |

---

### Task 1: `dkt new` takes a free suffix; the kind `fix` goes

**Scenarios:** change-creation#slug-active-in-another-worktree, change-creation#slug-archived, change-creation#unreadable-worktree-blocks-creation, change-creation#concurrent-runs-keep-the-first, change-creation#missing-source, change-creation#unknown-kind

**Review:** judgment

**Files:**
- Modify: `internal/change/new.go`
- Modify: `internal/cli/cmd_new.go`
- Test: `internal/change/new_test.go`
- Test: `internal/cli/cmd_new_test.go`

**Interfaces:**
- Produces:
  - `change.Kinds` is `[]string{"feature", "change-request", "baseline"}`.
  - `const change.MaxSlug = 200` replaces `maxSlug`.
  - `change.StartsWithDate(s string) bool`: `s` starts with `\d{4}-\d{2}-\d{2}`. `ValidateSlug` uses it.
  - `change.Suffixed(slug string, n int) string`: `slug` when `n <= 1`, else `slug + "-" + n`.
  - `change.NewResult` gains `Requested string` (JSON `requested`) and `Used []string` (JSON `used`, never null).
  - In package `cli`:
    - `usedPlaces(inv scan.Inventory, slug string) []string`: `"active in <sources>"` for the active change, then `"archived as <folder> in <sources>"` for each archived use, as the refusal words them today.
    - `usedLine(requested string, used []string, created string) string`: `<requested> is used: <used joined by "; ">; created <created>`.

- [x] **Step 1: Extract the used-slug places (structural)**
  - In `cmd_new.go`, move the building of `parts` into `usedPlaces`. The refusal still reads `slug %q is already used: <parts joined by "; ">`.
  - Run `go test ./internal/cli/ -run 'TestSlug|TestUnreadable|TestConcurrent'` and `go vet ./...`. Expected: PASS, with no test file changed.
  - Commit as `refactor(new): build the used-slug places in one function`, with the trailers.

- [x] **Step 2: Write the failing tests**

  **`internal/change/new_test.go`:**
  - `TestSuffixed`: `("add-x", 0)` and `("add-x", 1)` give `add-x`; `("add-x", 2)` gives `add-x-2`; `("add-x", 12)` gives `add-x-12`.
  - `TestStartsWithDate`: true for `2026-10-30-fix-x` and `2026-10-30`; false for `add-x`, `2026-10-x`, `20261030-x`.

  **`internal/cli/cmd_new_test.go`**, changing the existing tests where named:
  - `TestUnknownKind`: `--kind fix` exits 2; stderr holds `use one of feature, change-request, baseline`.
  - `TestNewInPathWithSpaces`: `--kind feature` instead of `fix`; otherwise unchanged.
  - `TestSlugActiveInThisWorktree`: worktree A itself holds `openspec/changes/add-x/` before the run. Exit 0, `result.slug` `add-x-2`, `result.used` is exactly one string, `active in <A>`.
  - `TestSlugActiveInAnotherWorktree`:
    - exit 0; `result.slug` is `add-x-2`, `result.requested` is `add-x`;
    - `result.used` is one string holding `active in` and worktree B's path;
    - `openspec/changes/add-x-2/docket.yaml` exists in A, and `openspec/changes/add-x` does not;
    - without `--json`, stdout has the line `add-x is used: active in <B>; created add-x-2`.
  - `TestSlugArchived`: exit 0; `result.slug` is `add-x-2`; `result.used` holds `archived as 2025-01-02-add-x`.
  - `TestSlugSuffixSkipsUsed`: worktree B holds `add-x` and worktree C holds `add-x-2`. Exit 0, `result.slug` is `add-x-3`, `result.used` names only B.
  - `TestUnreadableSourceWithUsedSlug` (replaces `TestUnreadableSourceWithOtherFindings`): B holds `add-x`, C is removed. Exit 3; `unreadable` names only C; no findings; `noChange`.
  - `TestConcurrentRunsKeepTheFirst`: same setup as today. The run exits 0 with `result.slug` `add-x-2`; `result.used` is `["openspec/changes/add-x exists in this worktree"]`; `marker.txt` still holds `first\n`; the fake OpenSpec log has a line starting `new change add-x-2` and none equal to `new change add-x --schema docket`.
  - `TestChangeCreated`: also `result.requested` is `add-x` and `result.used` is an empty array (decode and check it is `[]any{}`, not nil).
  - `TestNewSuffixTooLong`: a 199-character slug held by worktree B. Exit 1, `slug-invalid` whose message holds `201`; `noChange`.
  - `TestUnreadableWorktreeBlocksCreation` and the missing-source cases of the usage test stay as they are and must pass.

- [x] **Step 3: Run them to see them fail**

  Run: `go test ./internal/change/ -run 'TestSuffixed|TestStartsWithDate'` and `go test ./internal/cli/ -run 'TestUnknownKind|TestSlug|TestUnreadableSource|TestConcurrent|TestChangeCreated|TestNewSuffix'`.
  Expected: FAIL (`undefined: Suffixed`, exit 1 where 0 is expected).

- [x] **Step 4: Implement**
  - `change.go` items as in Interfaces.
  - In `newCommand`:
    - Usage and flag help list the three kinds.
    - Drop the pre-lock refusal; keep `out.Unreadable` and return it when not empty.
    - Under the lock, for `n := 1, 2, ...`: `cand := change.Suffixed(slug, n)`. `ValidateSlug(cand)` failing gives `slug-invalid` with its message. A candidate is taken when `inv.Uses(cand)` reports anything or `Lstat(openspec/changes/<cand>)` finds an entry; a non-`ErrNotExist` Lstat error is returned. Take the first candidate that is not taken.
    - `used` is `usedPlaces(inv, slug)`; only when that is empty and the Lstat found the requested slug is it `["openspec/changes/<slug> exists in this worktree"]`. Each place appears once. `used` is `[]string{}` when nothing was found.
    - The `safepath.Check`, the OpenSpec run, `CreateDocket` and the cleanup use the chosen slug, unchanged otherwise.
    - Human output: `usedLine(slug, used, chosen)` first when the chosen slug differs, then `created <dir>`.

- [x] **Step 5: Run the tests to see them pass**

  Run: `go test ./internal/change/ ./internal/cli/` and `go vet ./...`.
  Expected: PASS.

- [x] **Step 6: Commit**

  Commit as `feat(new): take a free suffix for a used slug; drop the kind fix`, with the trailers.

### Task 2: The scan lists records

**Scenarios:** none

**Review:** judgment

**Files:**
- Create: `internal/record/record.go`
- Test: `internal/record/record_test.go`
- Modify: `internal/scan/scan.go`
- Test: `internal/scan/scan_test.go`

**Interfaces:**
- Consumes: Task 1's `change.StartsWithDate`.
- Produces:
  - `const record.Dir = "docket/records"`.
  - `record.SlugOf(name string) (string, bool)`: for a file name (no folder), the name without `.md` when it is a date, a hyphen and at least one more character, then `.md`; else false.
  - `scan.Record{Slug string; Sources []Source}`, `Sources[0]` ranked as for changes.
  - `scan.Inventory.Records []Record`, sorted by slug, never nil.
  - `(inv Inventory) RecordSources(slug string) []Source`: the sources holding the record, nil when none.

- [x] **Step 1: Write the failing tests**
  - `internal/record/record_test.go: TestSlugOf`: `2026-10-30-fix-x.md` gives `2026-10-30-fix-x`; false for `README.md`, `2026-10-30-fix-x.txt`, `.md`, `2026-10-30-fix-x`, `2026-10-30.md`, `2026-10-30-.md`, `notes.md`.
  - `internal/scan/scan_test.go`:
    - **`TestScanListsRecords`**:
      - the default branch commits `docket/records/2026-10-01-fix-z.md`;
      - unmerged branch `b` commits `docket/records/2026-10-29-fix-y.md` and `docket/records/sub/2026-10-29-deep.md`;
      - the main worktree has, uncommitted, `docket/records/2026-10-30-fix-x.md`, `docket/records/README.md`, `docket/records/2026-10-30-notes.txt`, and `docket/records/2026-10-30-link.md`, a symbolic link to `2026-10-30-fix-x.md`.
      - `inv.Records` slugs are exactly `2026-10-01-fix-z`, `2026-10-29-fix-y`, `2026-10-30-fix-x`.
      - `RecordSources("2026-10-30-fix-x")` is the worktree; `RecordSources("2026-10-29-fix-y")` is `refs/heads/b`; `RecordSources("2026-10-01-fix-z")` includes the default branch; `RecordSources("2026-10-30-none")` is nil.
      - `inv.Unreadable` is empty.
    - **`TestScanNoRecordsFolder`**: a repository with no `docket/`. `inv.Records` is empty and not nil; no unreadable.
    - **`TestScanRecordsUnreadable`** (skipped as root): worktree B with `openspec/changes/add-x/` and `docket/records/` at mode `000`. `inv.Unreadable` has one entry for B with `Path` `docket/records`; `add-x` is still in `inv.Changes`.
    - **`TestScanRecordsNotAFolder`**: branch `b` commits a file named `docket/records`. One unreadable for `refs/heads/b` with `Path` `docket/records`.
  - Existing scan tests that compare whole inventories get `Records: []Record{}`.

- [x] **Step 2: Run them to see them fail**

  Run: `go test ./internal/record/ ./internal/scan/`.
  Expected: FAIL (`undefined: SlugOf`, `inv.Records undefined`).

- [x] **Step 3: Implement**
  - `record.go`: package comment "Package record holds the rules of records: one file per piece of light-review work under docket/records." Then `Dir` and `SlugOf`.
  - In `scan.go`:
    - `readDisk` and `readTree` also return the record slugs and a record error.
      - Disk: `os.ReadDir(<root>/docket/records)`. A missing folder is no records; another error is the record error. Keep regular files (`e.Type().IsRegular()`) that pass `SlugOf`, as `ListFiles` keeps only regular files in a tree.
      - Tree: `g.ListFiles(rev, record.Dir)`; keep paths with no `/` after `docket/records/` that pass `SlugOf`. An error is the record error.
    - A record error adds `Unreadable{Source: src, Path: record.Dir, Reason: err.Error()}`; the source's changes are still added.
    - `found` gains `records map[string][]Source`; `add` takes the record slugs. Build `inv.Records` like `inv.Changes`.

- [x] **Step 4: Run the tests to see them pass**

  Run: `go test ./internal/record/ ./internal/scan/ ./internal/cli/` and `go vet ./...`.
  Expected: PASS. The `cli` run shows no command's output changed.

- [x] **Step 5: Commit**

  Commit as `feat(scan): list the records of every source`, with the trailers.

### Task 3: `dkt record` writes a record from the shipped template

**Scenarios:** records#record-written, records#unknown-kind, records#name-with-a-date, records#same-name-same-day, records#unreadable-source-blocks-the-record, records#concurrent-records, records#symlinked-records-folder-refused

**Review:** judgment

**Files:**
- Create: `record/template.md`
- Modify: `schema.go`
- Test: `schema_test.go`
- Modify: `internal/record/record.go`
- Test: `internal/record/record_test.go`
- Create: `internal/cli/cmd_record.go`
- Modify: `internal/cli/run.go`
- Create: `internal/cli/cmd_record_test.go`

**Interfaces:**
- Consumes: Task 1's `change.Suffixed`, `change.MaxSlug`, `change.StartsWithDate`, `usedLine`; Task 2's `record.Dir`, `Inventory.RecordSources`.
- Produces:
  - `record.Kinds = []string{"fix", "refactor", "test", "prompt", "checks"}`.
  - `record.Sections = []string{"What", "Why", "Verification", "Review"}`.
  - `record.ValidateName(name string) error`:
    - a name starting with a date: `name %q must not start with a date: dkt record adds the date`;
    - otherwise `change.ValidateSlug(name)`;
    - then a name longer than `change.MaxSlug - 11` (the date and its hyphen): the slug-length message for the full slug.
  - `record.Slug(now time.Time, name string) string`: `now.Format("2006-01-02") + "-" + name`.
  - `record.Render(title, kind, source, template string) ([]byte, error)`: `---\n`, `change.Docket{Title, Kind, Source}.Marshal()`, `---\n\n`, then the template.
  - `record.Create(dir, slug string, used func(slug string) bool, content []byte) (string, error)`:
    - tries `change.Suffixed(slug, n)` for `n = 1, 2, ...`;
    - skips a candidate when `used` says so, or when creating `<dir>/<candidate>.md` with `O_WRONLY|O_CREATE|O_EXCL|O_NOFOLLOW` (mode `0644`) fails with `fs.ErrExist` and `Lstat` shows no symbolic link there;
    - a candidate that is a symbolic link stops it with a `*record.LinkError{Name: "<candidate>.md"}`, so nothing is written past a link;
    - a candidate longer than `change.MaxSlug` stops it with an error wrapping `record.ErrTooLong`, naming the length;
    - writes `content` and returns the candidate written; when the write fails it removes the file it created.
  - `var record.ErrTooLong`; `record.LinkError{Name string}` with an `Error` method naming the file.
  - `record.Result{Slug, Path, Title, Kind, Source, Requested string; Used []string}`, JSON keys `slug`, `path`, `title`, `kind`, `source`, `requested`, `used`.
  - `docket.RecordTemplate() string`, from `//go:embed record/template.md`.
  - `recordCommand()` in package `cli`, registered after `newCommand` in `commands`.

- [x] **Step 1: Write the failing tests**

  **`schema_test.go: TestRecordTemplate`**: `RecordTemplate()` has no front matter; its level-two headings are `record.Sections` in order; each section holds one HTML comment and nothing else; one comment comes before the first heading.

  **`internal/record/record_test.go`:**
  - `TestValidateName`: `fix-x` passes; `2026-10-30-fix-x` fails holding `dkt record adds the date`; `Fix_X` fails; a 189-character name passes and a 190-character one fails holding `200`.
  - `TestSlugUsesTheTimesLocation`: `Slug(time.Date(2026, 10, 31, 1, 0, 0, 0, time.FixedZone("", 8*3600)), "fix-x")` is `2026-10-31-fix-x`.
  - `TestRender`: the front matter parses with `change.ParseDocket` to the three values, no `base`; a title holding `: ` and `#` reads back unchanged; the rest is the template byte for byte.
  - `TestCreate`, in a temporary folder:
    - free: writes `x.md` with the content, returns `x`;
    - `used("x")` true: writes `x-2.md`;
    - `x.md` exists and `used` is always false: returns `x-2`, `x.md` unchanged;
    - `x` used and `x-2.md` existing: returns `x-3`;
    - `x` used and `x-2.md` a symbolic link to a file outside the folder: a `*LinkError` naming `x-2.md`; the target unchanged and no `x-3.md`.
  - `TestCreateTooLong`: a 199-character slug that `used` reports: `errors.Is(err, ErrTooLong)`, nothing written.

  **`internal/cli/cmd_record_test.go`**, with `runRecord(t, dir, env, args...)` passing `Now: 2026-10-30 09:00 UTC`:
  - **`TestRecordWritten`**: in `cleanRepo`, `dkt record fix-lint-timeout --kind fix --title "Lint timeout" --source "review of #40" --json`.
    - Exit 0.
    - `docket/records/2026-10-30-fix-lint-timeout.md` holds the three values in its front matter and `RecordTemplate()` after it.
    - `result.slug` is `2026-10-30-fix-lint-timeout`, `result.path` is `docket/records/2026-10-30-fix-lint-timeout.md`, `result.requested` equals the slug, `result.used` is `[]`.
    - Without `--json`, stdout is `created docket/records/2026-10-30-fix-lint-timeout.md`.
  - **`TestRecordUsage`**: each exits 2 and creates no `docket/`:
    - `--kind chore` (stderr lists all five kinds);
    - no `--kind`; no `--source`; `--source "  "`; no `--title`; a title holding a newline; no name; two names.
  - **`TestRecordNameWithADate`**: `2026-10-30-fix-x`. Exit 1, `slug-invalid` holding `dkt record adds the date`, no `docket/`.
  - **`TestRecordNameTooLong`**: a 190-character name. Exit 1, `slug-invalid`.
  - **`TestRecordSameNameSameDay`**: worktree B holds `docket/records/2026-10-30-fix-x.md`, uncommitted. In A: exit 0, `result.slug` `2026-10-30-fix-x-2`, the file exists in A, `result.used` is one string `in <B's path>`; without `--json`, the `is used` line names B.
  - **`TestRecordUsedOnABranch`**: unmerged branch `b` commits `docket/records/2026-10-30-fix-x.md`. `result.slug` is `2026-10-30-fix-x-2`, `result.used` names `refs/heads/b`.
  - **`TestRecordUnreadableSource`**: worktree B removed. Exit 3, `unreadable` names B, no `docket/` in A.
  - **`TestRecordConcurrent`**: three goroutines run the same `dkt record fix-x ...` in one worktree. All exit 0; the slugs are exactly `2026-10-30-fix-x`, `-2`, `-3`; each file is complete.
  - **`TestRecordSymlinked`**, one sub-test each; every one exits 1 with `symlink-refused` naming the link, and writes nothing on either side of it:
    - `docket/records` a symbolic link to a temporary folder outside the repository (the outside folder stays empty, nothing else is under `docket/`);
    - `docket` a symbolic link to such a folder;
    - `docket/records/2026-10-30-fix-x.md` a symbolic link to a file outside (the file is unchanged);
    - worktree B holds `docket/records/2026-10-30-fix-x.md`, and A has `docket/records/2026-10-30-fix-x-2.md` as a symbolic link: the finding names `docket/records/2026-10-30-fix-x-2.md`.
  - **`TestRecordNoDefaultBranch`**: a repository set up as `TestNoDefaultBranchForRouting` does (only a branch `topic`), without `--default-branch`. Exit 4, no `docket/`.

- [x] **Step 2: Run them to see them fail**

  Run: `go test . -run TestRecordTemplate`, `go test ./internal/record/`, `go test ./internal/cli/ -run TestRecord`.
  Expected: FAIL (`undefined: RecordTemplate`, `unknown command "record"`).

- [x] **Step 3: Write the template**

  `record/template.md`, exactly:

  ```markdown
  <!-- Each section: at most 60 words, outside comments; the details belong
  in the code and the pull request. Replace every comment with your text. -->

  ## What

  <!-- What changed, in a few sentences: the files or the behavior touched.
  fix: the behavior restored. refactor: what moved or was restructured.
  test: which behavior the new tests cover. prompt: which text changed, and
  which copies. checks: which checks run differently. -->

  ## Why

  <!-- Why it was needed. fix: the symptom and its cause. refactor: what the
  old shape made hard. test: what was not covered. prompt: what agents did
  wrong or could not do. checks: what the old settings missed. -->

  ## Verification

  <!-- How it was checked and the result, by the light review steps of the
  workflow file: the regression test seen failing, then passing; the
  mutation that made a test fail; the mechanical check of the prompt; the
  pull request's own check run. -->

  ## Review

  <!-- Who reviewed the diff (a model, or a person), what the review found,
  and how each finding was handled. -->
  ```

- [x] **Step 4: Implement**
  - `record.go` items as in Interfaces.
  - `cmd_record.go`, `recordCommand()`:
    - `Name: "record"`, `Usage: "record <name> --kind <fix|refactor|test|prompt|checks> --source <text> --title <text>"`, flags `--kind`, `--source`, `--title` as in `newCommand`.
    - Argument checks as `dkt new` makes them, with `record.Kinds`, giving `UsageError`; `--source` must not be blank after trimming.
    - `record.ValidateName` failing gives `slug-invalid`.
    - `slug := record.Slug(c.Now, name)`; `rel := record.Dir + "/" + slug + ".md"`.
    - `safepath.Check(c.Repo.Root, rel)` before anything else touches the disk; a link gives `refuse`.
    - `scanDefaultBranch`; unreadable sources are returned, nothing written.
    - `os.MkdirAll` of the records folder, then `record.Create` with `used` from `inv.RecordSources(cand) != nil` and `record.Render(title, kind, source, docket.RecordTemplate())`. `ErrTooLong` gives `slug-invalid`; a `*LinkError` gives `symlink-refused` naming `docket/records/<name>`.
    - `used`: `"in " + sourceNames(inv.RecordSources(slug))` when the scan has the slug; else `"<rel> exists in this worktree"` when the written slug differs; else `[]string{}`.
    - Human output: `usedLine(slug, used, written)` when the written slug differs, then `created <path>`.

- [x] **Step 5: Run the tests to see them pass**

  Run: `go test ./...` and `go vet ./...`.
  Expected: PASS.

- [x] **Step 6: Commit**

  Commit as `feat(record): dkt record writes a record from the shipped template`, with the trailers.

### Task 4: `dkt lint` checks records

**Scenarios:** records#record-kind-not-known, records#record-section-empty, records#template-text-left, records#section-over-the-limit, records#chinese-text-counted, records#record-tasks-checked, records#record-without-tasks, change-lint#lint-every-active-change, change-lint#named-change-missing, change-lint#no-openspec-installed, change-lint#lint-records-the-branch-adds

**Review:** judgment

**Files:**
- Modify: `internal/record/record.go`
- Create: `internal/record/parse.go`
- Test: `internal/record/parse_test.go`
- Modify: `internal/lint/lint.go`, `internal/lint/plan.go`, `internal/lint/placeholder.go`
- Create: `internal/lint/record.go`
- Create: `internal/lint/record_test.go`
- Modify: `internal/cli/cmd_lint.go`
- Test: `internal/cli/cmd_lint_test.go`

**Interfaces:**
- Consumes: Task 2's `record.Dir`, `record.SlugOf`; Task 3's `record.Kinds`, `record.Sections`, `docket.RecordTemplate`.
- Produces:
  - `const record.TasksSection = "Tasks"`, `const record.WordLimit = 60`.
  - `record.Words(text string) float64`, counted as Global Constraints says.
  - `record.Parse(b []byte) record.File`:
    - `File{FrontMatter change.Docket; FrontMatterErr error; Sections []Section; Plan change.Plan}`;
    - `Section{Name string; Line int; Text string}`: every level-two heading after the front matter, in order; `Line` is 1-based; `Text` is the body up to the next heading of level one or two, HTML comments blanked;
    - the front matter is lines 1 to the next `---` line when line 1 is `---`; `FrontMatterErr` says it is missing, or is `change.ParseDocket`'s error;
    - front matter lines and HTML comments (`markdown.BlankComments`) are blanked before headings and `change.ParsePlan` are read, so a heading inside a comment is not a section or a task, and line numbers stay.
  - `lint.Input` gains `AddedRecords []string` (repository paths the branch adds under `docket/records`), `AddedRecordsErr error`, `RecordTemplate string`.
  - `lint.Report` gains `Records []string`, sorted.
  - `checkTaskHeadings(r *run, c *loaded, file string, p change.Plan)` and `checkPlaceholderFile(r *run, c *loaded, name string, b []byte)` in `internal/lint`.
  - `addedFiles(c *Context, dirs ...string) ([]string, error)` in package `cli`.
  - `LintResult` gains `Records []string` (JSON `records`, never null).

- [x] **Step 1: Share the task heading and placeholder checks (structural)**
  - `plan.go`: move the `BadHeadings` loop and the number check into `checkTaskHeadings(r, c, file, p)`; `checkPlan` calls it with `planFile` and `c.Plan`.
  - `placeholder.go`: move the body of the per-file loop into `checkPlaceholderFile`; `checkPlaceholders` calls it for each file.
  - `cmd_lint.go`: `addedFiles(c, dirs...)`; the call passes `lint.RoutedDirs...`.
  - Run `go test ./internal/lint/ ./internal/cli/` and `go vet ./...`. Expected: PASS, with no test file changed.
  - Commit as `refactor(lint): share the task heading and placeholder checks`, with the trailers.

- [x] **Step 2: Write the failing tests**

  **`internal/record/parse_test.go`:**
  - `TestWords`, table: `""` 0; `"one two"` 2; `"don't"` 2; `"v0.9.0 dkt-record"` 5; `"中文字"` 1.5; `"中文abc"` 2; `"かなカナ"` 2; `"한국"` 1; `"2026"` 1; `"- [ ] **bold** text"` 2.
  - `TestParse`: front matter, `## What` with a comment and text, `## Tasks` with `### Task 1: Move`. Sections, lines and texts as written; `Plan.Tasks[0].Line` is the heading's file line; no comment text in `Text`.
  - `TestParseNoFrontMatter`: `FrontMatterErr` not nil; sections still read.

  **`internal/lint/record_test.go`**, through `Run` with an `fstest.MapFS` holding `docket/records/<slug>.md`, the slug in `AddedRecords` as `docket/records/<slug>.md`, and `RecordTemplate: docket.RecordTemplate()`. A helper `goodRecord()` returns a record that gives no finding.
  - `TestGoodRecord`: no finding; `Report.Records` is the slug.
  - `TestRecordFrontMatter`, one sub-test each, every finding on line 1 with `File` the record path and `Change` empty:
    - no front matter; front matter `- a` (not a mapping): `record-front-matter`;
    - no title; a blank `source`: `record-front-matter`;
    - `kind: chore`, and no `kind`: `record-kind` (scenario record-kind-not-known).
  - `TestRecordSections`, one sub-test each, rule `record-section`, the message naming the section:
    - `## Why` missing (no line);
    - an extra `## Notes` (on its line);
    - `## Why` before `## What` (on the later of the two);
    - `## Why` twice;
    - `## Tasks` before `## Review`;
    - `## Review` with nothing under it (scenario record-section-empty: names `Review`, on its heading line).
  - `TestRecordTemplateTextLeft`: `## Why` holds only its template comment. `placeholder` on the comment's line, and `record-section` naming `Why`.
  - `TestRecordTooLong`: `## What` with 61 English words gives `record-too-long` naming `What` and `61`, on its heading line; 60 words give none; a comment of 100 words beside 10 words gives none.
  - `TestRecordChineseCounted`: `## Why` with 110 Chinese characters and 4 English words gives no `record-too-long`; 121 Chinese characters give one naming `60.5`.
  - `TestRecordTasks`:
    - `## Tasks` with `### Task 1: Move` and `### Task 1: Rename`: `task-heading` on the second, its message naming both lines (scenario record-tasks-checked);
    - `### Task two: x` under `## Tasks`: `task-heading`;
    - `### Task 1: x` under `## What`: `task-heading` saying task headings belong under `## Tasks`;
    - no `## Tasks` section: no task finding (scenario record-without-tasks).
  - `TestRecordFrontMatterIsNotMarkdown`: front matter holding the YAML comment `# Task 1 x` and `title: "## Why"`, otherwise good: no finding.
  - `TestRecordCommentedHeadings`: a good record whose `## What` also holds `<!-- ## Notes -->` and a comment spanning `### Task 1: x` and `### Task 1: y`: no finding.
  - `TestLintRecordUnreadable`: a `failingFS` that fails to open the record: one unreadable naming its path; the slug is not in `Report.Records`; no finding for it.
  - `TestLintIgnoresNonRecords`: `AddedRecords` holds `docket/records/README.md` and `docket/records/sub/2026-10-30-x.md`: `Report.Records` empty, no finding.
  - `TestLintDeletedRecordSkipped`: `AddedRecords` names a record not in the file system: not linted, no finding, no unreadable.
  - `TestLintNamedRecord`: `Slugs: ["2026-10-01-fix-a"]`, a bad record not in `AddedRecords`: it is linted.
  - `TestLintRecordMissing`: `Slugs: ["2026-10-30-fix-z"]`: `record-missing` naming it; also for a name that is a symbolic link and for `2026-10-30-a/b`.
  - `TestLintMixedNames`: `Slugs: ["add-x", "2026-10-30-fix-b"]` with both present: `Report.Changes` is `add-x`, `Report.Records` is the record. `Slugs: ["2026-10-30-fix-b"]` alone: `Report.Changes` empty and no spec title finding though a main spec has a bad title.
  - `TestLintRecordsUnknown`: `AddedRecordsErr` set and no names: one unreadable with source `records`.

  **`internal/cli/cmd_lint_test.go`:**
  - **`TestLintRecordsTheBranchAdds`**: the default branch commits a bad `docket/records/2026-10-01-fix-a.md`; branch `topic` commits a good `2026-10-30-fix-b.md` and has a good `2026-10-30-fix-c.md` uncommitted. `dkt lint --json`: exit 0, `result.records` is `["2026-10-30-fix-b", "2026-10-30-fix-c"]`, no finding names `fix-a`.
  - `TestLintEveryActiveChangeJSON` also checks `result.records` is `[]`.
  - `TestNoDefaultBranchForRouting`: `unreadable` is now exactly `routing` and `records`, each with the `--default-branch` hint.
  - `TestLintNamedChanges`, `TestNoOpenSpecInstalled` and the named-change-missing case stay and pass.

- [x] **Step 3: Run them to see them fail**

  Run: `go test ./internal/record/ ./internal/lint/` and `go test ./internal/cli/ -run TestLint`.
  Expected: FAIL (`undefined: Words`, `unknown field AddedRecords`).

- [x] **Step 4: Implement**
  - `parse.go`: `Words`, `Parse` as in Interfaces.
  - `lint.go`:
    - With names, a name for which `change.StartsWithDate` holds is a record name; the others go through `scope` as today. Without names, the records are `AddedRecords` filtered to `docket/records/<name>` with `record.SlugOf(name)`, sorted, without duplicates.
    - A record is looked up among the entries of `docket/records`, as `changeEntry` does for changes. A named record that is missing, not a regular file, or not a valid path gives `record-missing`: `record %q does not exist under docket/records`. An unnamed one in that state is skipped.
    - `AddedRecordsErr` without names gives `r.unreadable("records", err)`.
    - Each record: read it; a read error gives `r.unreadable(<path>, err)` and the record is left out. Otherwise append its slug to `Report.Records` and run `checkRecord` with `&loaded{Dir: record.Dir, Files: {<slug>.md: b}}`, so findings carry the path and no change.
    - Spec titles are checked only when no name is given, as today.
  - `placeholder.go`: `readPlaceholders` also collects the placeholders of `in.RecordTemplate`, as from a template named `record/template.md`, when it is not empty.
  - `record.go` in `internal/lint`, `checkRecord(r, c, f record.File)`:
    - **Front matter** (line 1): `record-front-matter` for `FrontMatterErr`, an empty title (`ParseDocket` blanks an invalid one: "front matter needs a `title` on one line"), a blank source; `record-kind` for a kind not in `record.Kinds`, naming the five.
    - **Sections:** `record-section` for a level-two heading that is not one of the four or `Tasks`; a known section repeated, or out of the order What, Why, Verification, Review, Tasks; each of the four that is missing; each of the four whose `Text` is blank.
    - **Words:** `record-too-long` for each of the four over `record.WordLimit`: "`## What` has 61 words; the limit is 60", the count formatted with `strconv.FormatFloat(w, 'f', -1, 64)`.
    - **Tasks:** `checkTaskHeadings(r, c, <slug>.md, f.Plan)`; then `task-heading` for each task heading above the `## Tasks` heading, or in a record without one.
    - **Placeholders:** `checkPlaceholderFile`.
  - `cmd_lint.go`: a second `addedFiles(c, record.Dir)` call feeds `AddedRecords` and `AddedRecordsErr`; `RecordTemplate: docket.RecordTemplate()`; `LintResult{Changes, Records}`, both `[]string{}` when empty.

- [x] **Step 5: Run the tests to see them pass**

  Run: `go test ./...` and `go vet ./...`.
  Expected: PASS.

- [x] **Step 6: Commit**

  Commit as `feat(lint): check the records the branch adds`, with the trailers.

### Task 5: The `docket` skill keeps a record on the light review path

**Scenarios:** records#light-review-opens-a-record

**Review:** checks

**Files:**
- Modify: `.agents/skills/docket/SKILL.md`
- Modify: `.claude/skills/docket/SKILL.md`

- [x] **Step 1: Edit `.agents/skills/docket/SKILL.md`**
  - **Full review, step 1:** the command becomes `dkt new <slug> --kind feature|change-request|baseline --source "<where the request came from>" --title "<one line>"`, and after the sentence on `docket.yaml` add: "When the slug is taken, `dkt new` creates the change under the first free suffix (`-2`, `-3`, ...) and reports it; use the slug it reports."
  - **`## Light review`** becomes, exactly:

    ```markdown
    ## Light review

    No change is opened; a record in `docket/records/` says what the work
    did. In a git worktree:

    1. Before the first commit, open the record:
       `dkt record <name> --kind fix|refactor|test|prompt|checks --source "<where the request came from>" --title "<one line>"`.
       The kind follows triage: a fix (`fix`), a restructuring
       (`refactor`), tests (`test`), prompt text (`prompt`), check settings
       (`checks`). Use the slug `dkt record` reports: it adds the date, and
       a suffix when the name is taken.
    2. Make the edit following the steps under `## Light review` in the
       workflow file. Fill the record's sections as the work and its review
       go, in a few sentences each, replacing the comments.
    3. Every commit for the work carries `Record: <slug>` in the final
       paragraph of its message. Larger work may list tasks under
       `## Tasks` in the plan's format; a commit for one also carries
       `Task: <N>`.
    4. Run `dkt lint` and fix what it reports. Then open the pull request
       and merge it with a merge commit.
    ```

- [x] **Step 2: Copy and check**
  - Copy the file to `.claude/skills/docket/SKILL.md`.
  - Run: `cmp .agents/skills/docket/SKILL.md .claude/skills/docket/SKILL.md`. Expected: no output.
  - Run: `go test . -run 'TestSkill|TestInstalledSkill|TestBlock'`. Expected: PASS (the frontmatter still loads, and the installed copy is the shipped one).

- [x] **Step 3: Commit**

  Commit as `docs(skill): keep a record on the light review path`, with the trailers.

### Task 6: Docket's docs

**Scenarios:** none

**Review:** checks

**Files:**
- Modify: `README.md`, `README.zh-TW.md`
- Modify: `docs/ARCHITECTURE.md`
- Modify: `AGENTS.md`
- Modify: `docket/ideas/2026-10-29-records.md`
- Modify: `docket/ideas/2026-10-29-framework-skill.md`

- [x] **Step 1: Edit the docs**
  - **`README.md`**:
    - the command list: `dkt new <slug> --kind feature|change-request|baseline ...`; add `dkt record <name> --kind fix|refactor|test|prompt|checks --source "<origin>" --title "<title>"   # a record for light-review work`; `dkt lint [<slug>|<record>...]`;
    - where the flow is described, one sentence: light-review work opens no change but keeps a record in `docket/records/`, written by `dkt record` and checked by `dkt lint`, and its commits carry `Record: <slug>`;
    - one sentence that `dkt new` and `dkt record` take a free `-2` suffix when the slug is used;
    - the role table's Check row: `lint` also checks records (front matter, sections, word limits);
    - the paragraph on the commands that write files adds `record` (a record under `docket/records/`).
  - **`README.zh-TW.md`**: the same edits in Traditional Chinese.
  - **`docs/ARCHITECTURE.md`**:
    - Codemap: a row for `internal/record`: "Records: kinds, name and slug, the template rendering, creating a file under a free name, parsing and word counting"; `schema.go` also embeds the record template.
    - Capability index, new rows: "Pick a free slug" → `change.Suffixed` with `scan.Inventory.Uses` or `RecordSources`; "Where a record slug is used" → `scan.Inventory.RecordSources`; "Count a record section's words" → `record.Words`; "Check task headings in any file" → `lint.checkTaskHeadings`.
  - **`AGENTS.md`**, outside the Docket block: the status line lists `dkt record`; the prompt paths add `record/template.md`.
  - **`docket/ideas/2026-10-29-records.md`**: add `## Status`: "`add-records` made records exist: `dkt record`, the template, lint, the skill text. It settled some points differently from Decided: slugs carry the date and take a `-2` suffix instead of being refused; a fifth kind, `checks`; one template with four shared sections, each limited to 60 words, instead of sections per kind. Tracking remains for a second change: the scan reads records, `dkt sync` ticks their tasks, `dkt status` lists them, and `Record:` trailers are read."
  - **`docket/ideas/2026-10-29-framework-skill.md`**: in the open question on queued ideas, note that `add-records` settled the records idea.

- [x] **Step 2: Verify**

  Run: `rg -n "change-request\|fix\|baseline" README.md README.zh-TW.md .agents .claude --hidden`.
  Expected: no output.

  Run: `go test . -run 'TestBlock|TestInstalledSkill'`.
  Expected: PASS. The Docket block is unchanged.

- [x] **Step 3: Commit**

  Commit as `docs: records and dkt record`, with the trailers.

### Task 7: Trial of the light review path in a scratch project

**Scenarios:** records#light-review-opens-a-record

**Review:** checks

**Files:**
- Modify, only if the trial calls for new wording: `.agents/skills/docket/SKILL.md`, `.claude/skills/docket/SKILL.md`, `record/template.md`

The lead runs this task.
- Transcripts and a results table go in the scratchpad; the summary goes in the pull request.
- Every Claude Code run:
  - is `claude -p` with `--output-format stream-json --verbose`, `--setting-sources project,local`, and `--plugin-dir` pointing at the installed superpowers;
  - runs on two models, the session's default and Sonnet;
  - has the branch's `dkt`, built into the scratchpad, first on `PATH`;
  - has `XDG_CONFIG_HOME` set to a scratch directory, so the user's own instructions and workflow file are never read or written.

- [x] **Step 1: Set up a scratch project**
  - A new git repository on `main` holding a README and `prompts/greeting.md`, a prompt with the typo `recieve`, listed as a prompt path in the project's `AGENTS.md`. No remote is configured, so a run may stop at the pull request; that is not a failure.
  - Run the branch's `dkt init` and commit.
  - Write `<XDG_CONFIG_HOME>/docket/workflow.md` with the current template's hash and text, with Light review step 2 replaced by "Have the whole diff reviewed once by running `./review-light.sh`." The script appends its arguments to a log in the scratchpad and prints `no findings`.

- [x] **Step 2: Run the light review**

  Prompt "Fix the typo in prompts/greeting.md.", with `--max-turns 40` and the tools the work needs.

  Expected, for each model:
  - the agent states the light review path;
  - a Bash command running `dkt record` with `--kind prompt` comes before the first `git commit`;
  - `docket/records/<date>-<name>.md` exists with all four sections filled and no template comment, and `dkt lint` reports nothing on it;
  - every commit made carries `Record: <slug>` naming that record.

- [x] **Step 3: Adjust and rerun on failure**

  When a run fails its expectation:
  1. change the wording;
  2. run `go test ./...`;
  3. rebuild the scratchpad `dkt`, and rerun both models.

  Commit any change, with the trailers.
