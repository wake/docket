# Design

## Context

`domain.Load` reads `docket/domains.md` and the capabilities of the project: each main spec, and each delta of an active change. `dkt domains` works out the unclassified capabilities from it, those no node holds, in `internal/cli/cmd_domains.go`; `dkt lint`, when no change is named, reports the tree's own findings (`domain-format`, `domain-duplicate`, `domain-unknown`) from the same `Load`. `dkt snapshot` reads the tree of the default branch through its own path. In the docket skill, the "Domain tree" section and Full review step 2 apply only once `docket/domains.md` exists; the rules for editing the tree are in `domain-tree.md`, in the skill's folder, which the skill points to only for a project that has the file. See proposal.md - Why for the motivation.

## Goals / Non-Goals

**Goals:**
- One definition of an unclassified capability, shared by `dkt domains` and `dkt lint`.
- A suggestion that reaches the agent in a project with no tree, at the step where a capability is added.

**Non-Goals:**
- Changing how `dkt snapshot` computes its unclassified list.

## Decisions

### Unlisted capabilities come from `domain.Load`

`domain.Loaded` gains the sorted paths of the capabilities no node holds. `dkt domains` builds `result.unclassified` from it, in a structural commit before the rule, and `dkt lint` reports `domain-unlisted` for the same paths, so the two cannot disagree.

Lint reports them only when the tree lists at least one item, `Load` did not stop on a format problem or duplicate, and nothing it reads was unreadable: in those cases what is unlisted is not known, and the other findings already say why. `Unlisted` itself still returns what was read, so `dkt domains` keeps listing it as today; lint applies the condition. A `domain-unknown` finding does not hold it back, as it concerns a line of the tree, not a capability.

With no change named, lint already loads the tree and reports every unlisted capability. With changes named, which is how archiving runs lint, it loads the tree too and reports the unlisted capabilities with a delta in a named change, with the finding's change set to that change; a capability outside the named changes is left to the lint that names none, so a change is not stopped by an older omission it does not touch. Every capability with a delta in a named change is checked, including one the change modifies and one in a change of another schema: with the tree in place it should be listed either way, and `dkt domains` counts it the same way. The exception is a capability the change retires, whose delta removes every requirement of its main spec and adds, modifies and renames none: the skill has the change delete its line from the tree, while its main spec and the delta stay until archiving, so reporting it would stop the archive the skill asks for, and keeping the line would leave a `domain-unknown` once archived. Lint reads the linted changes' deltas with `specdoc.ParseDelta` and the main spec's requirement names with `specdoc.Requirements`, comparing names with runs of space folded, as OpenSpec matches them; `domain.Load` is left alone, so `dkt domains` still lists such a capability as unclassified. A delta that removes only some requirements leaves a live capability, so it is reported. With no change named, a retired capability is not reported at all. The rule does not depend on the change's schema: a capability every requirement of which is removed is gone after archiving under any schema, so keeping its line would leave `domain-unknown`. A capability with deltas in two named changes is reported once for each, so each change's finding stands on its own.

The finding names `docket/domains.md`, with no line, and its message is fixed: `<capability> is not listed in docket/domains.md; list it under a node there, then run dkt domains to check the tree`.

Alternatives considered:
- Reporting only capabilities a named change adds. Closer to the change's step, but lint cannot tell an added capability from a modified one without the main specs of the default branch, and an unlisted modified capability is a gap all the same.
- Leaving lint with changes named untouched. Archiving would then never see the finding.

### The record of a declined tree is a tree with no items

When the user declines a tree, the agent has no memory across sessions, so the decision must sit in the repository. The smallest record that needs no new setting is the tree file itself: a `docket/domains.md` with no items, holding only a comment line such as `# This project keeps no domain tree.`. `dkt domains` already reads such a file as found with no nodes. Lint gives it no `domain-unlisted`, and the skill suggests a tree only when the file does not exist, so the agent does not ask again. Deleting the file, or adding a first node, brings the tree back.

Alternatives considered:
- A sentence in `AGENTS.md`. It reaches every agent but no tool, so lint would still report every capability in a project that later adds an empty file for another reason, and the skill would have to read prose to decide.
- A new key in a configuration file. The proposal rules out a new setting for this.

### Where the suggestion lives

`domain-tree.md` is read only by an agent the skill sends there, and the skill sends it there only when `docket/domains.md` exists. So the condition for suggesting a tree goes in `SKILL.md`, which every agent working in the repository loads: Full review step 2, where the change's capabilities are named. It says when, and sends the agent to `domain-tree.md`, whose new section says how: draft a first tree from `dkt domains --json`, write it in the same change once the user agrees, and when the user declines, offer to record it with a tree with no items. The "Domain tree" section of `SKILL.md` also points to step 2, so an agent looking for the tree's rules finds the suggestion.

The step 2 bullet that lists a new capability in the tree now applies when the file lists any item, so a declined tree asks nothing of the agent.

### Ten main specs

The skill suggests a tree once the project already has ten or more main specs, counted before the change, when the change adds a capability; a project with nine that adds its tenth is not asked until the next one. The number is a judgment, not a measurement: the one data point is a project that asked for its tree by hand at 27 capabilities and found it late, as every capability then had to be placed at once. Ten is well before that, keeps the first tree to about ten lines to agree on, and is a number the agent can check with `ls openspec/specs` or `dkt domains --json`. It counts main specs, not changes in flight, since main specs are what pile up.

## Risks / Trade-offs

- [A project that already has a tree and left capabilities out is stopped at its next archive] → That is the intent of the idea; the message says which file to edit and which command shows the tree, and the finding names each capability once.
- [A tree with no items written for another reason silences the rule] → The comment line written with it says what it records, and `dkt domains` still lists every capability as unclassified.
- [The count of ten is a judgment] → It is written once in the skill and can be changed in a later release.
