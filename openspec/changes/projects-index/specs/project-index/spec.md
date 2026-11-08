## Purpose

Defines the index of Docket projects that dkt keeps on a machine, the `dkt projects` command that edits and lists it, and how dkt adds a project on its own, so that a tool working across a machine's projects can read one list instead of keeping its own.

## ADDED Requirements

### Requirement: Index file
The index SHALL be the file `docket/projects.json` under `$XDG_STATE_HOME` when that variable is an absolute path, and under `$HOME/.local/state` otherwise. It SHALL hold one JSON object whose `projects` is an array of entries, each with `path` (an absolute path), `origin` (a URL or null), `added` (an RFC 3339 time in UTC) and `ignored` (a boolean). A missing file SHALL be an empty index. dkt SHALL write it by replacing the whole file, creating its folders when needed.

#### Scenario: Index under the state home
- **WHEN** `XDG_STATE_HOME` is `/s` and a project is added
- **THEN** the entry is written to `/s/docket/projects.json`

#### Scenario: Index without a state home
- **WHEN** `XDG_STATE_HOME` is not set, `HOME` is `/h`, and a project is added
- **THEN** the entry is written to `/h/.local/state/docket/projects.json`, and the folders are created

#### Scenario: Relative state home
- **WHEN** `XDG_STATE_HOME` is `state`, `HOME` is `/h`, and a project is added
- **THEN** the entry is written to `/h/.local/state/docket/projects.json`

### Requirement: Index that cannot be read
When the index exists but cannot be read, is not valid JSON, or does not have the form under Index file, `dkt projects` SHALL list it in `unreadable` with the reason, SHALL NOT write it, and SHALL exit with 3.

#### Scenario: Index that is not JSON
- **WHEN** the index holds `not json` and a user runs `dkt projects add --json` in a Docket project
- **THEN** `unreadable` lists the index, the file still holds `not json`, and dkt exits with 3

### Requirement: Project path
A project SHALL be identified by the path of its repository's main worktree, the first one `git worktree list` reports. Given a directory, dkt SHALL use the main worktree of the repository holding it, so that a subdirectory or a linked worktree gives the same path. A Docket project SHALL be one whose worktree holding the given directory has a regular file `docket/workflow.md`.

#### Scenario: Linked worktree
- **WHEN** a user runs `dkt projects add` in a linked worktree of the repository at `/w/app`, and that worktree has `docket/workflow.md`
- **THEN** the entry's path is `/w/app`

#### Scenario: Subdirectory
- **WHEN** a user runs `dkt projects add` in `/w/app/internal` of a Docket project at `/w/app`
- **THEN** the entry's path is `/w/app`

### Requirement: Add a project
`dkt projects add [<path>]` SHALL add the Docket project holding `<path>`, or the current directory when none is given, with its `origin` remote's URL (null without one), the current time and `ignored` false. `result.state` SHALL be `added`. When an entry with that path exists and is ignored, it SHALL set `ignored` to false and report `unignored`; when it exists and is not ignored, it SHALL write nothing and report `exists`. `result.project` SHALL be the entry as it stands afterwards.

#### Scenario: New project
- **WHEN** a user runs `dkt projects add --json` in a Docket project not in the index
- **THEN** the index holds its entry with `ignored` false, `result.state` is `added`, and dkt exits with 0

#### Scenario: Project already in the index
- **WHEN** a user runs `dkt projects add --json` in a Docket project already in the index and not ignored
- **THEN** the index file is not written, `result.state` is `exists`, and dkt exits with 0

#### Scenario: Ignored project added
- **WHEN** a user runs `dkt projects add --json` in a Docket project whose entry is ignored
- **THEN** the entry's `ignored` is false, its `added` is unchanged, `result.state` is `unignored`, and dkt exits with 0

### Requirement: Add refuses other directories
`dkt projects add` SHALL write nothing and exit with 1, reporting `not-a-repository` when the path is not in a git repository, and `not-a-project` when it is in one without `docket/workflow.md`.

