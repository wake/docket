# Conflict Detection Specification

## Purpose
Defines how Docket finds active changes that touch the same requirement, so that work in parallel worktrees and branches learns of an overlap while it can still be avoided, long before archiving would overwrite one change with the other.

## Requirements

### Requirement: Overlapping changes
`dkt conflicts` SHALL report a requirement as overlapping when two or more distinct active changes touch the same capability path and requirement name. A change touches a requirement when one of its delta specs adds, modifies or removes a requirement with that name, or renames a requirement from or to that name.

#### Scenario: Both modify one requirement
- **WHEN** `add-x` and `add-y` each have a MODIFIED block for `booking` / "Cancel booking"
- **THEN** the command reports one overlap on `booking` / "Cancel booking" naming both changes

#### Scenario: Rename against modify
- **WHEN** `add-x` renames `booking` / "Cancel booking" to "Cancel a booking" and `add-y` modifies "Cancel booking"
- **THEN** the command reports an overlap on "Cancel booking" naming `add-x` with RENAMED-FROM and `add-y` with MODIFIED

#### Scenario: Same capability, different requirements
- **WHEN** `add-x` modifies `booking` / "Cancel booking" and `add-y` adds `booking` / "Refund points"
- **THEN** the command reports no overlap

#### Scenario: Both add the same name
- **WHEN** `add-x` and `add-y` each add `booking` / "Waitlist cap"
- **THEN** the command reports an overlap naming both changes

### Requirement: Requirement names compared
Requirement names that differ only in case or in runs of whitespace SHALL count as one name, since OpenSpec takes such spellings for one requirement. The overlap SHALL be reported under the spelling used by the first of its changes by slug. Capability paths SHALL match exactly.

#### Scenario: Names differ in case
- **WHEN** `add-x` modifies `booking` / "Cancel booking" and `add-y` modifies `booking` / "Cancel  Booking"
- **THEN** the command reports one overlap on `booking` / "Cancel booking" naming both changes

### Requirement: Where changes are found
`dkt conflicts` SHALL consider every active change that the scan defined by `workspace-scan` finds, in every source it is found in, and SHALL NOT consider a slug that has an archived use in any source. A change SHALL be compared with other changes only, never with its own copies, and it touches a requirement when any of its copies does.

#### Scenario: Overlap with another worktree
- **WHEN** worktree A holds `add-x`, worktree B holds `add-y` uncommitted, both modify the same requirement, and the command runs in worktree A
- **THEN** the overlap is reported with worktree B as the source of `add-y`

#### Scenario: Archived change left on a branch
- **WHEN** `add-x` is archived on the default branch, an unmerged branch still holds `openspec/changes/add-x/`, and `add-y` modifies a requirement that copy modifies
- **THEN** the command reports no overlap

#### Scenario: Copies that differ
- **WHEN** `add-x` modifies "Cancel booking" on its pushed branch, its worktree now modifies "Refund points" instead, and `add-y` modifies "Refund points"
- **THEN** the command reports an overlap on "Refund points" whose entry for `add-x` names only the worktree as its source

#### Scenario: Own copies not compared
- **WHEN** `add-x` is in worktree A and on `refs/remotes/origin/add-x`, both copies modify "Cancel booking", and no other change touches it
- **THEN** the command reports no overlap

### Requirement: Delta specs read as OpenSpec reads them
`dkt conflicts` SHALL read the delta specs of each copy as OpenSpec parses them, whether or not the change's other files are complete. A copy that OpenSpec cannot parse, or whose files cannot be read, SHALL be listed as unreadable with its source, its folder and the reason. When fewer than two active changes exist, the command SHALL report no overlap without running OpenSpec.

#### Scenario: Change still being proposed
- **WHEN** `add-y`'s `proposal.md` holds only the front matter `dkt new` writes, and its delta modifies `booking` / "Cancel booking", as `add-x`'s does
- **THEN** the command reports the overlap

#### Scenario: Copy OpenSpec cannot parse
- **WHEN** OpenSpec fails on the copy of `add-y` in worktree B, and `add-x` and `add-z` modify the same requirement
- **THEN** the overlap of `add-x` and `add-z` is reported, the copy of `add-y` in worktree B is listed as unreadable with OpenSpec's message, and the command exits with 3

#### Scenario: Linked delta spec in another worktree
- **WHEN** worktree B's `openspec/changes/add-y/specs/booking/spec.md` is a symbolic link
- **THEN** the copy of `add-y` in worktree B is listed as unreadable, naming the link, and what the link points to is not read

#### Scenario: One active change
- **WHEN** `add-x` is the only active change and no OpenSpec CLI can be found
- **THEN** the command reports no overlap and exits with 0

### Requirement: One change in focus
With a slug, `dkt conflicts` SHALL report only the overlaps that involve that change. When the slug is not an active change, it SHALL report `change-missing`, naming the folder where the slug is archived when it is, and exit with 1; while a source could not be read and the slug is archived nowhere, it SHALL report only the unreadable source.

#### Scenario: Unrelated overlap hidden
- **WHEN** `add-x` and `add-y` overlap, `add-z` overlaps nothing, and a user runs `dkt conflicts add-z`
- **THEN** the command reports no overlap and exits with 0

#### Scenario: Slug not active
- **WHEN** a user runs `dkt conflicts add-q`, `add-q` is archived as `openspec/changes/archive/2026-11-01-add-q`, and no source holds it active
- **THEN** the findings hold `change-missing` naming that archive folder, and the command exits with 1

#### Scenario: Slug possibly in an unreadable worktree
- **WHEN** a user runs `dkt conflicts add-q`, no readable source holds `add-q`, and worktree B cannot be read
- **THEN** the findings do not hold `change-missing`, worktree B is listed as unreadable, and the command exits with 3

### Requirement: Overlap report
Each overlap SHALL name the capability and the requirement, and for each change involved: its slug, every operation it applies to that requirement (ADDED, MODIFIED, REMOVED, RENAMED-FROM, RENAMED-TO, in that order), and every source whose copy touches it. The JSON result SHALL hold `changes`, the sorted slugs compared, and `overlaps`, sorted by capability, requirement and slug. Each overlap SHALL also be a `requirement-overlap` finding.

#### Scenario: Sources listed
- **WHEN** `add-y` is both in worktree B and on `refs/remotes/origin/add-y`, and both copies modify the requirement `add-x` modifies
- **THEN** the entry for `add-y` lists worktree B's path and `refs/remotes/origin/add-y` as its sources

#### Scenario: Overlap found
- **WHEN** `add-x` and `add-y` both modify `booking` / "Cancel booking" and a user runs `dkt conflicts --json`
- **THEN** `result.overlaps` holds that overlap, `findings` hold one `requirement-overlap`, and the command exits with 1

#### Scenario: No overlaps
- **WHEN** no two active changes touch the same requirement
- **THEN** `result.overlaps` is an empty array, `findings` is empty, and the command exits with 0

#### Scenario: Unreadable worktree
- **WHEN** one listed worktree cannot be read and two other changes overlap
- **THEN** the overlap is reported, the worktree is listed as unreadable, and the command exits with 3
