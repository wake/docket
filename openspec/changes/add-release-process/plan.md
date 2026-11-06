# Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A pushed `vX.Y.Z` tag publishes dkt's executables and `checksums.txt` as a GitHub release, `install.sh` installs them with their digest verified, and the plugin's version and marketplace source follow the same version.

**Architecture:** `.goreleaser.yaml` holds the build; `.github/workflows/release.yml` runs it on a tag, and a new job in `ci.yml` runs it as a snapshot on every pull request. `install.sh` is a POSIX shell script with everything inside `main`. `release_test.go`, in the root package, tests `install.sh` through fake `curl` and `uname` on `PATH`, the snapshot end to end, and the plugin's version.

**Tech Stack:** POSIX sh, GoReleaser v2, GitHub Actions, Go 1.24 standard library for tests.

**Spec:** `openspec/changes/add-release-process/proposal.md`, `design.md`, `specs/release/spec.md`.

## Global Constraints

- Targets: `darwin` and `linux`, each on `amd64` and `arm64`. No Windows.
- Asset names: `dkt-<os>-<arch>` and `checksums.txt`, with no version.
- Stamp: `-X github.com/wake/docket/internal/cli.Version={{ .Tag }}`, as `tools/dev-bin/dkt` stamps the same variable.
- Releases URL: `https://github.com/wake/docket/releases`; latest at `/latest/download/<name>`, a version at `/download/<tag>/<name>`.
- Install target: `~/.local/bin/dkt`. `install.sh` has no setting for the directory or the URL, and no test hook.
- Plugin version: `0.16.0` in `plugin.json`, `v0.16.0` as the marketplace's `ref`.
- Do not push a tag, create a release or run GoReleaser without `--snapshot` or `check`.
- Do not touch `AGENTS.md`, `shipped/`, `skills/`, `internal/setup/`, `docket/`, or any Go code outside `release_test.go`.
- No new Go dependency. Commits carry `Change: add-release-process/<N>` in the last paragraph, with no other trailer. `git add` file by file.
- `go vet ./...` and `go test ./...` pass after every task.

## Review Focus

- The two shells: `sh` is dash on the Ubuntu runner and bash in POSIX mode on macOS, and both must pass. Pinned by running `release_test.go` under both in CI's existing `test` job.
- `curl | sh` feeds the script on standard input. Pinned in Task 1: every test runs the script that way.
- A failure must never cost the user a working dkt, including an executable that verifies but does not run. Pinned in Task 1 (`TestInstallRefuses`): every case starts with an earlier `dkt` and requires it unchanged.
- `~/.local/bin/dkt` as a symbolic link: the script must replace the link and leave its target alone. Pinned in Task 1 (`TestInstallReplacesLink`).
- A home directory with a space in its path. Pinned in Task 1: every test's home is `<tmp>/home dir`.

## Reuse

| Need | Existing (path:symbol) | Decision | Reason | Evidence |
|---|---|---|---|---|
| Stamping the version | internal/cli/run.go:Version | reuse | the variable `tools/dev-bin/dkt` already stamps | `rg -n "cli.Version" tools/ internal/` |
| Version the release prints | internal/cli/version.go:ResolveVersion | reuse | a stamped value wins; nothing changes | `rg -n "ResolveVersion" cmd/ internal/` |
| Writing test files | devbin_test.go:writeFiles | reuse | same package, same shape | `rg -n "func writeFiles" *_test.go` |
| Skipping POSIX tests on Windows | devbin_test.go:TestDevWrapper | reuse | same `runtime.GOOS` check | `rg -n "GOOS" devbin_test.go` |
| Test gated by an environment variable | internal/cli/realopenspec_test.go:TestRealOpenSpec | extend | same `DOCKET_TEST_*` naming; here the value is the path of the build, where that test takes `1` | `rg -n "DOCKET_TEST_OPENSPEC" internal/ .github/` |
| Building releases | external: goreleaser | reuse | the settled choice | `.goreleaser.yaml` does not exist yet |
| Running it in CI | external: goreleaser/goreleaser-action | reuse | GoReleaser's documented way, `version: '~> v2'` | https://goreleaser.com/ci/actions/ |
| Installing the executable | none | new | nothing exists; `install.sh` is the URL docket-init already names | `rg --files \| rg '(^\|/)(install\|setup\|bootstrap)(\.sh\|$)'`; `rg -n "install\.sh" . --glob '!openspec/changes/add-release-process/**'` |
| Plugin metadata | .claude-plugin/plugin.json | extend | add `version` | `rg --files -g 'plugin.json'` |
| Marketplace | none | new | required by Claude Code's plugin install | `rg --files -g 'marketplace.json'` |

