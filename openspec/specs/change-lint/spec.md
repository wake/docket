# Change Lint Specification

## Purpose
Defines the rules `dkt lint` checks in a change and in the main specs' titles, so that the rules joining superpowers and OpenSpec fail visibly when they are broken instead of living only in prompts.

## Requirements

### Requirement: What is linted
`dkt lint` SHALL lint each change named on the command line, or, when none is named, every active change in the worktree's `openspec/changes/`, then the title of every main spec under `openspec/specs/`, then the domain tree as `domain-tree` defines. It SHALL also lint records and issues, as their specs define. It SHALL read the worktree's files as they are, uncommitted edits included, and SHALL report `change-missing` for a named change that does not exist. A name that is empty, holds `/` or `\`, or is `.` or `..` SHALL be a usage error.

#### Scenario: Lint every active change
- **WHEN** the worktree has the active changes `add-x` and `add-y` and a user runs `dkt lint`
- **THEN** both changes are linted, the archive folder is not, and the main specs' titles are checked

#### Scenario: Named change missing
- **WHEN** a user runs `dkt lint add-z` and the worktree has no change `add-z`
- **THEN** the finding `change-missing` names `add-z` and dkt exits with 1

#### Scenario: Named change skips main specs
- **WHEN** a main spec has no title and a user runs `dkt lint add-x`
- **THEN** no finding is about that main spec

#### Scenario: Named change skips records
- **WHEN** the branch adds a record that breaks a rule and a user runs `dkt lint add-x`
- **THEN** no finding is about that record

#### Scenario: Lint reports the domain tree
- **WHEN** `docket/domains.md` has a line that is not an item, lists a capability twice, or lists a capability that does not exist, and a user runs `dkt lint`
- **THEN** the findings hold `domain-format`, `domain-duplicate` or `domain-unknown` as `domain-tree` defines, naming `docket/domains.md` and the line, and dkt exits with 1

#### Scenario: Lint with no domain tree
- **WHEN** there is no `docket/domains.md` and a user runs `dkt lint`
- **THEN** no finding is about the domain tree

#### Scenario: Linked domain tree
- **WHEN** `docket/domains.md` is a symbolic link and a user runs `dkt lint`
- **THEN** `unreadable` lists it and dkt exits with 3

#### Scenario: Linked specs folder with a domain tree
- **WHEN** `docket/domains.md` exists, `openspec/specs` is a symbolic link, and a user runs `dkt lint`
- **THEN** `unreadable` lists `openspec/specs` and dkt exits with 3

#### Scenario: Linked specs folder with no domain tree
- **WHEN** there is no `docket/domains.md`, `openspec/specs` is a symbolic link, and a user runs `dkt lint`
- **THEN** the domain tree adds nothing to `unreadable`

#### Scenario: Named change skips the domain tree
- **WHEN** `docket/domains.md` has a line that is not an item and a user runs `dkt lint add-x`
- **THEN** no finding is about the domain tree

### Requirement: Front matter checked
For a change of the `docket` schema, `dkt lint` SHALL report `title-missing` when `proposal.md` has no front matter or no title that is one line of text, `title-too-long` giving the length and the limit for a title over the limit `change-metadata` defines, and `metadata-invalid` for a front matter it cannot read and for a `kind` or `source` that is missing or not of the form `change-metadata` defines.

#### Scenario: Proposal without front matter
- **WHEN** `openspec/changes/add-x/proposal.md` starts with `# Proposal`
- **THEN** `dkt lint add-x` reports `title-missing` naming that file

#### Scenario: Unknown kind
- **WHEN** the front matter of `add-x`'s proposal has `kind: fix`
- **THEN** `dkt lint add-x` reports `metadata-invalid` naming `kind` on that line

#### Scenario: Change title too long
- **WHEN** the front matter of `add-x`'s proposal has a title of 121 bytes
- **THEN** `dkt lint add-x` reports `title-too-long` on that line, giving 121 and 120, and no `title-missing`

