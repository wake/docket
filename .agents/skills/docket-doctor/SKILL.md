---
name: docket-doctor
description: >-
  Check this project's Docket setup (the files dkt init installs, the
  OpenSpec CLI, the user's workflow file, merged changes that lost their
  commit trailers, and what needs judgment), then help the user fix what
  is wrong. Runs only when the user invokes it.
disable-model-invocation: true
---

# Docket doctor

Check this project's Docket setup, list every problem, then fix them with
the user one at a time. Change nothing the user has not agreed to.

## 1. Run `dkt inspect`

Run `dkt inspect --json` before anything else. It writes nothing. Its
`result` holds:

- `files`: each file `dkt init` installs, as `same`, `missing` or
  `differs`;
- `openspec`: the OpenSpec CLI's `version` (null when none could be
  started) and whether it is `supported`;
- `workflow`: the user's workflow file, as `dkt workflow` reports it;
- `default_branch`: the branch whose trailers were checked; null when none
  was found, as in a repository with no commit, and `changes` is then
  empty;
- `changes`: the active changes on the default branch whose commit
  trailers were checked, and those skipped, with the reason.

Its `findings` and `unreadable` are the mechanical problems. They include
`install-ignored` (a path `dkt init` installs that git ignores),
`title-missing` (a main spec without a title) and `other-schema` (an active
change made with a schema other than `docket`). When it exits
with 2 or 4 it checked nothing: tell the user what it said and stop.

## 2. Check what needs judgment

Read the workflow file at `result.workflow.path` when it exists, then
check:

- What each step needs: for each step of the workflow file, whether this
  environment has what it names, such as another model to review with,
  subagents, or a command, skill or plugin. Where a required review's
  reviewer is missing, say which level of the fallback under the file's
  `## Reviews` applies: a reviewer on a different model, else the same
  model in a fresh context, else a note in the pull request that the work
  had no independent review.
- Drift: where the workflow file and the user's instructions already in
  your context (their global agent instructions, memory, this project's
  `AGENTS.md` and `CLAUDE.md`) say different things about the same step,
  such as another reviewer, model or effort. Quote both.
- Conflicts with the framework: instructions in your context that
  Docket's rules do not allow, such as squash merges (`dkt sync` reads
  each commit's trailers), or rules that open a change or a release where
  the triage would not.
- superpowers: whether its skills (brainstorming, writing-plans,
  subagent-driven-development) are available; the full review path uses
  them.

Do not read, list or search user-level files that are not already in your
context, such as global instruction files or another agent's
configuration, unless the user agrees first.

## 3. List every problem

Before fixing anything, list every problem from steps 1 and 2, each with
what it is, why it matters and the fix you propose. When there is none,
say so and stop.

## 4. Fix them one at a time

Take the problems in the order listed. For each, say what you will do,
and do it only after the user agrees; when they decline, go to the next.

- The workflow file (`workflow-missing`, `workflow-incomplete`,
  `workflow-stale`, or a step this environment cannot do): run the
  `docket-setup` skill. When `result.workflow.path` is null, no home
  directory was found to hold the file: say so instead; setup cannot run
  until `HOME` is set.
- Drift: ask which side the user wants to keep. The workflow file changes
  through `docket-setup`; for their own instructions, propose the
  wording.
- The installed files (`schema-missing`, `skill-missing`,
  `agents-block-missing`, `schema-differs`, `skill-differs`,
  `agents-block-differs`, `user-skill-missing`, `user-skill-differs`,
  `schema-project-copy`, `config-schema-default`):
  take them together, since one command fixes them all. `dkt init` writes
  every missing file, removes a project copy of a shipped schema once the
  per-machine copy matches the shipped one, sets
  `schema: docket` in `openspec/config.yaml`, and leaves differing files
  alone; when there is no `openspec` directory it first runs
  `openspec init`. `dkt init --force` also replaces every differing file.
  The schema's files and the `docket-init` copies are per-machine, named
  by absolute path. First show how each differing file differs from the
  shipped one: in a new temporary directory, run `git init`, create an
  empty `openspec` directory, and run `dkt init` there with `HOME`,
  `XDG_DATA_HOME` and `CLAUDE_CONFIG_DIR` set to new temporary directories,
  so nothing on your machine is written; compare the file at the same
  path, or per-machine files under those directories. Then name
  every file the command will write or remove, and run it only after the
  user agrees.
- `schema-edited`: show how the project's `openspec/schemas/docket` differs
  from the shipped schema. Say that OpenSpec uses it instead of the
  per-machine one and that forks are not supported yet. Removing it loses
  the edits: only after the user agrees, remove it and run `dkt init`.
- `config-schema-other`: say that `openspec/config.yaml` names another
  schema and `dkt new` uses `docket` either way. Propose setting
  `schema: docket`; edit the file only after the user agrees.
- `symlink-refused` or `agents-block-malformed`: explain what stops
  `dkt init` and propose how to fix it; do not change the link or the
  markers yourself.
- `openspec-missing` or `openspec-unsupported`: say which versions Docket
  supports (the finding names them) and that `DOCKET_OPENSPEC` can name
  the command that runs one. Install nothing.
- `install-ignored`: group the findings by rule (the pattern in the
  message). Before proposing, list the files the fix would let git see,
  including files under that directory that are ignored today
  (`git ls-files --others --ignored --exclude-standard -- <dir>`). Propose
  the narrowest rule that frees the installed paths (for `.claude/`:
  `.claude/*` and `!.claude/skills/`). Say whether the ignore file is
  shared (`.gitignore`) or only on this machine (`.git/info/exclude`,
  `core.excludesFile`). Edit it only after the user agrees. When
  `openspec/` is kept out of git on purpose, do not edit anything: point
  to the README's adoption steps.
- `title-missing`: read each spec's Purpose and draft a title, one line of
  text that is not empty (the form `dkt lint` checks). List all the drafts,
  write every `openspec/specs/<capability>/docket.yaml` (`title: <text>`)
  only after the user agrees, then run `dkt lint`.
- `other-schema`: when the finding says 0 steps ticked, propose converting
  the change by `convert.md` in this skill's folder. Otherwise propose no
  conversion: say it is finished with its own files and archived with
  `dkt archive`, and list the capabilities it adds that have no
  `docket.yaml` title beside their delta
  (`openspec/changes/<slug>/specs/<capability>/docket.yaml`); offer to
  draft them as for `title-missing`.
- A conflict with the framework: propose the wording for the user's
  instructions; never edit their files.
- `no-task-commits`: explain that the change's commits on the default
  branch lost their `Change` and `Task` trailers, as a squashed pull
  request does, and that `dkt archive` will refuse the change. Leave the
  decision to the user.
- An unreadable source, or no superpowers: say what is wrong; the fix is
  the user's.

## 5. Check again

Run `dkt inspect` again and report what is left.
