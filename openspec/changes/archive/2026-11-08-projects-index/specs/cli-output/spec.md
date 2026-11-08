## ADDED Requirements

### Requirement: Writes outside the repository
No command SHALL write outside the repository it runs in, except the project index that `project-index` defines: `dkt projects` edits it, and any other command may add the project it ran in. Where a requirement says that a command writes nothing, it SHALL mean nothing in the repository or its worktrees.

#### Scenario: Inspect in a new project
- **WHEN** a user runs `dkt inspect --json` in a Docket project not in the index
- **THEN** no file in the repository is written, and the index holds the project
