# Design

## Context

See proposal.md for the motivation. The docket skill says to take `result.next`, and what an idea file holds, but no code reads either, and `docket/order.md` has no defined format. `internal/scan` already finds changes and records in every worktree and branch, and `gitx.DefaultBranch` names the tip every other check uses. v0.15.0 had a much larger `dkt status`; it is a reference only.

## Goals / Non-Goals

**Non-Goals:** see proposal.md. This design also does not decide whether an idea is finished: "taken" is as far as it goes.

## Decisions

**One test for "still to do": is anyone on it?** An idea stops being next once any change or record, anywhere, names it. That covers both "finished" and "someone is on it in another worktree", which matters for the session asking: both mean "do not start this". It needs no state. The skill already says part of an idea that must wait becomes a new idea, so one piece of work is enough to take one. Alternative: every handler merged, as v0.15.0 did: it needs state and answers a question this version does not ask.

**The source must be exactly the path.** The skill says `--source` holds only the path. v0.15.0 matched a path followed by other text; that invites `docket/ideas/a.md.bak` and `docket/ideas/a.md, not this one`.

**Read the order and ideas from the default branch tip.** Edits to them are direct merges, so the tip is what has been agreed, and every worktree of a repository gets the same answer. Reading the current worktree would give a stale worktree a stale answer. The cost: an edit is not seen until it is merged and fetched. Reading git objects also means no symbolic link can be followed: a linked file reads as its target's name, which then fails to parse.

**`- docket/ideas/<name>.md` per line.** Reordering moves one line, with no numbers to rewrite, unlike v0.15.0's numbered lines. Headings are ignored so that the file can be titled. Anything else is an error rather than skipped, because a skipped line could be the first idea, and the agent would start the wrong work. The finding states the form, since no shipped text does.

**Issues in the order, local ones only.** An issue has a settled expected behavior and can be as next as any idea. It stays out of the rest: no `dropped`, since an issue no longer wanted leaves the order; `needs` names only ideas, and either can be widened later without breaking anything. An address in another tool is refused because dkt could not read its title or tell whether it is closed, and a differently written address would look untaken.

**`result.next` has no `kind`.** The path already says it, under `docket/ideas/` or `docket/issues/`, and a field can be added later without breaking callers.

**One-line `needs`.** The front matter reader in `internal/change` takes one line per key apart from `specs`, and this keeps it: no YAML dependency for one list. A block list fails with a finding that shows the form.

**Honor `needs`.** The skill's rule for work that has to wait writes the waiting part as an idea that needs the original, and that idea sits behind one in progress: ignoring `needs` would make it next too early. A need is met only when the work is on the tip, since the skill says dependent work waits for the merge.

**A dropped need counts as met.** Otherwise the idea would wait forever and say only `waiting`. The agent reads the idea before starting, and the person who dropped the need can drop or edit this one too. Alternative: report a problem: that blocks the whole order on a decision that may be fine.

**A committed symbolic link names nothing.** `ReadBlobs` returns a link's target as text, which has no front matter, so a linked `proposal.md` on a branch counts as no work rather than as unreadable, unlike one in a worktree. `dkt new` refuses links, so this takes a link committed by hand, and telling them apart needs another git call per commit.

**Any problem makes `next` null, even a later line.** A simple rule, and `ok: false` beside a `next` would ask the agent to weigh them. The fix is a one-line direct merge. Alternative: answer from the lines before the first problem: more cases, and it hides a broken order until it is reached.

**Only the items the order reaches are read.** The order's lines and their needs. Linting every idea is `dkt lint`'s job, later.

**With no order or a problem in it, no scan.** `next` is null whatever the work says, so reading every worktree and branch would only add to the report. Fixing the order brings any unreadable source to light on the next run.

**An unreadable source does not stop the answer.** `next` is still reported with exit 3, so a broken proposal in someone's worktree does not stop every session; the cost is that the unread proposal might have named that idea, which `unreadable` shows. The rest of Docket treats an unreadable place the same way.

**`result.next` is an object.** `{path, title}` can grow for the workbench without breaking callers, where a bare path could not, and the title is read anyway.

**The result stays small.** `next`, `reason` and `waiting_on`. `reason` lets a caller tell "nothing ordered" from "all done" without parsing text.

## Risks / Trade-offs

- [The skill says what `result.next` is for but not what to do when it is null] → An agent reads `reason`; the skill's text is the maintainer's to change, and is not touched here.
- [Nothing shipped states the order's format] → The `order-format` finding states it.
- [A record's `scan.Use` has no commit, so `FindWork` looks its ref up in a second `Refs` call; a ref that moves in between is read at its new commit] → Accepted: the window is milliseconds and the answer is one a run a moment later would give. Setting the commit in `scan.Records` is a one-line change for later, in a file `add-dkt-issue` is changing now.
- [A worktree's file that disappears between the scan and the read names nothing] → Accepted: a run a moment later would give the same answer, and a change folder without `proposal.md` looks the same.
- [The tip is only as fresh as the last fetch] → Accepted: `dkt` does not contact a remote, and `default-branch` defines the tip this way everywhere.
