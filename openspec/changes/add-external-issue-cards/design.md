# Design

## Context

See proposal.md - Why. Today:

- `shipped/skill/SKILL.md` ("Ideas, issues and the order") says an issue goes to the tool `## Issues` in `AGENTS.md` names, that such issues are not listed in the order, and that `--source` may hold the issue's address in that tool. dkt never reads `AGENTS.md`.
- Whether an issue is taken is decided in two places, both by exact match of a work's `source` with the issue's path: `idea.FindWork` and `idea.Next` for `dkt status`, and the issue `handlers` in `internal/snapshot/backlog.go` for `dkt snapshot`.
- The issue rules of `dkt lint` live in `internal/lint/record.go` (`issueKind`, `lintDated`, `checkDatedFront`, `checkDatedSections`); the record rules share them, and `dated.after` already allows sections after the required ones (records use it for `## Tasks`).
- `dropped` and `needs` on an issue are ignored by `dkt status` (ideas spec, "Next item").

## Goals / Non-Goals

Goals: one issue model for local issues and issues kept in another tool, so that state and handlers come from Docket's own data and stay readable when the other tool cannot be reached; the least data from the other tool (address, number, title); the issue read in full from the tool whenever work on it starts.

Non-goals beyond the proposal's: no change to how a work names an item, and no reading of `AGENTS.md` by dkt.

## Approaches

- **A. A card is an issue, and dkt stays offline** (chosen). A card is an issue file with two more fields and no body. The order, `scan.Issues`, `dkt status`, `dkt snapshot` and viewers treat it as any issue; the flow with the other tool is text in the skill, carried out by the coding agent with the tool `## Issues` names. Trade-off: the notes written to the other tool are worded by each agent.
- **B. As A, with dkt printing the text of each note.** Uniform wording, and dkt could list the pending `unsynced` notes. Costs a command and its spec, and dkt has no pull request address to put in them. Left for later, if wording turns out to vary too much.
- **C. A separate kind, such as `docket/external/`.** Clearer separation, but the order, the scans, the snapshot and viewers each need a second path, against the goal of one issue model. Rejected.
- dkt calling each tool's API was ruled out: it breaks Docket's independence from outside systems and needs code per tool.

The maintainer chose A on 2026-11-12.

## Decisions

### The flow, in the skill

The skill's "Ideas, issues and the order" section gains the flow below; it applies only when `AGENTS.md` has `## Issues`. Reading the tool ("syncing") means reading the issue's content, comments and last update time with the tool `## Issues` names.

| Case | What happens |
|---|---|
| New issue | File it in the tool, once the user agrees, with the sections from `dkt issue template`; then make its card. When it cannot be filed, write a local issue with the four sections and `unsynced: [open]`; once it can be filed, file it, add `external` and `external-id`, empty the body and drop `open`. |
| Existing issue in the tool | Make its card, with only the card's fields; no proposal or record is written for it now. |
| The user asks to sync | The user says which issues. Those without a card get one, as above. For those with a card, align the card: update `title` when the tool's title changed; when the tool has closed it and it is not done, list it for the user and, once they agree, drop it with the reason. The edits are a direct merge of their own. |
| Work starts on a card | Before brainstorming or the record, sync the issue: align the card as above, and put its content and comments, with the tool's last update time as read, into the proposal or the record. When the tool has closed it, stop and ask the user. Card edits go into the work's first commit. |
| Before the pull request | Sync again, and add what changed since work started. |

Making a card: check that no card has the address (`dkt issue` refuses one that does), then `dkt issue <name> --external <address> --external-id <id> --title "<the tool's title>" --source "<source>"`, then `dkt lint`. The source is where the issue was found for a new issue, and its address for an existing one. A card is not added to the order on its own.

Notes written to the tool, when the agent can write there:

| When | Note |
|---|---|
| A change or record takes the card | which work |
| Its pull request is opened | the pull request's address |
| It is merged | the merge commit; whether to close the issue too is for `## Issues` to say |
| The card is dropped | the reason |

Notes are comments, which every tool has; closing, status fields and labels differ between tools and are left to `## Issues`. Writing to the tool is visible to others, so the agent makes only the writes `## Issues` allows and asks the user before any other. When a note cannot be written, its step goes into the card's `unsynced`; once the tool can be reached, the agent writes it from the current local data and removes the step. Nothing written to the tool is copied into the card: who handles it and whether it is merged come from the works and git, and the dropped reason is in `## Dropped`.

