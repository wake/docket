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
plan.md SHALL have a `## Reuse` section with a table whose columns are Need, Existing (path:symbol), Decision and Reason; the table MAY have no rows. Decision SHALL be `extend` or `new`; for `extend`, the path before the first `:` of Existing SHALL exist in the worktree unless Existing starts with `external:`; for `new`, Reason SHALL not be empty. A break SHALL be reported as `reuse-table`.

#### Scenario: Reuse row with a missing path
- **WHEN** an `extend` row names `internal/gitx/run.go:Run` and that file does not exist
- **THEN** lint reports `reuse-table` on that row

#### Scenario: External dependency row
- **WHEN** an `extend` row's Existing is `external: git CLI`
- **THEN** lint reports nothing about that row

#### Scenario: Empty reuse table
- **WHEN** the Reuse table has its header and no rows
- **THEN** lint reports nothing about the Reuse section

#### Scenario: Reuse decision in the table
- **WHEN** a row of the four-column table has the Decision `reuse`
- **THEN** lint reports `reuse-table` on that row

### Requirement: Reuse table in the earlier format
A Reuse table whose columns are Need, Existing (path:symbol), Decision, Reason and Evidence SHALL be accepted under the rules of the Reuse table requirement, except that Evidence SHALL not be checked and Decision MAY also be `reuse`, whose Existing SHALL be checked as for `extend`.

#### Scenario: Earlier table with an empty Evidence
- **WHEN** the Reuse table has the five columns and a `reuse` row names an existing file and has no Evidence
- **THEN** lint reports nothing about that row

#### Scenario: Reuse row in the earlier table with a missing path
- **WHEN** a `reuse` row of the five-column table names `internal/gitx/run.go:Run` and that file does not exist
- **THEN** lint reports `reuse-table` on that row

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
- **WHEN** plan.md holds a comment made of three lines, a comment opener, a line of three backticks and a comment closer, followed by `### Task 2: Real`
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
- **WHEN** `openspec/specs/orders/spec.md` holds a comment made of three lines, a comment opener, a line of three backticks and a comment closer, followed by `### Requirement: Persist orders`, and a user runs `dkt lint` naming no change
- **THEN** lint reports `fence-in-comment` on the line of three backticks of that spec

### Requirement: Backtick in a backtick fence's info string
In every Markdown file of a `docket` change, in each record and issue it lints and in each main spec whose title it checks, `dkt lint` SHALL report `fence-info-backtick`, naming the file and line, for each line that opens with three or more backticks, holds another backtick after them, and lies outside fenced code as OpenSpec 1.16 and later read the file. dkt and OpenSpec 1.14 read such a line as fenced code; OpenSpec 1.16 and later do not.

#### Scenario: Inline code at the start of a line
- **WHEN** plan.md holds a line made of three backticks, `npm test` and three backticks, followed by `### Task 2: Real`
- **THEN** lint reports `fence-info-backtick` on that line

#### Scenario: Backtick fence with a plain info string
- **WHEN** the design holds a line of three backticks followed by `go`, a line of code and a line of three backticks
- **THEN** lint reports no `fence-info-backtick`

#### Scenario: Tilde fence whose info string holds a backtick
- **WHEN** the proposal holds a line of three tildes followed by `` a`b ``, a line of code and a line of three tildes
- **THEN** lint reports no `fence-info-backtick`

#### Scenario: Line inside fenced code
- **WHEN** a code block fenced by lines of three tildes in the design holds a line made of three backticks, `x` and three backticks
- **THEN** lint reports no `fence-info-backtick`

#### Scenario: Line both rules read inside fenced code
- **WHEN** the design holds, in order, a line made of three backticks, `a` and a backtick, a line of three backticks, a line made of three backticks, `b` and a backtick, and a line of three backticks
- **THEN** lint reports `fence-info-backtick` on the first line only

#### Scenario: Line after the two readings swap
- **WHEN** the design holds, in order, a line made of three backticks, `a` and a backtick, two lines of three backticks, a line made of three backticks, `b` and a backtick, and a line of three backticks
- **THEN** lint reports `fence-info-backtick` on the first and the fourth line

#### Scenario: Line in a record
- **WHEN** a record the branch adds holds a line made of three backticks, `make` and three backticks
- **THEN** lint reports `fence-info-backtick` on that line of the record

#### Scenario: Line in a main spec
- **WHEN** `openspec/specs/orders/spec.md` holds a line made of three backticks, `npm test` and three backticks, followed by `### Requirement: Persist orders`, and a user runs `dkt lint` naming no change
- **THEN** lint reports `fence-info-backtick` on that line of that spec

### Requirement: Comment opener inside a code span
In every Markdown file of a `docket` change, in each record and issue it lints and in each main spec whose title it checks, `dkt lint` SHALL report `comment-in-code-span`, naming the file and line, for each comment opener that dkt reads as the start of an HTML comment while it lies inside a code span, the code spans of a line being found before its comments. The message SHALL name the line where dkt's comment ends, or say that it runs to the end of the file.

#### Scenario: Opener in a code span hides the rest of the file
- **WHEN** plan.md holds a line with a comment opener in a code span, followed by `### Task 2: Report it` and no comment closer
- **THEN** lint reports `comment-in-code-span` on the line of the opener, saying that the comment runs to the end of the file

