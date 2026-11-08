# Design

## Context

See `proposal.md` - Why. What dkt already has:

- `internal/scan` finds every change folder, record and issue in every worktree, local branch and remote-tracking branch (`workspace-scan`), with each use's source.
- `internal/idea` reads ideas, the order and the work naming each item, from the default branch tip (`ideas`); `dkt status` reports only the next item.
- `internal/change` parses plan tasks and steps; `internal/record` parses records and their `## Tasks`; `dkt sync` takes trailers reachable from HEAD (`plan-sync`).
- `internal/domain` loads the domain tree from an `fs.FS` (`domain-tree`).
- `dkt lint` and `dkt conflicts` read delta specs through the OpenSpec CLI; nothing reads them without it.
- Measured on this repository: `status` 1.1 s, `lint` 0.69 s, `domains` 0.24 s; looking up merges and tags one at a time starts git 221 times (6.0 s), batched 2 times (0.10 s).

## Goals / Non-Goals

**Goals:**

- One run gives a viewer every Docket fact it shows, with no inference left to it: stages, states, positions and merges are decided here.
- About one second on a repository the size of this one.
- A source that cannot be read costs only what it holds.

**Non-Goals:**

- The proposal's non-goals. At design level, also: reading changes of another schema beyond their proposal and deltas, and any field a viewer can compute from the others (counts, sums, labels).

## Output

`dkt snapshot --json` prints the standard envelope; `result` is the object below. Arrays are never null. Times are RFC 3339 with the committer's offset, as `git log --format=%cI` gives them; dates are `YYYY-MM-DD`. Commits are full SHAs. Paths are relative to the repository root and slash-separated, except a worktree's path, which is absolute.

### `result`

| Field | Type | Meaning |
|---|---|---|
| `default_branch` | object or null | The default branch (`default-branch`); null when there is none. |
| `works` | array of work | Every change and record found, sorted by `created`, then `type`, then `slug`. |
| `ideas` | array of idea | Every idea on the default branch tip, sorted by path. |
| `issues` | array of issue | Every issue found, sorted by slug. |
| `order` | array of order item | The items of `docket/order.md` on the tip, in order. |
| `specs` | array of spec | Every main spec on the tip, sorted by capability. |
| `domains` | object | The domain tree on the tip. |

### `default_branch`

| Field | Type | Example |
|---|---|---|
| `name` | string | `"main"` |
| `ref` | string | `"refs/remotes/origin/main"`: the ref its tip is read from. |
| `commit` | string | `"56adc09d8bfe381381e2f17236b2e6eccca5db58"` |
| `time` | string | `"2026-11-08T10:12:03+08:00"` |
| `subject` | string | `"Merge pull request #113 from wake/worktree-move-viewer-ideas"` |

### work

| Field | Type | Meaning |
|---|---|---|
| `type` | `"change"` or `"record"` | |
| `slug` | string | The change's slug, or the record's slug (`2026-11-07-fix-x`). |
| `title`, `kind`, `source` | string or null | From the front matter; null when it cannot be read. |
| `schema` | string | `"docket"`, or the other schema a change names; always `"docket"` for a record. |
| `created` | string (date) or null | A change: `created` in its `.openspec.yaml`. A record: the date in its slug. |
| `proposed` | string (date) or null | The `date` of the idea or issue its `source` names; otherwise `created`. |
| `position` | integer or null | The `position` of the order item its `source` names; null when it names none. |
| `read_from` | string | The location the fields of this work were read from (see Copy read). |
| `archived` | string or null | The archive folder, such as `openspec/changes/archive/2026-11-08-add-x`; null for an active change and for a record. |
| `stages` | array of string | The stages of its chain it has reached, in chain order (see Stages): for a change some of `proposal`, `design`, `plan`, `dev`, `merged`, `archived`; for a record some of `record`, `dev`, `merged`. |
| `phase` | string | The last entry of `stages`. |
| `progress` | object | `{done, total}`: how many of its `tasks` are done, out of how many; `{0, 0}` without tasks. |
| `merged` | boolean | The default branch tip holds the work, active or archived. |
| `merge` | object or null | `{commit, time, subject}`: the commit of the default branch that brought it in (see Merge). |
| `release` | object or null | `{tag, time}`: the first tag that contains `merge`. |
| `last_commit` | object or null | `{commit, time}`: the newest commit, in any location, whose trailer names the work. |
| `locations` | array of location | Every source holding the work, in the order of `workspace-scan`. |
| `tasks` | array of task, or null | From `plan.md` or the record's `## Tasks`, in order; null for a change of another schema. |
| `documents` | array of document | The work's files and its source document. |
| `deltas` | array of delta | A change's delta specs, sorted by capability; empty for a record. |

