## ADDED Requirements

### Requirement: Origin without user information
When the URL of the `origin` remote has the scheme `http` or `https`, `repo.origin` SHALL hold it with its user information removed, so that a token in the URL is never printed; the rest of the URL, and any other URL, SHALL be printed as it is.

#### Scenario: Token in the origin URL
- **WHEN** the `origin` remote is `https://x-access-token:secret@example.com/org/app.git` and a user runs any command with `--json`
- **THEN** `repo.origin` is `https://example.com/org/app.git`

#### Scenario: Token in a URL that is not well formed
- **WHEN** the `origin` remote is `https://user:p%zz@example.com/org/app.git` and a user runs any command with `--json`
- **THEN** `repo.origin` is `https://example.com/org/app.git`

#### Scenario: SSH remote kept
- **WHEN** the `origin` remote is `ssh://git@example.com/org/app.git` or `git@example.com:org/app.git`
- **THEN** `repo.origin` is that URL as it is
