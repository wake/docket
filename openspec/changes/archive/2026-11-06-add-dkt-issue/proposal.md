---
title: "Keep issues as files with dkt issue"
kind: feature
source: "Rebuild plan, step 4: dkt issue"
specs:
  issues: Issues
---

# Proposal

## Why

The docket skill tells agents to keep work whose expected behavior is already settled, but that is put off, as an issue: `dkt issue <name>` writes it under `docket/issues/`, or, for a project that names its own issue tool, `dkt issue template` gives the sections to file it there. Neither command exists yet, so agents fall back to telling the user, and nothing checks an issue file.

## What Changes

- New command `dkt issue <name> --title "<one line>" --source "<where it was found>"`: writes `docket/issues/<date>-<name>.md` with the title and source in its front matter and the shipped issue template after it. Names, slugs, the folder and failed writes follow the rules records have: kebab-case names without a date, the slug never reused across the scanned sources, no write through a symbolic link, nothing left behind on failure.
- `dkt issue template` prints the template for an issue kept in another tool: the same four sections, with guidance that does not assume `dkt lint` or a `source` field.
- `dkt lint` checks the issue files the branch adds, and a dated name given to it now names a record or an issue: front matter (`issue-front-matter`, `title-too-long`), the sections `## Current`, `## Expected`, `## Unchanged` and `## Evidence` (`issue-section`, `placeholder`), and their length (`issue-too-long`, the 1,440-byte limit records have). Its result lists the issue slugs it linted.
- The workspace scan finds issue files the way it finds records, so a slug is never reused.
- The record code is first made to take the folder, the sections and the command name as parameters, without changing what records do, so that issues share it.

## Non-Goals

- An issue's state (pending, in progress, handled) and listing issues: that is `dkt status`, next.
- Checking that a change or record whose `source` names an issue points to an existing file.
- Writing to an outside issue tool: the agent files the issue there with the tool's own commands, as the skill says.
- Moving this repository's GitHub issue into a file.

## Capabilities

### New Capabilities
- `issues`: issue files under `docket/issues/`, how `dkt issue` writes one, what `dkt issue template` prints, and how `dkt lint` checks them.

### Modified Capabilities
- `records`: a dated name given to `dkt lint` names a record or an issue.
- `change-lint`: `dkt lint` also lints issues, and its result lists them.
- `workspace-scan`: a scan for issues reads the same sources as a scan for records.
- `cli-output`: `dkt issue` needs a git repository.

## Impact

- New `internal/issue` package and `internal/cli/cmd_issue.go`; `dkt --help` lists the command.
- `internal/record`, `internal/scan`, `internal/lint` and `internal/cli/cmd_record.go`: the record code takes the folder, sections and command name as parameters.
- New shipped files `shipped/issue/template.md` and `shipped/issue/external.md`; the second is prompt text the maintainer edits before it ships.
- `AGENTS.md`: `dkt issue` no longer listed as missing.
