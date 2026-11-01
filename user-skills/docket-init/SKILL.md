---
name: docket-init
description: >-
  Set this project up for Docket: check that git, dkt, the OpenSpec CLI
  and superpowers are in place, run dkt init, write the user's workflow
  file when it is missing, check the setup, and list the files to
  commit. Runs only when the user invokes it.
disable-model-invocation: true
---

# Docket init

Set this project up for Docket, step by step. This command writes only
through `dkt init`, `docket-setup` and `docket-doctor`, each with the
consent it asks for. It never installs anything and never commits.

## 1. Check the environment, writing nothing

Run every check below and collect what is missing before saying anything.
A check that cannot run because an earlier one failed is listed as not
checked yet, with the reason.

- git: run `git rev-parse --show-toplevel`. If it fails, the user should
  open the project's repository or create one with `git init`.
- `dkt`: if it cannot be started, the user should install it with
  `go install github.com/wake/docket/cmd/dkt@latest`.
- OpenSpec, only inside a work tree: run `dkt inspect --json`. On
  `openspec-missing` or `openspec-unsupported`, the user should install
  OpenSpec 1.14.x (`npm install -g @fission-ai/openspec@1.14`) or set
  `DOCKET_OPENSPEC` to a command that runs it, such as
  `npx -y @fission-ai/openspec@1.14.3`. On an exit with 2 or 4, report
  what it said.
- superpowers: check your own skill list for brainstorming, writing-plans
  and subagent-driven-development (Claude Code lists them as
  `superpowers:brainstorming` and so on). If any is missing: in Claude
  Code, `/plugin install superpowers@claude-plugins-official`; in Codex,
  `/plugins` and search for superpowers; in other agents, see
  https://github.com/obra/superpowers. Then start a new session.

When anything is missing, report all of it together, with how to install
each, and stop. Install nothing.

## 2. Run `dkt init --json`

It writes the project's files, the per-machine schema and this command's
user-level copies. For any finding it reports, do what `docket-doctor`
says for that finding (read its `SKILL.md`, see step 4): show how a
differing file differs, and run `dkt init --force` only after the user
agrees. When it exits with 2, 3 or 4, tell the user what it said and stop.

## 3. The workflow file

Run `dkt workflow --json`.

- `missing` or `incomplete`: read `docket-setup/SKILL.md` from this
  repository's `.claude/skills/` (Claude Code) or `.agents/skills/`
  (Codex) and follow it. Say that skills `dkt init` added during this
  session may not be loaded yet, which is why it is read from the file.
- `stale`: say so and offer setup.
- `ok`: go on.

## 4. Check the setup

Read `docket-doctor/SKILL.md` the same way and follow it.

## 5. Report

List what `git status --short` shows as created or changed by the setup,
for the user to commit (`openspec/`, `.claude/skills/`, `.agents/skills/`,
`AGENTS.md`, `CLAUDE.md`). Say that the workflow file, the per-machine
schema and the `docket-init` copies are outside the repository. Do not
commit.

- In a repository with no commit, say that the first commit gives it the
  default branch `dkt new` needs.
- In Claude Code, when `.claude/skills/` did not exist before this run,
  say to type `/reload-skills`. In Codex, say to restart when `$docket`
  does not show.

End by saying development or a refactoring can start: the user describes
the work, and the `docket` skill triages it.