When work handles a card, `--source` holds the card's path, as for any issue; the skill and the proposal template stop offering the address.

### The card's fields

`external` is the key: lint and `dkt issue` compare it exactly. `external-id` is for people (`#92`, `PROJ-123`, what commit messages use) and is not compared, because two tools in one project can both have a `#92`. The two are given together, each in the form of a source; `external` is not required to be a URL, since a tool may address an issue by a path.

### Taken stays the card's path

A work names an issue only when its `source` is the card's path. Also counting a `source` equal to a card's `external` was considered: it needs changes in both places that decide taken, and exact matching of addresses misses on a trailing `/` or a `#` part. With a card made as soon as an issue enters the flow, new work never needs it.

### Dropped, for every issue

`dropped: true` works on local issues and cards alike, with the reason under `## Dropped`, the last section, added to `issueKind.after`. A field such as `dropped-reason` was considered; the reason is prose, and the section reuses the parsing and checks the records already have. `dkt snapshot` reports a dropped issue as `dropped` whatever its handlers; `dkt status` skips it.

### Unsynced steps

`unsynced` takes a fixed list of steps (`open`, `taken`, `pr`, `merged`, `dropped`), written in the one-line list form of an idea's `needs`, so that lint can check it and viewers can show which notes are missing. `open` belongs to an issue not yet filed, so never to a card; the other steps belong only to cards.

### Duplicate addresses, in the worktree

`dkt issue` and `dkt lint` look for another issue with the same `external` in the worktree's `docket/issues/` only. Reading every branch's issues would need their contents, where `scan.Issues` lists only their paths; two branches that card the same issue meet once one is rebased on the other, and lint reports it then.

### Edge cases settled for the build

- A card is an issue whose front matter has the `external` key, whatever its value. A blank or one-sided pair is an `issue-front-matter` finding; the card's section rules still apply, so a card is never also checked as a local issue.
- A dropped issue is not next, whether or not a work names it, and no finding is reported for a work that names a dropped issue: the work may be what found that it should be dropped.
- `external` values are compared exactly, as the front matter reads them (after unquoting). An issue keeps its address for good, dropped or done: when the tool reopens an issue, its card is used again (remove `dropped`, and new work names the card). Lint reports the issue it lints, naming the other.
- `unsynced` is not checked against the issue's state: its steps are notes still to write, not a state.
- `dkt snapshot` reads `dropped` as true only when it is `true`, and `unsynced` only in its one-line form, empty otherwise; it reports the findings `dkt status` would (a bad `dropped` on an issue in the order) and leaves the rest of the form to `dkt lint`.
- `dkt issue` reports `external-taken` through the findings it already uses for `slug-invalid` (exit 1), and an unreadable issue through its existing `unreadable` path (exit 3).
- Editing an existing card and naming it to `dkt lint` already works: the records spec ("Records named") lints a named issue whatever its state.

### The skill text is a shipped prompt

The edits to `shipped/skill/SKILL.md` and `shipped/schema/templates/proposal.md` are drafted in Traditional Chinese on the prompt page and land only once the maintainer has edited them; the dkt tasks do not wait for them. After they land, `dkt init --force` refreshes the installed copies, and `internal/setup/history.go` gets the new hashes.

## Risks / Trade-offs

- [A card's title can be stale in the order and in viewers until work starts or the user syncs] → Accepted: the read before work corrects it, so no work starts from a stale card.
- [An agent skips the read before work] → The skill makes it the first step of work on a card, and the proposal or record records the last update time it read, so a reviewer can see it was done.
- [Edits to an existing card are not linted, as `dkt lint` covers the issues a branch adds] → The skill says to run `dkt lint <slug>` naming the card after editing it; named issues are linted whatever their state.
- [Notes in the tool are visible to others] → Only what `## Issues` allows is written without asking.
- [An issue that had `dropped: true` was ignored and now is skipped] → A change in behavior; it is unlikely that projects wrote `dropped` on issues while it did nothing. The release notes say so.

## Migration Plan

Nothing to migrate: local issues are unchanged, and a project starts using cards when it wants to. Works whose `source` is an address keep naming no issue, as today. The release after merging is a minor version.