#### Scenario: Repository without Docket
- **WHEN** a user runs `dkt projects add --json` in a git repository that has no `docket/workflow.md`
- **THEN** the findings hold `not-a-project`, the index is not written, and dkt exits with 1

#### Scenario: Not a repository
- **WHEN** a user runs `dkt projects add /tmp/plain --json` and `/tmp/plain` is a folder outside any git repository
- **THEN** the findings hold `not-a-repository` and dkt exits with 1

### Requirement: List the projects
`dkt projects ls` SHALL report every entry in `result.projects`, sorted by path, each with `path`, `origin`, `added`, `ignored` and `missing`. `missing` SHALL be true when nothing exists at the path, and the entry SHALL stay in the index. A path that cannot be checked for another reason SHALL be listed in `unreadable` with `missing` false. Without `--json`, it SHALL print one line per entry: the path, followed by `missing` and `ignored` when they apply.

#### Scenario: Project on an unmounted disk
- **WHEN** an entry's path does not exist and a user runs `dkt projects ls --json`
- **THEN** that entry has `missing` true, the index still holds it, and dkt exits with 0

#### Scenario: Ignored entries listed
- **WHEN** the index holds one ignored entry and one that is not, and a user runs `dkt projects ls --json`
- **THEN** `result.projects` holds both, with `ignored` true on the first

#### Scenario: Path that cannot be checked
- **WHEN** an entry's path is inside a folder dkt may not search, and a user runs `dkt projects ls --json`
- **THEN** that entry is listed with `missing` false, `unreadable` names its path, and dkt exits with 3

#### Scenario: Empty index
- **WHEN** there is no index file and a user runs `dkt projects ls --json`
- **THEN** `result.projects` is an empty array, no file is written, and dkt exits with 0

### Requirement: Path argument
`dkt projects rm` and `dkt projects ignore` SHALL take one path. When it is a directory in a git repository, the project path under Project path SHALL be used; otherwise the path itself, made absolute against the current directory and cleaned, so that an entry whose path is gone can still be named.

#### Scenario: Path that is gone
- **WHEN** the index holds `/w/old`, nothing exists there, and a user runs `dkt projects rm /w/old --json`
- **THEN** the entry is removed

### Requirement: Remove a project
`dkt projects rm <path>` SHALL remove the entry with that path and report `result.state` `removed`. When no entry has that path, it SHALL write nothing, report `not-in-index` and exit with 1.

#### Scenario: Remove an entry
- **WHEN** the index holds the project at `/w/app` and a user runs `dkt projects rm /w/app --json`
- **THEN** the index no longer holds it, `result.state` is `removed`, and dkt exits with 0

#### Scenario: Remove an unknown path
- **WHEN** no entry has the path `/w/none` and a user runs `dkt projects rm /w/none --json`
- **THEN** the findings hold `not-in-index`, the index is not written, and dkt exits with 1

### Requirement: Ignore a project
`dkt projects ignore <path>` SHALL set `ignored` true on the entry with that path and report `ignored`, or report `exists` and write nothing when it is already ignored. When no entry has that path, it SHALL add one with `ignored` true, the origin when the path is in a repository (null otherwise) and the current time, and report `ignored`. The path need not be a Docket project.

#### Scenario: Ignore a registered project
- **WHEN** the index holds the project at `/w/app`, not ignored, and a user runs `dkt projects ignore /w/app --json`
- **THEN** the entry's `ignored` is true, `result.state` is `ignored`, and dkt exits with 0

#### Scenario: Ignore before registering
- **WHEN** no entry has the path of a Docket project and a user runs `dkt projects ignore` on it
- **THEN** the index holds an entry for it with `ignored` true, and a later `dkt status` in that project does not change the entry

