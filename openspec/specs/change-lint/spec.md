# change-lint Specification

## Purpose
Defines the structural rules `dkt lint` enforces on a change folder, so that
the rules joining superpowers and OpenSpec fail visibly when they are broken
instead of living only in prompts.

## Requirements

### Requirement: What is linted
`dkt lint` SHALL lint the changes named on the command line, or every active change in the current worktree's `openspec/changes/` when none is named. It SHALL read the worktree's files as they are, uncommitted edits included, and SHALL NOT require the OpenSpec CLI.

#### Scenario: Lint every active change
- **WHEN** the worktree has the active changes `add-x` and `add-y` and a user runs `dkt lint`
- **THEN** both changes are linted and the archive folder is not

#### Scenario: Named change missing
- **WHEN** a user runs `dkt lint add-z` and the worktree has no change `add-z`
- **THEN** the command reports that `add-z` does not exist and exits with 1

#### Scenario: No OpenSpec installed
- **WHEN** no OpenSpec CLI is installed and a user runs `dkt lint`
- **THEN** the command lints normally

### Requirement: Proposal sections
proposal.md SHALL have the sections Why, What Changes, Non-Goals, Builds On, Capabilities, Impact, Implementation Contract and Change Log, and Implementation Contract SHALL have the subsections Observable Behavior, Interfaces, Failure Modes and Acceptance Criteria. Non-Goals, Builds On and each of the four subsections SHALL hold text other than whitespace and HTML comments.

#### Scenario: Non-Goals missing
- **WHEN** proposal.md has no Non-Goals section
- **THEN** lint reports a `proposal-section` finding naming Non-Goals

#### Scenario: Empty Failure Modes
- **WHEN** the Failure Modes subsection holds only the template's HTML comment
- **THEN** lint reports a `proposal-section` finding naming Failure Modes

#### Scenario: Builds On left empty
- **WHEN** the Builds On section has no text
- **THEN** lint reports a `proposal-section` finding naming Builds On

### Requirement: Change Log rows
Every row of the Change Log table after its header SHALL have a date in the form YYYY-MM-DD and a non-empty Source. A Change Log table with only its header SHALL pass.

#### Scenario: Change Log row without source
- **WHEN** a Change Log row has a date and an empty Source cell
- **THEN** lint reports a `change-log` finding on that row's line

#### Scenario: Header-only Change Log
- **WHEN** the Change Log table has its header and no rows
- **THEN** lint reports nothing about the Change Log

### Requirement: Leftover template text
lint SHALL report, in every file of the change, any HTML comment or angle-bracket placeholder that appears verbatim in the shipped `docket` schema templates, except inside code spans and fenced code blocks.

#### Scenario: Template comment left
- **WHEN** proposal.md still contains the template comment under Why
- **THEN** lint reports a `placeholder` finding on that line

### Requirement: Plan rules only with a plan
The rules on plan.md and on scenario claims SHALL apply only when the change has a plan.md. A change without one SHALL be linted on its proposal and deltas alone.

#### Scenario: Plan not written yet
- **WHEN** a change has proposal.md and deltas but no plan.md
- **THEN** lint reports no plan or `scenario-unclaimed` findings for it

### Requirement: Task headings
In plan.md, every heading whose text begins with the word Task followed by a number SHALL be a level-three heading of the exact form `### Task N: <name>`, with N a positive integer. Task numbers SHALL be unique across the tasks and the entries under `## Cancelled Tasks`.

#### Scenario: Task heading at the wrong level
- **WHEN** plan.md has the heading `## Task 3: Parser`
- **THEN** lint reports a `task-heading` finding on that line

#### Scenario: Duplicate task number
- **WHEN** two tasks are both numbered 2
- **THEN** lint reports a `task-heading` finding naming both lines

### Requirement: Reuse table
plan.md SHALL have a `## Reuse` section holding a table with the columns Need, Existing (path:symbol), Decision, Reason and Evidence, and at least one row. Decision SHALL be `reuse`, `extend` or `new`. For `reuse` and `extend`, the path before the first colon in Existing SHALL exist in the worktree, unless Existing starts with `external:`, which names a dependency outside the repository; for `new`, Reason SHALL be non-empty. Every row SHALL have a non-empty Evidence.

#### Scenario: Reuse row with a missing path
- **WHEN** a `reuse` row names `internal/gitx/run.go:Run` and that file does not exist
- **THEN** lint reports a `reuse-table` finding on that row

#### Scenario: New row without a reason
- **WHEN** a `new` row has an empty Reason
- **THEN** lint reports a `reuse-table` finding on that row

