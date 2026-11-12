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
- A URL that `net/url` accepts but `u.String()` would re-encode (`https://tok@example.com/a b.git`): the rest stays byte for byte, so an implementation that keeps `url.Parse` for well-formed input fails. Tested in Task 1.
- An error while reading the origin: still reported as `no-repository`, as `identifyError` does today. Tested in Task 2.
- An `@` after the authority (`https://example.com/path@x`): the URL is unchanged. Tested in Task 1.
- A password with a raw `@` (`https://user:p@ss@example.com/app.git`): all of the user information goes. Tested in Task 1.
- An upper-case scheme (`HTTPS://tok@example.com/app.git`): the user information goes and the scheme keeps its case. Tested in Task 1.
- `ssh://git@...` and `git@host:path`: unchanged in `repo.origin` too. Tested in Task 2.

## Reuse

| Need | Existing (path:symbol) | Decision | Reason | Evidence |
|---|---|---|---|---|
| Read the origin URL without its user information | internal/projects/locate.go:Origin | extend | The index already applies the rule through it (`register.go:50`, `locate.go:42`); it only needs to stop depending on `url.Parse` | `rg -n 'projects\.Origin\|Origin\(' internal --glob '*.go'` |
| Read the origin for the envelope | internal/cli/run.go:identify | extend | It is the only place `repo.origin` is set | `rg -n "Origin = " internal/cli/` |
| Map a git error while identifying | internal/cli/run.go:identifyError | reuse | Same error rule as today | `rg -n "func identifyError" internal/cli/` |
| A repository with a remote in tests | internal/testutil/repo.go:NewRepo | reuse | Both test files already use it (`docketRepo` wraps it in `locate_test.go`) | `rg -n "func NewRepo" internal/testutil/` |
| Clean the entries the index already holds | internal/projects/files.go:Load | extend | Every reader of the index and every later `Save` goes through it | `rg -n "projects.Load\|Load\(" internal --glob '*.go'` |

## Revision 1 (2026-11-13): clean the entries the index already holds

The codex pre-merge review (job review-mwdxi0s6-jomq2h, high) found that an entry stored before this change from an `http` or `https` URL that did not parse keeps its token, and `dkt projects ls --json` prints it. That breaks project-index's "Origin recorded" ("never stored or printed"). Task 3 is added; the proposal and design say so.

## Revision 2 (2026-11-13): the rule is its own requirement

At archive time, `openspec validate --specs --strict` found Repository identity over 500 characters with the rule added to it. The rule moved, word for word, into a new requirement, Origin without user information, with the same three scenarios; Repository identity is left as it was. No behavior or test changes; Task 2 claims the three scenarios.

---

### Task 1: Remove the user information by its place in the URL

**Scenarios:** none

**Review:** judgment

**Files:**
- Modify: `internal/projects/locate.go`
- Modify: `internal/projects/locate_test.go`

This fixes what project-index's "Origin recorded" already requires, for URLs `url.Parse` rejects; Task 2 relies on it for the cli-output scenarios.

- [x] **Step 1: Write the failing test**

Add these rows to the table in `TestOrigin` (`internal/projects/locate_test.go`), keeping the existing rows:

| in | want |
|---|---|
| `https://user:p%zz@example.com/org/app.git` | `https://example.com/org/app.git` |
| `https://tok@example.com/a%zz.git` | `https://example.com/a%zz.git` |
| `https://user:p ss@example.com/app.git` | `https://example.com/app.git` |
| `https://tok@example.com/a b.git` | `https://example.com/a b.git` |
| `https://user:p@ss@example.com/app.git` | `https://example.com/app.git` |
| `HTTPS://tok@example.com/app.git` | `HTTPS://example.com/app.git` |
| `https://tok@example.com` | `https://example.com` |
| `https://example.com/path@x` | `https://example.com/path@x` |
| `https://example.com?q=a@b` | `https://example.com?q=a@b` |
| `ftp://u:p@example.com/app.git` | `ftp://u:p@example.com/app.git` |

- [x] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/projects/ -run TestOrigin -v`
Expected: FAIL on the rows `url.Parse` rejects (the first three), on the `a b.git` row (re-encoded to `a%20b.git`) and on the upper-case scheme row.

- [x] **Step 3: Implement `withoutUserInfo(raw string) string` in `internal/projects/locate.go`**

`Origin` returns `withoutUserInfo(raw)`; `withoutUserInfo` follows the rule in Global Constraints with `strings` only. Update `Origin`'s doc comment to say the user information is removed by its place, so a URL that does not parse loses it too.

- [x] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/projects/ -v -run TestOrigin`
Expected: PASS

- [x] **Step 5: Commit**

