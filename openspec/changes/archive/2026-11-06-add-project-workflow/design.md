# Design

## Context

`dkt init` installs Docket-owned files (the schema, the operations section, the docket skill) and replaces them on upgrade: equal to an earlier release's file means `updated`, anything else is `kept` unless `--force`. `internal/setup.Decide` makes that call per target; `dkt inspect` reads the same decisions. The skill currently ends with "Default steps", which a project cannot edit without `dkt init` reporting `skill-differs`. See proposal.md for why the steps move.

## Goals / Non-Goals

**Goals:**
- A boundary that says, for any step, whether it lives in the skill or in `docket/workflow.md`.
- A check that tells a project its workflow file is behind the shipped template or lacks a section the skill looks up, without reading what the sections say.

**Non-Goals:**
- Merging template versions; that is docket-doctor's job.

## Decisions

**What stays in the skill.** A step stays in the skill when dkt depends on it or would break if it changed: the triage, the order of the steps on each path, `dkt new`, where the design documents and the plan go, `openspec instructions`, trailers, `dkt record`, `dkt sync` and `dkt lint`, no squash, archiving with `/opsx:archive`, `dkt conflicts`. Everything else is a section of `docket/workflow.md`. The alternative, moving each path's whole workflow into the project file, would let a project drop a step dkt relies on, and every template change would touch every project's copy.

**Headings are the keys.** The skill names a section by quoting its heading with the file name: `"Plan review" in docket/workflow.md`. Angle brackets were rejected because `<...>` already means a placeholder in the skill and Markdown renderers drop it. The required set is the shipped template's level-3 headings (the level-2 headings only group them); a file may use any heading level, since only the text is a key. "Reviewers" is required although only other sections refer to it.

**Template version is an integer, not a hash.** The file is meant to be edited, so its bytes cannot say which template it came from, unlike the files `setup.Earlier` tracks. `template: N` in the front matter is bumped only when the template changes in substance. A lower version is `outdated`; a missing, unreadable or higher one is `template-unknown`, since this release cannot say what changed. Sections are checked only at the shipped version: at another version, a missing heading may be one the project's template never had, which docket-doctor sorts out while merging.

**Pinning the version.** A test in the root package keeps a map from each template version to the SHA-256 of the template as shipped at that version. Editing the template without raising `template` and adding the new hash fails it, so a substantive change cannot ship under an old version. Old entries stay, as the history docket-doctor will merge from.

**The skill and the template are checked against each other.** A second test collects every `"<heading>" in docket/workflow.md` reference in `shipped/skill/SKILL.md` and requires each to be a level-3 heading of the template, so renaming one side without the other fails.

**Never replaced, even with `--force`.** `--force` exists to restore Docket-owned files; the workflow file is the project's, and replacing it would discard the project's steps. `dkt init` lists an existing file as `kept` and adds no finding for that alone.

**Where the code goes.** `setup.Decide` gains a `workflow` kind with its own decision, like the config kind: missing means create; otherwise the action is always keep, and a `Workflow` state on the decision carries the file's version, the shipped version and the missing headings. The status (`same`, `outdated`, `differs`) and the findings come from that state in `internal/cli`, as the config findings do. The version is read with `change.ReadFrontMatter`; the headings with `markdown.Headings` over `markdown.FenceMask`. Symbolic links and unreadable files are handled as for every other target.

**This repository's workflow file.** `dkt init --force` from this branch's build creates `docket/workflow.md` here; then "When to archive" says to archive each change in its own pull request until a release process exists, which is the rule AGENTS.md holds today. AGENTS.md drops its "Default steps" line and the archive line, and lists `docket/workflow.md` as a prompt file.

## Risks / Trade-offs

- [The skill tells the agent to stop when a section is missing, and docket-doctor does not exist yet] → The finding names the heading, so the user can copy it back from the template by hand.
- [A heading changed only in case or spacing is reported missing] → Intentional: the skill quotes it exactly, so an agent looking for it would not find it either.
- [An agent edits a section's heading while customizing] → docket-init and the template both say headings are not renamed or translated, and `dkt inspect` reports it.
