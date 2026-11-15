# Implementation Plan

**Goal:** `dkt lint` reports `domain-unlisted` for each capability the domain tree leaves out, and the docket skill tells the agent when to suggest starting a tree and how to record that the user declined one.

**Architecture:** `domain.Loaded` gains the sorted list of capabilities no node holds; `dkt domains` builds `result.unclassified` from it, and `dkt lint` reports the same list, all of it with no change named and the part with a delta in a named change otherwise, leaving out a capability a linted change retires. The skill's Full review step 2 says when to suggest a tree and sends the agent to `domain-tree.md`, whose new section says how.

**Tech Stack:** Go, standard library only; Markdown prompt text.

**Spec:** `openspec/changes/domain-tree-upkeep/proposal.md`, `design.md`, `specs/change-lint/spec.md`, `specs/domain-tree/spec.md`.

## Global Constraints

- `dkt domains` output is unchanged; `dkt snapshot` is not touched.
- Structural and behavior edits go in separate commits. No new dependency.
- The skill text follows the Chinese drafts kept outside the repository; `shipped/` and its installed copies, in `.claude/skills/docket/` and `.agents/skills/docket/`, say the same.

## Review Focus

- A tree with no items, a format problem, a duplicate or an unreadable source gives no `domain-unlisted`; a `domain-unknown` finding does not hold it back, and `dkt domains` still lists what it could read.
- With changes named, a capability outside them is not reported, even when it is unlisted.
- An agent in a project without `docket/domains.md` reaches the suggestion from `SKILL.md` alone.

## Reuse

| Need | Existing (path:symbol) | Decision | Reason |
|---|---|---|---|
| The capabilities no node holds | internal/domain/load.go:Loaded | extend | `Load` already reads the tree and every capability; the list belongs next to them |
| `result.unclassified` of `dkt domains` | internal/cli/cmd_domains.go:runDomains | extend | Builds the list from `Loaded` instead of its own walk |
| The domain findings of lint | internal/lint/lint.go:Run | extend | Already loads the tree and reports its findings when no change is named |
| The tree's rules for agents | shipped/skill/domain-tree.md | extend | Gains the section on starting a tree |
| The docket skill | shipped/skill/SKILL.md | extend | Step 2 and the Domain tree section point to it |
| Hashes of earlier shipped files | internal/setup/history.go:Earlier | extend | `dkt init` must recognize the two files as shipped before this change |

## Revision 1 (2026-11-16): a capability a change retires

The pre-merge review on the other model found that a change removing a capability was stopped: the skill has it delete the capability's line from the tree, but until archiving the main spec and the removing delta remain, so `domain-unlisted` reported it, and keeping the line instead leaves `domain-unknown` after archiving. Lint now leaves out a capability whose delta in a linted change only removes requirements; the delta adds the requirement Capability retired by a change, and Task 1 gains the acceptance item for it, replacing the case that expected the removing change to be reported. Requested by the reviewer, in the pre-merge review.

---

### Task 1: The domain-unlisted rule

**Scenarios:** change-lint#main-spec-left-out-of-the-tree, change-lint#new-capability-in-a-named-change, change-lint#capability-outside-the-named-change, change-lint#tree-with-no-items, change-lint#tree-with-a-format-problem, change-lint#capability-being-removed, change-lint#removal-with-an-addition, domain-tree#declined-tree, domain-tree#spec-outside-the-tree

**Review:** judgment

**Files:**
- Modify: `internal/domain/load.go`
- Modify: `internal/domain/load_test.go`
- Modify: `internal/cli/cmd_domains.go`
- Modify: `internal/cli/cmd_domains_test.go`
- Modify: `internal/lint/lint.go`
- Modify: `internal/lint/lint_test.go`
- Modify: `internal/snapshot/snapshot_test.go`

