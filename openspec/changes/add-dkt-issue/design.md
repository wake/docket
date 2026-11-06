# Design

## Context

Records already have everything an issue needs: a dated slug that is never reused across worktrees and branches (`scan.Records`), a write that refuses links, takes the first free suffix under `O_EXCL` and cleans up after a failure (`runRecord`), and lint rules for front matter, sections, placeholders and length (`internal/lint/record.go`). Each of these names `record.Dir`, the record sections or "dkt record" directly. See proposal.md for why issues are needed.

## Goals / Non-Goals

**Goals:**
- Issues behave like records wherever the specs say "as records": names, slugs, links, failed writes, the lint scope and the length rule, by sharing the code rather than copying it.

**Non-Goals:**
- Issue state and listing (`dkt status`).

## Decisions

**Parameterize the record code first, in its own task.** The record functions take the folder, the sections and the command name as parameters (a scan for files under a folder, a dated write under a folder, a descriptor for the lint rules), and records call them with their own values. This is a structural change: the record tests pass unchanged before and after, and it is committed apart from the issue behavior. Copying `runRecord` and `lint/record.go` for issues was the alternative; it would double the link and cleanup logic, which is where the subtle cases are.

**A small `issue` package beside `record`.** It holds the folder, the four sections, the path of a slug and the limit (the record limit). Issues have no kind and no tasks, so they do not go into the record package.

**Two shipped templates.** `shipped/issue/template.md` is the file template `dkt issue` writes. `shipped/issue/external.md` is what `dkt issue template` prints for a project that keeps issues in another tool: its opening comment does not mention `dkt lint`, and its Evidence comment asks where the issue was found, since such a tool has no `source` field. Printing the file template with sentences cut out by code was rejected: the text would differ from what the maintainer approved without anyone editing it. A test checks that both templates have the four sections in order, so they cannot drift apart in structure.

**`template` is a word, not a name.** `dkt issue template` prints; with any other argument or flag it is a usage error. No issue can be named `template`, which costs nothing since names are free-form otherwise.

**Dated names in `dkt lint`.** A record and an issue share the slug form, so a dated name is looked up in both folders and every match is linted. `record-missing` keeps its name when neither exists, so nothing that reads findings today breaks; its message says neither a record nor an issue was found.

**Rule names.** The issue rules mirror the record ones with an `issue-` prefix (`issue-front-matter`, `issue-section`, `issue-too-long`); `title-too-long`, `placeholder` and `slug-invalid` are shared, as they are between changes and records.

## Risks / Trade-offs

- [The structural task touches every record path at once] → It changes no behavior and the existing record tests, which cover links, suffixes, concurrency and cleanup, must pass untouched; mutating the changed lines must make one fail.
- [The external template waits for the maintainer's edit] → It is the last behavior task; everything else can merge without waiting on it only if the plan is split, so it stays in this change and the change waits.
