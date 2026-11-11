---
title: "Cards for issues kept in another tool"
kind: feature
source: "docket/ideas/external-issue-cards.md"
---

# Proposal

## Why

A project whose `AGENTS.md` has a `## Issues` section files its issues in another tool, and those issues have no place in Docket: they cannot be listed in `docket/order.md` (an address there makes the whole order `order-invalid`), `dkt status` never names them as next, and `dkt snapshot` reports no state or handlers for them, so viewers show them only as a line of text in a work's `source`. Projects already need to order such issues.

## What Changes

- An issue kept in another tool is represented in Docket by a **card**: an issue file in `docket/issues/` whose front matter adds `external` (its address) and `external-id` (the number people use, such as `#92`), and whose body is empty. Its content stays in the other tool; nothing else is copied in.
- A card is an issue in every other respect: the order lists it by path, `dkt status` can name it as next, and its state and handlers come from the changes and records whose `source` is its path, exactly as for a local issue. Docket stays complete on its own: when the other tool cannot be reached, everything but the content kept there can still be read locally.
- `dkt issue <name> --external <address> --external-id <id>` writes a card, and refuses when an issue in the worktree already has that address.
- `dkt lint` checks a card's two fields, asks for no sections in a card, and reports two issues with the same address.
- An issue, local or card, can be dropped: `dropped: true` with the reason under `## Dropped`. `dkt status` skips it, and `dkt snapshot` reports it as `dropped`.
- An issue can record the writes to the other tool that could not be made, as `unsynced`; `dkt lint` checks its form and `dkt snapshot` reports it.
- `dkt snapshot` reports each issue's `external`, `external_id`, `dropped` and `unsynced`.
- The docket skill describes the flow: when a card is made (an issue Docket files in the tool, or an existing one there), that the issue is read in full from the tool before work starts and again before the pull request, what is written back to the tool and when, and what is noted locally when it cannot be. The sentences saying that an issue in another tool is named by its address in `--source` and is not listed in the order change to match, as does the `source` line of the proposal template.

## Non-Goals

- dkt does not reach the other tool: every read and write there is made by the coding agent with the tool `## Issues` names.
- No mirror of the other tool: labels, assignees, content and comments are not copied into the card, and no card is made until an issue enters Docket's flow or the user asks for one.
- No background sync: cards are brought in line with the other tool only when work on one starts, before its pull request, or when the user asks.
- A work whose `source` is an address rather than a card's path still names no issue.
- Viewer features specific to one tool (showing its comments, labels or status) belong to the viewer, not to this change.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `issues`: card fields and their checks, the `--external` and `--external-id` flags, `dropped` with `## Dropped`, `unsynced`, and duplicate addresses.
- `ideas`: the order lists cards like any issue, a dropped issue is not next, and `dkt status` reports a `dropped` that is not `true` or `false`.
- `project-snapshot`: issues carry `external`, `external_id`, `dropped` and `unsynced`, and a dropped issue's state is `dropped`.

## Impact

- Code: `internal/cli/cmd_issue.go`, `internal/lint/record.go` (the issue rules), `internal/idea` (issue `dropped` in the order and next), `internal/snapshot` (issue entries and state).
- Prompts: `shipped/skill/SKILL.md` and `shipped/schema/templates/proposal.md`, with their installed copies and new hashes in `internal/setup/history.go`.
- Output: `dkt snapshot --json` gains issue fields and the `dropped` state; viewers that read it can show cards with links to the other tool.
