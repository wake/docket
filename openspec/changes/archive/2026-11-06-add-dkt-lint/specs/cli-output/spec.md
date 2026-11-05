## MODIFIED Requirements

### Requirement: Repository required
A command that needs a git repository, `dkt init`, `dkt inspect`, `dkt new`, `dkt sync` and `dkt lint`, SHALL do nothing outside one, report `no-repository`, and exit with 4. When `git` cannot be run, it SHALL report `git-missing` and exit with 4.

#### Scenario: Outside a repository
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
