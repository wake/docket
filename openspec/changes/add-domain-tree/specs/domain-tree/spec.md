# Spec Delta

## Purpose

Defines `docket/domains.md`, an optional tree of a project's areas, and `dkt domains`, which reports it, so that capabilities and the changes touching them can be viewed by area whatever path they are stored at.

## ADDED Requirements

### Requirement: Tree file
A project MAY keep its domain tree in `docket/domains.md`. Each line that is not blank and does not start with `#` SHALL be an item: `- <text>`, indented by two spaces per level. An item SHALL be a capability when its text is one code span holding a capability path, and otherwise a node named by its text. A node MAY be empty, or hold nodes and capabilities. A capability SHALL sit under a node and hold nothing.

#### Scenario: Nested nodes read
- **WHEN** `docket/domains.md` holds the lines `# Areas`, `- Users`, `  - Payments`, ``    - `refund` `` and `- Billing`, and a user runs `dkt domains --json`
- **THEN** `result.nodes` holds `Users`, whose child `Payments` holds the capability `refund`, then `Billing`, in the order written

#### Scenario: Empty node
- **WHEN** a node has no item under it
- **THEN** the node is reported with `capabilities` and `children` as empty arrays

#### Scenario: Name in any language
- **WHEN** `docket/domains.md` holds the line `- 使用者`
- **THEN** the node's name is `使用者`

#### Scenario: Windows line endings
- **WHEN** the lines of `docket/domains.md` end in CRLF
- **THEN** the tree reads the same as with LF

#### Scenario: Node apart from the storage path
- **WHEN** the node `Payments` holds the capability `billing/invoice`, stored at `openspec/specs/billing/invoice/spec.md`
- **THEN** `billing/invoice` is reported under `Payments`, and no file is moved

### Requirement: Read from the worktree
`dkt domains` SHALL read the worktree's files as they are, uncommitted edits included. It SHALL write nothing, contact no remote and not run OpenSpec.

#### Scenario: Uncommitted edit
- **WHEN** `docket/domains.md` has an edit that is not committed, and a user runs `dkt domains --json`
- **THEN** `result.nodes` follows the edit

#### Scenario: No OpenSpec on the machine
- **WHEN** OpenSpec cannot be run and a user runs `dkt domains --json`
- **THEN** the tree is reported and dkt exits with 0

### Requirement: Capabilities reported
A capability SHALL be a main spec under `openspec/specs/` or a delta spec under `specs/` in an active change. For each one, `dkt domains` SHALL report `path`; `main`, whether its main spec exists; `changes`, the active changes holding a delta for it, sorted; and `title`: the main spec's title as `change-metadata` defines it, else its name under `specs:` in the proposal of the first of those changes that gives one, else empty.

#### Scenario: Main spec with a title
- **WHEN** `openspec/specs/refund/spec.md` starts with `# Refund Specification` and no active change touches it
- **THEN** the capability `refund` has the title `Refund`, `main` true and `changes` empty

#### Scenario: Main spec without a title
- **WHEN** `openspec/specs/refund/spec.md` starts with `# refund Specification`
- **THEN** the capability `refund` has an empty title

#### Scenario: New capability in an active change
- **WHEN** the active change `add-x` has the delta `specs/refund/spec.md`, its proposal gives `refund: Refund` under `specs:`, and there is no main spec
- **THEN** the capability `refund` has the title `Refund`, `main` false and `changes` `["add-x"]`

#### Scenario: Delta to an existing capability
- **WHEN** the active changes `add-y` and `add-x` both hold a delta for `refund`, which has a main spec
- **THEN** `changes` is `["add-x", "add-y"]` and `main` is true

#### Scenario: Nested capability path
- **WHEN** the main spec is `openspec/specs/billing/invoice/spec.md`
- **THEN** the capability's `path` is `billing/invoice`

#### Scenario: Archived change ignored
- **WHEN** the only delta for `refund` is in a folder under `openspec/changes/archive/`, and there is no main spec
- **THEN** `refund` is not a capability

#### Scenario: Front matter that cannot be parsed
- **WHEN** the proposal of the active change `add-x`, the only one holding a delta for `refund`, has front matter that cannot be parsed
- **THEN** `refund` has an empty title, `dkt domains` reports no finding and nothing unreadable about the proposal, and `dkt lint` reports `metadata-invalid` as `change-lint` defines

#### Scenario: Linked spec or proposal
- **WHEN** `openspec/specs/refund/spec.md` is a symbolic link, or the proposal of an active change is one
- **THEN** Docket does not read through it: the link gives no capability and no title, and `unreadable` does not list it

