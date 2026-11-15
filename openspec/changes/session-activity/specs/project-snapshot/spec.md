## ADDED Requirements

### Requirement: Work activity
Each work's `activity` SHALL be `working` when a record that counts under `session-activity` has the state `working` and a directory in a worktree that holds the work, at a location that is not archived, while the work is not merged; `waiting` when such records exist and none is `working`; and null otherwise. A directory is in the deepest worktree whose path is the directory or one of its parents.

#### Scenario: Session working on a change
- **WHEN** a linked worktree holds the unmerged change `add-x`, and a record from 2 minutes ago has the state `working` and a directory in that worktree
- **THEN** `add-x` has `activity` `working`, and every other work has `activity` null

#### Scenario: Session waiting
- **WHEN** the only record for that worktree has the state `waiting`
- **THEN** `add-x` has `activity` `waiting`

#### Scenario: Merged work
- **WHEN** a session is working in the main worktree, which holds only works the tip holds
- **THEN** every work has `activity` null

#### Scenario: Worktree inside another
- **WHEN** a linked worktree at `/w/app/.claude/worktrees/x` holds `add-x`, the main worktree `/w/app` holds the unmerged `add-y`, and a session is working in `/w/app/.claude/worktrees/x/internal`
- **THEN** `add-x` has `activity` `working` and `add-y` has `activity` null

#### Scenario: No records
- **WHEN** the state home has no `docket/sessions/` folder
- **THEN** every work has `activity` null, and nothing is reported about it

### Requirement: Activity records that cannot be read
When `docket/sessions/` exists but cannot be listed, `dkt snapshot` SHALL list it in `unreadable`. A record file that cannot be read or parsed SHALL be skipped.

#### Scenario: Folder that cannot be listed
- **WHEN** `docket/sessions/` under the state home cannot be listed
- **THEN** `unreadable` names the folder, every work has `activity` null, and dkt exits with 3
