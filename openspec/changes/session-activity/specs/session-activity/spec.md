## Purpose
Defines how a development session reports whether it is working on something: the `dkt activity` command a coding agent's hooks call, the state records it keeps on the machine, and how long a record counts, so that `dkt snapshot` can tell viewers which works a session is on.

## ADDED Requirements

### Requirement: Activity command
`dkt activity <state>`, with the state `working` or `waiting`, SHALL record the calling session's state, its directory and the time of the write in a file of its own in the folder `docket/sessions/` under `$XDG_STATE_HOME` when that variable is an absolute path, and under `$HOME/.local/state` otherwise. With the state `ended`, it SHALL remove the session's file. It SHALL write nothing else.

#### Scenario: Working recorded
- **WHEN** `XDG_STATE_HOME` is `/s` and a session in `/w/app` runs `dkt activity working`
- **THEN** a file in `/s/docket/sessions/` holds the state `working`, the directory `/w/app` and the time of the run

#### Scenario: Later state replaces the earlier
- **WHEN** a session runs `dkt activity working` and then `dkt activity waiting`
- **THEN** one file holds that session's record, with the state `waiting`

#### Scenario: Ended removes the record
- **WHEN** a session that ran `dkt activity working` runs `dkt activity ended`
- **THEN** no file holds a record of that session

### Requirement: Session and directory
When standard input holds a JSON object, its string `session_id` SHALL name the session and its string `cwd` SHALL be the directory. Without `cwd`, the directory SHALL be the command's working directory; without `session_id`, the directory SHALL name the session. The directory SHALL be recorded with its symbolic links resolved. Standard input that is a terminal SHALL NOT be read.

#### Scenario: Two sessions in one directory
- **WHEN** two runs in `/w/app` pass `{"session_id": "a", "cwd": "/w/app"}` and `{"session_id": "b", "cwd": "/w/app"}`
- **THEN** two files hold records, one for each session

#### Scenario: No hook input
- **WHEN** `dkt activity working` runs in `/w/app` with no standard input, twice
- **THEN** one file holds a record with the directory `/w/app`

### Requirement: Activity never stops the agent
`dkt activity` SHALL print nothing to standard output or standard error and SHALL exit with 0 in every case, including a missing or unknown state, extra arguments, standard input that is not JSON, no location for the state home, and a folder it cannot write. It SHALL start no process.

#### Scenario: Unknown state
- **WHEN** a session runs `dkt activity busy`
- **THEN** nothing is printed, no file is written, and dkt exits with 0

#### Scenario: Folder that cannot be written
- **WHEN** `docket/sessions/` under the state home is not writable and a session runs `dkt activity working`
- **THEN** nothing is printed and dkt exits with 0

#### Scenario: No home
- **WHEN** neither `HOME` nor `XDG_STATE_HOME` is set and a session runs `dkt activity working`
- **THEN** nothing is printed and dkt exits with 0

### Requirement: Activity window
A record SHALL count for 15 minutes after the time it was written; after that, it SHALL be treated as absent.

#### Scenario: Record past the window
- **WHEN** the only record for a worktree was written 16 minutes ago
- **THEN** readers treat that worktree as having no session

### Requirement: Old records removed
Each write by `dkt activity` SHALL remove the files in `docket/sessions/` whose last write is older than the window.

#### Scenario: Old record removed
- **WHEN** a record was written 20 minutes ago and another session runs `dkt activity working`
- **THEN** the old record's file no longer exists
