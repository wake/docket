# Spec Delta

## MODIFIED Requirements

### Requirement: Slug never reused
`dkt new` SHALL NOT create a change under a slug that any scanned source already uses, whether active or archived. When the slug is used, it SHALL create the change under the first of `<slug>-2`, `<slug>-3`, ... that no scanned source uses, and SHALL report that slug and every place the given slug is used. When any source is unreadable it SHALL create nothing and exit with 3, because it cannot show that the slug is unused. Of two runs that create the same slug in one worktree at the same time, one SHALL create the slug and the other SHALL take the next suffix without touching what the first created.

#### Scenario: Slug active in another worktree
- **WHEN** worktree B holds `add-x` and a user runs `dkt new add-x --kind feature --source x --title X` in worktree A
- **THEN** the command creates `openspec/changes/add-x-2/`, `result.slug` is `add-x-2`, the output names worktree B, and it exits with 0

#### Scenario: Slug archived
- **WHEN** the default branch tip holds `openspec/changes/archive/2025-01-02-add-x/` and a user runs `dkt new add-x --kind feature --source x --title X`
- **THEN** the command creates `openspec/changes/add-x-2/`, the output names the archived folder, and it exits with 0

#### Scenario: Unreadable worktree blocks creation
- **WHEN** a listed worktree cannot be read and a user runs `dkt new add-x --kind feature --source x --title X`
- **THEN** nothing is created and the command exits with 3

#### Scenario: Concurrent runs keep the first
- **WHEN** two runs of `dkt new add-x --kind feature --source x --title X` start at the same time in one worktree
- **THEN** one creates `openspec/changes/add-x/` and the other `openspec/changes/add-x-2/`, both exit with 0, and `add-x` keeps everything the first run wrote

### Requirement: Kind and source
`dkt new` SHALL require `--kind` with one of the values `feature`, `change-request` or `baseline`, and SHALL require a non-empty `--source` saying where the request came from. A fix that restores a known answer is a record, not a change.

#### Scenario: Missing source
- **WHEN** a user runs `dkt new add-x --kind feature`
- **THEN** the command exits with 2

#### Scenario: Unknown kind
- **WHEN** a user runs `dkt new add-x --kind fix --source x`
- **THEN** the command exits with 2 and lists the three kinds
