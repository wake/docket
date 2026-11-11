# Project Snapshot Specification

## Purpose
Defines what `dkt snapshot` reports about a whole project in one run: the default branch, every change and record with where it is and how far it has come, ideas, issues, the order, main specs and the domain tree, so that a viewer can show a project from one read-only command.

## Requirements

### Requirement: Snapshot command
`dkt snapshot` SHALL take no argument and need a repository. It SHALL write nothing, contact no remote and not run OpenSpec. With `--json`, `result` SHALL hold `default_branch`, `works`, `ideas`, `issues`, `order`, `specs` and `domains`; arrays SHALL never be null. Times SHALL be RFC 3339 with the commit's own offset, dates `YYYY-MM-DD`, commits full SHAs, and paths relative to the repository root, apart from a worktree's absolute path.

#### Scenario: Snapshot of an empty project
- **WHEN** a user runs `dkt snapshot --json` in a repository with one commit on `main` and no Docket files
- **THEN** `works`, `ideas`, `issues`, `order` and `specs` are empty arrays, `domains.found` is false, and dkt exits with 0

#### Scenario: Snapshot offline
- **WHEN** the network is unavailable and a user runs `dkt snapshot --json` in a repository with remote-tracking branches
- **THEN** it completes from the refs as they were at the last fetch

#### Scenario: No OpenSpec needed
- **WHEN** no OpenSpec CLI can be found and a user runs `dkt snapshot --json` in a project with active changes
- **THEN** the deltas of those changes are reported and dkt exits with 0

### Requirement: Default branch reported
`result.default_branch` SHALL hold the default branch's `name`, the `ref` its tip is read from, and the tip's `commit`, `time` and `subject`, as `default-branch` defines the branch and its tip; it SHALL be null when there is no default branch.

#### Scenario: Tip from the remote-tracking ref
- **WHEN** `refs/remotes/origin/main` and `refs/heads/main` both exist and point to different commits
- **THEN** `default_branch.ref` is `refs/remotes/origin/main` and `commit` is that ref's commit

#### Scenario: No default branch
- **WHEN** the repository has no branch named `main` or `master` and no `origin/HEAD`
- **THEN** `default_branch` is null, and `ideas`, `order`, `specs` and `domains.nodes` are empty

### Requirement: Works found
`result.works` SHALL hold one entry for each change and each record found as `workspace-scan` defines, keyed by its type and slug, however many locations hold it, sorted by `created`, then `type`, then `slug`, with a null `created` last. Each entry's `type` SHALL be `change` or `record`.

#### Scenario: One change in three locations
- **WHEN** `add-x` is in a linked worktree, on `refs/heads/worktree-add-x` and on `refs/remotes/origin/worktree-add-x`
- **THEN** `works` holds one entry for `add-x`, with three `locations`

#### Scenario: Archived change
- **WHEN** the default branch holds `openspec/changes/archive/2026-11-01-add-x/`
- **THEN** `works` holds `add-x` with `archived` that folder

### Requirement: Locations
Each work's `locations` SHALL list every source holding it, in the order `workspace-scan` reads them, each with `source`, `kind` (`worktree`, `branch` or `remote-branch`), `branch`, `default`, `commit`, `time`, `path` and `archived`. For a worktree, `branch` is its checked-out branch or null when detached, and `commit` its HEAD or null before the first commit. `default` SHALL be true only for the ref `default_branch.ref` names.

#### Scenario: Detached worktree
- **WHEN** a worktree with a detached HEAD holds `add-x`
- **THEN** its location has `kind` `worktree`, `branch` null, and `commit` its HEAD

#### Scenario: Default ref marked
- **WHEN** `add-x` is on `refs/heads/main` and on `refs/remotes/origin/main`, and the tip is read from the remote-tracking ref
- **THEN** only the location `refs/remotes/origin/main` has `default` true

### Requirement: Copy read
A work's fields SHALL be read from one location, named in `read_from`: the first location that holds it archived, and when none does, the first location. `archived` SHALL be that location's archive folder, or null.

