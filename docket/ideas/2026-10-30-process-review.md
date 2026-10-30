---
title: Review the workflow against development data
---

Recorded 2026-10-30, while designing `add-doctor`. The maintainer asked
where a review of the workflow from real development data belongs: whether
the steps pay off, and where they could be lighter or faster.

## Problem

The workflow file says what each path does, but nothing shows whether it
is worth it: how much review happens per change, what it finds, what gets
fixed, and how long the work takes. The doctor answers whether the setup
is right now; this answers whether the process pays off over many pieces
of work, and its output is a proposed edit to the workflow file.

## Decided

- Not part of the doctor (`docket-doctor`): different inputs (history across changes,
  not the current state), cadence and output.
- Two layers, split by the rule that work involving the agent never goes in
  the CLI:
  - collecting the numbers is mechanical, for `dkt`;
  - reading them and proposing changes is a command the user invokes,
    which shows the whole revised workflow file and writes it only when the
    user agrees, as setup does.
- It reads only the repository: archived changes, their `evidence.md`,
  records and git history. Token use, model quota and session time live
  outside the repository and stay out.

## Depends on

- `2026-10-28-evidence-findings-and-sizes.md`: sizes and review findings
  per change in `evidence.md`.
- `2026-10-29-records.md`: records, so that light-review work is counted.

## Open questions

- Which numbers the review reads first, and over how many changes.
- Whether `dkt status` or a new command prints the numbers across changes.