### location

| Field | Type | Example |
|---|---|---|
| `source` | string | `"/w/app/.claude/worktrees/add-x"`, `"refs/heads/worktree-add-x"`, `"refs/remotes/origin/worktree-add-x"` |
| `kind` | `"worktree"`, `"branch"` or `"remote-branch"` | |
| `branch` | string or null | A worktree's checked-out branch, null when detached; the branch name for the others (`"worktree-add-x"`, `"origin/worktree-add-x"`). |
| `default` | boolean | This is the ref `default_branch.ref` names. |
| `commit` | string or null | The worktree's HEAD or the ref's commit; null for a worktree with no commit yet. |
| `time` | string or null | That commit's time. |
| `path` | string | The change folder or record file in this source. |
| `archived` | boolean | The folder is under `openspec/changes/archive/`. |

### task

| Field | Type | Meaning |
|---|---|---|
| `number` | integer | N of `### Task N:`. |
| `name` | string | |
| `steps` | integer | Its `- [ ]` and `- [x]` steps, as `plan-sync` counts them. |
| `ticked` | integer | The steps ticked in the file. |
| `commits` | array of string | Commits naming it, oldest first, reachable from the commit of `read_from`. |
| `state` | `"done"` or `"pending"` | `done` when a commit in `commits` names it or every step is ticked. |

### document

| Field | Type | Example |
|---|---|---|
| `kind` | string | `proposal`, `design`, `plan`, `spec-delta`, `record`, `idea`, `issue` or `other` |
| `path` | string | `"openspec/changes/add-x/specs/booking/spec.md"` |
| `source` | string | The location it is read from: `read_from` for the work's own files, `default_branch.ref` for an idea, an issue's first location for an issue. |

### delta

| Field | Type | Example |
|---|---|---|
| `capability` | string | `"booking"` |
| `path` | string | `"openspec/changes/add-x/specs/booking/spec.md"` |
| `added` | array | `[{"name": "Join when full", "scenarios": ["Full slot", "Slot frees up"]}]` |
| `modified` | array | same form as `added` |
| `removed` | array | `[{"name": "Walk-in mark", "reason": "Walk-ins moved to paper."}]` |
| `renamed` | array | `[{"from": "Party limit", "to": "Group size limit"}]` |

### idea

| Field | Type | Meaning |
|---|---|---|
| `path` | string | `"docket/ideas/dkt-projects-index.md"` |
| `title` | string or null | null when the front matter cannot be read. |
| `needs` | array of string | |
| `dropped` | boolean | |
| `date` | string (date) or null | See Idea date; null when no commit on the tip's history added it. |
| `state` | string | `queued`: in the order and still to do; `pool`: not in the order and still to do; `dropped`; `taken`: a change or record names it. |
| `waiting_on` | array of string | The needs not yet met, as `ideas` defines them; empty unless `queued` or `pool`. |
| `position` | integer or null | Its order item's `position`. |
| `taken_by` | array of work ref | `{type, slug}` of each work whose `source` is its path. |

### issue

| Field | Type | Meaning |
|---|---|---|
| `slug` | string | `"2026-11-07-title-missing-wrong-path"` |
| `path` | string | |
| `title`, `source` | string or null | From the front matter, read from its first location. |
| `date` | string (date) | The date in its slug. |
| `state` | string | `open`: no handler; `in-progress`: a handler is not merged; `done`: every handler is merged. |
| `position` | integer or null | Its order item's `position`. |
| `handlers` | array | `{type, slug, merged}` of each work whose `source` is its path. |
| `locations` | array of location | Every source holding the file, as for a work. |