---

### Task 1: install.sh

**Scenarios:** release#latest-release-installed, release#each-machine-gets-its-executable, release#earlier-dkt-replaced, release#version-with-a-v, release#version-without-a-v, release#empty-version, release#digest-differs, release#no-entry-for-the-executable, release#no-digest-tool, release#either-digest-tool, release#unsupported-system, release#unsupported-architecture, release#download-fails, release#target-is-a-directory, release#downloaded-file-does-not-run, release#interrupted-during-a-download, release#script-cut-short, release#directory-not-on-path, release#directory-on-path

**Review:** judgment

**Files:**
- Create: `install.sh`
- Create: `release_test.go`

**Interfaces:**
- Produces, in `release_test.go`, package `docket`: `type installEnv struct { home, fake, releases string }`, `newInstallEnv(t *testing.T) installEnv` and `(e installEnv) run(t *testing.T, env ...string) (stdout, stderr string, err error)`, reused by Task 2. `run` pipes `install.sh` to `sh` on standard input with `HOME`, `PATH` (the fake directory first), `FAKE_RELEASES`, `FAKE_OS=Linux`, `FAKE_ARCH=x86_64` and then `env`, whose values win; nothing else from the test's environment reaches it, so `DKT_VERSION` is unset unless a test sets it.

- [x] **Step 1: Write the failing tests** in `release_test.go`, skipping on Windows as `TestDevWrapper` does. Every test's home is `<t.TempDir()>/home dir`.

  The fake `curl`, written into the fake directory, takes the URL and `-o <file>` from its arguments, appends the URL to `$FAKE_RELEASES/requests`, strips `https://github.com/wake/docket/releases/` and copies that path under `$FAKE_RELEASES` to the output; a missing file prints an error to stderr and exits 22. The fake `uname` prints `$FAKE_OS` for `-s` and `$FAKE_ARCH` for `-m`, and `-m`/`-s` only. A published fake executable is a shell script printing `dkt v9.9.9`; a helper publishes it with a correct `checksums.txt` in the `<hash>  <name>` format under a given directory, such as `latest/download`.

  - `TestInstallLatest`, one subtest per row of the table `Darwin/arm64`, `Darwin/x86_64`, `Linux/aarch64`, `Linux/arm64`, `Linux/x86_64`, `Linux/amd64`, to `dkt-darwin-arm64`, `dkt-darwin-amd64`, `dkt-linux-arm64`, `dkt-linux-arm64`, `dkt-linux-amd64`, `dkt-linux-amd64`: exit 0; `requests` holds exactly the executable's and `checksums.txt`'s `latest/download` URLs; `~/.local/bin/dkt` has the published bytes and mode with the owner's execute bit; running it prints `dkt v9.9.9`; stdout contains the path and `dkt v9.9.9`; `~/.local/bin` holds only `dkt`.
  - `TestInstallReplacesEarlier`: an earlier `~/.local/bin/dkt` holds other bytes; after a run it holds the published bytes and `~/.local/bin` holds only `dkt`.
  - `TestInstallReplacesLink`: `~/.local/bin/dkt` is a symbolic link to a file outside; after a run the path is a regular file with the published bytes and the link's target is unchanged.
  - `TestInstallVersion`: `DKT_VERSION=v0.16.0` and `DKT_VERSION=0.16.0` both request only under `download/v0.16.0/`; `DKT_VERSION=` requests only under `latest/download/`.
  - `TestInstallRefuses`, one subtest per case, each starting with an earlier `~/.local/bin/dkt`; each exits non-zero, writes a message to stderr holding the given text, leaves the earlier file byte for byte and leaves `~/.local/bin` holding only `dkt`:
    - wrong digest in `checksums.txt`: `SHA-256`
    - `checksums.txt` without the executable's line: `checksums.txt`
    - executable not published: its URL
    - `checksums.txt` not published: its URL
    - `DKT_VERSION=v0.1.0`, a release that does not exist: `download/v0.1.0/`
    - `FAKE_OS=FreeBSD`: `FreeBSD` and the releases URL, with `requests` absent
    - `FAKE_OS=MINGW64_NT-10.0`: `MINGW64_NT-10.0` and the releases URL, with `requests` absent
    - `FAKE_ARCH=riscv64`: `riscv64` and the releases URL, with `requests` absent
    - a published executable that exits 3: `does not run`
  - `TestInstallTargetIsDirectory`: `~/.local/bin/dkt/keep` exists; exit non-zero, stderr holds `is a directory`, and `keep` is still there.
  - `TestInstallDigestTools`: `PATH` is only a directory of symbolic links to the tools the script and the fake `curl` need, plus the fakes. With neither digest tool: exit non-zero, stderr holds `sha256sum`, `requests` absent. With `sha256sum` alone and with `shasum` alone: exit 0 and installed; skip a subtest whose tool is not on the machine.
  - `TestInstallInterrupted`, one subtest each for `SIGHUP`, `SIGINT` and `SIGTERM`, skipped when `signal.Ignored` says the test process ignores that signal, since `sh` would inherit that and could not trap it: the fake `curl`, given `FAKE_CURL_SLEEP=1`, sleeps after recording its URL. Start the script, wait for `requests` to appear, send the signal to `sh`, wait for it to exit (10-second limit) and require a non-zero exit. A shell runs the trap when its child returns, so the fake `curl` is done by then. `~/.local/bin` holds only the earlier `dkt`, unchanged.
  - `TestInstallCutShort`: the file's last line is `main`. Cut `install.sh` at every line boundary and at every byte of the last line short of the complete `main`, and feed each cut to `sh`; afterwards `requests` is absent and the home directory is empty. Every byte of the file would take about 40 seconds, and a cut inside a line inside a function is a syntax error whichever line it is: a command that acts at the top level already fails at the line boundary after it.
  - `TestInstallPathHint`: with `PATH` not holding `~/.local/bin`, stdout holds `is not on your PATH`; with it, stdout does not hold `PATH`.

