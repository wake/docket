# Implementation Plan

**Goal:** `dkt lint` reports `domain-unlisted` for each capability the domain tree leaves out, and the docket skill tells the agent when to suggest starting a tree and how to record that the user declined one.

**Architecture:** `domain.Loaded` gains the sorted list of capabilities no node holds; `dkt domains` builds `result.unclassified` from it, and `dkt lint` reports the same list, all of it with no change named and the part with a delta in a named change otherwise. The skill's Full review step 2 says when to suggest a tree and sends the agent to `domain-tree.md`, whose new section says how.

**Tech Stack:** Go, standard library only; Markdown prompt text.

**Spec:** `openspec/changes/domain-tree-upkeep/proposal.md`, `design.md`, `specs/change-lint/spec.md`, `specs/domain-tree/spec.md`.

## Global Constraints

- `dkt domains` output is unchanged; `dkt snapshot` is not touched.
- Structural and behavior edits go in separate commits. No new dependency.
- The skill text follows the Chinese drafts kept outside the repository; `shipped/` and its installed copies, in `.claude/skills/docket/` and `.agents/skills/docket/`, say the same.

## Review Focus

- A tree with no items, a format problem, a duplicate or an unreadable source gives no `domain-unlisted`; a `domain-unknown` finding does not hold it back.
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

---

### Task 1: The domain-unlisted rule

**Scenarios:** change-lint#main-spec-left-out-of-the-tree, change-lint#new-capability-in-a-named-change, change-lint#capability-outside-the-named-change, change-lint#tree-with-no-items, change-lint#tree-with-a-format-problem, domain-tree#declined-tree, domain-tree#spec-outside-the-tree

**Review:** judgment

**Files:**
- Modify: `internal/domain/load.go`
- Modify: `internal/domain/load_test.go`
- Modify: `internal/cli/cmd_domains.go`
- Modify: `internal/cli/cmd_domains_test.go`
- Modify: `internal/lint/lint.go`
- Modify: `internal/lint/lint_test.go`

**Interfaces:**
- `(domain.Loaded).Unlisted() []string`: the paths of the capabilities no node holds, sorted.
- Rule `domain-unlisted`, file `docket/domains.md`, no line, message exactly `<capability> is not listed in docket/domains.md; list it under a node there, then run dkt domains to check the tree`.

- [ ] `Unlisted` returns the capabilities no node holds, sorted, for a tree with nodes, for a tree with no items, and for no tree; tested in `internal/domain`
- [ ] `dkt domains` gives the same `result.unclassified` as before, its existing tests passing unchanged, and the declined-tree case reports found, no nodes and the capability unclassified, with no finding; tested in `internal/cli/cmd_domains_test.go`
- [ ] `dkt lint` naming no change reports `domain-unlisted` with the fixed message for an unlisted main spec and for an unlisted capability with only a delta in an active change, and none for a listed one; tested in `internal/lint/lint_test.go`
- [ ] `dkt lint add-refund` reports it for `refund`, with the change set to `add-refund`, and `dkt lint add-x` reports none for an unlisted main spec outside `add-x`; tested in `internal/lint/lint_test.go`
- [ ] No `domain-unlisted` for a tree with no items, a tree with a format problem, a tree with a duplicate, or an unreadable source, with and without a change named; a `domain-unknown` line does not hold it back; no tree gives no domain finding; tested in `internal/lint/lint_test.go`

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

- [ ] Full review step 2 lists a new capability in the tree when `docket/domains.md` lists any item, and, when the project has no `docket/domains.md` and ten or more main specs and the change adds a capability, sends the agent to `domain-tree.md` to suggest starting the tree
- [ ] The Domain tree section of `SKILL.md` points to step 2 for when to suggest a tree
- [ ] `domain-tree.md` says that lint reports `domain-unlisted` once the file lists items, and has a section on starting the tree: a first tree drafted from `dkt domains --json` and shown as a suggestion, written in the same change only once the user agrees, and, when the user declines and agrees to record it, a file with only a comment line and no items, which stops the suggestion and the rule
- [ ] The English text says what the Chinese drafts say
- [ ] `dkt init --force`, run from this branch's build, leaves both installed skill folders byte-for-byte equal to `shipped/skill/`, and `TestEarlierHoldsShippedFiles` passes with the new hashes recorded for v0.29.0