### order item

| Field | Type | Example |
|---|---|---|
| `position` | integer | 1 for the first item line, counting only item lines. |
| `path` | string | `"docket/ideas/codex-support.md"` |

### spec

| Field | Type | Example |
|---|---|---|
| `capability` | string | `"change-lint"` |
| `title` | string | `"Change Lint"`: the first line's title, as `change-metadata` defines it. |
| `path` | string | `"openspec/specs/change-lint/spec.md"` |
| `requirements` | array of string | `["What is linted", "Front matter checked"]`, in file order. |

### `domains`

| Field | Type | Meaning |
|---|---|---|
| `found` | boolean | `docket/domains.md` exists on the tip. |
| `nodes` | array of node | `{name, capabilities, children}`: `capabilities` is an array of capability paths; titles are under `specs`, and the changes touching a capability under `works[].deltas`. |
| `unclassified` | array of string | Capabilities of the main specs and of the active changes' deltas that no node holds, sorted. |

This differs from `dkt domains --json`, whose capabilities also carry a title, `main` and `changes` read from the worktree: here those facts are elsewhere in the same result, read from the tip and from every location.

### Example

Abridged: one change in progress in a linked worktree, one archived change, one queued idea.

```json
{
  "default_branch": {"name": "main", "ref": "refs/remotes/origin/main", "commit": "56adc09d8bfe381381e2f17236b2e6eccca5db58", "time": "2026-11-08T10:12:03+08:00", "subject": "Merge pull request #113 from wake/worktree-move-viewer-ideas"},
  "works": [
    {
      "type": "change", "slug": "add-domain-tree", "title": "Add a domain tree for viewing specs by area", "kind": "feature",
      "source": "Maintainer request: classify specs in a tree for viewing", "schema": "docket",
      "created": "2026-11-07", "proposed": "2026-11-07", "position": null,
      "read_from": "/w/docket", "archived": "openspec/changes/archive/2026-11-07-add-domain-tree",
      "stages": ["proposal", "design", "plan", "dev", "merged", "archived"], "phase": "archived", "progress": {"done": 1, "total": 1}, "merged": true,
      "merge": {"commit": "a1b2c3d4e5f60718293a4b5c6d7e8f9012345678", "time": "2026-11-07T21:40:11+08:00", "subject": "Merge pull request #106 from wake/worktree-add-domain-tree"},
      "release": {"tag": "v0.18.0", "time": "2026-11-07T22:05:00+08:00"},
      "last_commit": {"commit": "0f1e2d3c4b5a69788796a5b4c3d2e1f0a9b8c7d6", "time": "2026-11-07T21:30:52+08:00"},
      "locations": [
        {"source": "/w/docket", "kind": "worktree", "branch": "main", "default": false, "commit": "56adc09d8bfe381381e2f17236b2e6eccca5db58", "time": "2026-11-08T10:12:03+08:00", "path": "openspec/changes/archive/2026-11-07-add-domain-tree", "archived": true},
        {"source": "refs/heads/main", "kind": "branch", "branch": "main", "default": false, "commit": "56adc09d8bfe381381e2f17236b2e6eccca5db58", "time": "2026-11-08T10:12:03+08:00", "path": "openspec/changes/archive/2026-11-07-add-domain-tree", "archived": true},
        {"source": "refs/remotes/origin/main", "kind": "remote-branch", "branch": "origin/main", "default": true, "commit": "56adc09d8bfe381381e2f17236b2e6eccca5db58", "time": "2026-11-08T10:12:03+08:00", "path": "openspec/changes/archive/2026-11-07-add-domain-tree", "archived": true}
      ],
      "tasks": [
        {"number": 1, "name": "Parse the tree file", "steps": 5, "ticked": 5, "commits": ["9d8c7b6a5f4e3d2c1b0a99887766554433221100"], "state": "done"}
      ],
      "documents": [
        {"kind": "proposal", "path": "openspec/changes/archive/2026-11-07-add-domain-tree/proposal.md", "source": "/w/docket"},
        {"kind": "design", "path": "openspec/changes/archive/2026-11-07-add-domain-tree/design.md", "source": "/w/docket"},
        {"kind": "plan", "path": "openspec/changes/archive/2026-11-07-add-domain-tree/plan.md", "source": "/w/docket"},
        {"kind": "spec-delta", "path": "openspec/changes/archive/2026-11-07-add-domain-tree/specs/domain-tree/spec.md", "source": "/w/docket"}
      ],
      "deltas": [
        {"capability": "domain-tree", "path": "openspec/changes/archive/2026-11-07-add-domain-tree/specs/domain-tree/spec.md",
         "added": [{"name": "Tree file", "scenarios": ["Nested nodes read", "Empty node"]}], "modified": [], "removed": [], "renamed": []}
      ]
    },
    {
      "type": "change", "slug": "projects-index", "title": "A local index of the projects on a machine", "kind": "feature",
      "source": "docket/ideas/dkt-projects-index.md", "schema": "docket",
      "created": "2026-11-08", "proposed": "2026-11-08", "position": null,
      "read_from": "/w/docket/.claude/worktrees/projects-index", "archived": null,
      "stages": ["proposal", "design", "plan", "dev"], "phase": "dev", "progress": {"done": 1, "total": 3}, "merged": false, "merge": null, "release": null,
      "last_commit": {"commit": "13c8503aa0b1c2d3e4f5a6b7c8d9e0f1a2b3c4d5", "time": "2026-11-08T11:02:44+08:00"},
      "locations": [
        {"source": "/w/docket/.claude/worktrees/projects-index", "kind": "worktree", "branch": "worktree-projects-index", "default": false, "commit": "13c8503aa0b1c2d3e4f5a6b7c8d9e0f1a2b3c4d5", "time": "2026-11-08T11:02:44+08:00", "path": "openspec/changes/projects-index", "archived": false},
        {"source": "refs/heads/worktree-projects-index", "kind": "branch", "branch": "worktree-projects-index", "default": false, "commit": "13c8503aa0b1c2d3e4f5a6b7c8d9e0f1a2b3c4d5", "time": "2026-11-08T11:02:44+08:00", "path": "openspec/changes/projects-index", "archived": false}
      ],
      "tasks": [
        {"number": 1, "name": "The index and settings files", "steps": 5, "ticked": 0, "commits": ["b0c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9"], "state": "done"},
        {"number": 2, "name": "The project of a directory", "steps": 5, "ticked": 1, "commits": [], "state": "pending"},
        {"number": 3, "name": "dkt projects add and ls", "steps": 6, "ticked": 0, "commits": [], "state": "pending"}
      ],
      "documents": [
        {"kind": "idea", "path": "docket/ideas/dkt-projects-index.md", "source": "refs/remotes/origin/main"},
        {"kind": "proposal", "path": "openspec/changes/projects-index/proposal.md", "source": "/w/docket/.claude/worktrees/projects-index"},
        {"kind": "spec-delta", "path": "openspec/changes/projects-index/specs/project-index/spec.md", "source": "/w/docket/.claude/worktrees/projects-index"}
      ],
      "deltas": [
        {"capability": "project-index", "path": "openspec/changes/projects-index/specs/project-index/spec.md",
         "added": [{"name": "Index file", "scenarios": ["Index under the state home"]}], "modified": [], "removed": [], "renamed": []}
      ]
    }
  ],
  "ideas": [
    {"path": "docket/ideas/codex-support.md", "title": "Verify Docket on Codex and ship its plugin", "needs": [], "dropped": false, "date": "2026-11-04", "state": "queued", "waiting_on": [], "position": 1, "taken_by": []},
    {"path": "docket/ideas/dkt-projects-index.md", "title": "A local index of the projects on a machine", "needs": [], "dropped": false, "date": "2026-11-08", "state": "taken", "waiting_on": [], "position": null, "taken_by": [{"type": "change", "slug": "projects-index"}]}
  ],
  "issues": [
    {"slug": "2026-11-07-title-missing-wrong-path", "path": "docket/issues/2026-11-07-title-missing-wrong-path.md", "title": "dkt lint calls a delta on a wrong path a new capability", "source": "Found while designing add-domain-tree, 2026-11-07", "date": "2026-11-07", "state": "open", "position": null, "handlers": [],
     "locations": [{"source": "refs/remotes/origin/main", "kind": "remote-branch", "branch": "origin/main", "default": true, "commit": "56adc09d8bfe381381e2f17236b2e6eccca5db58", "time": "2026-11-08T10:12:03+08:00", "path": "docket/issues/2026-11-07-title-missing-wrong-path.md", "archived": false}]}
  ],
  "order": [
    {"position": 1, "path": "docket/ideas/codex-support.md"},
    {"position": 2, "path": "docket/issues/2026-11-06-new-link-swap.md"},
    {"position": 3, "path": "docket/ideas/windows-executables.md"}
  ],
  "specs": [
    {"capability": "domain-tree", "title": "Domain Tree", "path": "openspec/specs/domain-tree/spec.md", "requirements": ["Tree file", "Tree read from the worktree"]}
  ],
  "domains": {"found": false, "nodes": [], "unclassified": ["domain-tree", "project-index"]}
}
```

