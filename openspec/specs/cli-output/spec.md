# CLI Output Specification

## Purpose
Defines what every `dkt` command prints and returns, so that scripts, CI and coding agents can rely on the same output shape and exit codes from any command.

## Requirements

### Requirement: JSON output
Every command and `--version` SHALL accept `--json`. With `--json`, standard output SHALL hold exactly one JSON object, on success and on failure alike, and diagnostics SHALL go to standard error; `--help` is the exception, see Help. Without `--json`, output SHALL be human-readable text.

#### Scenario: JSON on success
- **WHEN** a user runs `dkt --version --json`
- **THEN** standard output parses as a single JSON object whose `ok` is true

#### Scenario: JSON on failure
- **WHEN** a user runs `dkt nosuch --json`
- **THEN** standard output parses as a single JSON object whose `ok` is false and whose `findings` hold one with the `rule` `usage`

### Requirement: Result envelope
Every JSON result SHALL contain the eight fields `tool` (always `dkt`), `version`, `command`, `repo`, `ok`, `findings`, `unreadable` and `result`, including the result of a usage error. `command` is the name of the command run: `version` for `--version`, and the empty string when no command is recognized. `findings` and `unreadable` are arrays, empty when there is nothing to report. Each finding SHALL carry `rule` and `message`, and `change`, `file` and `line` when they apply.

#### Scenario: Envelope fields present
- **WHEN** a user runs `dkt --version --json`
- **THEN** the object has all eight fields, `command` is `version`, and `findings` and `unreadable` are empty arrays

#### Scenario: Envelope on a usage error
- **WHEN** a user runs `dkt nosuch --json`
- **THEN** the object also has all eight fields, and `command` is the empty string

### Requirement: Repository identity
Inside a git repository, `repo` SHALL hold the path of the repository root, the full SHA of HEAD, the current branch name and the URL of the `origin` remote. The branch is null when HEAD is detached, HEAD is null when there is no commit yet, and the URL is null when there is no `origin` remote. Outside a git repository, a command that does not need one SHALL report `repo` as null and run as usual.

#### Scenario: Detached HEAD
- **WHEN** HEAD is detached and a user runs `dkt --version --json`
- **THEN** `repo.branch` is null and `repo.head` is the full SHA of the checked-out commit

#### Scenario: No origin remote
- **WHEN** the repository has no `origin` remote and a user runs `dkt --version --json`
- **THEN** `repo.origin` is null and the rest is as usual

#### Scenario: No commits yet
- **WHEN** a user runs `dkt --version --json` in a repository just made with `git init`, with no commit
- **THEN** `repo.head` is null and `repo.root` is that repository's root

#### Scenario: Outside a repository
- **WHEN** a user runs `dkt --version --json` outside any git repository
- **THEN** the command exits with 0 and `repo` is null

### Requirement: Version
`dkt --version` SHALL print one line `dkt <version>` and exit with 0; with `--json`, the version goes in `result.version`. The version SHALL be the one stamped at build time; for a build by `go install` at a module version, it is that module version; for a build from a working tree, it is `dev`. The envelope's `version` field SHALL hold the same version.

#### Scenario: Release build
- **WHEN** the build stamps the version `v0.16.0` and a user runs `dkt --version`
- **THEN** standard output is `dkt v0.16.0` and the command exits with 0

#### Scenario: Working tree build
- **WHEN** dkt is built from a working tree and a user runs `dkt --version --json`
- **THEN** both `result.version` and `version` are `dev`

### Requirement: Exit codes
Every command SHALL exit with 0 when it succeeded and has nothing to report, 1 when it ran and found problems or refused the request, 2 on a usage error, 3 when it could not read a source it needed, and 4 when its environment is unusable, such as no git repository, `git` missing, or no usable OpenSpec CLI for a command that needs one.

#### Scenario: Unknown command
- **WHEN** a user runs `dkt nosuch`
- **THEN** the command exits with 2, prints the usage to standard error, and standard output is empty

#### Scenario: No command
- **WHEN** a user runs `dkt` with no arguments
- **THEN** the command exits with 2 and prints the usage to standard error

#### Scenario: Help
- **WHEN** a user runs `dkt --help`
- **THEN** the usage is printed to standard output and the command exits with 0

### Requirement: Unreadable sources
When a command cannot read a source it needs, it SHALL list that source with the reason in `unreadable`, SHALL NOT treat it as empty or missing, and SHALL exit with 3, even when it also has other findings.

#### Scenario: Unreadable installed file
- **WHEN** `.claude/skills/docket/SKILL.md` cannot be read and a user runs `dkt inspect --json`
- **THEN** `unreadable` lists the file with the reason, `result.files` does not list it as `missing`, and dkt exits with 3

### Requirement: Repository required
A command that needs a git repository, `dkt init`, `dkt inspect`, `dkt new`, `dkt sync`, `dkt lint` and `dkt conflicts`, SHALL do nothing outside one, report `no-repository`, and exit with 4. When `git` cannot be run, it SHALL report `git-missing` and exit with 4.

#### Scenario: Init outside a repository
- **WHEN** a user runs `dkt init --json --tools claude` outside any git repository
- **THEN** no file is written, `repo` is null, the findings hold `no-repository`, and dkt exits with 4

#### Scenario: New outside a repository
- **WHEN** a user runs `dkt new add-x --json --kind feature --source x --title X` outside any git repository
- **THEN** nothing is created, OpenSpec is not run, the findings hold `no-repository`, and dkt exits with 4

#### Scenario: Sync outside a repository
- **WHEN** a user runs `dkt sync add-x --json` outside any git repository
- **THEN** nothing is written, the findings hold `no-repository`, and dkt exits with 4

#### Scenario: Lint outside a repository
- **WHEN** a user runs `dkt lint --json` outside any git repository
- **THEN** the findings hold `no-repository` and dkt exits with 4

#### Scenario: Conflicts outside a repository
- **WHEN** a user runs `dkt conflicts --json` outside any git repository
- **THEN** OpenSpec is not run, the findings hold `no-repository`, and dkt exits with 4

### Requirement: Help
`dkt --help` SHALL print dkt's usage, and `dkt <command> --help` SHALL print that command's usage and flags. Both SHALL print text to standard output, whatever `--json` says, and exit with 0.

#### Scenario: Command help
- **WHEN** a user runs `dkt init --help`
- **THEN** standard output lists the flags `--tools`, `--force` and `--json` of `dkt init`, and dkt exits with 0

#### Scenario: Help ignores --json
- **WHEN** a user runs `dkt --help --json`
- **THEN** standard output is the usage text, not JSON, and dkt exits with 0