```bash
git add internal/projects/locate.go internal/projects/locate_test.go
git commit -m "Remove the user information from an origin URL by its place, not by parsing it"
```

with the trailer `Change: origin-without-token/1`.

### Task 2: Print `repo.origin` without the user information

**Scenarios:** cli-output#token-in-the-origin-url, cli-output#token-in-a-url-that-is-not-well-formed, cli-output#ssh-remote-kept

**Review:** checks

**Files:**
- Modify: `internal/cli/run.go`
- Modify: `internal/cli/repo_test.go`

**Interfaces:**
- Consumes: `projects.Origin(g gitx.Git) (*string, error)` from Task 1.

Repository identity is unchanged; its scenarios stay covered by `TestRepoDetachedHead`, `TestRepoNoOrigin`, `TestRepoNoCommits` and `TestRepoOutsideRepository`.

- [x] **Step 1: Write the failing test**

Add `TestRepoOriginWithoutUserInfo` to `internal/cli/repo_test.go`: for each row, a `testutil.NewRepo` with one commit and `git remote add origin <in>`, then `identified(t, r).Origin` must equal want.

| in | want |
|---|---|
| `https://x-access-token:secret@example.com/org/app.git` | `https://example.com/org/app.git` |
| `https://user:p%zz@example.com/org/app.git` | `https://example.com/org/app.git` |
| `ssh://git@example.com/org/app.git` | `ssh://git@example.com/org/app.git` |
| `git@example.com:org/app.git` | `git@example.com:org/app.git` |

Add `TestRepoOriginReadFails` to the same file: in a `testutil.NewRepo` with one commit, put a `git` script first on PATH (as `internal/testutil/repo.go` does) that exits 3 when its arguments include `remote.origin.url` and otherwise runs the real git (`exec.LookPath("git")` before PATH changes). `dkt domains --json` in that repository must exit 4 with one finding, rule `no-repository`, whose message starts with `cannot identify the repository`. This test passes before the change too; it guards the error mapping Step 3 rewrites.

- [x] **Step 2: Run the tests to verify the new one fails**

Run: `go test ./internal/cli/ -run 'TestRepoOrigin' -v`
Expected: `TestRepoOriginWithoutUserInfo` FAILS on the two `https` rows; `TestRepoOriginReadFails` passes.

- [x] **Step 3: Take the origin through `projects.Origin` in `identify` (`internal/cli/run.go`)**

Replace the `c.Git.RemoteURL("origin")` block with `projects.Origin(c.Git)`; an error still returns `identifyError(err)`, and a nil result leaves `r.Origin` nil.

- [x] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/cli/ -run TestRepo -v`
Expected: PASS, including the existing `TestRepo*` tests.

- [x] **Step 5: Run the full checks**

Run: `gofmt -l .`, `go vet ./...`, `go test ./...`
Expected: no output from gofmt, vet clean, all tests pass.

- [x] **Step 6: Commit**

```bash
git add internal/cli/run.go internal/cli/repo_test.go
git commit -m "Print repo.origin without the user information of an http or https URL"
```

with the trailer `Change: origin-without-token/2`.

### Task 3: Clean the entries the index already holds

**Scenarios:** none

**Review:** checks

**Files:**
- Modify: `internal/projects/files.go`
- Modify: `internal/projects/files_test.go`

**Interfaces:**
- Consumes: `withoutUserInfo(raw string) string` from Task 1 (same package).

This fixes what project-index's "Origin recorded" already requires for entries stored before Task 1.

- [x] **Step 1: Write the failing test**

Add `TestLoadRemovesUserInfo` to `internal/projects/files_test.go`: with the `write` helper, an index holding two entries, `/w/a` with `origin` `https://user:p%zz@example.com/org/app.git` and `/w/b` with `origin` null (both `added` `2026-11-08T09:10:11Z`, `ignored` false). After `Load`, `/w/a`'s `Origin` is `https://example.com/org/app.git` and `/w/b`'s is nil. Then `Save` to a new path and read the file back: it does not contain `p%zz`.

- [x] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/projects/ -run TestLoadRemovesUserInfo -v`
Expected: FAIL, the origin still holds `user:p%zz@`.

- [x] **Step 3: In `Load` (`internal/projects/files.go`), replace each non-nil `Origin` with `withoutUserInfo` of it, after the checks**

- [x] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/projects/ -v`
Expected: PASS

- [x] **Step 5: Run the full checks**

Run: `gofmt -l .`, `go vet ./...`, `go test ./...`
Expected: no output from gofmt, vet clean, all tests pass.

- [x] **Step 6: Commit**

```bash
git add internal/projects/files.go internal/projects/files_test.go
git commit -m "Remove the user information from the origins the index already holds"
```

with the trailer `Change: origin-without-token/3`.
