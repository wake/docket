# Spec Delta

## Purpose

Defines ideas and `docket/order.md`, the order they are taken in, and how `dkt status` names the next one, so that an agent picking up work gets one answer wherever it runs.

## ADDED Requirements

### Requirement: Idea file
An idea SHALL be a file `docket/ideas/<name>.md` whose front matter has `title` on one line, optionally `dropped: true`, and optionally `needs`, the paths of the ideas it depends on as one line: `needs: [docket/ideas/a.md, docket/ideas/b.md]`. Free text follows.

#### Scenario: Idea read
- **WHEN** `docket/ideas/b.md` has the front matter `title: B` and `needs: [docket/ideas/a.md]`, and `docket/ideas/a.md` has `title: A` and `dropped: true`
- **THEN** both are ideas, and neither is reported as a problem by `dkt status`

### Requirement: Order file
`docket/order.md` SHALL list the ideas to work on next, in order, one per line, as `- docket/ideas/<name>.md`. Blank lines and lines starting with `#` SHALL be ignored. A line MAY stay when its idea is done.

#### Scenario: Order read
- **WHEN** `docket/order.md` holds `# Order`, a blank line, `- docket/ideas/a.md` and `- docket/ideas/b.md`
- **THEN** the order is `docket/ideas/a.md`, then `docket/ideas/b.md`

### Requirement: Read from the default branch tip
`dkt status` SHALL read `docket/order.md` and the ideas from the tip of the default branch, as `default-branch` defines it, and not from any worktree. It SHALL write nothing, SHALL NOT contact a remote, and SHALL NOT need the OpenSpec CLI.

#### Scenario: Order edited in a worktree only
- **WHEN** the tip's order holds `- docket/ideas/a.md`, and the current worktree's uncommitted `docket/order.md` holds `- docket/ideas/b.md` first
- **THEN** `result.next` is `docket/ideas/a.md`

#### Scenario: Idea only on a branch
- **WHEN** an unmerged branch adds `docket/order.md` and its ideas, and the tip has neither
- **THEN** `result.next` is null and `result.reason` is `no-order`

#### Scenario: No OpenSpec
- **WHEN** the OpenSpec CLI cannot be found and a user runs `dkt status --json`
- **THEN** dkt answers as usual and exits with 0

### Requirement: Next idea
`result.next` SHALL be the first idea in the order that is still to do, with its `path` and `title`, or null when there is none. An idea SHALL be still to do when it is not dropped, no change or record names it, and every idea it needs is met.

#### Scenario: First idea next
- **WHEN** the order names `docket/ideas/a.md`, then `docket/ideas/b.md`, and nothing has been done on either
- **THEN** `result.next` is the path `docket/ideas/a.md` with its title, `result.reason` is empty, and dkt exits with 0

#### Scenario: Dropped idea skipped
- **WHEN** the first idea in the order has `dropped: true` and the second has nothing done on it
- **THEN** `result.next` is the second

#### Scenario: Taken idea skipped
- **WHEN** a change names the first idea in the order and the second has nothing done on it
- **THEN** `result.next` is the second

### Requirement: Work that names an idea
A change or record SHALL name an idea when the `source` in its front matter is exactly the idea's path: the `proposal.md` of a change, active or archived, or a record file. Changes and records SHALL be found as `workspace-scan` defines: in every worktree, local branch and remote-tracking branch. One is enough for the idea to be taken.

#### Scenario: Change in another worktree
- **WHEN** another worktree holds an uncommitted `openspec/changes/add-x/proposal.md` whose `source` is `docket/ideas/a.md`
- **THEN** `docket/ideas/a.md` is taken

#### Scenario: Record on another branch
- **WHEN** an unmerged branch holds a record whose `source` is `docket/ideas/a.md`
- **THEN** `docket/ideas/a.md` is taken

#### Scenario: Archived change
- **WHEN** the tip holds `openspec/changes/archive/2026-11-01-add-x/proposal.md` whose `source` is `docket/ideas/a.md`
- **THEN** `docket/ideas/a.md` is taken

#### Scenario: Source with other text
- **WHEN** a change has the `source` `docket/ideas/a.md, part two`
- **THEN** it does not name `docket/ideas/a.md`

#### Scenario: Work with no front matter
- **WHEN** an archived change's `proposal.md` has no front matter
- **THEN** it names no idea and is not reported as a problem

### Requirement: Needs
An idea it needs SHALL be met when that idea is dropped, or when a change or record that names it is on the default branch tip. Work only in a worktree or on another branch SHALL NOT meet it.

#### Scenario: Need merged
- **WHEN** `docket/ideas/b.md` needs `docket/ideas/a.md`, and the tip holds a record that names `docket/ideas/a.md`
- **THEN** `docket/ideas/b.md` can be next

#### Scenario: Need in progress
- **WHEN** the order names `docket/ideas/a.md`, then `docket/ideas/b.md`, which needs it, and only an unmerged branch holds a change that names `docket/ideas/a.md`
- **THEN** `result.next` is null, `result.reason` is `waiting`, and `result.waiting_on` holds `docket/ideas/a.md`

