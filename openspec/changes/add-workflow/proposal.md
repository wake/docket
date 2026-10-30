# Proposal

## Why

The `docket` skill names each path's review and verification steps in
capabilities ("a reviewer on a different model", "a fresh agent"), and an
agent maps them to tools anew every time. A user who already has tools and
habits, such as a cross-model reviewer they always use, has no place to say
so once (`docket/ideas/2026-10-29-framework-skill.md`, the second of its
three changes; the first shipped in v0.6.0).

## What Changes

- Docket ships a **template**: for each of the four paths (full review,
  light review, check, direct merge), its review and verification steps in
  capabilities, with the review fallback. It is embedded in `dkt`;
  `dkt workflow template` prints it.
- Each user has one **materialized workflow** at
  `$XDG_CONFIG_HOME/docket/workflow.md` (`~/.config/docket/workflow.md` when
  unset): the four paths' steps brought as close to the template as the
  user's tools and habits allow, with the template's hash in its
  frontmatter. It is used by every project and never kept in one.
- `dkt workflow` reports where the file is and whether it is missing, lacks
  a path, or was made from an older template.
- The skill keeps the framework rules: what `dkt` reads or checks and where
  things go (`dkt new`, the change folder, the plan's format, trailers,
  merge commits, `dkt archive`). Each path's steps come from the
  materialized workflow:
  - with none, or one that lacks a path, the agent runs setup before the
    work: it drafts the four paths from the template, the user's
    instructions it already has and the tools it can find, shows the draft,
    and writes the file only when the user agrees;
  - with one made from an older template, it keeps working with it and
    tells the user once what the current template does differently.
- The Docket block in `AGENTS.md` says that the four paths' steps follow the
  materialized workflow even where the user's other instructions differ;
  otherwise the block still does not override the user's instructions.
  Projects set up by v0.6.0 see `agents-block-differs` until `dkt init
  --force` replaces the block.

## Non-Goals

- No reference mode: the materialized workflow never points into the
  user's own instructions; drift between the two is `docket doctor`'s to
  find, in the next change.
- No project template layer; the lookup order stays planned for later.
- `dkt init` says nothing about setup; the skill starts it at the first
  piece of work.
- `dkt` never reads or judges the workflow's prose beyond the four section
  headings and the hash.
- No command that writes the workflow file; the agent writes it after the
  user agrees.

## Builds On

- `docket.Skill` and `docket.Block` (`schema.go`): the embedding pattern;
  the template is embedded the same way.
- `setup.Init` (`internal/setup/setup.go`): installs every file of the
  skill, so a skill file added here ships with no installer change.
- `internal/cli` command table (`run.go`), `Outcome`, findings, `unreadable`
  and exit codes (`envelope.go`).
- `openspec.envValue`-style lookups of `Options.Env`: only `cmd/dkt` reads
  the process environment.
- The `project-setup` capability, whose skill and block requirements change.

## Capabilities

### New Capabilities

- `materialized-workflow`: the template Docket ships, where a user's
  materialized workflow lives and what it holds, and what `dkt workflow`
  reports about it.

### Modified Capabilities

- `project-setup`: the skill's content (framework rules here, each path's
  steps from the materialized workflow, setup when it is missing) and the
  block's content (the materialized workflow wins for the four paths).

## Impact

- `schema.go` and a new template source; a new `internal/cli/cmd_workflow.go`
  and the code that reads the workflow file.
- Shipped prompt text: `.agents/skills/docket/` and `.claude/skills/docket/`
  (the skill and a setup file), the template, the Docket block in
  `AGENTS.md`; the prompt path list in `AGENTS.md`.
- Both READMEs, `docs/ARCHITECTURE.md`, the framework-skill idea (reference
  mode dropped).
- Projects set up by v0.6.0: `dkt init --force` to take the new block and
  skill.

## Implementation Contract

### Observable Behavior

- With no `workflow.md`, `dkt workflow` reports `missing` and exits with 1;
  an agent starting development work runs setup first.
- After setup, `dkt workflow` reports `ok` and exits with 0, and the agent
  follows the file's section for the path it took.
- After an upgrade that changes the template, `dkt workflow` reports
  `stale`; the agent keeps working with the file and says so once.
- `dkt workflow template` prints the template, and with `--json` its text
  and hash.

### Interfaces

- `dkt workflow [--json]`: `result` has `path`, `status` (`ok`, `missing`,
  `incomplete`, `stale`), `template` (the shipped template's hash),
  `recorded` (the hash in the file, or null) and `missing_sections`.
  Findings: `workflow-missing`, `workflow-incomplete` (naming each missing
  section), `workflow-stale`.
- `dkt workflow template [--json]`: prints the template; `result` has
  `template` (its hash) and `text`.
- `workflow.md`: YAML frontmatter with `template: <hash>`, then the
  sections `## Full review`, `## Light review`, `## Check`,
  `## Direct merge`.
- A hash is the first 12 hexadecimal digits of the SHA-256 of the template
  text.

### Failure Modes

- The file cannot be read: listed as unreadable, exit 3.
- The frontmatter is missing, does not parse, or has no `template`:
  `recorded` is null and the status is `stale`.
- A missing section and an old hash together: `incomplete`, which asks for
  setup, wins over `stale`.
- `XDG_CONFIG_HOME` is set but empty or relative: it is ignored, as the XDG
  specification says, and the home directory's `.config` is used.
- No home directory can be found: `dkt workflow` exits with 4.

### Acceptance Criteria

- `materialized-workflow#workflow-in-the-config-home`
- `materialized-workflow#workflow-under-xdg-config-home`
- `materialized-workflow#xdg-config-home-ignored-when-relative`
- `materialized-workflow#template-printed`
- `materialized-workflow#template-names-capabilities`
- `materialized-workflow#no-workflow`
- `materialized-workflow#complete-workflow`
- `materialized-workflow#a-path-missing`
- `materialized-workflow#made-from-an-older-template`
- `materialized-workflow#no-frontmatter`
- `materialized-workflow#unreadable-workflow`
- `project-setup#skill-loads-for-development-work`
- `project-setup#skill-not-loaded-for-a-question`
- `project-setup#setup-before-the-first-work`
- `project-setup#steps-from-the-workflow`
- `project-setup#stale-workflow-kept`
- `project-setup#skill-wins-over-superpowers`
- `project-setup#workflow-wins-over-user-instructions`

## Change Log

| Date | Source | Change | Impact |
|---|---|---|---|
