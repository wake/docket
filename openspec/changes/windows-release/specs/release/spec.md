## MODIFIED Requirements

### Requirement: Publishing a version tag
Pushing a tag of the form `vX.Y.Z` on a commit of the default branch SHALL publish a GitHub release of that tag with one executable per target, macOS, Linux and Windows each on Intel and ARM, named `dkt-<os>-<arch>`, with `.exe` on Windows and no version in the name, and `checksums.txt` holding the SHA-256 digest of each. Any other tag SHALL publish nothing.

#### Scenario: Version tag pushed
- **WHEN** a maintainer pushes the tag `v0.16.0` on a commit whose plugin files name that version
- **THEN** the GitHub release `v0.16.0` holds `dkt-darwin-amd64`, `dkt-darwin-arm64`, `dkt-linux-amd64`, `dkt-linux-arm64`, `dkt-windows-amd64.exe`, `dkt-windows-arm64.exe` and `checksums.txt`, which lists each of the six

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

### Requirement: Snapshot on every pull request
CI SHALL build every target with the release configuration on each pull request and push to the default branch, publish nothing, check that the executables it built carry the names a release publishes and that `checksums.txt` holds the digest of each, and install the build for its own machine with `install.sh` from a local copy of the snapshot, checking that the installed dkt reports the version the build stamped.

#### Scenario: Snapshot built and installed
- **WHEN** CI runs on a pull request
- **THEN** every target is built, no release or tag is created, and the snapshot installed by `install.sh` prints the version the build stamped

#### Scenario: Asset names drift
- **WHEN** a pull request changes the release configuration so that an executable has another name than `install.sh` asks for, or a Windows executable loses its `.exe`
- **THEN** the check of the snapshot fails and so does CI

#### Scenario: Checksum missing
- **WHEN** the snapshot's `checksums.txt` lacks an executable, or holds another digest for it than the file built
- **THEN** the check of the snapshot fails and so does CI
