# Platform support Specification

## Purpose
Says which operating systems dkt supports, what is verified on a system before it is supported, and how dkt keeps the protections other specs require on a system that lacks the means other systems use for them.

## Requirements

### Requirement: Supported systems
dkt SHALL be supported on macOS and Linux. It SHALL build for Windows, where only the protections this spec names are verified; running its commands on Windows SHALL NOT be described as supported, by the documentation or by the name of a check, until a spec says so.

#### Scenario: Build for Windows
- **WHEN** dkt is built for Windows on amd64 or arm64
- **THEN** the build succeeds

#### Scenario: Windows check names its limit
- **WHEN** the CI job that runs on Windows is listed among a pull request's checks
- **THEN** its name says that it covers only the build and the protections, and that dkt is not yet supported on Windows

### Requirement: Links refused on Windows
Wherever a requirement says that dkt does not read or write through a symbolic link, on Windows a symbolic link and a junction SHALL both count as one, whether it is the file itself or a folder on the path to it.

#### Scenario: Symbolic link read on Windows
- **WHEN** on Windows, a file dkt reads without following links is a symbolic link to a regular file
- **THEN** the read fails and nothing of the target is read

#### Scenario: Junction on the path on Windows
- **WHEN** on Windows, a folder on the path dkt checks for links is a junction to a folder inside the repository
- **THEN** dkt treats it as a symbolic link at that folder

### Requirement: New files never through a link
When dkt creates a file that must not exist yet, such as a new record, a new proposal, or the temporary file a write goes through, and a symbolic link or, on Windows, a junction is at that name, the creation SHALL fail and SHALL leave the link's target untouched, on macOS, Linux and Windows.

#### Scenario: Link at a new file's name
- **WHEN** a symbolic link to a file outside the folder is at the name dkt is about to create
- **THEN** the creation fails as if the file existed, and the file outside is unchanged

### Requirement: Runs that write hold each other off on Windows
The runs that other specs require not to interleave, such as two runs of `dkt new` or of `dkt issue` with a card in one worktree, SHALL hold each other off on Windows as they do on macOS and Linux, when they use the same temporary folder, as the runs of one user do: a second run waits until the first is done with the folder it locked. Runs with different temporary folders, such as those of two users, do not hold each other off on Windows.

#### Scenario: Second run waits
- **WHEN** on Windows, one run holds the lock on a folder and a second run asks for the lock on the same folder
- **THEN** the second run waits until the first releases it, and then takes it

### Requirement: No protection dropped in silence
When dkt cannot set up a protection that a spec requires, such as the lock or the refusal of a link, it SHALL fail with a message naming what it could not do, and SHALL write nothing; it SHALL NOT go on without the protection.

#### Scenario: Lock cannot be set up
- **WHEN** on Windows, the folder that holds dkt's locks cannot be created
- **THEN** the command fails with a message naming that folder and the reason, and writes nothing

### Requirement: Programs found on Windows
On Windows, wherever dkt looks for a program, git, OpenSpec or the program `DOCKET_OPENSPEC` names, it SHALL try the name with each extension in `PATHEXT`, in order, a dot added to one written without (`.com`, `.exe`, `.bat`, `.cmd` when it is unset or empty), or the name alone when it already ends in one of them, and SHALL NOT require an execute bit. The other rules for finding the program stay the same on every system.

#### Scenario: git on Windows
- **WHEN** on Windows, the environment's `Path` lists a folder holding `git.exe` and a user runs `dkt status`
- **THEN** dkt runs that `git.exe` and reports no `git-missing`

#### Scenario: OpenSpec installed by npm on Windows
- **WHEN** on Windows, `DOCKET_OPENSPEC` is unset, the environment's `Path` lists a folder holding `openspec.cmd`, and a user runs `dkt init`
- **THEN** dkt runs that `openspec.cmd` and reports no `openspec-missing`

#### Scenario: Extension given
- **WHEN** on Windows, `DOCKET_OPENSPEC` names `C:\tools\openspec.exe`, which exists
- **THEN** dkt runs that file

#### Scenario: Name alone on other systems
- **WHEN** on macOS or Linux, a folder on `PATH` holds `git.exe` but no `git`
- **THEN** dkt does not run `git.exe`

#### Scenario: Extensions in order
- **WHEN** on Windows, `PATHEXT` is `.CMD;.EXE` and a folder on `Path` holds both `openspec.exe` and `openspec.cmd`
- **THEN** dkt runs `openspec.cmd`

