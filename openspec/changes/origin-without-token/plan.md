# Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `repo.origin` in every command's envelope, and the project index, never carry the user information of an `http` or `https` origin URL.

**Architecture:** `projects.Origin` removes the user information by its place in the URL instead of parsing it; `identify` in the cli package takes the origin through `projects.Origin`.

**Tech Stack:** Go 1.25, standard library only.

**Spec:** `openspec/changes/origin-without-token/proposal.md`, `specs/cli-output/spec.md`, `design.md` in the same folder.

## Global Constraints

- The scheme check is `http` or `https`, compared without regard to case; every other URL is returned byte for byte.
- The authority is the text after `://` up to the first `/`, `?` or `#`, or the end; when it holds an `@`, everything up to and including its last `@` is removed. Every other byte is kept.
- `identify` keeps mapping a git error to `identifyError`.
- No new dependency; `net/url` is no longer imported by `internal/projects/locate.go` if nothing else there uses it.
- Check with `gofmt -l .` (prints nothing), `go vet ./...` and `go test ./...`.

## Review Focus

- A URL that `net/url` rejects (`https://user:p%zz@example.com/org/app.git`, `https://tok@example.com/a%zz.git`, a space in the password): the user information goes and the rest stays byte for byte. Tested in Task 1.
- An `@` after the authority (`https://example.com/path@x`): the URL is unchanged. Tested in Task 1.
- A password with a raw `@` (`https://user:p@ss@example.com/app.git`): all of the user information goes. Tested in Task 1.
- An upper-case scheme (`HTTPS://tok@example.com/app.git`): the user information goes and the scheme keeps its case. Tested in Task 1.
- `ssh://git@...` and `git@host:path`: unchanged in `repo.origin` too. Tested in Task 2.

## Reuse

| Need | Existing (path:symbol) | Decision | Reason | Evidence |
|---|---|---|---|---|
| Read the origin URL without its user information | internal/projects/locate.go:Origin | extend | The index already applies the rule; it only needs to stop depending on `url.Parse` | `rg -n "func Origin" internal/` |
| Read the origin for the envelope | internal/cli/run.go:identify | extend | It is the only place `repo.origin` is set | `rg -n "Origin = " internal/cli/` |
| Map a git error while identifying | internal/cli/run.go:identifyError | reuse | Same error rule as today | `rg -n "func identifyError" internal/cli/` |
| A repository with a remote in tests | internal/testutil/repo.go:NewRepo | reuse | Both test files already use it (`docketRepo` wraps it in `locate_test.go`) | `rg -n "func NewRepo" internal/testutil/` |

---

### Task 1: Remove the user information by its place in the URL

**Scenarios:** none

**Review:** judgment

**Files:**
- Modify: `internal/projects/locate.go`
- Modify: `internal/projects/locate_test.go`

This fixes what project-index's "Origin recorded" already requires, for URLs `url.Parse` rejects; Task 2 relies on it for the cli-output scenarios.

- [ ] **Step 1: Write the failing test**

Add these rows to the table in `TestOrigin` (`internal/projects/locate_test.go`), keeping the existing rows:

| in | want |
|---|---|
| `https://user:p%zz@example.com/org/app.git` | `https://example.com/org/app.git` |
| `https://tok@example.com/a%zz.git` | `https://example.com/a%zz.git` |
| `https://user:p ss@example.com/app.git` | `https://example.com/app.git` |
| `https://user:p@ss@example.com/app.git` | `https://example.com/app.git` |
| `HTTPS://tok@example.com/app.git` | `HTTPS://example.com/app.git` |
| `https://tok@example.com` | `https://example.com` |
| `https://example.com/path@x` | `https://example.com/path@x` |
| `https://example.com?q=a@b` | `https://example.com?q=a@b` |
| `ftp://u:p@example.com/app.git` | `ftp://u:p@example.com/app.git` |

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/projects/ -run TestOrigin -v`
Expected: FAIL on the rows `url.Parse` rejects (the first three) and on the upper-case scheme row.

- [ ] **Step 3: Implement `withoutUserInfo(raw string) string` in `internal/projects/locate.go`**

`Origin` returns `withoutUserInfo(raw)`; `withoutUserInfo` follows the rule in Global Constraints with `strings` only. Update `Origin`'s doc comment to say the user information is removed by its place, so a URL that does not parse loses it too.

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/projects/ -v -run TestOrigin`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/projects/locate.go internal/projects/locate_test.go
git commit -m "Remove the user information from an origin URL by its place, not by parsing it"
```

with the trailer `Change: origin-without-token/1`.

### Task 2: Print `repo.origin` without the user information

**Scenarios:** cli-output#detached-head, cli-output#no-origin-remote, cli-output#token-in-the-origin-url, cli-output#token-in-a-url-that-is-not-well-formed, cli-output#ssh-remote-kept, cli-output#no-commits-yet, cli-output#outside-a-repository

**Review:** checks

**Files:**
- Modify: `internal/cli/run.go`
- Modify: `internal/cli/repo_test.go`

**Interfaces:**
- Consumes: `projects.Origin(g gitx.Git) (*string, error)` from Task 1.

The scenarios Detached HEAD, No origin remote, No commits yet and Outside a repository are unchanged and stay covered by `TestRepoDetachedHead`, `TestRepoNoOrigin`, `TestRepoNoCommits` and `TestRepoOutsideRepository`.

- [ ] **Step 1: Write the failing test**

Add `TestRepoOriginWithoutUserInfo` to `internal/cli/repo_test.go`: for each row, a `testutil.NewRepo` with one commit and `git remote add origin <in>`, then `identified(t, r).Origin` must equal want.

| in | want |
|---|---|
| `https://x-access-token:secret@example.com/org/app.git` | `https://example.com/org/app.git` |
| `https://user:p%zz@example.com/org/app.git` | `https://example.com/org/app.git` |
| `ssh://git@example.com/org/app.git` | `ssh://git@example.com/org/app.git` |
| `git@example.com:org/app.git` | `git@example.com:org/app.git` |

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/cli/ -run TestRepoOriginWithoutUserInfo -v`
Expected: FAIL on the two `https` rows.

- [ ] **Step 3: Take the origin through `projects.Origin` in `identify` (`internal/cli/run.go`)**

Replace the `c.Git.RemoteURL("origin")` block with `projects.Origin(c.Git)`; an error still returns `identifyError(err)`, and a nil result leaves `r.Origin` nil.

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/cli/ -run TestRepo -v`
Expected: PASS, including the existing `TestRepo*` tests.

- [ ] **Step 5: Run the full checks**

Run: `gofmt -l .`, `go vet ./...`, `go test ./...`
Expected: no output from gofmt, vet clean, all tests pass.

- [ ] **Step 6: Commit**

```bash
git add internal/cli/run.go internal/cli/repo_test.go
git commit -m "Print repo.origin without the user information of an http or https URL"
```

with the trailer `Change: origin-without-token/2`.
