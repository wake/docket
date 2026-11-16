## ADDED Requirements

### Requirement: Absolute paths as Windows writes them
On Windows, every absolute path dkt prints or compares, including the repository root and the worktree paths it reads from git, SHALL be written with backslashes, as Windows writes it, with the drive letter as git gives it; paths relative to the repository root SHALL keep forward slashes on every system. On macOS and Linux paths SHALL stay as they are.

#### Scenario: Repository root on Windows
- **WHEN** on Windows, a command runs in a repository whose root git prints as `C:/work/repo`
- **THEN** `repo.root` in its JSON output is `C:\work\repo`

#### Scenario: Worktree path compared with one dkt builds
- **WHEN** on Windows, a linked worktree's path comes from `git worktree list` and the same worktree's root from `git rev-parse --show-toplevel`
- **THEN** both are written with backslashes and are equal

#### Scenario: Relative path keeps forward slashes
- **WHEN** on Windows, a finding names a file of the repository
- **THEN** its path relative to the repository root, such as `openspec/changes/add-x/plan.md`, keeps forward slashes
