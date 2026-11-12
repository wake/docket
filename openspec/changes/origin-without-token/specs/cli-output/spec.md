## MODIFIED Requirements

### Requirement: Repository identity
Inside a git repository, `repo` SHALL hold the path of the repository root, the full SHA of HEAD, the current branch name and the URL of the `origin` remote. The branch is null when HEAD is detached, HEAD is null when there is no commit yet, and the URL is null when there is no `origin` remote. When the URL has the scheme `http` or `https`, its user information SHALL be removed first, so that a token in the URL is never printed; the rest of the URL, and any other URL, SHALL be printed as it is. Outside a git repository, a command that does not need one SHALL report `repo` as null and run as usual.

#### Scenario: Detached HEAD
- **WHEN** HEAD is detached and a user runs `dkt --version --json`
- **THEN** `repo.branch` is null and `repo.head` is the full SHA of the checked-out commit

#### Scenario: No origin remote
- **WHEN** the repository has no `origin` remote and a user runs `dkt --version --json`
- **THEN** `repo.origin` is null and the rest is as usual

#### Scenario: Token in the origin URL
- **WHEN** the `origin` remote is `https://x-access-token:secret@example.com/org/app.git` and a user runs any command with `--json`
- **THEN** `repo.origin` is `https://example.com/org/app.git`

#### Scenario: Token in a URL that is not well formed
- **WHEN** the `origin` remote is `https://user:p%zz@example.com/org/app.git` and a user runs any command with `--json`
- **THEN** `repo.origin` is `https://example.com/org/app.git`

#### Scenario: SSH remote kept
- **WHEN** the `origin` remote is `ssh://git@example.com/org/app.git` or `git@example.com:org/app.git`
- **THEN** `repo.origin` is that URL as it is

#### Scenario: No commits yet
- **WHEN** a user runs `dkt --version --json` in a repository just made with `git init`, with no commit
- **THEN** `repo.head` is null and `repo.root` is that repository's root

#### Scenario: Outside a repository
- **WHEN** a user runs `dkt --version --json` outside any git repository
- **THEN** the command exits with 0 and `repo` is null
