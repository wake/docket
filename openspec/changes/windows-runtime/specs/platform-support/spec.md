# Spec Delta

## ADDED Requirements

### Requirement: Programs found on Windows
On Windows, wherever dkt looks for a program, git, OpenSpec or the program `DOCKET_OPENSPEC` names, it SHALL try the name with each extension in `PATHEXT`, in order (`.com`, `.exe`, `.bat`, `.cmd` when it is unset or empty), or the name alone when it already ends in one of them, and SHALL take any regular file found as one it can run. The other rules for finding the program stay the same on every system.

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
