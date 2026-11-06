# Design

## Context

See proposal.md for why. What exists today:

- `dkt new` checks its flags, scans every worktree and branch (`scan.Changes`), picks the first free slug and reports where the requested one is used.
- `dkt lint` lints changes through an `fs.FS`, and finds what a branch added with `gitx.AddedSince`.
- `dkt sync` reads `Change` trailers (`gitx.ChangeTrailers`), parses the plan (`change.ParsePlan`) and ticks it (`change.Tick`).
- The previous version, tag `v0.15.0`, had records with a per-section word limit. The maintainer settled a different length rule and template for this rebuild, so that code is a reference, not something to move back.

The maintainer settled these points; the rest are this design's calls, marked "(my call)":

- A title is limited on its own, and everything else as one body, with no per-section limit.
- Length is `len` of the text in UTF-8 bytes, so that no language needs a rule. A title holds at most 120 bytes and a body 1,440: 40 and 480 Chinese characters, about 20 and 250 English words.
- A change's title takes the same limit, in this change.
- The template carries no numbers and no language; it says `dkt lint` checks the length.
- The two skill edits ship with this change.

## Goals / Non-Goals

**Goals:**

- The light review path in the docket skill works from end to end: `dkt record`, `Record` trailers, `dkt sync`, `dkt lint`.
- One length rule, in one place, for every title and record body.

**Non-Goals:** see proposal.md. In addition, this design does not make `dkt lint` read `Change` trailers, and does not change how a change's plan is synced.

## Decisions

### Approaches: how much of records goes in this change

- **A. Only `dkt record` and lint.** Smaller, but the shipped skill would still tell agents to run `dkt sync <record>`, which would fail with `change-missing`.
- **B. Everything the skill's light review path names, without `dkt status` (chosen; my call).** `dkt record`, lint, `dkt sync` and the trailer check. `dkt status` is its own change and does not yet exist for changes either.
- **C. Everything including `dkt status`.** The previous version split this in two for size, and `dkt status` has more to it than records.

### Length: `change.Size`

`change.Size(text)` is `len(strings.Join(strings.Fields(text), " "))`. `strings.Fields` also removes what `markdown.BlankComments` leaves behind, so a comment costs nothing. It lives in `internal/change` beside `ValidLine`, with `TitleLimit = 120`. `record.BodyLimit = 1440` lives with records. `dkt issue` will use both.

