# Releasing Docket

A release is a `vX.Y.Z` tag on `main`. Pushing the tag runs the `release` workflow, which builds `dkt` and publishes a GitHub release.

## Current state

No release has been made with this process yet. The latest release, v0.15.0, has no executables, so `install.sh` fails until the first release made this way.

## Steps

1. In a pull request, set `version` in `.claude-plugin/plugin.json` to `X.Y.Z` and `ref` in `.claude-plugin/marketplace.json` to `vX.Y.Z`. Merge it.
2. Tag that merge commit on `main` and push the tag:

   ```sh
   git tag vX.Y.Z <merge-commit>
   git push origin vX.Y.Z
   ```

Both files already hold the next version, 0.16.0, so the first release needs only the tag.

Release versions in ascending order: GitHub marks each new release as the latest, and that is what `install.sh` installs by default.

## What the workflow does

1. Checks that the tagged commit is on `main`. If not, the workflow fails and nothing is published.
2. Runs `TestPluginVersion` with the pushed tag. If the tag differs from either file, or is not plain `vX.Y.Z` (for example `v0.16.0-rc.1`), the workflow fails here and nothing is published.
3. Runs GoReleaser (`.goreleaser.yaml`), which publishes the release with the executables for each platform and a checksums file.

Pull requests and pushes to `main` run a snapshot build with the same configuration. The release workflow itself runs only on a tag.

## After the first release

Only a real release exercises the tag filter, the publish step and the install paths. After the first one, look at:

- The workflow ran for the tag, and both checks passed.
- The release holds the four executables and `checksums.txt`.
- `install.sh` installs it, and `dkt --version` prints the tag.
- `claude plugin marketplace add wake/docket` and `claude plugin install docket@docket` install the plugin.

## Installing a version

`install.sh` installs the latest release. To choose one:

```sh
DKT_VERSION=vX.Y.Z sh install.sh
```
