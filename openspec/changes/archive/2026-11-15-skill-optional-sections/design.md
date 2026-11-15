## Context

`dkt init` installs the docket skill as one file, `SKILL.md`, in `.claude/skills/docket/` and `.agents/skills/docket/`, the folders Claude Code and Codex read skills from. The idea left one question open: whether Codex and other coding agents load a file a skill points to as reliably as Claude Code does. Claude Code's skills are built for it: a skill folder holds other files that SKILL.md refers to, read when needed. Codex documents the same layout, a skill folder with SKILL.md plus optional references, read under progressive disclosure, but does not say whether the model opens a file SKILL.md points to.

## Goals / Non-Goals

**Goals:**
- A project that uses neither optional feature reads neither section.
- A project upgrading from an earlier release gets the new files from `dkt init` without any step of its own.

**Non-Goals:**
- Splitting any other part of the skill.

## Decisions

### Codex reads the side files: tested

In a scratch repository with the skill split as this change does, `codex exec` (codex-cli 0.153.4, gpt-5.6-sol at its default low effort) was given three tasks that did not name the skill or the side files: twice adding a capability to `docket/domains.md`, once bringing an issue from the tracker named in `AGENTS.md` `## Issues` into the backlog. All three times it read `SKILL.md`, then at once the side file the task called for, and followed it: it made the card with `--external` and `--external-id`, its address as `--source`, outside the order, and took the request as the user's agreement to add a node. The runs used an earlier wording of the two pointer lines; the meaning was the same.

### The headings stay, each with one pointer line

`SKILL.md` keeps `## Issues in another tool` and `## Domain tree`, each holding one line: when the condition holds, read the file in this skill's folder. The skill's reference "see "Issues in another tool"" keeps leading to a heading, and someone reading the skill still sees that both features exist. Nothing else refers to either section: docket-doctor, the issue templates and the schema do not name them.

Alternative: drop the headings and put the pointers into the Issue bullet and Full review step 2 (the reference would need rewriting, and the domain tree would be named in two places instead of one).

### `dkt init` installs every file of the shipped skill folder

The skill targets come from walking `shipped/skill/`, as the schema targets come from walking `shipped/schema/`: each file goes to both skill folders, `SKILL.md` first. Each file is decided on its own by the existing rules (same, missing, outdated, differs), and each needs its hashes in `internal/setup/history.go`.

### An earlier release's skill folder

It holds only `SKILL.md`. `dkt inspect` reports `skill-outdated` for each `SKILL.md` and `skill-missing` for each new file, which docket-doctor already handles by running `dkt init`; `dkt init` lists `SKILL.md` as `updated` and the new files as `created`. A user who edited `SKILL.md` keeps it (`skill-differs`) and still gets the new files; their `SKILL.md` keeps the full sections, so nothing is lost. The reload step stays tied to a created `SKILL.md`: a new side file is read when the skill points to it, with no reload.

### AGENTS.md's prompt file list

It names the skill folders, `.claude/skills/docket/` and `.agents/skills/docket/`, and `shipped/`, so the new files are already in it; it needs no edit.

## Risks / Trade-offs

- [A coding agent other than Claude Code and Codex does not open the side file] → it misses only the optional features' rules, in projects that use them; the pointer line still says the file exists.
- [Codex reads `SKILL.md` 240 lines at a time, and in two of the three runs read no further] → the pointers sit near line 130. The skill is about 300 lines today and about 255 after the split, so "Commit trailers" and "Parallel work" can still be missed; that is outside this change and is reported for a follow-up.

## Migration Plan

Nothing to run: `dkt init`, or docket-doctor and docket-upgrade through it, updates `SKILL.md` and adds the new files.
