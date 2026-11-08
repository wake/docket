---
title: A read-only command that outputs a whole project at once
---

A viewer of a project's work, such as a desktop app that reads through a local server passing dkt's output on unchanged, reads this command's output directly. So the output is written in Docket's own terms, and makes sense without any viewer. Each existing command scans the project again: measured on the Docket repository, `status` takes 1.1 s, `lint` 0.69 s and `domains` 0.24 s. Run one by one they repeat the scan, so one command gives it all at once.

## In scope

- Project overview: the default branch, and its latest commit and time.
- Work list: each change and record with its type, slug, title, kind, source, and created and proposed dates.
- Locations: the worktrees, local branches and remote branches each piece of work appears in, each with its commit and time; the scope follows `workspace-scan`.
- Task progress: each task's number, name, number of steps, number ticked and commits; the rules follow `plan-sync`, for all the work at once.
- The work's documents: each document's kind (proposal, design, plan, spec delta, record, idea, issue, evidence), path and the source it is read from.
- The domain tree: what `dkt domains` outputs today.
- The sections below (phase, merge and release, spec content, ideas and issues) are part of this command's output too, not commands of their own. The local project index (`docket/ideas/dkt-projects-index.md`) is not: it belongs to no single project.

## Requirements

- Read-only: writes nothing except the registration into the local project index (see `docket/ideas/dkt-projects-index.md`); no network, and no OpenSpec CLI needed.
- The standard result envelope; sources that cannot be read are listed in `unreadable`, and the rest is output as usual.
- Query git in batches. Measured on the Docket repository: looking up merges and tags one by one starts git 221 times and takes 6.0 s; batched, it takes 2 runs and 0.10 s, with the same results.
- Target: about 1 s for a repository the size of Docket's. Once everything is batched, an estimated 10 to 15 git runs and 0.6 to 0.8 s; not measured yet.

## Phase of the work

Docket has no notion of a phase yet; it has to be defined first. The definition taken from the prototype:

- Design: has a proposal, no plan yet.
- Ready: has a plan, no task started yet.
- In progress: some task has started, not all are done.
- Done: every task is done, merged or not.

## Merge and release of each piece of work

- The merge that brought each piece of work into the default branch: its commit, time and subject, matched by the `Change:` and `Record:` trailers.
- The first tag that contains that merge, and the tag's time.
- The archive folder of an archived change.

The prototype looked these up one at a time, starting git several times for each merge. A batched approach was tested and works: one `git log --parents` reads the trailers, one `git for-each-ref` lists the tags by creation time, and the commit graph is then walked in memory. On the Docket repository, the results for 110 merges matched the one-at-a-time lookup exactly.

## Spec content

- The requirements (names) of each main spec.
- What each spec delta changes: which requirements it adds, modifies, removes and renames; the scenario names of added and modified ones, the old and new names of renamed ones, and the reason for removed ones.

## Ideas, the order and issues

`dkt status` already decides, following the `ideas` spec, which items are still to do, but outputs only the next one.

- Each idea's title, needs and date, in three groups: queued (in the order and still to do), pool (not in the order and still to do), dropped.
- Each idea's and issue's position in the order. For an idea that a piece of work took, the position follows that work.
- Which pieces of work took each idea.
- Each issue's title, source, state (open, in progress), handlers (the changes and records whose source it is, and whether each is merged) and location.

## Open questions

- The names of the command and of its fields.
- Phase: which phase a record without tasks is in, and a change with a plan but no tasks; how a task counts as started: a commit carries its trailer, or one of its steps is ticked.
- Spec content: whether dkt parses the Markdown itself or goes through OpenSpec, given that the command must not need the OpenSpec CLI.
- Ideas: where an idea's date comes from, since Docket's idea file names carry no date.
- Issues: in the prototype an issue with a handler is in progress; what it is once every handler is merged is not settled.
