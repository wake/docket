# Implementation Plan

**Goal:** The docket skill's sections for issues in another tool and the domain tree move into files next to `SKILL.md`, and `dkt init` and `dkt inspect` cover every file of the skill folder.

**Architecture:** `targetsOf` in `internal/setup/setup.go` walks `shipped/skill/` as it walks `shipped/schema/`, giving each file a target in both skill folders; the existing per-file rules decide each one. The two sections' text moves word for word into `shipped/skill/issues-in-another-tool.md` and `shipped/skill/domain-tree.md`, and each heading in `SKILL.md` keeps one pointer line.

**Tech Stack:** Go; the shipped skill, installed by `dkt init`.

**Spec:** `proposal.md`, `specs/project-setup/spec.md`, `specs/setup-inspection/spec.md` and `design.md` in this change folder.

## Global Constraints

- The new files are `issues-in-another-tool.md` and `domain-tree.md`. Each starts with the section's heading as a level-one heading (`# Issues in another tool`, `# Domain tree`), followed by the section's text from `SKILL.md`, unchanged.
- In `SKILL.md`, `## Issues in another tool` holds only: When the project's `AGENTS.md` has a `## Issues` section, read `issues-in-another-tool.md` in this skill's folder before working with any issue.
- In `SKILL.md`, `## Domain tree` holds only: When the project has `docket/domains.md`, read `domain-tree.md` in this skill's folder before editing it.
- Skill targets are listed per skill folder, `.claude/skills/docket/` then `.agents/skills/docket/`, each folder's files in path order, so `SKILL.md` comes first.
- New hashes are recorded for v0.28.0.

## Review Focus

- A user who edited `SKILL.md`: it is kept, and the new files are still created.
- A file of the user's own in a skill folder: neither listed nor touched.
- The reload next step: printed when a `SKILL.md` is created, not when only the other skill files are.
- After `dkt init`, every skill file is byte-for-byte the same in both folders.

## Reuse

| Need | Existing (path:symbol) | Decision | Reason |
|---|---|---|---|
| Skill targets for every file of the shipped skill folder | `internal/setup/setup.go:targetsOf` | extend | It already walks `shipped/schema/` the same way |
| A test shipped tree with one file replaced or added | `internal/cli/cmd_inspect_test.go:withShippedWorkflow` | extend | It copies the shipped tree and replaces the workflow; any path is the same work |
| Hashes of the new skill files and the new `SKILL.md` | `internal/setup/history.go:Earlier` | extend | Every shipped target's hashes are kept there |

---

### Task 1: Init and inspect cover every file of the skill folder

**Scenarios:** project-setup#fresh-repository-gets-the-files, project-setup#skill-folder-from-an-earlier-release, setup-inspection#skill-folder-from-an-earlier-release-inspected, setup-inspection#before-init, setup-inspection#edited-skill, setup-inspection#file-from-an-earlier-release

**Review:** checks

**Files:**
- Modify: `internal/setup/setup.go`
- Modify: `internal/setup/setup_test.go`
- Modify: `internal/cli/cmd_init_test.go`
- Modify: `internal/cli/cmd_inspect_test.go`

- [x] With a shipped skill folder of `SKILL.md` and one other file, `Decide` lists both files under `.claude/skills/docket/` and then under `.agents/skills/docket/`, `SKILL.md` first in each, all of kind `skill`.
- [x] Fresh `dkt init` with that shipped tree writes both files to both folders, each byte-for-byte the same in both.
- [x] Both folders holding only an earlier release's `SKILL.md`: `dkt init` lists each `SKILL.md` as `updated` and the other file as `created` in both folders and exits with 0; `dkt inspect` reports `skill-outdated` for each `SKILL.md` and `skill-missing` for the other file in both folders and exits with 1.
- [x] An edited `SKILL.md` is listed as `kept` with `skill-differs`, and the other file is still `created`.
- [x] The other file gets each rule `SKILL.md` gets, tested on it: edited, it is `kept` with `skill-differs`, and `--force` makes it `overwritten`; equal to an earlier release's hash, it is `updated` by `dkt init` and `outdated` with `skill-outdated` in `dkt inspect`; a symbolic link in its place makes `dkt init` write nothing and report `symlink-refused` naming it, and `dkt inspect` report `symlink-refused` without reading through it.
- [x] A file of the user's own in a skill folder is not listed and stays unchanged.
- [x] The reload step is printed when a `SKILL.md` is created, and not when only the other file is created.

### Task 2: Move the two sections into files of their own

**Scenarios:** none

**Review:** checks

**Files:**
- Modify: `shipped/skill/SKILL.md`
- Create: `shipped/skill/issues-in-another-tool.md`
- Create: `shipped/skill/domain-tree.md`
- Modify: `.claude/skills/docket/SKILL.md`
- Modify: `.agents/skills/docket/SKILL.md`
- Create: `.claude/skills/docket/issues-in-another-tool.md`
- Create: `.agents/skills/docket/issues-in-another-tool.md`
- Create: `.claude/skills/docket/domain-tree.md`
- Create: `.agents/skills/docket/domain-tree.md`
- Modify: `internal/setup/history.go`

- [x] The body of each new file, after its heading, equals the section's text in `SKILL.md` before this task, byte for byte.
- [x] `SKILL.md` keeps both headings, each with exactly the pointer line in Global Constraints, and is otherwise unchanged.
- [x] The copies in both skill folders equal `shipped/skill/`, installed by this worktree's own build with `init --force`.
- [x] `internal/setup/history.go` holds the new `SKILL.md` hash and both new files' hashes for both folders under v0.28.0, and `go test ./...` passes, `TestEarlierHoldsShippedFiles` included.
- [x] `dkt inspect` from this worktree's build reports every skill file as `same`.