**Interfaces:**
- `(domain.Loaded).Unlisted() []string`: the paths of the capabilities no node holds, sorted, including those read when another source was unreadable; lint, not `Unlisted`, holds the rule back in that case.
- Rule `domain-unlisted`, file `docket/domains.md`, no line, message exactly `<capability> is not listed in docket/domains.md; list it under a node there, then run dkt domains to check the tree`.

- [x] `Unlisted` returns the capabilities no node holds, sorted, for a tree with nodes, for a tree with no items, and for no tree, and still returns the readable unlisted capabilities when an active change's `specs` folder is a symbolic link; tested in `internal/domain`
- [x] `dkt domains` gives the same `result.unclassified` as before, its existing tests passing unchanged, and the declined-tree case reports found, no nodes and the capability unclassified, with no finding; tested in `internal/cli/cmd_domains_test.go`
- [x] `dkt snapshot` reads a tree holding only a comment line as found, with no nodes and every capability unclassified, as before; tested in `internal/snapshot/snapshot_test.go`
- [x] `dkt lint` naming no change reports `domain-unlisted` with the fixed message for an unlisted main spec and for an unlisted capability with only a delta in an active change, and none for a listed one; with no `openspec/changes` folder it still reports the unlisted main specs; tested in `internal/lint/lint_test.go`
- [x] `dkt lint add-refund` reports it for `refund`, with the change set to `add-refund`; `dkt lint add-x` reports none for an unlisted main spec outside `add-x`; naming two changes that both hold a delta for an unlisted capability reports it once for each, with each change set; a change of another schema is reported too; `dkt lint missing` reports `change-missing` and no `domain-unlisted`; tested in `internal/lint/lint_test.go`
- [x] No `domain-unlisted` for a tree with no items, a tree with a format problem, a tree with a duplicate, an active change's `specs` folder that is a symbolic link, or an `openspec/changes` folder that is a symbolic link, with and without a change named; no tree gives no domain finding; tested in `internal/lint/lint_test.go`
- [x] A change whose delta for `refund` only removes requirements gives no `domain-unlisted`, named or not; a delta that also adds a requirement, or holds none, is reported; with a removing change and a modifying change both named, only the modifying one is; tested in `internal/lint/lint_test.go`
- [x] With a tree that lists `gone`, which does not exist, and leaves out `refund`, which has a main spec and a delta in `add-refund`: `dkt lint` reports `domain-unknown` for `gone` and `domain-unlisted` for `refund`, and `dkt lint add-refund` reports only `domain-unlisted` for `refund`; tested in `internal/lint/lint_test.go`

### Task 2: The skill suggests starting a tree

**Scenarios:** none

**Review:** judgment

**Files:**
- Modify: `shipped/skill/SKILL.md`
- Modify: `shipped/skill/domain-tree.md`
- Modify: `.claude/skills/docket/SKILL.md`
- Modify: `.claude/skills/docket/domain-tree.md`
- Modify: `.agents/skills/docket/SKILL.md`
- Modify: `.agents/skills/docket/domain-tree.md`
- Modify: `internal/setup/history.go`

- [x] Full review step 2 keeps the tree up to date (listing a new capability, and updating or deleting the line of one whose path changes or that is removed) only when `docket/domains.md` lists any item, so a tree with no items asks for nothing; and, when the project has no `docket/domains.md` and already has ten or more main specs, counted before the change, and the change adds a capability, it sends the agent to `domain-tree.md` to suggest starting the tree
- [x] The Domain tree section of `SKILL.md` points to step 2 for when to suggest a tree
- [x] `domain-tree.md` says that lint reports `domain-unlisted` once the file lists items, and has a section on starting the tree: a first tree drafted from `dkt domains --json` and shown as a suggestion, written in the same change only once the user agrees, and, when the user declines and agrees to record it, a file with only a comment line and no items, which stops the suggestion and the rule
- [x] The English text says what the Chinese drafts say
- [x] `dkt init --force`, run from this branch's build, leaves both installed skill folders byte-for-byte equal to `shipped/skill/`, and `TestEarlierHoldsShippedFiles` passes with the new hashes recorded for v0.29.0