#### Scenario: Linked change folder
- **WHEN** `openspec/changes/add-x` is a symbolic link to a folder holding `specs/refund/spec.md`
- **THEN** it is not a change, as `change-lint` and `workspace-scan` treat it: `refund` is not a capability, and `unreadable` does not list it

### Requirement: Unclassified capabilities
`result.unclassified` SHALL hold every capability no node holds, sorted by path. A capability being unclassified SHALL NOT be a finding.

#### Scenario: Spec outside the tree
- **WHEN** `openspec/specs/invoice/spec.md` exists and no node holds `invoice`
- **THEN** `result.unclassified` lists `invoice`, there is no finding, and dkt exits with 0

### Requirement: No tree
When `docket/domains.md` does not exist, `result.found` SHALL be false and `result.nodes` empty, every capability SHALL be unclassified, there SHALL be no finding, and dkt SHALL exit with 0. When it exists, `result.found` SHALL be true.

#### Scenario: Project without a tree
- **WHEN** there is no `docket/domains.md`, `openspec/specs/refund/spec.md` exists, and a user runs `dkt domains --json`
- **THEN** `result.found` is false, `result.nodes` is empty, `result.unclassified` lists `refund`, and dkt exits with 0

#### Scenario: File with no items
- **WHEN** `docket/domains.md` holds only the line `# Areas`
- **THEN** `result.found` is true and `result.nodes` is empty

#### Scenario: File in place of the docket folder
- **WHEN** `docket` is a regular file
- **THEN** `docket/domains.md` does not exist, `result.found` is false, and dkt exits with 0

### Requirement: Format problems
`dkt domains` SHALL report `domain-format`, naming `docket/domains.md` and the line, for each line that is not an item of the form above, or that breaks its rules: an indent that is not two spaces per level, a level skipped, a capability at the top level, an item under a capability, a backtick anywhere but around one whole code span, or a capability path that is not a valid relative path or starts with `-`. `result` SHALL be null, and dkt SHALL exit with 1.

#### Scenario: Line that is not an item
- **WHEN** a line of `docket/domains.md` is `Users`
- **THEN** the finding `domain-format` names that line, `result` is null, and dkt exits with 1

#### Scenario: Tab indent
- **WHEN** an item is indented with a tab
- **THEN** the finding `domain-format` names that line

#### Scenario: Odd indent
- **WHEN** an item is indented by three spaces
- **THEN** the finding `domain-format` names that line

#### Scenario: Level skipped
- **WHEN** the item after `- Users` is indented by four spaces
- **THEN** the finding `domain-format` names that line

#### Scenario: Capability at the top level
- **WHEN** a line with no indent is ``- `refund` ``
- **THEN** the finding `domain-format` names that line

#### Scenario: Item under a capability
- **WHEN** an item is indented under the capability `refund`
- **THEN** the finding `domain-format` names that line

#### Scenario: Code span mixed with text
- **WHEN** an item's text is ``Payments `refund` ``
- **THEN** the finding `domain-format` names that line

#### Scenario: Stray backtick
- **WHEN** an item's text is a name holding one backtick
- **THEN** the finding `domain-format` names that line

#### Scenario: Capability path outside the specs
- **WHEN** an item's text is `` `../refund` ``
- **THEN** the finding `domain-format` names that line

#### Scenario: Capability path starting with a dash
- **WHEN** an item's text is `` `-refund` ``
- **THEN** the finding `domain-format` names that line

#### Scenario: Every bad line reported
- **WHEN** two lines of `docket/domains.md` break the form
- **THEN** there is one `domain-format` finding for each

### Requirement: Duplicates
`dkt domains` SHALL report `domain-duplicate`, naming the later line and saying which line came first, when a node has the same name as a node under the same parent, or a capability is listed a second time anywhere in the tree. `result` SHALL be null, and dkt SHALL exit with 1. Names SHALL be compared exactly, after surrounding whitespace is removed.

#### Scenario: Sibling nodes with one name
- **WHEN** two top-level nodes are both named `Users`
- **THEN** the finding `domain-duplicate` names the second line and gives the first, `result` is null, and dkt exits with 1

#### Scenario: One name under different parents
- **WHEN** `Users` and `Billing` each hold a node named `History`
- **THEN** there is no finding

#### Scenario: Capability under two nodes
- **WHEN** `Users` and `Billing` both list the capability `refund`
- **THEN** the finding `domain-duplicate` names the second line and gives the first

### Requirement: Unknown capability
`dkt domains` SHALL report `domain-unknown`, naming the line, when the tree lists a capability with neither a main spec nor a delta in an active change. The capability SHALL stay in `result.nodes`, with `main` false and `changes` empty, and dkt SHALL exit with 1.