- [x] **Step 2: Run** `go test -run TestInstall .`; expect FAIL: `install.sh` does not exist.

- [x] **Step 3: Write `install.sh`**, mode 0755, starting `#!/bin/sh` and `set -eu`, with a comment saying what it does and why everything is in `main`. A `die` function prints `install.sh: <message>` to stderr and exits 1. In `main`, in this order:
  1. Map `uname -s` and `uname -m` as the scenarios say; anything else dies naming the value and `https://github.com/wake/docket/releases`.
  2. Pick the digest command, `sha256sum` then `shasum -a 256`; neither dies naming both.
  3. Pick the base URL from `${DKT_VERSION:-}`: empty is latest, `v*` as is, otherwise with `v` added.
  4. Die when `$HOME/.local/bin/dkt` is a directory. `mkdir -p "$HOME/.local/bin"`.
  5. `tmp=$(mktemp -d "$dir/.dkt-install.XXXXXX")`, with `trap 'rm -rf "$tmp"' EXIT` and `trap 'exit 1' HUP INT TERM`.
  6. Download the executable and `checksums.txt` with `curl -fsSL "$url" -o "$out"`; a failure dies naming the URL.
  7. Read the expected digest with `awk -v f="$file" '$2 == f { print $1 }'`; empty dies naming `checksums.txt`. Compare with the first field of the digest command's output; a difference dies with `SHA-256 of <file> does not match checksums.txt`.
  8. `chmod 755`, then run the temporary file with `--version`, capturing its output; a failure dies with `<file> does not run on this machine`.
  9. `mv -f` it over the target and print `Installed <version output> at <path>`.
  10. Print `<dir> is not on your PATH; add it to run dkt.` when `:$PATH:` does not hold the directory.

  The last line of the file is `main`, with no arguments: `main` takes none.

