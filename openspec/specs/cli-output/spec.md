# cli-output Specification

## Purpose
Defines what every `dkt` command prints and returns, so that scripts, CI jobs
and agents can rely on the same output shape and exit codes from any command.

## Requirements

### Requirement: JSON output
Every command SHALL accept `--json`. With `--json`, the command SHALL write exactly one JSON object to standard output and nothing else, on success and on failure alike, and SHALL write diagnostics to standard error. Without `--json`, output SHALL be human-readable text.

#### Scenario: JSON on success
- **WHEN** a user runs `dkt init --json` in a repository where it succeeds
- **THEN** standard output parses as a single JSON object whose `ok` is true

#### Scenario: JSON on failure
- **WHEN** a user runs `dkt new Bad_Slug --kind feature --source x --title X --json`
- **THEN** standard output parses as a single JSON object whose `ok` is false and whose findings name the invalid slug

### Requirement: Result envelope
Every JSON result SHALL contain the fields `tool` (always "dkt"), `version`, `command`, `repo`, `ok`, `findings`, `unreadable` and `result`, including results of usage and environment errors. `repo` SHALL be null when the repository could not be identified. Each finding SHALL carry `rule` and `message`, and `change`, `file` and `line` when they apply.

#### Scenario: Envelope fields present
- **WHEN** any command runs with `--json`
- **THEN** the object has all eight fields, and `findings` and `unreadable` are arrays, empty when there is nothing to report

### Requirement: Repository identity
Every result SHALL identify what it describes: the repository root path, the full SHA of HEAD, the current branch name or null when HEAD is detached, and the URL of the `origin` remote or null when there is none.

#### Scenario: Detached HEAD
- **WHEN** a command runs with HEAD detached
- **THEN** `repo.branch` is null and `repo.head` is the full SHA of the checked-out commit

#### Scenario: No origin remote
- **WHEN** the repository has no `origin` remote
- **THEN** `repo.origin` is null and the command otherwise behaves as usual

### Requirement: Exit codes
Every command SHALL exit with 0 when it succeeded and has nothing to report, 1 when it ran and found problems or refused the request, 2 on a usage error, 3 when it could not read a source it needed, and 4 when its environment is unusable: no git repository, `git` missing, or no usable OpenSpec CLI for a command that needs one.

#### Scenario: Usage error
- **WHEN** a user runs `dkt new` without a slug
- **THEN** the command exits with 2 and prints its usage, or with `--json` a JSON object whose finding has the rule `usage`

#### Scenario: Not a git repository
- **WHEN** a user runs any command with `--json` outside a git repository
- **THEN** the command exits with 4 and prints one JSON object whose `repo` is null and whose finding has the rule `environment`

### Requirement: Unreadable is not empty
When a command cannot read a source it needed, it SHALL list that source in `unreadable` with the reason, SHALL NOT report that source as holding no changes, and SHALL exit with 3 even when it also found problems elsewhere.

#### Scenario: Unreadable source with other findings
- **WHEN** a command finds a problem in one worktree and cannot read another
- **THEN** it reports both and exits with 3
