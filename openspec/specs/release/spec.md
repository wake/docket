# Release Specification

## Purpose
Defines how a version of dkt is published as a GitHub release, how the Claude Code plugin follows that version, and how `install.sh` installs a published dkt on a machine.

## Requirements

### Requirement: Publishing a version tag
Pushing a tag of the form `vX.Y.Z` on a commit of the default branch SHALL publish a GitHub release of that tag with one executable per target, macOS and Linux each on Intel and ARM, named `dkt-<os>-<arch>` with no version in the name, and `checksums.txt` holding their SHA-256 digests. Any other tag SHALL publish nothing.

#### Scenario: Version tag pushed
- **WHEN** a maintainer pushes the tag `v0.16.0` on a commit whose plugin files name that version
- **THEN** the GitHub release `v0.16.0` holds `dkt-darwin-amd64`, `dkt-darwin-arm64`, `dkt-linux-amd64`, `dkt-linux-arm64` and `checksums.txt`

#### Scenario: Another tag pushed
- **WHEN** a maintainer pushes the tag `v0.16.0-rc.1`
- **THEN** nothing is published, whether or not the workflow's tag filter lets it through

#### Scenario: Tag outside the default branch
- **WHEN** a maintainer pushes the tag `v0.17.0` on a commit that is not on `main`, whose plugin files name that version
- **THEN** the release workflow fails and nothing is published

#### Scenario: Two tags on one commit
- **WHEN** a commit whose plugin files name `0.16.0` holds the tags `v0.16.0` and `v0.17.0`
- **THEN** the run for `v0.16.0` publishes only that tag, with executables that report it, and the run for `v0.17.0` fails

#### Scenario: Executable reports the tag
- **WHEN** a user runs `dkt --version` with the executable from the release `v0.16.0`
- **THEN** it prints `dkt v0.16.0`

### Requirement: Plugin follows the version
`.claude-plugin/plugin.json` SHALL hold the version as `X.Y.Z`, and `.claude-plugin/marketplace.json` SHALL list the plugin `docket` with this repository as its source, at the tag `vX.Y.Z` of the same version.

#### Scenario: Marketplace names the same version
- **WHEN** `plugin.json` says `0.16.0`
- **THEN** the source of the plugin in `marketplace.json` is the repository `wake/docket` at the ref `v0.16.0`

#### Scenario: One file changed without the other
- **WHEN** a pull request sets `plugin.json` to `0.17.0` and leaves `marketplace.json` at `v0.16.0`
- **THEN** the tests fail

#### Scenario: Tag differs from the plugin
- **WHEN** a maintainer pushes the tag `v0.17.0` on a commit whose `plugin.json` says `0.16.0`
- **THEN** the release workflow fails and nothing is published

### Requirement: Snapshot on every pull request
CI SHALL build every target with the release configuration on each pull request and push to the default branch, publish nothing, and install the build for its own machine with `install.sh` from a local copy of the snapshot, checking that the installed dkt reports the version the build stamped.

#### Scenario: Snapshot built and installed
- **WHEN** CI runs on a pull request
- **THEN** every target is built, no release or tag is created, and the snapshot installed by `install.sh` prints the version the build stamped

#### Scenario: Asset names drift
- **WHEN** a pull request changes the release configuration so that an executable has another name than `install.sh` asks for
- **THEN** the install from the snapshot fails and so does CI

### Requirement: Installing a release
Run as `curl -fsSL https://raw.githubusercontent.com/wake/docket/main/install.sh | sh`, `install.sh` SHALL download the executable for the machine and `checksums.txt` from the latest GitHub release, and install the executable at `~/.local/bin/dkt`, creating the directory when missing and replacing any earlier dkt in one step. It SHALL print the installed path and version.

#### Scenario: Latest release installed
- **WHEN** a user on Linux on ARM runs the script
- **THEN** it downloads `dkt-linux-arm64` and `checksums.txt` from `releases/latest/download/`, `~/.local/bin/dkt` is that executable and can be run, and it prints the path and the version

#### Scenario: Each machine gets its executable
- **WHEN** `uname` reports `Darwin` with `arm64` or `x86_64`, or `Linux` with `aarch64`, `arm64`, `x86_64` or `amd64`
- **THEN** it downloads `dkt-darwin-arm64`, `dkt-darwin-amd64`, `dkt-linux-arm64` or `dkt-linux-amd64` accordingly