- [x] **Step 4: Run** `go test -run TestInstall .`; expect PASS. If `shellcheck` is available, run `shellcheck install.sh`; the unquoted digest command is intended and takes a `# shellcheck disable=SC2086` comment.

- [x] **Step 5: Mutate** and see a test fail for each, then restore: remove the `main` wrapper; skip the digest comparison; drop the `v` normalization; run the version check after `mv`; remove the signal trap.

- [x] **Step 6: Run** `go vet ./...` and `go test ./...`; expect PASS.

- [x] **Step 7: Commit** `install.sh` and `release_test.go` with the trailer `Change: add-release-process/1`.

### Task 2: Snapshot build in CI

**Scenarios:** release#snapshot-built-and-installed, release#asset-names-drift, release#executable-reports-the-tag

**Review:** judgment

**Files:**
- Create: `.goreleaser.yaml`
- Modify: `.github/workflows/ci.yml`
- Modify: `.gitignore`
- Modify: `release_test.go`

**Interfaces:**
- Consumes: `newInstallEnv` and `(installEnv).run` from Task 1, with no fake `uname` on `PATH` for this test.

- [x] **Step 1: Write the failing test** `TestInstallSnapshot` in `release_test.go`: skip unless `DOCKET_TEST_DIST` is set. Read `<dist>/artifacts.json` (fields `name`, `path`; `path` is relative to the repository root) and `<dist>/metadata.json` (field `tag`). The artifacts named `dkt-*` are exactly the four target names. Copy those and `checksums.txt` into `<releases>/latest/download/`, run `install.sh` with the real `uname`, expect exit 0, and require the installed `dkt --version` to print `dkt <tag>`.

- [x] **Step 2: Run** `go test -run TestInstallSnapshot .`; expect SKIP. Run it with `DOCKET_TEST_DIST=dist`; expect FAIL: no `artifacts.json`.

- [x] **Step 3: Write `.goreleaser.yaml`**: `version: 2`; one build with `main: ./cmd/dkt`, `binary: dkt`, `env: [CGO_ENABLED=0]`, `goos: [darwin, linux]`, `goarch: [amd64, arm64]`, `flags: [-trimpath]`, `ldflags: [-s -w -X github.com/wake/docket/internal/cli.Version={{ .Tag }}]`; `archives: [{ formats: [binary], name_template: dkt-{{ .Os }}-{{ .Arch }} }]`, with a comment saying why the names hold no version; `checksum: { name_template: checksums.txt }`. Add `/dist/` to `.gitignore` under a comment naming GoReleaser.

- [x] **Step 4: Build the snapshot** with GoReleaser v2 (the task's brief gives the path of a verified binary; do not install it system-wide): `goreleaser check`, then `goreleaser release --snapshot --clean`, with no `*_TOKEN` variable in its environment. Expect success and the four names in `dist/checksums.txt`.

- [x] **Step 5: Run** `DOCKET_TEST_DIST=dist go test -run TestInstallSnapshot -v .`; expect PASS, not SKIP. Change `name_template` to `dkt_{{ .Os }}_{{ .Arch }}`, rebuild and see it fail, then change the stamp to `{{ .Version }}`, rebuild and see it fail; restore both.

- [x] **Step 6: Add the job** `release-snapshot` to `.github/workflows/ci.yml`, on `ubuntu-latest`: `actions/checkout@v7` with `fetch-depth: 0` and a comment that GoReleaser reads the history and tags; `actions/setup-go@v7` with `go-version: stable`; `goreleaser/goreleaser-action@v7` with `version: '~> v2'` and `args: check`; the same action with `args: release --snapshot --clean`; a step `TestInstallSnapshot` running `go test -run TestInstallSnapshot -v .` with `DOCKET_TEST_DIST: dist`. It passes no token, and the workflow's `permissions: contents: read` stays. Lint the workflow with `actionlint` (the task's brief gives the path).