#### Scenario: Comment closed on a later line
- **WHEN** the proposal holds a line with a comment opener in a code span, and a later line holds a comment closer
- **THEN** lint reports `comment-in-code-span` on the line of the opener, naming the line of the closer

#### Scenario: Comment closed on the same line
- **WHEN** the design holds a line with a comment opener in one code span and a comment closer in a later one
- **THEN** lint reports `comment-in-code-span` on that line, naming that line

#### Scenario: Comment holding a code span
- **WHEN** the design holds a line made of a comment opener, a code span and a comment closer
- **THEN** lint reports no `comment-in-code-span`

#### Scenario: Comment starting a line
- **WHEN** the design holds a line that starts with a comment, closes it, and then holds a comment opener in a code span
- **THEN** lint reports no `comment-in-code-span`

#### Scenario: Opener in an HTML block
- **WHEN** the design holds a line `<div>`, a line with a comment opener in a code span and a comment closer, and a line `</div>`
- **THEN** lint reports no `comment-in-code-span`

#### Scenario: Backticks inside comments
- **WHEN** the design holds a line made of two comments, each holding one backtick
- **THEN** lint reports no `comment-in-code-span`

#### Scenario: Escaped backtick
- **WHEN** the design holds a line made of a backslash and a backtick, a comment that opens and closes, and a backtick
- **THEN** lint reports no `comment-in-code-span`

#### Scenario: Opener in fenced code
- **WHEN** a fenced code block in the design holds a line with a comment opener in a code span
- **THEN** lint reports no `comment-in-code-span`

#### Scenario: Opener in a record
- **WHEN** a record the branch adds holds a line with a comment opener in a code span
- **THEN** lint reports `comment-in-code-span` on that line of the record

#### Scenario: Opener in an issue
- **WHEN** an issue the branch adds holds a line with a comment opener in a code span
- **THEN** lint reports `comment-in-code-span` on that line of the issue

#### Scenario: Opener in a main spec
- **WHEN** `openspec/specs/orders/spec.md` holds a line with a comment opener in a code span, followed by `### Requirement: Persist orders`, and a user runs `dkt lint` naming no change
- **THEN** lint reports `comment-in-code-span` on that line of that spec

### Requirement: Capability missing from the domain tree
When `docket/domains.md` lists at least one item, has no format problem or duplicate, and every source `dkt domains` reads can be read, `dkt lint` SHALL report `domain-unlisted`, naming `docket/domains.md`, for each capability `dkt domains` reports as unclassified: with no change named, every one; with changes named, each with a delta in a named change. The message SHALL name the capability and say to list it under a node in `docket/domains.md`, then run `dkt domains`.

#### Scenario: Main spec left out of the tree
- **WHEN** `docket/domains.md` holds the lines `- Billing` and ``  - `invoice` ``, `openspec/specs/invoice/spec.md` and `openspec/specs/refund/spec.md` exist, and a user runs `dkt lint`
- **THEN** the finding `domain-unlisted` names `docket/domains.md` and `refund`, there is none for `invoice`, and dkt exits with 1

#### Scenario: New capability in a named change
- **WHEN** the tree lists `invoice` only, the change `add-refund` holds a delta for `refund`, and a user runs `dkt lint add-refund`
- **THEN** the finding `domain-unlisted` names `refund` and the change `add-refund`

#### Scenario: Capability outside the named change
- **WHEN** the tree lists `invoice` only, `openspec/specs/legacy/spec.md` exists, the change `add-x` holds a delta for `invoice` only, and a user runs `dkt lint add-x`
- **THEN** there is no `domain-unlisted` finding

#### Scenario: Tree with no items
- **WHEN** `docket/domains.md` holds only the line `# This project keeps no domain tree.`, `openspec/specs/refund/spec.md` exists, and a user runs `dkt lint`
- **THEN** there is no `domain-unlisted` finding

#### Scenario: Tree with a format problem
- **WHEN** `docket/domains.md` holds a line that is not an item, `openspec/specs/refund/spec.md` exists, no node lists it, and a user runs `dkt lint`
- **THEN** the finding `domain-format` is reported and there is no `domain-unlisted` finding

### Requirement: Capability retired by a change
`dkt lint` SHALL NOT report `domain-unlisted` for a capability that a linted change retires, as the change drops its line from the tree: with changes named, not for that change; with none named, not at all. A change retires a capability when its delta for it removes every requirement of the main spec, names compared with runs of space folded, and adds, modifies and renames none.

#### Scenario: Capability being removed
- **WHEN** the tree lists `invoice` only, `openspec/specs/refund/spec.md` holds the one requirement `Old rule`, the change `remove-refund` holds a delta for `refund` that only removes `Old rule`, and a user runs `dkt lint remove-refund` or `dkt lint`
- **THEN** there is no `domain-unlisted` finding

#### Scenario: Partial removal
- **WHEN** the tree lists `invoice` only, `openspec/specs/refund/spec.md` holds the requirements `Old rule` and `Y`, the change `remove-x` holds a delta for `refund` that only removes `Old rule`, and a user runs `dkt lint remove-x`
- **THEN** the finding `domain-unlisted` names `refund` and the change `remove-x`

#### Scenario: Removal with an addition
- **WHEN** the tree lists `invoice` only and the change `rework-refund` holds a delta for `refund` with an ADDED and a REMOVED requirement, and a user runs `dkt lint rework-refund`
- **THEN** the finding `domain-unlisted` names `refund` and the change `rework-refund`
