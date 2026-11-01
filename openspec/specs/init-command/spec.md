# init-command Specification

## Purpose
Defines the `docket-init` command, which `dkt init` installs once per
machine and the user invokes in any project to set it up for Docket: what
it checks first, what it runs, and what it leaves to the user.

## Requirements

### Requirement: Init command runs only when invoked
Docket SHALL ship a `docket-init` skill that runs only when the user invokes it. Its `SKILL.md` SHALL start with frontmatter whose `name` is `docket-init`, whose `description` says only when to use it in at most 1,024 characters, and which has `disable-model-invocation: true`. Its folder SHALL hold `agents/openai.yaml` with `policy: allow_implicit_invocation: false`.

#### Scenario: Not run on its own
- **WHEN** an agent with `docket-init` installed is asked to add a feature, or to set up a test framework
- **THEN** it does not load `docket-init` and does not run `dkt init`

#### Scenario: Invoked
- **WHEN** the user types `/docket-init` in Claude Code or `$docket-init` in Codex
- **THEN** the agent follows `docket-init`

### Requirement: Environment checked first
Before anything that writes a file, the command SHALL check, writing nothing, that the working directory is in a git work tree, that `dkt` runs, that the OpenSpec CLI `dkt` finds is in its supported range, and that the agent has the superpowers skills. When any is missing, it SHALL stop, name each missing one, say how to install it, and SHALL NOT install it.

#### Scenario: Environment first
- **WHEN** the user invokes `docket-init` in a git repository with OpenSpec 1.14.3 and superpowers available
- **THEN** the agent checks the four before it runs `dkt init`

#### Scenario: Missing OpenSpec
- **WHEN** `dkt inspect --json` reports `openspec-missing` or `openspec-unsupported`
- **THEN** the agent stops before running `dkt init`, says how to install OpenSpec 1.14.x or point `DOCKET_OPENSPEC` at it, and no file is created or changed

#### Scenario: Missing superpowers
- **WHEN** the agent has no superpowers skills
- **THEN** it stops before running `dkt init` and says how to install superpowers in that agent

#### Scenario: Not a git repository
- **WHEN** the user invokes `docket-init` in a directory outside any git work tree
- **THEN** the agent stops and says to open the project's repository or create one

### Requirement: Setup through existing steps
After the checks pass, the command SHALL run `dkt init`; then, when `dkt workflow` reports `missing` or `incomplete`, follow `docket-setup`; then check the setup as `docket-doctor` does. It SHALL follow those skills from the files `dkt init` wrote in the project, since an agent may not load skills added during the session. Where a step it follows asks before acting, it SHALL ask the same.

#### Scenario: New project
- **WHEN** the user invokes `docket-init` in a git repository with no commit, no `openspec/` and no workflow file, with the environment in place
- **THEN** the agent runs `dkt init`, shows a draft of the workflow file and writes it only after the user agrees, then runs `dkt inspect` and lists any problem it reports

#### Scenario: Differing files
- **WHEN** `dkt init` reports `skill-differs` for `.claude/skills/docket/SKILL.md`
- **THEN** the agent shows how the file differs from the shipped one and runs `dkt init --force` only after the user agrees

### Requirement: Init command report
The command SHALL end by listing the files the setup created or changed in the repository, for the user to commit, and by saying that development or a refactoring can start. It SHALL say how to load skills added during the session when the agent may not have them. It SHALL NOT commit.

#### Scenario: Files to commit
- **WHEN** `docket-init` has set up a new project
- **THEN** the agent lists `openspec/`, the skill folders, `AGENTS.md` and `CLAUDE.md` as files to commit, makes no commit, and in Claude Code says to type `/reload-skills` when `.claude/skills/` did not exist before