- [x] **Step 7: Run** `go vet ./...` and `go test ./...`; expect PASS. `git status --short` must not list `dist/`.

- [x] **Step 8: Commit** `.goreleaser.yaml`, `.github/workflows/ci.yml`, `.gitignore` and `release_test.go` with the trailer `Change: add-release-process/2`.

### Task 3: Release workflow, plugin version and marketplace

**Scenarios:** release#version-tag-pushed, release#another-tag-pushed, release#marketplace-names-the-same-version, release#one-file-changed-without-the-other, release#tag-differs-from-the-plugin

**Review:** judgment

**Files:**
- Create: `.github/workflows/release.yml`
- Create: `.claude-plugin/marketplace.json`
- Create: `docs/release.md`
- Modify: `.claude-plugin/plugin.json`
- Modify: `release_test.go`

- [x] **Step 1: Write the failing test** `TestPluginVersion` in `release_test.go`. Read both JSON files. `plugin.json`'s `version` matches `^\d+\.\d+\.\d+$`. `marketplace.json` has exactly one plugin, named `docket`, whose `source` has `source: github`, `repo: wake/docket` and `ref` equal to `v` plus the version. When `DOCKET_RELEASE_TAG` is not empty, it equals `v` plus the version; the failure message names the tag and the file. A comment says the release workflow passes the tag.

- [x] **Step 2: Run** `go test -run TestPluginVersion .`; expect FAIL: no version, no marketplace.

- [x] **Step 3: Add** `"version": "0.16.0"` to `.claude-plugin/plugin.json` after `name`. Create `.claude-plugin/marketplace.json`: `name: docket`, `owner: { name: wake }`, and `plugins` with one entry: `name: docket`, the same `description` as `plugin.json`, and `source: { source: github, repo: wake/docket, ref: v0.16.0 }`. Do not repeat `version` in the entry: the docs warn when both are set.

- [x] **Step 4: Run** `go test -run TestPluginVersion .`; expect PASS. Run it with `DOCKET_RELEASE_TAG=v0.16.0`; expect PASS. With `DOCKET_RELEASE_TAG=v0.17.0` and with `DOCKET_RELEASE_TAG=v0.16.0-rc.1`; expect FAIL for each. Set the marketplace's `ref` to `v0.17.0`; expect FAIL; restore. Run `claude plugin validate .` at the repository root; expect exit 0. `--strict` also fails on optional metadata this change does not add (a marketplace description, a plugin author) and on the repository's own `CLAUDE.md`, so it is not the check here.

- [x] **Step 5: Write `.github/workflows/release.yml`**: `name: release`; `on: push: tags: ['v[0-9]+.[0-9]+.[0-9]+']`; `permissions: contents: write`; one job on `ubuntu-latest`: `actions/checkout@v7` with `fetch-depth: 0`; `actions/setup-go@v7` with `go-version: stable`; a step named for the check, running `go test -run TestPluginVersion .` with `DOCKET_RELEASE_TAG: ${{ github.ref_name }}`; `goreleaser/goreleaser-action@v7` with `version: '~> v2'`, `args: release --clean` and `GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}`. Lint with `actionlint` (the task's brief gives the path). This workflow cannot run in a pull request; its first real run is the first release, and the tag filter is backed by the check in the step before GoReleaser.

- [x] **Step 6: Write `docs/release.md`**, in English, for maintainers: what a release is; the steps (in a pull request, set `version` in `.claude-plugin/plugin.json` to `X.Y.Z` and `ref` in `.claude-plugin/marketplace.json` to `vX.Y.Z`; merge; tag that merge commit on `main` and push the tag); that both files already hold the next version, so a first release needs only the tag; what the workflow publishes; that a mismatch fails the workflow before anything is published; how to install a chosen version with `DKT_VERSION`. Keep it under 40 lines, with no host-specific details.

- [x] **Step 7: Run** `go vet ./...` and `go test ./...`; expect PASS.

- [x] **Step 8: Commit** the five files with the trailer `Change: add-release-process/3`.