#### Scenario: Worktree read first
- **WHEN** `add-x` is active in a linked worktree with an uncommitted edit to its title, and on its remote-tracking branch with the old title
- **THEN** `read_from` is the worktree's path and `title` is the edited one

#### Scenario: Archived copy wins
- **WHEN** a linked worktree holds `add-x` active and the default branch holds it archived
- **THEN** `read_from` is the first location holding the archived folder, and `archived` is that folder

### Requirement: Work metadata
`title`, `kind` and `source` SHALL come from the front matter of the change's `proposal.md` or of the record. `schema` SHALL be the change's schema as `change-metadata` defines it, and `docket` for a record. `created` SHALL be the `created` date in a change's `.openspec.yaml`, or the date in a record's slug, and null when there is none.

#### Scenario: Record metadata
- **WHEN** `docket/records/2026-11-06-fix-x.md` has the title `Fix X`, kind `fix` and source `review of #40`
- **THEN** its work has `type` `record`, `slug` `2026-11-06-fix-x`, those three values, `schema` `docket` and `created` `2026-11-06`

### Requirement: Unreadable work front matter
When the front matter of a change's `proposal.md` or of a record cannot be read, its `title`, `kind` and `source` SHALL be null, and that file SHALL be listed in `unreadable` with its source, as `ideas` lists it, since which idea or issue the work takes depends on it.

#### Scenario: Unreadable front matter
- **WHEN** a change's `proposal.md` starts with `---` and has no closing line
- **THEN** its `title`, `kind` and `source` are null, `unreadable` lists that `proposal.md` with its source, the rest is reported, and dkt exits with 3

### Requirement: Tasks reported
`tasks` SHALL list the `### Task N:` sections of a change's `plan.md`, or of a record's `## Tasks`, in file order, each with `number`, `name`, `steps` and `ticked` counted as `plan-sync` counts them, and `commits`: the commits whose `Change` or `Record` trailer names that task, oldest first, reachable from the commit of `read_from`. A change without `plan.md` and a record without `## Tasks` SHALL have an empty array, and a change of another schema null.

#### Scenario: Task commits from a worktree
- **WHEN** a linked worktree's HEAD carries `Change: add-x/1` and no other ref reaches that commit
- **THEN** with `read_from` that worktree, Task 1 of `add-x` lists the commit

#### Scenario: Other schema
- **WHEN** a change's `.openspec.yaml` has `schema: spec-driven`
- **THEN** its `tasks` is null

### Requirement: Task state and progress
Each task's `state` SHALL be `done` when a commit in its `commits` names it or every step is ticked, and `pending` otherwise. A work's `progress` SHALL be `{done, total}`: the number of its tasks that are done, and the number of its tasks, both 0 when it has none.

#### Scenario: Task with a commit and no ticks
- **WHEN** Task 1 has three unticked steps and a commit carries `Change: add-x/1`
- **THEN** Task 1's `state` is `done`

#### Scenario: Task ticked in part
- **WHEN** Task 2 has one of three steps ticked and no commit names it
- **THEN** Task 2's `state` is `pending`, and with Task 1 done, `progress` is `{done: 1, total: 2}`

### Requirement: Stages of a change
A change's `stages` SHALL list, in this order, those it has reached: `proposal` when the folder of `read_from` has `proposal.md`; `design` when it has a `specs/**/spec.md` or `design.md`; `plan` when its plan has a task; `dev` when a commit reachable from any location's commit carries `Change: <slug>/<N>`; `merged` when `merged` is true; `archived` when `archived` is not null. `phase` SHALL be the last of them.

#### Scenario: Change without design
- **WHEN** a change has `proposal.md` and a `plan.md` with tasks, no delta spec and no `design.md`, and no commit names a task
- **THEN** its `stages` are `["proposal", "plan"]` and its `phase` is `plan`

#### Scenario: Bare trailer is not dev
- **WHEN** the only commit for `add-x` carries `Change: add-x`
- **THEN** its `stages` do not hold `dev`

#### Scenario: Done but not merged
- **WHEN** every task of an unmerged change has a commit
- **THEN** its `phase` is `dev` and its `progress` is full