#### Scenario: External dependency row
- **WHEN** a `reuse` row's Existing is `external: git CLI`
- **THEN** lint does not look for a path and reports nothing about that row

#### Scenario: Empty reuse table
- **WHEN** the Reuse table has its header and no rows
- **THEN** lint reports a `reuse-table` finding on the table

#### Scenario: Row without evidence
- **WHEN** a `reuse` row has an empty Evidence cell
- **THEN** lint reports a `reuse-table` finding on that row

### Requirement: Scenario references
Every task SHALL have a `**Scenarios:**` line that is `none` or a comma-separated list of `<capability-path>#<scenario-slug>`. Each reference SHALL name a scenario in the change's deltas or in the current main spec. Every scenario under the change's ADDED and MODIFIED requirements SHALL be referenced by at least one task.

#### Scenario: Scenario nobody claims
- **WHEN** the delta adds the scenario "Join when full" to `booking` and no task lists `booking#join-when-full`
- **THEN** lint reports a `scenario-unclaimed` finding for it

#### Scenario: Reference to an unknown scenario
- **WHEN** a task lists `booking#join-when-ful`
- **THEN** lint reports a `scenario-ref` finding on that task

#### Scenario: Scaffolding task
- **WHEN** a task's line is `**Scenarios:** none`
- **THEN** lint reports nothing about that task's scenarios

### Requirement: Scenario slugs
A scenario's slug SHALL be its name in lower case, with every run of characters other than letters and digits, in any script, replaced by one hyphen, and no leading or trailing hyphen. Within one capability, taking the main spec's requirements, removing those the delta modifies or removes, and adding the delta's added and modified ones, no two scenarios SHALL share a slug.

#### Scenario: Two scenarios named Success
- **WHEN** two requirements in the delta for `booking` each have a scenario named "Success"
- **THEN** lint reports a `scenario-duplicate` finding naming both

### Requirement: Cancelled and compensating tasks
Every bullet under `## Cancelled Tasks` SHALL have the form `- Task N: <name> - <reason>` with a non-empty reason. Every `**Compensates:** Task N` line SHALL name a task that is a `### Task` heading in the same plan.

#### Scenario: Cancelled task without a reason
- **WHEN** a bullet under Cancelled Tasks reads `- Task 5: Front desk screen`
- **THEN** lint reports a `cancelled-task` finding on that line

#### Scenario: Compensating a cancelled task
- **WHEN** a task says `**Compensates:** Task 5` and Task 5 is listed under Cancelled Tasks
- **THEN** lint reports a `compensates` finding on that line

### Requirement: Routing
lint SHALL report every file under `docs/superpowers/specs/` or `docs/superpowers/plans/` that the current branch adds after its merge base with the default branch tip, whether committed or present uncommitted in the worktree, because designs and plans belong in the change folder. When no default branch can be found, lint SHALL list the routing check as unreadable and still run its other rules.

#### Scenario: Design written to the superpowers folder
- **WHEN** the worktree has a new untracked file `docs/superpowers/specs/2026-10-25-x-design.md`
- **THEN** lint reports a `routing` finding naming that file

#### Scenario: No default branch for routing
- **WHEN** the repository has no default branch Docket can find and a change has a `proposal-section` problem
- **THEN** lint reports the `proposal-section` finding, lists the routing check as unreadable, and exits with 3

### Requirement: Unparseable structure
lint SHALL report as `unparseable`, never skip, delta content OpenSpec 1.14 would not apply: a requirement header outside the ADDED, MODIFIED, REMOVED and RENAMED sections, a level-two section other than those four and Purpose, and a RENAMED FROM or TO line that does not form a pair. Headers inside fenced code blocks SHALL be ignored, as OpenSpec ignores them.

#### Scenario: Requirement outside any delta section
- **WHEN** a delta file has `### Requirement: Cap` under `## Notes`
- **THEN** lint reports an `unparseable` finding on that line

#### Scenario: Header inside a code block
- **WHEN** a requirement's text contains a fenced code block with a `### Requirement:` line
- **THEN** that line is not read as a requirement

#### Scenario: Unpaired rename
- **WHEN** the RENAMED section has a FROM line with no TO line after it
- **THEN** lint reports an `unparseable` finding on the FROM line

### Requirement: Agreement with OpenSpec
For every delta file OpenSpec 1.14 accepts, Docket's reading SHALL produce the same capability, operation, requirement names and scenario names, in the same order, as `openspec show <change> --type change --json --deltas-only`.

#### Scenario: Examples agree with OpenSpec
- **WHEN** the contract test runs over the shared examples with OpenSpec 1.14.3
- **THEN** Docket's reading and OpenSpec's output match for every example
