## ADDED Requirements

### Requirement: Ignored files inspected
`dkt inspect` SHALL report `install-ignored`, as `dkt init` would after writing, for each file in the repository that `dkt init` would install or keep and that git ignores and does not track, whether or not the file exists yet. It SHALL change no ignore file.

#### Scenario: Ignored file inspected
- **WHEN** `.git/info/exclude` holds `openspec/` on its seventh line and a user runs `dkt inspect --json`
- **THEN** the findings hold `install-ignored` naming `openspec/config.yaml` with `.git/info/exclude`, line 7 and `openspec/`, and `.git/info/exclude` is byte-for-byte unchanged

### Requirement: Main spec titles inspected
`dkt inspect` SHALL report `title-missing` for each main spec under the worktree's `openspec/specs/` that has no title, as `dkt lint` does when no change is named. When `openspec` itself is a symbolic link (`symlink-refused`), it SHALL read nothing under it for this check.

#### Scenario: Main spec title inspected
- **WHEN** `openspec/specs/booking/spec.md` exists without a `docket.yaml` beside it
- **THEN** `dkt inspect` reports `title-missing` naming `openspec/specs/booking/docket.yaml` and exits with 1

### Requirement: Changes of another schema inspected
`dkt inspect` SHALL report `other-schema` for each active change in the worktree's `openspec/changes/` that is a change of another schema, naming its `.openspec.yaml`, its schema, and how many checkboxes of its `tasks.md` are ticked out of how many. When `openspec` itself is a symbolic link (`symlink-refused`), it SHALL read nothing under it for this check.

#### Scenario: Change of another schema inspected
- **WHEN** `openspec/changes/add-x/.openspec.yaml` holds `schema: spec-driven` and its `tasks.md` has four checkboxes, none ticked
- **THEN** `dkt inspect` reports `other-schema` naming `openspec/changes/add-x/.openspec.yaml`, `spec-driven` and 0 of 4

### Requirement: Doctor fixes adoption problems
The doctor SHALL take each `install-ignored`, `title-missing` and `other-schema` problem as the steps it lists, and act only after the user agrees: for an ignore rule, first show which files the change would let git see and whether the ignore file is shared or local; for titles, list a title drafted from each spec's Purpose, then write them together; for a change of another schema, propose converting it only when no step is ticked.

#### Scenario: Ignore rule changed with consent
- **WHEN** the doctor reaches `install-ignored` for files under `.claude/skills/`, ignored by `.claude/` in `.gitignore`
- **THEN** it proposes a rule that lets in `.claude/skills/`, lists the files git would then see, says `.gitignore` is shared with everyone who clones the project, and edits it only after the user agrees

#### Scenario: Titles written with consent
- **WHEN** the doctor reaches `title-missing` for three main specs
- **THEN** it lists a drafted title for each of the three before writing any, and writes the three `docket.yaml` files only after the user agrees

#### Scenario: Unstarted change converted
- **WHEN** the doctor reaches `other-schema` for `add-x` with 0 of 4 steps ticked and the user agrees to convert it
- **THEN** the agent converts `add-x` to the `docket` schema by the conversion guide shipped with the doctor, and `dkt lint add-x` then reports no finding

#### Scenario: Started change left
- **WHEN** the doctor reaches `other-schema` for `add-x` with 2 of 4 steps ticked
- **THEN** it proposes no conversion, says to finish `add-x` with its own files and archive it with `dkt archive`, and lists any capability `add-x` adds that has no title

## MODIFIED Requirements

### Requirement: Trailers on the default branch
For each active change whose folder is in the default branch's tree and whose plan there has a task, `dkt inspect` SHALL report `no-task-commits` when no commit reachable from the default branch's tip and not from the change's recorded base carries both `Change: <slug>` and a `Task:` trailer. A change whose base is missing or not an ancestor of the tip, or that is a change of another schema, SHALL be skipped. `result.changes` SHALL list each such change, with the reason when skipped.

#### Scenario: Squashed change
- **WHEN** `add-x` is in the default branch's tree with a plan of two tasks, and the only commit since its base on the default branch carries no `Change: add-x` trailer
- **THEN** the findings hold `no-task-commits` for `add-x`

#### Scenario: Merged with a merge commit
- **WHEN** `add-x` reached the default branch through a merge commit whose second parent's history has commits with `Change: add-x` and `Task: 1`
- **THEN** no finding names `add-x`

#### Scenario: Base not an ancestor
- **WHEN** `add-x` is in the default branch's tree and its recorded base is not an ancestor of the default branch's tip
- **THEN** no finding names `add-x`, and `result.changes` lists `add-x` as skipped with the reason

#### Scenario: Change of another schema skipped
- **WHEN** `add-x` is in the default branch's tree with `schema: spec-driven` and a `plan.md` with a task, and no commit carries `Change: add-x`
- **THEN** no finding names `add-x`, and `result.changes` lists `add-x` as skipped, saying it was made with the `spec-driven` schema