Without `--json`, the command prints one line per work (`<type> <slug>  <phase>  <title>`), then the counts of ideas, issues and specs.

## Decisions

**A new command, not new fields on old ones.** Each old command keeps its scope and its read source; one scan feeding everything is the reason for the change. The name `snapshot` says it is a read of the whole project at one moment.

**Where each part is read from.** Works and issues come from every location (`workspace-scan`), since work in progress lives in worktrees and branches. Ideas, the order, main specs and the domain tree come from the default branch tip, as `dkt status` reads ideas and the order: they describe the project as merged. Without a default branch, those four are empty and `default_branch` is null; works and issues are still reported.

**Copy read.** A work found in several locations is reported once, its fields read from one copy, named in `read_from`: the first location, in `workspace-scan` order (worktrees as `git worktree list` gives them, then local branches, then remote-tracking branches), that holds it archived; when none does, the first that holds it. Archiving is the last step of a change, so an archived copy is never older than an active one. A worktree comes first because it holds uncommitted edits. An issue is read from its first location in the same order.

**Stages, not a judged phase.** The maintainer chose a chain of stages over a judged phase such as "in progress". Each stage is a fact a reader can check, and `stages` lists the ones reached, so a stage that was skipped (a change without design) shows as missing between reached ones, and a viewer greys the rest without inferring anything. For a change: `proposal` when the folder of `read_from` has `proposal.md`; `design` when it has a delta spec or `design.md`; `plan` when its plan has a task; `dev` when a commit reachable from any location carries `Change: <slug>/<N>`, naming a task; `merged` when the default branch tip holds it; `archived` when `archived` is set. For a record: `record` always; `dev` when any commit reachable from any location carries its `Record` trailer, with or without a task; `merged` as for a change. `phase` is the last stage reached. Whether someone is working on it now, and where, is not a stage: a viewer sees it from `locations` and `last_commit`.