### Requirement: New capabilities named
For each capability the change has a delta for and the worktree has no main spec for, `dkt lint` SHALL report `title-missing` naming the capability when the proposal's front matter gives it no name under `specs:`. This holds for a change of any schema.

#### Scenario: New capability without a name
- **WHEN** `add-x` has a delta for `booking`, there is no `openspec/specs/booking/spec.md`, and the front matter's `specs:` has no `booking`
- **THEN** `dkt lint add-x` reports `title-missing` for `booking`

#### Scenario: Existing capability needs no name
- **WHEN** `add-x` has a delta for `booking` and `openspec/specs/booking/spec.md` exists
- **THEN** no `title-missing` finding names `booking`

### Requirement: Proposal sections
The proposal of a `docket` change SHALL have the level-two sections Why, What Changes, Non-Goals, Capabilities and Impact, and Non-Goals SHALL hold text other than whitespace and HTML comments; otherwise `dkt lint` SHALL report `proposal-section` naming the section.

#### Scenario: Non-Goals missing
- **WHEN** the proposal has no Non-Goals section
- **THEN** lint reports `proposal-section` naming Non-Goals

#### Scenario: Non-Goals holds only the template comment
- **WHEN** the Non-Goals section holds only the template's HTML comment
- **THEN** lint reports `proposal-section` naming Non-Goals

### Requirement: Leftover template text
In every Markdown file of a `docket` change, `dkt lint` SHALL report `placeholder` for each HTML comment or angle-bracket placeholder that appears verbatim in the shipped `docket` templates, except inside code spans and fenced code blocks, naming the file and line.

#### Scenario: Template comment left
- **WHEN** the proposal still holds the template comment under Why
- **THEN** lint reports `placeholder` on that line

#### Scenario: Placeholder in a code span
- **WHEN** the design mentions `` `<capability-path>` `` in a code span
- **THEN** lint reports nothing about it

### Requirement: Plan rules only with a plan
The rules on plan.md and on scenario claims SHALL apply only when the change has a plan.md; a change without one is linted on its proposal and deltas alone.

#### Scenario: Plan not written yet
- **WHEN** a change has a proposal and deltas but no plan.md
- **THEN** lint reports no plan rule and no `scenario-unclaimed` finding for it

### Requirement: Task headings
In plan.md, every heading outside fenced code and HTML comments whose text begins with `Task` followed by a number SHALL be a level-three heading of the form `### Task N: <name>`, N a positive integer, and task numbers SHALL be unique across the tasks and the bullets under `## Cancelled Tasks`; otherwise `dkt lint` SHALL report `task-heading`.

#### Scenario: Task heading at the wrong level
- **WHEN** plan.md has the heading `## Task 3: Parser`
- **THEN** lint reports `task-heading` on that line

#### Scenario: Duplicate task number
- **WHEN** two tasks are both numbered 2
- **THEN** lint reports `task-heading` naming both lines

### Requirement: Reuse table
plan.md SHALL have a `## Reuse` section with a table whose columns are Need, Existing (path:symbol), Decision, Reason and Evidence, with at least one row. Decision SHALL be `reuse`, `extend` or `new`; for `reuse` and `extend`, the path before the first `:` of Existing SHALL exist in the worktree unless Existing starts with `external:`; for `new`, Reason SHALL not be empty; every row SHALL have Evidence. A break SHALL be reported as `reuse-table`.

#### Scenario: Reuse row with a missing path
- **WHEN** a `reuse` row names `internal/gitx/run.go:Run` and that file does not exist
- **THEN** lint reports `reuse-table` on that row

#### Scenario: External dependency row
- **WHEN** a `reuse` row's Existing is `external: git CLI`
- **THEN** lint reports nothing about that row

#### Scenario: Empty reuse table
- **WHEN** the Reuse table has its header and no rows
- **THEN** lint reports `reuse-table` on the table

