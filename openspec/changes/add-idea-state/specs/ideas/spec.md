## Purpose

Defines ideas, the work that needs design before it becomes a change, and
`docket/order.md`, the order in which they are taken: their file formats,
how `dkt lint` checks them, how an idea's state is derived from the work
that names it, how `dkt status` reports the order and the ideas, and how the
`docket` skill uses them.

## ADDED Requirements

### Requirement: Idea file
An idea SHALL be a regular file `docket/ideas/<name>.md`: YAML front matter with a `title` of the form `docket-metadata` defines, optionally `dropped: true`, and optionally `needs`, a list of the repository-root paths of the ideas it depends on; then free text.

#### Scenario: Idea file read
- **WHEN** `docket/ideas/2026-11-02-x.md` has the front matter `title: X` and `needs: [docket/ideas/2026-11-01-y.md]`, `docket/ideas/2026-11-01-y.md` has `title: Y`, and a user runs `dkt lint`
- **THEN** lint reports no finding for either idea

### Requirement: Order file
`docket/order.md` SHALL hold the order of upcoming work and nothing else: the heading `# Order`, then lines of the form ``N. `<path>` ``, numbered from 1 with no gaps, each naming one idea by its repository-root path. Blank lines MAY stand between them. A line stays when its idea is done.

#### Scenario: Order file read
- **WHEN** `docket/order.md` holds `# Order`, a blank line, ``1. `docket/ideas/2026-11-01-a.md` `` and ``2. `docket/ideas/2026-11-02-b.md` ``, both ideas exist, and a user runs `dkt lint`
- **THEN** lint reports no finding for the order

### Requirement: Order checked
`dkt lint` SHALL report `order-format` when `docket/order.md` is not a regular file, its first line that is not blank is not `# Order`, a line is neither blank nor of the line form, or the lines are not numbered 1, 2, 3 and so on; and `order-idea` for a line whose path names no regular idea file in the worktree, names the idea an earlier line names, or names an idea with `dropped: true`. Each finding SHALL name its line.

#### Scenario: Order line with a checkbox
- **WHEN** line 3 of `docket/order.md` is ``3. [ ] `docket/ideas/2026-11-02-c.md` ``
- **THEN** `dkt lint` reports `order-format` on that line and exits with 1

#### Scenario: Gap in the order
- **WHEN** the lines of `docket/order.md` are numbered 1, 2 and 4
- **THEN** `dkt lint` reports `order-format` on the line numbered 4

#### Scenario: Order names a missing idea
- **WHEN** a line of `docket/order.md` names `docket/ideas/2026-11-02-none.md` and the worktree has no such file
- **THEN** `dkt lint` reports `order-idea` on that line

#### Scenario: Same idea twice
- **WHEN** lines 2 and 5 of `docket/order.md` name the same idea
- **THEN** `dkt lint` reports `order-idea` on line 5 and not on line 2

#### Scenario: Dropped idea in the order
- **WHEN** a line of `docket/order.md` names an idea whose front matter has `dropped: true`
- **THEN** `dkt lint` reports `order-idea` on that line

#### Scenario: Linked order file
- **WHEN** `docket/order.md` is a symbolic link to a file outside the repository
- **THEN** `dkt lint` reports `order-format` naming the link and does not read through it

### Requirement: Idea front matter checked
`dkt lint` SHALL report `idea-front-matter` for an idea whose front matter is missing or is not a YAML mapping, lacks a valid `title`, has `dropped` with a value other than `true`, or has a `needs` that is not a list of paths each naming another regular idea file in the worktree. A file under `docket/ideas/` that is not a regular file SHALL be skipped.

#### Scenario: Idea without a title
- **WHEN** `docket/ideas/2026-11-02-x.md` has front matter without `title`
- **THEN** `dkt lint` reports `idea-front-matter` for it and exits with 1

#### Scenario: Need names no idea
- **WHEN** an idea has `needs: [docket/ideas/2026-11-02-none.md]` and the worktree has no such file
- **THEN** `dkt lint` reports `idea-front-matter` naming that path

### Requirement: Idea sources checked
`dkt lint` SHALL report `idea-source` for each change it lints and each record the branch adds whose source names a path under `docket/ideas/`, as `docket-metadata` defines, that is no regular idea file in the worktree, naming the source.

#### Scenario: Source names no idea
- **WHEN** the change `add-x` has the source `docket/ideas/2026-11-02-none.md, item 4` and the worktree has no such file
- **THEN** `dkt lint add-x` reports `idea-source` naming that source and exits with 1

### Requirement: Idea state
An idea's handlers SHALL be the records and changes, found and read as an issue's handlers are, whose source names the idea's path as `docket-metadata` defines. Its state SHALL be `dropped` with `dropped: true`, and otherwise `pending` with no handler, `in-progress` when the default branch tip does not hold a handler, and `done` when it holds every handler. Ideas SHALL be read from the default branch tip only.

#### Scenario: Idea pending
- **WHEN** the default branch tip holds `docket/ideas/2026-11-02-x.md` and no record or change names it
- **THEN** the idea's state is `pending`

#### Scenario: Handler on a branch
- **WHEN** the default branch tip holds the idea and an unmerged branch holds the change `add-x`, whose `docket.yaml` has the idea's path as its source
- **THEN** the idea's state is `in-progress`, with `add-x` as a handler the tip does not hold

