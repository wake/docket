## MODIFIED Requirements

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