### Requirement: Scenario references
Every task SHALL have a `**Scenarios:**` line that is `none` or a comma-separated list of `<capability-path>#<scenario-slug>`, each naming a scenario of the change's deltas or of the current main spec; otherwise `dkt lint` SHALL report `scenario-ref`. Every scenario under the change's ADDED and MODIFIED requirements SHALL be claimed by a task; otherwise `scenario-unclaimed`.

#### Scenario: Scenario nobody claims
- **WHEN** the delta adds the scenario "Join when full" to `booking` and no task lists `booking#join-when-full`
- **THEN** lint reports `scenario-unclaimed` for it

#### Scenario: Reference to an unknown scenario
- **WHEN** a task lists `booking#join-when-ful`
- **THEN** lint reports `scenario-ref` on that task's line

#### Scenario: Scaffolding task
- **WHEN** a task's line is `**Scenarios:** none`
- **THEN** lint reports nothing about that task's scenarios

### Requirement: Scenario slugs
A scenario's slug SHALL be its name in lower case, with every run of characters other than letters and digits, in any script, replaced by one `-`, and no leading or trailing `-`. Within one capability, once the delta is applied to the main spec, no two scenarios SHALL share a slug; otherwise `dkt lint` SHALL report `scenario-duplicate` naming both.

#### Scenario: Two scenarios named Success
- **WHEN** two requirements in the delta for `booking` each have a scenario named "Success"
- **THEN** lint reports `scenario-duplicate` naming both

#### Scenario: Slug of a name with punctuation
- **WHEN** a scenario is named "Join twice — rejected"
- **THEN** its slug is `join-twice-rejected`

### Requirement: Cancelled and compensating tasks
Every bullet under `## Cancelled Tasks` SHALL have the form `- Task N: <name> - <reason>` with a reason (`cancelled-task` otherwise), and every `**Compensates:** Task N` line SHALL name a `### Task` heading of the same plan (`compensates` otherwise).

#### Scenario: Cancelled task without a reason
- **WHEN** a bullet under Cancelled Tasks reads `- Task 5: Front desk screen`
- **THEN** lint reports `cancelled-task` on that line

#### Scenario: Compensating a cancelled task
- **WHEN** a task says `**Compensates:** Task 5` and Task 5 is listed under Cancelled Tasks
- **THEN** lint reports `compensates` on that line

### Requirement: Deltas read through OpenSpec
`dkt lint` SHALL read a change's deltas, and the main specs its references name, through the OpenSpec CLI's JSON output, not by parsing them itself, and, for a `docket` change, SHALL report as `openspec-invalid` each error `openspec validate` reports for it. When OpenSpec cannot show the change's deltas, it SHALL report `openspec-failed` as `openspec-runtime` says and skip the rules that need them.

#### Scenario: Validation error reported
- **WHEN** a delta requirement has no scenario
- **THEN** lint reports `openspec-invalid` with OpenSpec's message and exits with 1

#### Scenario: Proposal OpenSpec cannot read
- **WHEN** the proposal of `add-x` has no Why section, so OpenSpec cannot show its deltas
- **THEN** lint reports `proposal-section` for Why and `openspec-failed` for the deltas, and reports no `scenario-unclaimed` finding

### Requirement: Main spec titles
A main spec's title SHALL be read as `change-metadata` defines. When no change is named, `dkt lint` SHALL report `title-missing` for each main spec without one, naming its `spec.md`.

#### Scenario: Main spec with OpenSpec's default first line
- **WHEN** `openspec/specs/booking/spec.md` starts with `# booking Specification` and a user runs `dkt lint`
- **THEN** lint reports `title-missing` naming `openspec/specs/booking/spec.md`

#### Scenario: Main spec with a title
- **WHEN** `openspec/specs/booking/spec.md` starts with `# Booking Waitlist Specification`
- **THEN** lint reports nothing about its title