#### Scenario: Stages of an archived change
- **WHEN** the tip holds `add-x` archived and its tasks have commits
- **THEN** its `stages` end with `merged` and `archived`

### Requirement: Stages of a record
A record's `stages` SHALL list, in this order, those it has reached: `record` always; `dev` when a commit reachable from any location's commit carries its `Record` trailer, with or without a task; `merged` when `merged` is true. `phase` SHALL be the last of them.

#### Scenario: Record with a bare trailer
- **WHEN** a record without tasks is on a branch, and a commit there carries `Record: 2026-11-06-fix-x`
- **THEN** its `stages` are `["record", "dev"]`

#### Scenario: Merged record without trailers
- **WHEN** the tip holds a record and no commit anywhere carries its `Record` trailer
- **THEN** its `stages` are `["record", "merged"]`, without `dev`

### Requirement: Proposed date
A work's `proposed` SHALL be the `date` of the idea, or of the issue, that its `source` names, and its `created` otherwise.

#### Scenario: Proposed date from an idea
- **WHEN** `add-x` was created on 2026-11-08 with the `source` of an idea dated 2026-11-04
- **THEN** its `proposed` is `2026-11-04`

### Requirement: Documents
`documents` SHALL list each file in the change folder of `read_from`, with `kind` `proposal`, `design`, `plan` or `spec-delta` for `proposal.md`, `design.md`, `plan.md` and `specs/**/spec.md`, and `other` for the rest; for a record, the record file with `kind` `record`; and, when `source` is the path of an idea or an issue that is found, that file with `kind` `idea` or `issue`. Each document SHALL carry the `source` it is read from.

#### Scenario: Change from an idea
- **WHEN** `add-x` has `source: "docket/ideas/x.md"`, that idea is on the default branch, and the change folder holds `proposal.md` and `notes.md`
- **THEN** `documents` holds the idea with `source` `default_branch.ref`, the proposal with `kind` `proposal`, and `notes.md` with `kind` `other`

### Requirement: Deltas read
For a change, `deltas` SHALL list each `specs/<capability>/spec.md` of the change folder, sorted by capability, with `added`, `modified`, `removed` and `renamed`, read by dkt from the headings outside fenced code and HTML comments: a requirement's name and its scenario names for added and modified ones, its name and the `**Reason**` text for removed ones, and `from` and `to` for renamed ones. A record's `deltas` SHALL be empty.

#### Scenario: Every operation read
- **WHEN** a delta holds one ADDED requirement with two scenarios, one REMOVED requirement with a reason, and one RENAMED pair
- **THEN** its entry lists the added name with both scenario names, the removed name with its reason, and the pair as `from` and `to`

#### Scenario: Heading in a code block
- **WHEN** a delta holds `### Requirement: Example` inside fenced code
- **THEN** that heading is not reported

### Requirement: Merge and release
`merged` SHALL be true when the default branch tip holds the work. `merge` SHALL be the newest first-parent commit of the tip that brought in a commit whose `Change` or `Record` trailer names the work, with or without a task, with its `commit`, `time` and `subject`, or null. `release` SHALL be the tag, by creation time the oldest, whose commit has `merge` in its history, with its `tag` and `time`, or null.

#### Scenario: Merge commit found
- **WHEN** a merge commit on `main` brought in two commits carrying `Change: add-x/1` and `Change: add-x`
- **THEN** `merge` is that merge commit, with its time and subject

#### Scenario: Rebased work
- **WHEN** a commit carrying `Record: 2026-11-06-fix-x` was rebased onto `main` with no merge commit
- **THEN** `merge` is that commit

#### Scenario: First release
- **WHEN** the tags `v0.2.0` and then `v0.3.0` were made on commits after `merge`
- **THEN** `release.tag` is `v0.2.0`

#### Scenario: Merged without trailers
- **WHEN** the tip holds a change whose commits carry no trailer
- **THEN** `merged` is true and `merge` is null

### Requirement: Last commit
`last_commit` SHALL be the newest commit, by commit time, reachable from any location's commit, whose trailer names the work, with its `commit` and `time`; null when there is none.