**Tasks are done or pending.** A task is `done` when a commit reachable from `read_from` names it, or every step is ticked: `dkt sync` ticks every step of a task a commit names, so a task with a commit counts as ticked whether or not sync has run. Otherwise it is `pending`. `progress` counts them, so a change whose tasks are all done but which is not merged reads as `dev` with a full `progress`.

**Idea date.** The date of the oldest commit, in the history of the default branch tip, that added the idea's file (`git log --diff-filter=A`), in that commit's own offset. It needs no new field in an idea and works for every idea already written. An idea moved to a new name gets the date of the move.

**Issue state.** `open` with no handler, `in-progress` while some handler is not merged, `done` when every handler is merged. A handler is a change or record whose `source` is the issue's path, as `ideas` defines it.

**Merge.** dkt reads the first-parent history of the default branch tip once, with each commit's parents and trailers, plus the trailers of the commits each merge brought in. A work's `merge` is the newest first-parent commit that brought in a commit whose `Change:` or `Record:` trailer names the work, with or without a task: a merge commit for a merge, or the commit itself for a rebase. `merged` is separate: it says only that the tip holds the work, so a work without trailers is still merged.

**Release.** Tags are listed once with `git for-each-ref --sort=creatordate refs/tags`; `release` is the oldest tag whose commit has `merge` in its history. Its `time` is the tag's creation time (the tagger date of an annotated tag, the commit date of a lightweight one).