### Requirement: Lint result
The `result` of `dkt lint` SHALL hold `changes`, the slugs linted with their schema (`{"slug", "schema"}`), `records` and `issues`, the record and issue slugs linted, `specs`, the number of main spec titles checked (0 when a name was given), and `default_branch`, the default branch the routing rule used, or null when it was skipped. Every finding SHALL name its change in `change` when it is about one, and its file and line when it has them. Without `--json`, the output SHALL list the findings.

#### Scenario: Clean change
- **WHEN** `add-x` breaks no rule, the default branch is `main`, and a user runs `dkt lint add-x --json`
- **THEN** `result.changes` is `[{"slug": "add-x", "schema": "docket"}]`, `result.records` and `result.issues` are `[]`, `result.specs` is 0, `result.default_branch` is `main`, there is no finding, and dkt exits with 0

#### Scenario: Records in the result
- **WHEN** the branch adds the clean record `2026-11-06-fix-x` and a user runs `dkt lint --json`
- **THEN** `result.records` is `["2026-11-06-fix-x"]`

#### Scenario: Issues in the result
- **WHEN** the branch adds the clean issue `2026-11-06-lint-timeout` and a user runs `dkt lint --json`
- **THEN** `result.issues` is `["2026-11-06-lint-timeout"]`

### Requirement: Design and plan routing
`dkt lint` SHALL report `routing` for each file under `docs/superpowers/specs/` or `docs/superpowers/plans/` that is in the worktree but not at the merge base of HEAD and the default branch's tip, whether committed on the branch, staged, or untracked and not ignored, naming the file. It SHALL do so whether or not changes are named. When there is no default branch or no merge base, it SHALL skip this rule.

#### Scenario: Plan written to superpowers' folder
- **WHEN** the current branch adds `docs/superpowers/plans/2026-11-06-x.md`, not yet committed, and a user runs `dkt lint add-x`
- **THEN** the finding `routing` names `docs/superpowers/plans/2026-11-06-x.md`

#### Scenario: File already on the default branch
- **WHEN** `docs/superpowers/specs/old.md` exists at the merge base and the branch changes it
- **THEN** no `routing` finding names it

#### Scenario: Ignored file
- **WHEN** an untracked `docs/superpowers/plans/x.md` is ignored by git
- **THEN** no `routing` finding names it

#### Scenario: Routing without a default branch
- **WHEN** no default branch can be found and a user runs `dkt lint --json`
- **THEN** there is no `routing` finding and `result.default_branch` is null

### Requirement: Fence marker inside an HTML comment
In every Markdown file of a `docket` change, in each record and issue it lints and in each main spec whose title it checks, `dkt lint` SHALL report `fence-in-comment`, naming the file and line, for each line inside an HTML comment that would open fenced code if it stood outside the comment, when no line before the comment's end closes that code. dkt and OpenSpec find fenced code before HTML comments, so they read the lines after such a comment as code.

#### Scenario: Fence marker left in a comment
- **WHEN** plan.md holds a comment made of the lines `<!--`, a line of three backticks and `-->`, followed by `### Task 2: Real`
- **THEN** lint reports `fence-in-comment` on the line of three backticks

#### Scenario: Code block commented out whole
- **WHEN** a comment in the design holds a line of three backticks, a line of code and another line of three backticks
- **THEN** lint reports no `fence-in-comment`

#### Scenario: Comment marker inside fenced code
- **WHEN** a fenced code block in the proposal holds a line that opens an HTML comment, and a later line of three backticks closes the block
- **THEN** lint reports no `fence-in-comment`

#### Scenario: Fence marker in a record
- **WHEN** a record the branch adds holds a comment whose only other line is a line of three tildes
- **THEN** lint reports `fence-in-comment` on that line of the record

#### Scenario: Fence marker in a main spec
- **WHEN** `openspec/specs/orders/spec.md` holds a comment made of the lines `<!--`, a line of three backticks and `-->`, followed by `### Requirement: Persist orders`, and a user runs `dkt lint` naming no change
- **THEN** lint reports `fence-in-comment` on the line of three backticks of that spec