`ValidLine` stays as it is, because it also checks a source, which has no limit. The title check is one more case at each of its two call sites (`dkt new`, lint's `checkFront`) and in `dkt record`.

Alternatives, compared on one sentence translated into eleven languages and on 30,000 commit subjects in the maintainer's repositories:

- **Words, as `v0.15.0` counted them.** Needs a table of scripts, had a bug with one Japanese mark, and counts a Thai sentence as one word, so the limit stops working there.
- **Characters.** One line and fair to most scripts, but Chinese holds about three times as much as English before it reaches the limit.
- **Bytes (chosen by the maintainer).** One line, no table, and Chinese and English come out nearly even. Cyrillic, Devanagari and Thai reach the limit two to three times sooner. If that turns out to matter, weighing Chinese, Japanese and Korean characters as 3 and everything else as 1 takes about five lines. The code carries a `shortcut:` comment saying so.

### Rule names

`title-too-long` is shared by changes and records, and `dkt issue` will use it too. `record-too-long` is the body. Keeping `title-missing` for a title that is not one line of text means a too-long title does not read as missing (my call).

### Record packages

`internal/record` holds the kinds, the body limit, the path and date helpers, and `Parse`: the front matter, the sections with comments blanked, and the tasks. `dkt record`, `dkt sync` and lint all read through it.

Tasks are parsed by `change.ParsePlan` on the record with every line before `## Tasks` blanked, so line numbers stay right for `change.Tick` and a `### Task` heading under `## What` is not a task (my call: the alternative, parsing the whole file, would tick a heading the record does not treat as a task).

### The scan

`scan.Records(g)` is a function of its own beside `scan.Changes`, in its own file, with `gitx.TreeFiles`, from `add-dkt-conflicts`, for the branches. The two loops over sources look alike, and that is accepted (my call): `add-dkt-conflicts` is editing the loop in `Changes` at the same time, `dkt new` needs no records, and `dkt record` needs no changes, because a change slug cannot start with a date and a record slug always does.

### `dkt record`

- It checks flags (exit 2), then the name (`slug-invalid`, exit 1), then links (`safepath.Check`, `symlink-refused`, exit 1), then scans (unreadable, exit 3). Then it creates the folder and writes with `O_CREATE|O_EXCL|O_NOFOLLOW` through an `os.Root`, moving to the next suffix when the file exists, so two runs in a worktree end with two records and need no lock. The `os.Root` keeps a folder swapped for a link after the checks from taking the write out of the repository; `safepath.Check` still refuses a link that stays inside it. Cleanup removes by path, and a file another process swaps in at that path just after a failed write goes with it: the review judged that low and not worth more code.
- The name's form is checked by `dkt record` itself, as OpenSpec checks a change's: `^[a-z0-9]+(-[a-z0-9]+)*$`, at most 200 characters. A record does not go through OpenSpec.
- The date comes from a new `Options.Now`, which `cmd/dkt` leaves nil for `time.Now`, in local time, so a record belongs to the user's day. Tests give a fixed time.
- The result has the same fields as `dkt new`'s, by reusing its type and its code for the `used` list. That code moves to a helper in a separate structural commit.
- A failed write removes the file and any folder the run made, as `dkt new` does. A package variable lets a test make the write fail, the way `safefile.Rename` does.

### `dkt lint`

- A name starting with a date goes to the record side. With only such names, no change and no main spec title is linted. The rule is the same test `dkt new` uses to refuse a date.
- With no name, `cmd_lint` asks `AddedSince` for `docket/records` beside the routing folders, in a separate call so the routing rule never sees records, and passes the result in `lint.Input.Records`. Lint keeps only regular `.md` files directly in that folder.
- The trailers since the merge base go in `lint.Input.RecordTrailers`, so that lint stays free of git.
- The findings carry `File` and `Line` and leave `Change` empty, since a record is not a change. The file path names the record. Reusing `checkPlaceholderFile` and the task heading rules means giving them the file name and a `loaded` with an empty slug.
- The placeholder rule for a record knows only the record template, passed in `lint.Input.RecordTemplate`. A schema template's placeholder such as `<name>` is ordinary text in a record.
- The skipped-rule note in the human output now says that records were skipped too.

### `dkt sync`

`runSync` splits at the top: a slug starting with a date reads `docket/records/<slug>.md` and the `Record` trailer; everything from parsing the trailers to ticking is shared and takes the file, the tasks and the trailer's key. It keeps `Change` empty in a record's findings, and has no schema check. `gitx.ChangeTrailers` becomes a call on a general reader, with `Trailers.Change` renamed `Values`, as a structural commit.

### Template location

`shipped/record/template.md` is embedded like the rest of `shipped/` and is not written into a project; `dkt init` writes named paths under `shipped/`, so it does not pick it up. `TestShippedFileSet` gains the file.

## Risks / Trade-offs

- [An agent writes the name it gave instead of the slug `dkt` reported] → `record-trailer` reports it before the pull request.
- [Two machines pick the same name before either pushes] → the second merge has a conflict on the path; renaming is manual. Rare, and the date narrows it.
- [`add-dkt-conflicts` also edits `AGENTS.md`, `internal/cli/command.go`, `internal/scan/scan.go` and the requirement Repository required] → it merges first; this change rebases and takes its text.
- [Scanning for records adds a `git ls-tree` per commit on each `dkt record`] → the scan caches by commit, and records are opened a few times a day.
- [`dkt lint` makes one more `git log` for trailers] → it runs only with no names and a merge base, over the branch's own commits.
- [Bytes penalise some scripts] → accepted; see Length.

## Migration Plan

- No project uses records yet. This repository's `AGENTS.md` drops its stopgap in this change, and the next light-review work opens a record.
- `dkt new` and `dkt lint` become stricter for long change titles. The active and archived changes here have none over the limit.
- Rollback: revert the merge.
