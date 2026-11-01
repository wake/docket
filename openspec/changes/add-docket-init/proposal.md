# Proposal

## Why

Docket's skills reach a project only through `dkt init`, so a project that
has never run it has no Docket command to start from, and setting one up
takes several steps the user has to know: check that OpenSpec and
superpowers are there, run `dkt init`, write the workflow file, check the
setup. A command installed once per machine, like Claude Code's own
`/init`, can be typed in any project and walk through those steps.

## What Changes

- Docket ships a fourth skill, `docket-init`, that only the user invokes
  (`/docket-init` in Claude Code, `$docket-init` in Codex). It checks the
  environment first, writing nothing, and stops with install instructions
  when git, OpenSpec 1.14.x or superpowers is missing. It then runs
  `dkt init`, runs setup when the workflow file is missing or incomplete,
  checks the setup as `docket-doctor` does, and ends by listing the files
  to commit.
- `dkt init` installs `docket-init` once per machine, at user level:
  `~/.claude/skills/docket-init/` (or under `CLAUDE_CONFIG_DIR`) and
  `~/.agents/skills/docket-init/`, by the rules it already uses for the
  per-machine schema: a missing file is written and listed by its absolute
  path, a differing one is reported as `user-skill-differs` and kept unless
  `--force` is given, and a link at or below a `docket-init` folder makes
  it change nothing.
- `dkt inspect` reports those files with kind `user-skill`, as
  `user-skill-missing` or `user-skill-differs`.
- `dkt inspect` works in a repository whose default branch cannot be found,
  such as one with no commit yet: it skips the trailer check, says so, and
  reports `result.default_branch` as null instead of exiting with 4. The
  command's checks run in a new project.

## Non-Goals

- No survey of an existing project's code: the interface list, the
  adoption checks and a draft of capabilities are their own ideas
  (`2026-10-31-codebase-decomposition.md`, `2026-10-24-add-index.md`,
  `2026-10-31-existing-project-adoption.md`).
- Neither `dkt` nor the command installs git, OpenSpec, superpowers or
  `dkt`; installing writes outside the project.
- `dkt` does not look for superpowers or read an agent's configuration;
  only the agent can see which skills it has.
- No plugin or marketplace, and no prebuilt binaries
  (`2026-11-01-prebuilt-binaries.md`).
- `docket-init` is not written into projects' `.claude/skills/` or
  `.agents/skills/`.
- `dkt init` writes both user-level copies whichever agents are installed,
  and nothing removes them.
- The command does not commit.

## Builds On

- `project-setup` `Install the schema`, `Existing installed files` and
  `Files init writes`: the per-machine schema is the model for a file
  written outside the project. `internal/setup` (`Init`, `Inspect`,
  `targetsOf`, `decide`, `write`, `install`, and `target.base` for an
  absolute destination) installs the new files the same way;
  `safepath.Check` gives the link rule.
- `setup-inspection` `Installed files inspected` and `Doctor runs only when
  invoked`: the findings follow the existing `*-missing`/`*-differs` form,
  and `docket-init` uses `docket-doctor`'s frontmatter and
  `agents/openai.yaml` to run only when invoked. `cli.openSpecState` already
  reports the OpenSpec CLI that the command checks.
- `internal/userdirs` (`Home`, `Lookup`, and `DataHome`'s handling of a
  relative path) for the user-level directories.
- `docket.Skills` and the `docket-setup` and `docket-doctor` skills, which
  the command follows rather than repeats.
- `scan.ResolveDefaultBranch` and `scan.ErrNoDefaultBranch`, the error
  `dkt inspect` stops on today.
- Searched `rg -n 'claude/skills|agents/skills|CLAUDE_CONFIG_DIR' internal
  schema.go openspec/specs`: only the project skill roots exist; nothing
  writes under the user's agent directories.

## Capabilities

### New Capabilities

- `init-command`: the `docket-init` command, what it checks, runs and
  reports, and what it leaves to the user.

### Modified Capabilities

- `project-setup`: `dkt init` also installs `docket-init` at user level.
- `setup-inspection`: `dkt inspect` reports the user-level files, and runs
  in a repository without a default branch.

## Impact

- A new embedded skill folder `user-skills/docket-init/` (`SKILL.md`,
  `agents/openai.yaml`), a new prompt path.
- `internal/setup`, `internal/userdirs`, `internal/cli` (`init`,
  `inspect`), `schema.go`.
- `dkt init` and `dkt inspect` output list the new files; `dkt inspect
  --json` gains `result.default_branch`.
- README in both languages, `AGENTS.md` prompt paths,
  `docs/ARCHITECTURE.md`.

## Implementation Contract

### Observable Behavior

- On a machine where it never ran, `dkt init` lists
  `~/.claude/skills/docket-init/SKILL.md`,
  `~/.claude/skills/docket-init/agents/openai.yaml` and the same two under
  `~/.agents/skills/docket-init/` as `written`, with absolute paths.
- In any project, the user types `/docket-init` (or `$docket-init`); the
  agent checks the environment, then sets the project up step by step,
  asking where the steps it follows ask, and ends with the files to commit.
- `dkt inspect` in a repository with no commit reports the setup and
  `result.default_branch: null`, and exits with 0 or 1, not 4.

### Interfaces

- Skill `docket-init`: `SKILL.md` frontmatter `name: docket-init`,
  `disable-model-invocation: true`; `agents/openai.yaml` with
  `policy: allow_implicit_invocation: false`.
- `dkt init` and `dkt inspect` file kind `user-skill`; findings
  `user-skill-missing`, `user-skill-differs`.
- `CLAUDE_CONFIG_DIR`: when set and not empty, the Claude Code copy goes
  under `$CLAUDE_CONFIG_DIR/skills/`; it must be absolute.
- `dkt inspect --json`: `result.default_branch`, the branch whose trailers
  were checked, or null.

### Failure Modes

- No home directory, or a relative `CLAUDE_CONFIG_DIR`: `dkt init` exits
  with 4 as it does for the schema directory; `dkt inspect` reports
  `user-skill-missing` with the reason.
- A link at or below a user-level `docket-init` folder: `dkt init` writes
  nothing, names the link (`symlink-refused`) and exits with 1.
- Not a git repository, no OpenSpec 1.14.x, no superpowers, or no `dkt`:
  the command stops before writing and says what to install.
- `dkt init` reports differing files: the command shows the differences and
  runs `dkt init --force` only after the user agrees.

### Acceptance Criteria

- `project-setup#fresh-machine-gets-the-command`,
  `project-setup#claude-config-dir`, `project-setup#edited-command-copy`,
  `project-setup#copies-agree`,
  `project-setup#symlinked-command-folder-refused`,
  `project-setup#user-skills-directory-linked`,
  `project-setup#personal-skills-unchanged`.
- `setup-inspection#command-not-installed`,
  `setup-inspection#no-home-directory`, `setup-inspection#nothing-written`,
  `setup-inspection#repository-without-commits`,
  `setup-inspection#default-branch-found`.
- `init-command#not-run-on-its-own`, `init-command#invoked`,
  `init-command#environment-first`, `init-command#missing-openspec`,
  `init-command#missing-superpowers`, `init-command#not-a-git-repository`,
  `init-command#new-project`, `init-command#differing-files`,
  `init-command#files-to-commit`.

## Change Log

| Date | Source | Change | Impact |
|---|---|---|---|