#### Scenario: Newer commit on a branch
- **WHEN** the tip reaches a commit for `add-x` from 2026-11-01 and a remote-tracking branch reaches one from 2026-11-03
- **THEN** `last_commit` is the one from 2026-11-03

### Requirement: Order reported
`result.order` SHALL list the items of `docket/order.md` on the tip in order, each with its `path` and `position`: 1 for the first item line, counting only item lines. A work whose `source` names an item SHALL carry that item's `position`; otherwise its `position` SHALL be null.

#### Scenario: Positions skip other lines
- **WHEN** the order holds a heading, a blank line, `- docket/ideas/a.md`, a comment and `- docket/issues/2026-11-01-fix-x.md`
- **THEN** `order` holds `a.md` at position 1 and the issue at position 2

### Requirement: Ideas reported
`result.ideas` SHALL list every idea on the tip, sorted by path, with `path`, `title`, `needs`, `dropped`, `date`, `state`, `waiting_on`, `position` and `taken_by`. `state` SHALL be `dropped` when it is dropped, `taken` when a change or record names it as `ideas` defines, and otherwise `queued` when the order lists it and `pool` when it does not. `waiting_on` SHALL hold its needs that are not met, empty for `dropped` and `taken`.

#### Scenario: Idea in the pool
- **WHEN** an idea is not dropped, no work names it and the order does not list it
- **THEN** its `state` is `pool` and its `position` is null

#### Scenario: Taken idea
- **WHEN** a change on a branch has `source: "docket/ideas/a.md"`
- **THEN** that idea's `state` is `taken` and `taken_by` holds `{type: change, slug}` of that change

#### Scenario: Queued idea waiting
- **WHEN** the order lists `b.md`, which needs `a.md`, and nothing names `a.md`
- **THEN** `b.md` is `queued` with `waiting_on` `["docket/ideas/a.md"]`

#### Scenario: Need taken but not met
- **WHEN** `b.md` needs `a.md`, and only an unmerged branch holds a change whose `source` is `a.md`
- **THEN** `a.md` is `taken`, and `b.md` still has `waiting_on` `["docket/ideas/a.md"]`

### Requirement: Idea date
An idea's `date` SHALL be the date, in its own offset, of the oldest commit in the history of the default branch tip that added the idea's file, and null when there is none.

#### Scenario: Idea added and edited later
- **WHEN** `docket/ideas/a.md` was added by a commit of 2026-11-04 and edited by one of 2026-11-06
- **THEN** its `date` is `2026-11-04`

#### Scenario: Idea moved
- **WHEN** an idea was renamed from `docket/ideas/a.md` to `docket/ideas/b.md` by a commit of 2026-11-07
- **THEN** the `date` of `b.md` is `2026-11-07`

### Requirement: Issues reported
`result.issues` SHALL list every issue found as `workspace-scan` defines, sorted by slug, with `slug`, `path`, `date` from its slug, `state`, `position`, `locations` as for a work, `handlers` (each change or record whose `source` is its path, with `type`, `slug` and `merged`), and from its front matter read from its first location: `title`, `source`, `external` and `external_id` (null when absent), `dropped` (false when absent) and `unsynced` (empty when absent).

#### Scenario: Issue with a handler
- **WHEN** a record on a branch has `source: "docket/issues/2026-11-01-fix-x.md"`
- **THEN** that issue's `handlers` holds the record with `merged` false, the record's `documents` hold the issue with `kind` `issue`, and its `proposed` is `2026-11-01`

#### Scenario: Issue read from its first location
- **WHEN** a linked worktree holds an issue with an edited title, and the default branch holds it with the old title
- **THEN** the issue's `title` is the edited one

#### Scenario: Card reported
- **WHEN** a card has `external: "https://git.example.com/app/issues/92"`, `external-id: "#92"` and `unsynced: [taken]`
- **THEN** its entry has that `external`, `external_id` `#92`, `dropped` false and `unsynced` `["taken"]`

#### Scenario: Local issue reported
- **WHEN** an issue has neither `external` nor `unsynced`
- **THEN** its entry has `external` and `external_id` null and `unsynced` `[]`

