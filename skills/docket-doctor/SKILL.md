---
name: docket-doctor
description: >-
  Runs only when the user invokes it. Checks this project's Docket
  environment and setup and helps the user fix them. Can integrate the
  user's habits into `docket/workflow.md` again.
disable-model-invocation: true
---

# Docket doctor

Check this project's Docket setup, list every problem, and fix them one at
a time as the user allows. When the user asks to integrate their habits
into `docket/workflow.md` again, follow step 6.

## 1. Run `dkt inspect`

Run `dkt inspect --json` before anything else; it is a read-only command.
Its `result` holds:

- `files`: every Docket file `dkt init` writes (not the files
  `openspec init` generates), as `same`, `missing`, `outdated` or
  `differs`. Of these, `docket/workflow.md` (kind `workflow`) belongs to
  the project: `dkt init` writes it only when it is missing, and never
  changes it after that, with or without `--force`;
- `openspec`: the OpenSpec CLI's `version` (null when it cannot start) and
  whether it meets the minimum version;
- `default_branch`: the branch whose commit trailers were checked; null
  when none can be found (as in a repository with no commit), and the
  trailers are then not checked.

Its `findings` and `unreadable` are the problems found mechanically, among
them `install-ignored` (a path `dkt init` writes is ignored by git),
`title-missing` (a main spec's first line is still the generated id),
`trailer-missing` (no commit carries the trailer of a change on the default
branch), and the four starting with `workflow-` (`docket/workflow.md` is
missing, follows an older template, has an unknown version, or lacks a
section). When it exits with 2 or 4, nothing was checked: tell the user
what it said and stop.

Then run `dkt lint --json` and look only at `result.changes`: those whose
`schema` is not `docket` are active changes made with another schema. Its
other findings belong to each change's own work and are not handled here.
When it exits with 4 (for example, when OpenSpec cannot be used), skip
this and go on.

## 2. Check what needs judgment

- Reviewers: does this environment have the models, subagents and tools
  the sections of `docket/workflow.md` use? When it does not, say which
  fallback applies, as "Reviewers" in `docket/workflow.md` says.
- Conflicts with the docket skill's steps: whether the user's instructions
  in your context (their global agent instructions, memory, this project's
  `AGENTS.md` and `CLAUDE.md`) conflict with a step of the `docket` skill,
  such as squash merges, nothing extra in commit messages, plans kept
  outside the change folder, or opening changes with `/opsx:propose`.
  Quote both.
- superpowers: whether its skills (brainstorming, writing-plans,
  subagent-driven-development) are available; full review uses them.

Read user-level files that are not in your context, such as global
instruction files or another agent's configuration, only after the user
agrees.

## 3. List every problem

Before fixing anything, list every problem from steps 1 and 2, each with
what it is, why it matters and the fix you propose. When there is none, say
so and stop.

## 4. Fix them one at a time

Take the problems in the listed order. For each, say what the fix comes
down to and carry it out once the user allows it; when they do not, skip
it and go on to the next.

- Written files (`schema-missing`, `skill-missing`, `schema-outdated`,
  `skill-outdated`, `schema-differs`, `skill-differs`,
  `config-schema-default`, `guidance-missing`, `guidance-outdated`): handle
  them together, since one command fixes them. `dkt init` writes every
  missing file, updates the ones from an earlier release and leaves
  differing ones alone; `dkt init --force` also replaces each differing
  file, losing its edits. First show how each differing file differs from
  the shipped one: run `git init` in a new temporary directory, run
  `dkt init` there, and compare the files at the same paths. Then name
  every file the command will write, and run it only after the user
  agrees. Neither command changes an existing `docket/workflow.md`.
- `guidance-differs`: the `operations` section of `openspec/config.yaml` is
  the project's own, and `dkt init` keeps it with or without `--force`.
  Show how it differs from the shipped one; whether to change it is the
  user's call, and edit it only after the user agrees.
- `config-schema-other`: `openspec/config.yaml` names another schema.
  Propose `schema: docket`; edit the file only after the user agrees.
- `symlink-refused`: say which symbolic link stopped `dkt init` and how to
  deal with it; do not change the link yourself.
- `openspec-missing` or `openspec-unsupported`: say the minimum version
  Docket needs (the finding names it) and the install command; install
  nothing.
- `install-ignored`: group the findings by rule (the pattern in the
  message). Before proposing anything, list the files the fix would let git
  see, including files under that directory ignored today
  (`git ls-files --others --ignored --exclude-standard -- <dir>`). Propose
  the narrowest rule that lets the written paths through (for `.claude/`:
  `.claude/*` and `!.claude/skills/`). Say whether the ignore file is shared
  (`.gitignore`) or only on this machine (`.git/info/exclude`,
  `core.excludesFile`). Edit only after the user agrees. When `openspec/` is
  kept out of git on purpose, edit nothing.
- `title-missing`: read each spec's Purpose and draft a name. List all the
  drafts; after the user agrees, change the first line of each
  `openspec/specs/<capability>/spec.md` to `# <name> Specification`, then
  run `dkt lint`.
- `workflow-missing`: run `dkt init` to write the default
  `docket/workflow.md`, then integrate the user's instructions and habits
  by step 6.
- `workflow-outdated`: the project's file follows an older template. Get
  the current template (run `git init` and `dkt init` in a new temporary
  directory) and compare it with the project's file section by section:
  add the sections new in the template; for every other section, put the
  new template's text beside the project's and show the user what would
  change. Write it only after the user agrees, and set `template` in the
  front matter to the new version.
- `workflow-template-unknown`: the file has no `template` version that can
  be read, or one higher than this dkt ships. A higher version usually
  means the dkt on this machine is older than the one that wrote the file:
  propose upgrading dkt first. With no version, say how `template` is
  written (a positive integer in the front matter) and let the user decide
  which version it is; do not guess.
- `workflow-section-missing`: the finding names the missing section. Copy
  that section back from the current template, with its heading exactly as
  it is there; when the user wants it done their own way, rewrite its text
  by the rules of step 6. Write it only after the user agrees.
- Conflicts with the docket skill's steps: propose wording for the user's
  instructions; never edit their files.
- `trailer-missing`: say that this change's commits on the default branch
  lack the `Change: <slug>/<N>` trailer (as a squashed pull request does),
  that `dkt sync` cannot tick its plan from the trailers, and that the check
  before archiving will stop it. The decision is the user's.
- Changes made with another schema (found by `dkt lint` in step 1): when
  its `tasks.md` has nothing ticked yet, propose converting it by
  `convert.md` in this skill's folder. Otherwise do not propose converting:
  say it finishes with its own files and is archived with `/opsx:archive`
  all the same, and offer to draft names for the capabilities it adds,
  under `specs:` in the front matter of its `proposal.md`.
- Unreadable sources, or no superpowers: say what is wrong; the fix is the
  user's.

## 5. Check again

Run `dkt inspect` again and report what is left.

## 6. Integrate `docket/workflow.md` again

When the user asks for it, or to fix `workflow-missing`, integrate the
user's instructions and habits into `docket/workflow.md`:

- Sources: the user's instructions and memory already in your context
  (such as global instruction files), where they say how planning, reviews,
  verification and merging are done, and the tools you can use (other
  models, subagents, plugins, skills, commands). Read user-level files that
  are not in your context only with the user's consent.
- Start from the file as it is: keep what the project has already changed,
  unless the user asks otherwise.
- Keep every step of the template: where the user's instructions say how
  to do a step, rewrite it their way, naming the tools and settings; add
  the user's extra steps to the matching section, without adding sections;
  where they contradict the template, keep the template's step and mark the
  conflict for the user to decide. Instructions that conflict with the
  docket skill's steps (such as asking for squash merges) are listed as
  conflicts too.
- Keep the headings and the `template` value in the front matter as they
  are, and write in the file's own language.
- The file holds the project's shared steps: what you integrate is the
  user's personal habits, which affect everyone else on the project, so
  remind the user of that before changing it.
- Rather than showing the user the whole file, tell them the main points
  of the workflow and of the integration, listing where it differs from
  the template and the conflicts to decide; revise it until the user
  agrees, and only then write it and commit.