### Requirement: Batch files given only plain arguments
On Windows, `cmd.exe` reads the command line of a `.bat` or `.cmd` file itself, so `&`, `|`, `%` and the like in it can run other commands. When OpenSpec is such a file, dkt SHALL NOT start it with an argument holding anything but ASCII letters, digits and `-_.,/@+=:`, nor when its path holds anything else but `\`, spaces and `~`, and `(` `)` only with a space; the call SHALL fail with a message naming the word and the reason, and be reported as a failed OpenSpec call.

#### Scenario: Change name with a command separator
- **WHEN** on Windows, OpenSpec is `openspec.cmd`, a change folder is named `add-x&calc`, and a user runs `dkt lint`
- **THEN** dkt does not start OpenSpec for that change, and reports `openspec-failed` naming `add-x&calc`

#### Scenario: Program path with a command separator
- **WHEN** on Windows, `DOCKET_OPENSPEC` names `C:\tools\a&b\openspec.cmd` and a user runs `dkt init`
- **THEN** dkt does not start it, and reports `openspec-failed` naming that path

#### Scenario: Plain arguments pass
- **WHEN** on Windows, `DOCKET_OPENSPEC` is `npx -y @fission-ai/openspec@1.14.0`, found as `npx.cmd`, and a user runs `dkt new add-x --kind feature --source x --title X`
- **THEN** dkt runs it as on other systems

### Requirement: Variable names without case on Windows
On Windows, dkt SHALL read the environment variables it looks programs up with, `PATH` and `PATHEXT`, whatever the case of their names, as Windows does; when the environment holds several spellings of one, the last SHALL win.

#### Scenario: Path spelled as Windows does
- **WHEN** on Windows, the environment holds `Path` but no `PATH`, listing a folder holding `git.exe`
- **THEN** dkt finds that `git.exe`

### Requirement: Same shipped files from any checkout
A dkt built from a checkout of the repository on any system SHALL hold the shipped files byte for byte as committed, with LF line endings, so that what it installs, and the hashes it compares, are the same wherever it was built.

#### Scenario: Built from a Windows checkout
- **WHEN** dkt is built from a checkout made by Git for Windows with its default settings
- **THEN** the shipped files it holds have LF line endings and the hashes `internal/setup/history.go` records for this version

### Requirement: User folders on Windows
Wherever a requirement places dkt's files under `$HOME`, as for the index, the settings, the user template and the activity records, on Windows dkt SHALL use the folder `USERPROFILE` names, at the same paths under it, and SHALL NOT read `HOME`; a message that names `HOME` as missing SHALL name `USERPROFILE` instead. An absolute `XDG_CONFIG_HOME` or `XDG_STATE_HOME` SHALL be used as on other systems. These variables SHALL be read whatever the case of their names.

#### Scenario: Index under the user profile
- **WHEN** on Windows, `XDG_STATE_HOME` is not set, `USERPROFILE` is `C:\Users\me`, and a project is added
- **THEN** the entry is written to `C:\Users\me\.local\state\docket\projects.json`

#### Scenario: User template under the user profile
- **WHEN** on Windows, `XDG_CONFIG_HOME` is not set, `USERPROFILE` is `C:\Users\me`, and a user runs `dkt inspect --json`
- **THEN** `result.user_workflow.path` is `C:\Users\me\.config\docket\workflow.md`

#### Scenario: HOME not read on Windows
- **WHEN** on Windows, `HOME` is `C:\h`, neither `USERPROFILE` nor `XDG_STATE_HOME` is set, and a user runs `dkt projects ls --json`
- **THEN** the findings hold `no-home`, whose message names `USERPROFILE`, and dkt exits with 4

#### Scenario: State home kept on Windows
- **WHEN** on Windows, `XDG_STATE_HOME` is `D:\s` and a session in a folder runs `dkt activity working`
- **THEN** a file in `D:\s\docket\sessions\` holds the session's record

#### Scenario: Variable spelled in another case
- **WHEN** on Windows, the environment holds `UserProfile` set to `C:\Users\me` and no `USERPROFILE`, and a project is added
- **THEN** the entry is written to `C:\Users\me\.local\state\docket\projects.json`

#### Scenario: Other systems unchanged
- **WHEN** on macOS or Linux, `HOME` is `/h`, `USERPROFILE` is `/u`, `XDG_STATE_HOME` is not set, and a project is added
- **THEN** the entry is written to `/h/.local/state/docket/projects.json`