**Delta and spec reading without OpenSpec.** dkt reads the headings itself: `## ADDED|MODIFIED|REMOVED|RENAMED Requirements`, `### Requirement: <name>`, `#### Scenario: <name>`, a removed requirement's `**Reason**:` line, and a rename's `FROM:` and `TO:` lines, outside fenced code and HTML comments. This covers the format OpenSpec documents; a file OpenSpec would reject may still give a partial result here, and `dkt lint` remains the check that a delta is valid. Running OpenSpec would cost a Node.js start per change and break the no-OpenSpec requirement.

**Batched git, bounded by commits, not by works.** The scans of `workspace-scan` (`scan.Changes`, `scan.Records`, `scan.Issues`) run as one pass, `scan.Scan`: it lists the worktrees and refs once and each distinct ref commit once with one `git ls-tree -r`, which also lists the paths the snapshot reads, so the scans and the files of works share it. On top of it: one `git for-each-ref` for tags, one `git log` for the history and trailers over every location's commit, one for idea dates, and one `git cat-file --batch` for the files of every commit, so locations sharing a commit share the read. On a repository with C distinct ref commits the snapshot starts git C + 7 times, besides what every command runs to find the repository. Running the three scans separately cost 42 git runs and about 2.3 s on this repository on macOS, where a git run costs about 44 ms; Task 9 of the plan measured it, and Task 10 merged them.

**Findings and exit codes.** Problems `dkt status` or `dkt domains` would report on the tip (`order-format`, `order-item`, `idea-front-matter`, `idea-needs`, `issue-front-matter`, `domain-format`, `domain-duplicate`, `domain-unknown`) are findings here too, and the result is still output; dkt exits with 1 when there is any, and with 3 when a source is unreadable, as `cli-output` says. A work whose front matter cannot be read gives null fields and is listed in `unreadable`, as `ideas` lists it for `dkt status`: which idea or issue it takes depends on its `source`, so a taken state would otherwise be guessed. A domain tree with `domain-format` or `domain-duplicate` gives no nodes and every capability unclassified, where `dkt domains` gives a null `result`: here the rest of the project is still worth showing.

## Risks / Trade-offs

- [dkt's delta reader and OpenSpec's disagree on an odd file] → The snapshot shows what dkt read; `dkt lint` reports the file through OpenSpec. The reader follows the documented headings only.
- [A slow repository: many worktrees and branches] → Each distinct commit is read once; worktrees are read from disk. The target is measured on this repository before the change is merged.
- [A work in two worktrees with different edits] → One copy is reported, the first in scan order; `locations` still lists both.
- [An idea's history rewritten, or added by a squash] → Its date is that of the commit that exists now.

## Migration Plan

None. A new read-only command.

## Settled Questions

Five questions in the first draft changed the spec; the maintainer answered them on 2026-11-08, through the team lead:

1. to 3. The phase of a record without tasks, of a change whose plan has no task, and when a task counts as started: replaced by the chain of stages under Stages, with tasks only `done` or `pending` and `progress` given apart. The idea a work came from is shown by its `source`, not as a stage, and `release` stays its own field.
4. An issue whose handlers are all merged is `done`.
5. An idea's date is the date of the commit that added it on the default branch.

The maintainer will adjust these once real output can be looked at.