### Requirement: Issue state
An issue's `state` SHALL be `dropped` when it is dropped, and otherwise `open` when it has no handler, `done` when every handler is merged, and `in-progress` otherwise.

#### Scenario: Open issue
- **WHEN** no change or record names an issue
- **THEN** its `state` is `open`

#### Scenario: One handler merged of two
- **WHEN** an issue has a merged record and an unmerged change as handlers
- **THEN** its `state` is `in-progress`

#### Scenario: Every handler merged
- **WHEN** every handler of an issue is merged
- **THEN** its `state` is `done`

#### Scenario: Dropped issue with a handler
- **WHEN** an issue has `dropped: true` and an unmerged change as a handler
- **THEN** its `state` is `dropped`, and `handlers` still holds the change

### Requirement: Main specs reported
`result.specs` SHALL list every `openspec/specs/<capability>/spec.md` on the tip, sorted by capability, with `capability`, `path`, `title` as `change-metadata` defines a main spec's title, and `requirements`: the names of its `### Requirement:` headings outside fenced code and HTML comments, in file order.

#### Scenario: Nested capability
- **WHEN** the tip holds `openspec/specs/payments/refund/spec.md` with two requirements
- **THEN** `specs` holds the capability `payments/refund` with both names in order

### Requirement: Domain tree reported
`result.domains` SHALL hold `found`, `nodes` and `unclassified` for `docket/domains.md` on the tip, read as `domain-tree` defines. A node SHALL have `name`, `capabilities` (the capability paths it lists) and `children`. `unclassified` SHALL hold, sorted, the capabilities of `specs` and of the deltas of active changes that no node holds. When the tree has `domain-format` or `domain-duplicate`, `nodes` SHALL be empty and every capability unclassified, while the rest of `result` is reported.

#### Scenario: Tree with a new capability
- **WHEN** the tree lists `booking` under `Users`, and an active change adds a delta for `refund`
- **THEN** `nodes` holds `Users` with `["booking"]`, and `unclassified` holds `refund`

#### Scenario: Tree that cannot be used
- **WHEN** the tree has a line that is not an item, and the tip holds the main spec `booking`
- **THEN** the findings hold `domain-format`, `nodes` is empty, `unclassified` holds `booking`, `works` is still reported, and dkt exits with 1

#### Scenario: Stale capability in the tree
- **WHEN** the tree lists `gone` under `Users`, and no main spec or delta holds `gone`
- **THEN** the findings hold `domain-unknown`, and `Users` still lists `gone`

### Requirement: Problems and unreadable sources
`dkt snapshot` SHALL report the findings `dkt status` and `dkt domains` would report about the tip's order, ideas, issues and domain tree, and still output the result. Sources it cannot read SHALL be listed in `unreadable` as `workspace-scan` and `cli-output` define, and the rest SHALL be reported. It SHALL exit with 3 when anything is unreadable, else 1 when there is a finding, else 0.

#### Scenario: Unmounted worktree
- **WHEN** a listed worktree's directory is missing and another worktree holds `add-x`
- **THEN** `unreadable` lists the missing worktree, `works` holds `add-x`, and dkt exits with 3

#### Scenario: Ref whose tree cannot be read
- **WHEN** a remote-tracking ref points at an object that is not a commit, and a branch holds `add-x`
- **THEN** `unreadable` lists that ref, `works` holds `add-x`, and dkt exits with 3

#### Scenario: Idea without a title
- **WHEN** an idea in the order has front matter without `title`
- **THEN** the findings hold `idea-front-matter`, the rest of `result` is reported, and dkt exits with 1

### Requirement: Snapshot text
Without `--json`, `dkt snapshot` SHALL print one line per work, `<type> <slug>  <phase>  <title>`, and then one line with the numbers of ideas, issues and specs.

#### Scenario: Text output
- **WHEN** a project has one change `add-x` titled `Add X` whose `phase` is `plan`, and no idea, issue or spec
- **THEN** the output is `change add-x  plan  Add X` and `0 ideas, 0 issues, 0 specs`
