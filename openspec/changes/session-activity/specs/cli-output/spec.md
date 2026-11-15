## MODIFIED Requirements

### Requirement: JSON output
Every command other than `dkt activity`, and `--version`, SHALL accept `--json`. With `--json`, standard output SHALL hold exactly one JSON object, on success and on failure alike, and diagnostics SHALL go to standard error; `--help` is the exception, see Help. Without `--json`, output SHALL be human-readable text. `dkt activity` prints nothing, with or without `--json`, as `session-activity` says.

#### Scenario: JSON on success
- **WHEN** a user runs `dkt --version --json`
- **THEN** standard output parses as a single JSON object whose `ok` is true

#### Scenario: JSON on failure
- **WHEN** a user runs `dkt nosuch --json`
- **THEN** standard output parses as a single JSON object whose `ok` is false and whose `findings` hold one with the `rule` `usage`

#### Scenario: Activity with --json
- **WHEN** a session runs `dkt activity working --json`
- **THEN** standard output is empty and dkt exits with 0

### Requirement: Exit codes
Every command other than `dkt activity` SHALL exit with 0 when it succeeded and has nothing to report, 1 when it ran and found problems or refused the request, 2 on a usage error, 3 when it could not read a source it needed, and 4 when its environment is unusable, such as no git repository, `git` missing, or no usable OpenSpec CLI for a command that needs one. `dkt activity` always exits with 0, as `session-activity` says.

#### Scenario: Unknown command
- **WHEN** a user runs `dkt nosuch`
- **THEN** the command exits with 2, prints the usage to standard error, and standard output is empty

#### Scenario: No command
- **WHEN** a user runs `dkt` with no arguments
- **THEN** the command exits with 2 and prints the usage to standard error

#### Scenario: Help
- **WHEN** a user runs `dkt --help`
- **THEN** the usage is printed to standard output and the command exits with 0

#### Scenario: Activity usage error
- **WHEN** a session runs `dkt activity`
- **THEN** dkt exits with 0

### Requirement: Writes outside the repository
No command SHALL write outside the repository it runs in, except the project index that `project-index` defines, which `dkt projects` edits and any other command may add the project it ran in to, and the session records that `session-activity` defines, which only `dkt activity` writes. Where a requirement says that a command writes nothing, it SHALL mean nothing in the repository or its worktrees.

#### Scenario: Inspect in a new project
- **WHEN** a user runs `dkt inspect --json` in a Docket project not in the index
- **THEN** no file in the repository is written, and the index holds the project

#### Scenario: Snapshot leaves the session records
- **WHEN** the sessions folder holds a record past the window and a user runs `dkt snapshot`
- **THEN** every file in the sessions folder is unchanged