#### Scenario: Every handler merged
- **WHEN** the default branch tip holds the idea, a record whose source is its path, and an archived change whose source is its path followed by `, second part`
- **THEN** the idea's state is `done`

#### Scenario: One handler of two merged
- **WHEN** the default branch tip holds the idea and a record that names it, and worktree B holds an uncommitted change that names it
- **THEN** the idea's state is `in-progress`

#### Scenario: Dropped idea
- **WHEN** the idea has `dropped: true` and the default branch tip holds a record that names it
- **THEN** the idea's state is `dropped`

#### Scenario: Idea on a branch only
- **WHEN** an unmerged branch adds `docket/ideas/2026-11-02-z.md` and the default branch tip has no such file
- **THEN** `dkt status` reports no idea `docket/ideas/2026-11-02-z.md`

### Requirement: Order listed by status
`dkt status` SHALL list in `result.order` each line of the default branch tip's `docket/order.md` that has the line form, in order, with its number, path, its idea's title (null when not readable), state (`missing` when the tip holds no such idea), handlers, and `waiting_on`: the paths in the idea's `needs` whose idea is not `done`. `result.next` SHALL be the number of the first line whose state is `pending` or `in-progress`, or null.

#### Scenario: Order listed
- **WHEN** line 1 of the tip's order names an idea that is `done` and line 2 names one that a change on a branch names, and a user runs `dkt status --json`
- **THEN** `result.order` lists line 1 as `done` and line 2 as `in-progress` with that change, and `result.next` is 2

#### Scenario: Next skips done, dropped and missing
- **WHEN** the tip's order has line 1 `done`, line 2 naming a dropped idea, line 3 naming no idea the tip holds, and line 4 `pending`
- **THEN** line 3 has state `missing` and a null title, and `result.next` is 4

#### Scenario: Waiting on a need
- **WHEN** line 1 of the tip's order names an idea whose `needs` lists `docket/ideas/2026-11-01-b.md`, and that idea is `pending`
- **THEN** line 1's `waiting_on` is `["docket/ideas/2026-11-01-b.md"]`

#### Scenario: No order file
- **WHEN** the default branch tip has no `docket/order.md` and a user runs `dkt status --json`
- **THEN** `result.order` is empty, `result.next` is null, and no finding is reported for the order

#### Scenario: Unparsed order line
- **WHEN** the tip's `docket/order.md` holds ``1. `docket/ideas/2026-11-01-a.md` `` and a line `- later`
- **THEN** `result.order` lists line 1 only

### Requirement: Ideas outside the order listed by status
`dkt status` SHALL list in `result.ideas`, sorted by path, every idea the default branch tip holds that no line of its order names and whose state is `pending` or `in-progress`, each with its path, title, state and handlers. With slugs it SHALL list no idea and no order line, and `result.next` SHALL be null.

#### Scenario: Ideas outside the order
- **WHEN** the tip holds the ideas `a` (named by the order), `b` (pending), `c` (done) and `d` (dropped), and a user runs `dkt status --json`
- **THEN** `result.ideas` lists `b` only

#### Scenario: Order not listed with slugs
- **WHEN** the tip's order has a pending line and a user runs `dkt status add-x --json`
- **THEN** `result.order` and `result.ideas` are empty and `result.next` is null

### Requirement: Order and ideas in the text output
The text output of `dkt status` SHALL have, after the issues, a line for each order line that is not `done`, with its number, state, path and title; a line for each idea in `result.ideas` that is `in-progress`; and, when `result.next` is not null, a line naming it.

#### Scenario: Order in the text output
- **WHEN** line 1 of the tip's order is `done` and line 2 is `in-progress`, and a user runs `dkt status`
- **THEN** the output has a line for line 2, none for line 1, and a line naming 2 as the next

### Requirement: Ideas and the order have one writer
The `docket` skill SHALL have the agent leave `docket/order.md` and the files under `docket/ideas/` unchanged in the worktree of a change or a record. An idea found during the work, and any edit to the order, SHALL be made as a direct merge of its own.

#### Scenario: Idea found during a change
- **WHEN** an agent working on the change `add-x` finds further work that needs design
- **THEN** it writes the idea as a direct merge of its own, and the branch of `add-x` changes no file under `docket/ideas/`

### Requirement: Work names its idea
When work takes an idea, the `docket` skill SHALL have the agent pass the idea's path alone as the `--source` of the change or the record.

#### Scenario: Change takes an idea
- **WHEN** an agent opens a change for the idea `docket/ideas/2026-11-02-x.md`
- **THEN** it runs `dkt new` with `--source docket/ideas/2026-11-02-x.md`

### Requirement: One idea, one piece of work
The `docket` skill SHALL have the agent write a further part of an idea, one that the idea's change or record leaves for later, as a new idea whose `needs` names the first.

#### Scenario: Idea with a further part
- **WHEN** the change for `docket/ideas/2026-11-02-x.md` leaves a second part for later
- **THEN** the agent writes that part as a new idea with `needs: [docket/ideas/2026-11-02-x.md]`, as a direct merge of its own

### Requirement: Next work from status
The `docket` skill SHALL have the agent take the next piece of work from `result.next` of `dkt status --json`, not by reading the checkboxes or the first lines of `docket/order.md`.

#### Scenario: Agent picks the next work
- **WHEN** the user asks for the next piece of work
- **THEN** the agent runs `dkt status --json` and takes the idea of the line `result.next` names