### Requirement: Projects command form
`dkt projects` SHALL take one of the subcommands `add`, `ls`, `rm` and `ignore`; a missing or unknown subcommand, or a wrong number of paths for it, SHALL be a usage error. It SHALL run outside a git repository too, with `repo` null, and the envelope's `command` SHALL be `projects`. It SHALL NOT contact a remote. When the index has no location, because neither `HOME` nor an absolute `XDG_STATE_HOME` is set, it SHALL report `no-home` and exit with 4.

#### Scenario: Unknown subcommand
- **WHEN** a user runs `dkt projects prune --json`
- **THEN** the findings hold `usage` and dkt exits with 2

#### Scenario: List outside a repository
- **WHEN** a user runs `dkt projects ls --json` outside any git repository
- **THEN** `repo` is null, `result.projects` lists the index, and dkt exits with 0

#### Scenario: No home
- **WHEN** neither `HOME` nor `XDG_STATE_HOME` is set and a user runs `dkt projects ls --json`
- **THEN** the findings hold `no-home` and dkt exits with 4

### Requirement: Automatic registration
After any command other than `dkt projects` has run with a repository identified, whatever its exit code, dkt SHALL add the Docket project it ran in, as `dkt projects add` would, when no entry has that path, ignored or not. It SHALL NOT write the index otherwise. It SHALL NOT run for `--version`, `--help`, a usage error, or a directory that is not a Docket project.

#### Scenario: First command in a project
- **WHEN** a user runs `dkt status` in a Docket project not in the index
- **THEN** the index holds its entry with `ignored` false

#### Scenario: Project already registered
- **WHEN** a user runs `dkt lint` in a Docket project already in the index
- **THEN** the index file is not written

#### Scenario: Init registers the project
- **WHEN** a user runs `dkt init --tools claude` in a repository without Docket
- **THEN** once it has written `docket/workflow.md`, the index holds the project

#### Scenario: Failing command registers
- **WHEN** a user runs `dkt sync add-none` in a Docket project not in the index, and it exits with 1 because there is no such change
- **THEN** the index holds the project

#### Scenario: Status in a repository without Docket
- **WHEN** a user runs `dkt status` in a git repository that has no `docket/workflow.md`
- **THEN** the index is not written

### Requirement: Settings file
The settings SHALL be the file `docket/config.json` under `$XDG_CONFIG_HOME` when that variable is an absolute path, and under `$HOME/.config` otherwise. When its `projects.autoRegister` is false, automatic registration SHALL NOT run; when the file or the key is missing, it SHALL be true. dkt SHALL NOT write this file.

#### Scenario: Registration turned off
- **WHEN** the settings hold `{"projects": {"autoRegister": false}}` and a user runs `dkt status` in a Docket project not in the index
- **THEN** the index is not written

#### Scenario: No settings file
- **WHEN** there is no settings file and a user runs `dkt status` in a Docket project not in the index
- **THEN** the index holds the project

### Requirement: Registration does not fail the command
When automatic registration cannot complete, because the settings or the index cannot be read or do not have their form, neither `HOME` nor the variable for a file is set, or the index cannot be written, dkt SHALL print one warning line to standard error, naming the file, or the variables when there is no file, and the reason, and SHALL NOT register. The command's standard output and exit code SHALL be what they would be without registration.

#### Scenario: Settings that are not JSON
- **WHEN** the settings file holds `{` and a user runs `dkt status --json` in a Docket project
- **THEN** standard error holds a warning naming the settings file, the index is not written, and standard output and the exit code are as without registration

#### Scenario: Index that cannot be written
- **WHEN** the index's folder cannot be created and a user runs `dkt status --json` in a Docket project not in the index
- **THEN** standard error holds a warning naming the index, and the exit code is as without registration

#### Scenario: Broken index left alone
- **WHEN** the index holds `not json` and a user runs `dkt status --json` in a Docket project
- **THEN** standard error holds a warning naming the index, the file still holds `not json`, and standard output and the exit code are as without registration