#### Scenario: Capability that no longer exists
- **WHEN** a node lists `refund` and there is no such main spec or delta
- **THEN** the finding `domain-unknown` names that line, `result.nodes` still lists `refund`, and dkt exits with 1

#### Scenario: Capability with only a delta
- **WHEN** a node lists `refund`, there is no main spec, and the active change `add-x` holds a delta for it
- **THEN** there is no finding

### Requirement: Unreadable tree file
When a symbolic link is on the path to `docket/domains.md`, or the file is not a regular file or cannot be read, `unreadable` SHALL list it, `result` SHALL be null, and dkt SHALL exit with 3.

#### Scenario: Linked tree file
- **WHEN** `docket/domains.md` is a symbolic link
- **THEN** `unreadable` lists it, Docket does not read through it, `result` is null, and dkt exits with 3

#### Scenario: Linked docket folder
- **WHEN** `docket` is a symbolic link to a folder holding `domains.md`
- **THEN** `unreadable` lists `docket/domains.md` and names `docket` as the link, Docket does not read through it, `result` is null, and dkt exits with 3

### Requirement: Unreadable sources
When a symbolic link is on the path to `openspec/specs`, `openspec/changes` or a change's `specs` folder, or when `openspec/changes`, a spec or a proposal cannot be read, `unreadable` SHALL list it, nothing SHALL be read through the link, `result` SHALL be reported from what could be read, `domain-unknown` SHALL NOT be reported, and dkt SHALL exit with 3.

#### Scenario: Linked specs folder
- **WHEN** a node lists a capability that does not exist, and `openspec/specs`, or the `openspec` folder above it, is a symbolic link to a folder holding `refund/spec.md`
- **THEN** `unreadable` lists the link once, `refund` is not a capability, there is no `domain-unknown` finding, `result.nodes` is reported, and dkt exits with 3

#### Scenario: Linked changes folder
- **WHEN** a node lists a capability that does not exist, and `openspec/changes` is a symbolic link to a folder holding `add-x/specs/refund/spec.md`
- **THEN** `unreadable` lists the link, `refund` is not a capability, there is no `domain-unknown` finding, `result.nodes` is reported, and dkt exits with 3

#### Scenario: Linked specs folder in a change
- **WHEN** a node lists a capability that does not exist, and `openspec/changes/add-x/specs` is a symbolic link to a folder holding `refund/spec.md`
- **THEN** `unreadable` lists the link, `refund` is not a capability, there is no `domain-unknown` finding, `result.nodes` is reported, and dkt exits with 3

#### Scenario: File in place of the changes folder
- **WHEN** a node lists a capability that does not exist, and `openspec/changes` is a regular file
- **THEN** `unreadable` lists `openspec/changes`, there is no `domain-unknown` finding, `result.nodes` is reported, and dkt exits with 3

#### Scenario: Unreadable spec
- **WHEN** a node lists `refund`, and a file named `spec.md` under `openspec/specs/` cannot be read
- **THEN** `unreadable` lists that file, there is no `domain-unknown` finding, `result.nodes` is reported, and dkt exits with 3

#### Scenario: Unreadable proposal
- **WHEN** the proposal of the active change `add-x`, which holds a delta for `refund`, cannot be read, and a node lists a capability that does not exist
- **THEN** `unreadable` lists the proposal, there is no `domain-unknown` finding, `refund` is still reported with `changes` `["add-x"]` and an empty title, and dkt exits with 3

### Requirement: Result
`dkt domains` SHALL take no positional argument. With `--json`, `result` SHALL hold `found`, `nodes` and `unclassified`; each node `name`, `capabilities` and `children`; each capability `path`, `title`, `main` and `changes`. Arrays SHALL never be null.

#### Scenario: Argument given
- **WHEN** a user runs `dkt domains x`
- **THEN** dkt reports a usage error and exits with 2

### Requirement: Text
Without `--json`, `dkt domains` SHALL print the nodes indented by two spaces per level, each capability as its path followed by its title when it has one, then the unclassified capabilities under a line `Unclassified:`. It SHALL say when there is no `docket/domains.md`.

#### Scenario: Text output
- **WHEN** the node `付款` holds `refund`, titled `退款`, `invoice` is unclassified, and a user runs `dkt domains`
- **THEN** the output shows `付款`, then indented `refund: 退款`, then `Unclassified:` and indented `invoice`

#### Scenario: Text output without a tree
- **WHEN** there is no `docket/domains.md` and a user runs `dkt domains`
- **THEN** the output says that `docket/domains.md` does not exist
