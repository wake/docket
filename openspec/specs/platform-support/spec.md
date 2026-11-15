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
