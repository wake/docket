---
title: "Keep a token in the origin URL out of every command's output"
kind: change-request
source: "docket/ideas/origin-without-token.md"
---

## Why

Every command's JSON envelope prints `repo.origin` as git has it, so a remote such as `https://<token>@github.com/org/app.git` puts the token in every `--json` result, which scripts, CI and other tools pass on and log. The project index already removes the user information from `http` and `https` origins before storing them; `repo.origin` should follow the same rule.

## What Changes

- `repo.origin` in the envelope of every command drops the user information of an `http` or `https` URL, as the project index does. Any other URL is printed as it is.
- The removal no longer depends on the URL parsing cleanly. Today an `http` or `https` URL that does not parse, such as one with an invalid percent escape anywhere in it or a space in its password, keeps its user information in the project index; it loses it, in `repo.origin` and in the index alike, and the rest of the URL is kept as it is. An entry the index already holds with such a URL loses it when the index is read, so `dkt projects ls` never prints it, and the next write of the index stores it without.
- **BREAKING** for a script that read a token back out of `repo.origin`: it no longer can.

## Non-Goals

- No removal from other forms (`ssh://`, `git@host:path`, `ftp://`): their user part is a login name, not a secret, and a password in an `ssh://` URL is not used by git.
- No change to the remote itself: dkt only reads it.
- No change to the workbench, which reads the origin from git on its own.
- No warning when the origin holds a token.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `cli-output`: Repository identity prints the origin URL without the user information of an `http` or `https` URL.

## Impact

- `internal/cli/run.go` (`identify`) takes the origin through the same function the project index uses.
- `internal/projects/locate.go` (`Origin`) removes the user information by its place in the URL instead of by parsing it.
- Tests in `internal/cli/repo_test.go` and `internal/projects/locate_test.go`.