#### Scenario: Earlier dkt replaced
- **WHEN** `~/.local/bin/dkt` already exists and the script succeeds
- **THEN** the file holds the new executable and no temporary file is left in `~/.local/bin`

### Requirement: Choosing a version
When `DKT_VERSION` is set and not empty, `install.sh` SHALL download from the release of that version instead of the latest. The value MAY omit the leading `v`.

#### Scenario: Version with a v
- **WHEN** a user runs the script with `DKT_VERSION=v0.16.0`
- **THEN** it downloads from `releases/download/v0.16.0/`

#### Scenario: Version without a v
- **WHEN** a user runs the script with `DKT_VERSION=0.16.0`
- **THEN** it downloads from `releases/download/v0.16.0/`

#### Scenario: Empty version
- **WHEN** a user runs the script with `DKT_VERSION` set to the empty string
- **THEN** it downloads from `releases/latest/download/`

### Requirement: Verifying the download
`install.sh` SHALL compare the SHA-256 of the downloaded executable with its entry in the `checksums.txt` of the same release, and SHALL install it only when they are equal. A missing entry, or no tool to compute the digest, SHALL fail the same way as a mismatch: it SHALL NOT skip the check.

#### Scenario: Digest differs
- **WHEN** the digest of the download differs from its entry in `checksums.txt`
- **THEN** the script exits non-zero, says so on standard error, and `~/.local/bin/dkt` is as it was

#### Scenario: No entry for the executable
- **WHEN** `checksums.txt` has no line for the executable
- **THEN** the script exits non-zero and `~/.local/bin/dkt` is as it was

#### Scenario: No digest tool
- **WHEN** neither `sha256sum` nor `shasum` is on PATH
- **THEN** the script exits non-zero before downloading anything and `~/.local/bin/dkt` is as it was

#### Scenario: Either digest tool
- **WHEN** only one of `sha256sum` and `shasum` is on PATH
- **THEN** the script verifies the download with that tool and installs it

### Requirement: Failing without installing
`install.sh` SHALL exit non-zero, say why on standard error, leave `~/.local/bin/dkt` as it was and leave no temporary file in `~/.local/bin` when the machine has no published executable, `HOME` is not an absolute path, a download fails, or the executable cannot be run. It SHALL run the executable before putting it in place.

#### Scenario: Home is not an absolute path
- **WHEN** `HOME` is empty or a relative path
- **THEN** the script exits non-zero before downloading anything, its message names `HOME`, and it writes nothing

#### Scenario: Unsupported system
- **WHEN** `uname -s` reports `FreeBSD`, or a Windows shell such as `MINGW64_NT-10.0`
- **THEN** the script exits non-zero before downloading anything, and its message names the system and the releases page

#### Scenario: Unsupported architecture
- **WHEN** `uname -m` reports `riscv64`
- **THEN** the script exits non-zero before downloading anything, and its message names the architecture and the releases page

#### Scenario: Download fails
- **WHEN** the executable or `checksums.txt` cannot be downloaded, as when the release does not exist
- **THEN** the script exits non-zero, its message names the URL, and `~/.local/bin/dkt` is as it was

#### Scenario: Target is a directory
- **WHEN** `~/.local/bin/dkt` is a directory
- **THEN** the script exits non-zero and the directory is as it was

#### Scenario: Downloaded file does not run
- **WHEN** the verified executable cannot be run on the machine
- **THEN** the script exits non-zero, says so, and `~/.local/bin/dkt` is as it was

#### Scenario: Interrupted during a download
- **WHEN** the script receives a termination signal while downloading
- **THEN** it leaves no temporary file in `~/.local/bin` and `~/.local/bin/dkt` is as it was

### Requirement: A partial download runs nothing
`install.sh` SHALL take no action until the shell has read the whole script, so that a download cut short by `curl | sh` downloads and changes nothing.

#### Scenario: Script cut short
- **WHEN** the script reaches `sh` cut short at any byte before the end of its last command
- **THEN** nothing is downloaded and nothing is written under the home directory

### Requirement: Saying when the directory is not on PATH
`install.sh` SHALL say so on standard output when `~/.local/bin` is not on `PATH`, and SHALL NOT edit any shell configuration file.

#### Scenario: Directory not on PATH
- **WHEN** the script succeeds and `PATH` does not hold `~/.local/bin`
- **THEN** its output says that the directory is not on `PATH`

#### Scenario: Directory on PATH
- **WHEN** the script succeeds and `PATH` holds `~/.local/bin`
- **THEN** its output does not mention `PATH`
