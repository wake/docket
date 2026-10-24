# Spec Delta

## Purpose

Defines how Docket finds active changes that touch the same requirement, so
that parallel work in separate worktrees learns of an overlap while it can
still be avoided, long before archiving would overwrite one change with the
other.

## ADDED Requirements

### Requirement: Overlapping changes
`dkt conflicts` SHALL report a requirement as overlapping when two or more distinct active changes touch the same capability path and requirement name. A change touches a requirement when its delta modifies it, removes it, renames it from or to that name, or adds a requirement with that name.

#### Scenario: Both modify one requirement
- **WHEN** `add-x` and `add-y` each have a MODIFIED block for `booking` / "Cancel booking"
- **THEN** the command reports one overlap on `booking` / "Cancel booking" naming both changes

#### Scenario: Rename against modify
- **WHEN** `add-x` renames `booking` / "Cancel booking" to "Cancel a booking" and `add-y` modifies "Cancel booking"
- **THEN** the command reports an overlap naming both changes and their operations

#### Scenario: Same capability, different requirements
- **WHEN** `add-x` modifies `booking` / "Cancel booking" and `add-y` adds `booking` / "Refund points"
- **THEN** the command reports no overlap

#### Scenario: Both add the same name
- **WHEN** `add-x` and `add-y` each add `booking` / "Waitlist cap"
- **THEN** the command reports an overlap naming both changes

### Requirement: Where changes are found
`dkt conflicts` SHALL consider every active change the scan defined by `workspace-scan` finds, from all its sources, and SHALL NOT consider archived changes. A change SHALL be compared with other changes only, never with its own copies in other sources.

#### Scenario: Overlap with another worktree
- **WHEN** worktree A holds `add-x`, worktree B holds `add-y` uncommitted, both modify the same requirement, and the command runs in worktree A
- **THEN** the overlap is reported with worktree B as the source of `add-y`

#### Scenario: Archived change left on a branch
- **WHEN** `add-x` is archived on the default branch tip, an unmerged branch still holds `openspec/changes/add-x/`, and `add-y` modifies a requirement that copy modifies
- **THEN** the command reports no overlap

### Requirement: One change in focus
With a slug, `dkt conflicts` SHALL report only the overlaps that involve that change. The slug SHALL be an active change.

#### Scenario: Unrelated overlap hidden
- **WHEN** `add-x` and `add-y` overlap, `add-z` overlaps nothing, and a user runs `dkt conflicts add-z`
- **THEN** the command reports no overlap and exits with 0

#### Scenario: Slug not active
- **WHEN** a user runs `dkt conflicts add-q` and no active change is named `add-q`
- **THEN** the command reports that `add-q` is not active and exits with 1

### Requirement: Overlap report
Each overlap SHALL name the capability and the requirement, and for each change involved, its slug, every operation it applies to that requirement, and every source it was found in with that source's commit. The command SHALL exit with 1 when it reports any overlap and with 0 when it reports none, except as `cli-output` requires for unreadable sources. It SHALL NOT require the OpenSpec CLI.

#### Scenario: Sources listed
- **WHEN** `add-y` exists both in worktree B and on `refs/remotes/origin/add-y`
- **THEN** its entry in the overlap lists both sources with their commits

#### Scenario: No overlaps
- **WHEN** no two active changes touch the same requirement
- **THEN** the command reports no overlap and exits with 0

#### Scenario: Unreadable worktree
- **WHEN** one listed worktree cannot be read and two other changes overlap
- **THEN** the overlap is reported, the worktree is listed as unreadable, and the command exits with 3
