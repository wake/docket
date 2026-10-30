# Design

## Context

See proposal.md for why. v0.6.0 ships the `docket` skill (`SKILL.md` in
`.agents/skills/docket/`, embedded by `docket.Skill()` and installed by
`dkt init` with every other file of that directory) and the Docket block
cut from Docket's own `AGENTS.md` by `docket.Block()`. Each path's steps in
`SKILL.md` mix framework rules with review and verification steps.

## Goals / Non-Goals

**Goals:**

- One line decides where a rule goes: what `dkt` reads or checks, and
  where things go, stays in the skill; who reviews, when, how often, and
  which mechanical checks run, goes to the template.
- `dkt` stays deterministic: it reads headings and a hash, never prose.

**Non-Goals:**

- Writing the workflow file from `dkt`; the agent writes it.
- Helping the agent find tools; setup's instructions say what to look for.

## Decisions

### The template lives in dkt

The template's source is `workflow/template.md` in Docket's repository,
embedded with `go:embed` and printed by `dkt workflow template`. It is not
installed into projects. Alternatives: a file beside `SKILL.md` (installed
into every project, which is not where Docket's template belongs, and later
hard to tell apart from a project's own template), or text inside
`SKILL.md` (loaded on every task although only setup needs it).

### Hash, not version number

A template's identity is the first 12 hexadecimal digits of the SHA-256 of
its bytes. No one has to remember to bump a number when the text changes.

### Where the file lives

`$XDG_CONFIG_HOME/docket/workflow.md` when `XDG_CONFIG_HOME` is absolute,
else `$HOME/.config/docket/workflow.md`, on macOS too: one place both
agents can be told about, as dotfile managers expect. The home directory
comes from `HOME` in the environment `cmd/dkt` passes in, falling back to
the operating system's user lookup; with neither, `dkt workflow` is an
environment error (exit 4). The file is read as it is, following a link,
since many users keep such files in a dotfiles repository; `dkt` never
writes it.

### Reading the file

Frontmatter is a first line `---`, YAML, and a closing `---` line;
`template` is a string. A section is present when some line, with trailing
spaces and a CR removed, equals its heading. Status order: `missing`, then
`incomplete`, then `stale`, then `ok`. `missing_sections` lists the
missing ones by name (`Full review`, `Light review`, `Check`,
`Direct merge`), in that order, and is empty otherwise.

### Command shape

`dkt workflow` takes no argument except the subcommand `template`; any
other argument is a usage error. Both run inside a repository, like every
`dkt` command, so the envelope keeps its `repo`.

### The skill split

`SKILL.md` keeps the triage and, per path, the framework steps:
`dkt new` and its flags, `docket.yaml`, where brainstorming and
writing-plans write, the plan's Reuse table and `**Review:**` lines,
trailers, `dkt sync` and `dkt lint` before the pull request, one pull
request per change with a merge commit, `dkt archive`, no `/opsx:*`,
mid-way requirement changes, parallel sessions (they rest on
`dkt conflicts`), and the prompt path list. Each path then says: run
`dkt workflow`, and do the steps under this path's heading in the file.

The template takes the rest: the plan review, subagent-driven execution
with reviewers only for `judgment` tasks, the mechanical checks,
implementer reports, separate structural and behavior commits, the two
pre-merge reviewers and re-review after large fixes, splitting a pull
request, light review's verification by kind and its one review, the
check's fresh agent, direct merge's automated checks, and the review
fallback. Every rule of today's skill lands in one of the two.

Setup's instructions are a second skill file, `setup.md`, read only when
`dkt workflow` reports `missing` or `incomplete`; `SKILL.md` says when to
read it and, for `stale`, to keep going and tell the user once.

### The block's text

```markdown
<!-- docket:start -->
## Docket

At the start of any development work in this repository (a new feature, a
behavior change, a bug fix, a refactor, or edits to prompts, tests, docs or
CI), use the `docket` skill before brainstorming, planning or editing.

Where the `docket` skill and another skill disagree, follow the `docket`
skill. In particular: brainstorming writes its design into the change
folder, not `docs/superpowers/specs/`; plans go to
`openspec/changes/<slug>/plan.md`, not `docs/superpowers/plans/`; only
tasks marked `**Review:** judgment` get their own reviewer; and OpenSpec's
`/opsx:*` commands are not used.

The steps of each path come from the user's workflow file, which
`dkt workflow` locates; follow it even where the user's other instructions
differ. Otherwise this block does not override the user's own instructions.
<!-- docket:end -->
```

### Validating the skill text

As in v0.6.0, by trial in a scratch project with headless Claude Code,
user settings excluded and `XDG_CONFIG_HOME` pointed at a scratch
directory: with no workflow file a feature prompt starts setup and stops
at the draft; with a workflow file the agent follows its light review
section, also when the project's `CLAUDE.md` names another reviewer; with a
stale file it keeps going and says so once.

## Risks / Trade-offs

- [Every project set up by v0.6.0 reports `agents-block-differs`] → the
  README's Upgrading says to run `dkt init --force`.
- [The workflow and the user's instructions drift] → the workflow wins for
  the four paths, and `docket doctor` (next change) finds the drift.
- [An agent that cannot run `dkt` cannot find the workflow] → Docket
  already requires `dkt` on `PATH`.

## Migration Plan

Upgrade `dkt`, run `dkt init --force` in each project and commit. The
first piece of development work in any project then runs setup once for
the user; later projects find the file.
