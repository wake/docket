# Design

## Context

See proposal.md for the motivation. `dkt --version` already prints a version stamped at build time through `internal/cli.Version`, and the docket-init skill already names the `install.sh` URL. No release file exists yet, on `main` or at v0.15.0; that release has no assets.

## Goals / Non-Goals

**Non-Goals:** see proposal.md. In addition, this design does not protect against a compromised GitHub release: `checksums.txt` comes from the same place as the executable, so it catches a corrupted or truncated download, not a malicious one.

## Decisions

**Bare executables, named without a version.** GoReleaser's `binary` format uploads them as they are, and `name_template: dkt-{{ .Os }}-{{ .Arch }}` gives a stable name, so `releases/latest/download/<name>` works without asking the GitHub API for the version. An archive would need `tar` on every machine and an extra step for one file.

**The tag is the version.** The ldflags stamp `{{ .Tag }}` rather than `{{ .Version }}`, which drops the `v`; `cli-output` says the version is `v0.16.0`. In a snapshot `.Tag` is the most recent tag, which is good enough for checking that the stamp reaches the binary.

**macOS and Linux only for now.** The plan was to ship Windows too, but dkt does not build there: `syscall.O_NOFOLLOW` and `syscall.Flock` have no Windows counterpart, so the directory lock and the no-follow reads the specs require need a port, and the tests depend on `/bin/sh` fakes, `Umask` and `Mkfifo`, so CI could not verify the result. Adding Windows later takes that port and one word in `goos`. `install.sh` could not have installed a Windows executable anyway.

**One version check, in a test.** `TestPluginVersion` requires `plugin.json`'s version to match `marketplace.json`'s ref, and to match `DOCKET_RELEASE_TAG` when set. It runs with every `go test ./...`, which catches a half-done bump in its pull request, and the release workflow runs it with the tag before GoReleaser. Alternative: `jq` in the workflow, which no pull request would ever exercise.

**The source is `github` at a tag, not a relative path.** A relative path has no ref, so the plugin would follow whatever the marketplace's commit holds. The Claude Code docs do not mention a `github` source that points at the marketplace's own repository, so this is checked by hand in this change as far as it can be without a tag, and fully at the first release.

**`plugin.json` and `marketplace.json` hold the next version on `main` from now on.** With the files already at `0.16.0`, the first release is only a tag push. Until then the marketplace's ref does not exist, so `plugin install` fails; nobody can install Docket before the first release anyway.

**`install.sh` keeps its temporary directory inside `~/.local/bin`.** `mv` within one filesystem replaces the target in one step, and the new executable can be run before it replaces a working one, without depending on `/tmp`, which may be mounted `noexec`. The cost is an empty `~/.local/bin` after a failed first install, and traps to clean up on a signal.

**Everything in `install.sh` is inside `main`, called on the last line.** A shell reading from `curl` runs commands as they arrive; with the wrapper, a cut-short download holds no complete command that acts.

**No test hook in `install.sh`.** The tests put fake `curl` and `uname` on `PATH` and feed the script to `sh` on standard input, as `curl | sh` does. The script has no base URL setting: a variable that redirects where an executable comes from is one more thing to attack, and nobody asked for it.

**The snapshot test connects both sides.** `TestInstallSnapshot` runs only when `DOCKET_TEST_DIST` is set, as CI does after the snapshot build. It reads `artifacts.json`, serves those files through the fake `curl`, runs the real `install.sh` with the real `uname`, and requires the installed dkt to print the tag from `metadata.json`. So a change to the names, the ldflags or the checksum file fails a pull request, not a release.

## Risks / Trade-offs

- [The `github` source pointing at its own repository is undocumented] → Check it by hand before the first release; a relative path is the fallback, at the cost of the pin.
- [`.claude-plugin/marketplace.json` names a tag that does not exist until the first release] → `install` fails with a clear error until then, and the first release is planned.
- [A tag pushed on a commit that is not on `main`, or that failed CI, still publishes] → Only maintainers can push tags, and `docs/release.md` says to tag the merge commit.
- [Between a release's upload and the end of a mirror's propagation, `latest/download` could serve an executable and a `checksums.txt` of two releases] → The digests then differ and the script fails, installing nothing; running it again works.
