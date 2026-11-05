# Spec Delta

## ADDED Requirements

### Requirement: Unreadable sources
When a command cannot read a source it needs, it SHALL list that source with the reason in `unreadable`, SHALL NOT treat it as empty or missing, and SHALL exit with 3, even when it also has other findings.

#### Scenario: Unreadable installed file
- **WHEN** `.claude/skills/docket/SKILL.md` cannot be read and a user runs `dkt inspect --json`
- **THEN** `unreadable` lists the file with the reason, `result.files` does not list it as `missing`, and dkt exits with 3

### Requirement: Repository required
A command that needs a git repository, `dkt init` and `dkt inspect`, SHALL do nothing outside one, report `no-repository`, and exit with 4. When `git` cannot be run, it SHALL report `git-missing` and exit with 4.

#### Scenario: Outside a repository
- **WHEN** a user runs `dkt init --json --tools claude` outside any git repository
- **THEN** no file is written, `repo` is null, the findings hold `no-repository`, and dkt exits with 4

### Requirement: Help
`dkt --help` SHALL print dkt's usage, and `dkt <command> --help` SHALL print that command's usage and flags. Both SHALL print text to standard output, whatever `--json` says, and exit with 0.

#### Scenario: Command help
- **WHEN** a user runs `dkt init --help`
- **THEN** standard output lists the flags `--tools`, `--force` and `--json` of `dkt init`, and dkt exits with 0

#### Scenario: Help ignores --json
- **WHEN** a user runs `dkt --help --json`
- **THEN** standard output is the usage text, not JSON, and dkt exits with 0

## MODIFIED Requirements

### Requirement: JSON output
Every command and `--version` SHALL accept `--json`. With `--json`, standard output SHALL hold exactly one JSON object, on success and on failure alike, and diagnostics SHALL go to standard error; `--help` is the exception, see Help. Without `--json`, output SHALL be human-readable text.

#### Scenario: JSON on success
- **WHEN** a user runs `dkt --version --json`
- **THEN** standard output parses as a single JSON object whose `ok` is true

#### Scenario: JSON on failure
- **WHEN** a user runs `dkt nosuch --json`
- **THEN** standard output parses as a single JSON object whose `ok` is false and whose `findings` hold one with the `rule` `usage`