#### Scenario: Need ordered later
- **WHEN** the order names `docket/ideas/b.md`, which needs `docket/ideas/a.md`, then `docket/ideas/a.md`, and nothing has been done on either
- **THEN** `result.next` is `docket/ideas/a.md`

#### Scenario: Need dropped
- **WHEN** `docket/ideas/b.md` needs `docket/ideas/a.md`, which has `dropped: true`, and nothing has been done on `docket/ideas/b.md`
- **THEN** `docket/ideas/b.md` can be next

### Requirement: No next idea
When `result.next` is null, `result.reason` SHALL say why: `no-order` when there is no default branch, the tip has no `docket/order.md`, or the file names no idea; `order-invalid` when a problem is reported; `waiting` when ideas are left but none can start; `none-left` when every idea in the order is dropped or taken. `result.waiting_on` SHALL hold the unmet needs of the first idea that waits, and be empty otherwise. Only `order-invalid` SHALL make dkt exit with 1.

#### Scenario: No order file
- **WHEN** the tip has no `docket/order.md` and a user runs `dkt status --json`
- **THEN** `result.next` is null, `result.reason` is `no-order`, `result.waiting_on` is empty, the findings are empty, and dkt exits with 0

#### Scenario: Empty order file
- **WHEN** the tip's `docket/order.md` holds only a heading and blank lines
- **THEN** `result.next` is null, `result.reason` is `no-order`, and dkt exits with 0

#### Scenario: No default branch
- **WHEN** the repository has no commit
- **THEN** `result.next` is null, `result.reason` is `no-order`, and dkt exits with 0

#### Scenario: Nothing left
- **WHEN** every idea in the order is dropped or taken
- **THEN** `result.next` is null, `result.reason` is `none-left`, and dkt exits with 0

### Requirement: Problems in the order
`dkt status` SHALL report `order-format` for each line of `docket/order.md` that is not of the line form, naming the form, and `order-idea` for a line whose idea the tip does not hold. Each finding SHALL name `docket/order.md` and the line.

#### Scenario: Line not of the form
- **WHEN** line 3 of the tip's `docket/order.md` is `1. docket/ideas/a.md`
- **THEN** the findings hold `order-format` for `docket/order.md` line 3, naming the expected form, `result.next` is null, `result.reason` is `order-invalid`, and dkt exits with 1

#### Scenario: Order names a missing idea
- **WHEN** a line of the order names `docket/ideas/none.md` and the tip has no such file
- **THEN** the findings hold `order-idea` for that line, `result.next` is null, and dkt exits with 1

### Requirement: Problems in an idea
`dkt status` SHALL report `idea-front-matter` for an idea it reads whose front matter is missing, cannot be read, has no `title`, has a `dropped` other than `true` or `false`, or has a `needs` not of the line form, and `idea-needs` for a need the tip does not hold. Each finding SHALL name the idea's file.

#### Scenario: Idea without a title
- **WHEN** an idea in the order has front matter without `title`
- **THEN** the findings hold `idea-front-matter` for that file and dkt exits with 1

#### Scenario: Needs as a block
- **WHEN** an idea in the order writes `needs:` with the paths on the lines under it
- **THEN** the findings hold `idea-front-matter` for that file, naming the one-line form, and dkt exits with 1

#### Scenario: Need names a missing idea
- **WHEN** an idea in the order has `needs: [docket/ideas/none.md]` and the tip has no such file
- **THEN** the findings hold `idea-needs` for that idea's file and dkt exits with 1

### Requirement: A problem stops the answer
With any problem in the order or in an idea, `result.next` SHALL be null, `result.reason` `order-invalid`, and dkt SHALL exit with 1, even when an earlier idea could start: the order cannot be trusted.

#### Scenario: Later line broken
- **WHEN** the first idea in the order could start and a later line names a missing idea
- **THEN** the findings hold `order-idea`, `result.next` is null, and `result.reason` is `order-invalid`

### Requirement: Unreadable sources
A source the scan cannot read, and a change's `proposal.md` or a record whose front matter cannot be read, SHALL be listed in `unreadable`, and dkt SHALL exit with 3. `result.next` SHALL still be reported from what could be read.

#### Scenario: Broken proposal elsewhere
- **WHEN** another worktree's `proposal.md` has front matter that cannot be read, and the first idea in the order has nothing done on it
- **THEN** `unreadable` lists that worktree and file, `result.next` is that idea, and dkt exits with 3

#### Scenario: Linked proposal
- **WHEN** another worktree's `proposal.md` is a symbolic link
- **THEN** `unreadable` lists it, Docket does not read through it, and dkt exits with 3

### Requirement: Output
`dkt status` SHALL take no positional argument. With `--json`, `result` SHALL hold `next`, `reason` and `waiting_on`. Without it, dkt SHALL print the next idea's path and title, or one line saying why there is none.

#### Scenario: Text output
- **WHEN** a user runs `dkt status` and `docket/ideas/a.md`, titled `A`, is next
- **THEN** the output holds `docket/ideas/a.md` and `A`

#### Scenario: Argument given
- **WHEN** a user runs `dkt status add-x`
- **THEN** dkt reports a usage error and exits with 2
