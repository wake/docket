# Proposal

## Why

Step 3 of docket-init runs `dkt init --json --tools <coding agent>` and step 4 runs `dkt inspect --json`; neither command exists yet. `dkt init` writes Docket's schema, templates, archive guidance and the `docket` skill into a project; `dkt inspect` only checks, writing nothing, so that docket-init can confirm the setup and docket-doctor later knows what to fix. Both call OpenSpec, so locating OpenSpec and checking its version (openspec-runtime) come in this change too.

## What Changes

- Add `dkt init`:
  - When the project has no `openspec/`, it first runs `openspec init --tools <coding agent>`, passing the `--tools` value through as given.
  - It writes `openspec/schemas/docket/` (the schema and its templates), the `schema: docket` line and the archive guidance in `openspec/config.yaml`, `.claude/skills/docket/` and `.agents/skills/docket/`. It does not write `AGENTS.md` or `CLAUDE.md`.
  - Existing files: one equal to the shipped file is left alone; one equal to a file an earlier release shipped is updated; any other is kept and reported, and replaced only with `--force`.
  - The result lists what happened to each file, including the files `openspec init` created, and the next steps.
- Add `dkt inspect`: it checks the same files by the same rules, and also the OpenSpec version and paths git ignores; it writes nothing.
- Locate OpenSpec and check its version: the command named by `DOCKET_OPENSPEC`, else `openspec` on `PATH`; at least 1.7.0, with no upper bound; `OPENSPEC_TELEMETRY=0` on every invocation.
- cli-output gains the two rules held back until a command needed them: unreadable sources (exit 3), and exit 4 for a command that needs a repository outside one; and it states that `--help` always prints text, whatever `--json` says.
- CI gains a job that runs the tests calling the real OpenSpec once with OpenSpec 1.7.0 and once with the latest.
- The files written into a project ship with placeholder content for now: the schema, the templates, the archive guidance, the `docket` skill and the wording of `dkt init`'s next steps are all prompt text, replaced by the real content once the user has edited each. The placeholder schema is still one OpenSpec reads.
- Not in this change:
  - The other three checks of `dkt inspect`: commit trailers and main spec titles come with `dkt sync` and `dkt lint`, changes made with another schema with docket-doctor.
  - Checking for superpowers: docket-init and docket-doctor have the coding agent check its own skills.
  - docket-doctor itself.

## Capabilities

### New Capabilities
- `openspec-runtime`: how dkt finds the OpenSpec CLI, which versions it accepts, and how it reports a failed call.
- `project-setup`: which files `dkt init` writes, what it does with files that already exist, and its result and next steps.
- `setup-inspection`: what `dkt inspect` checks and reports.

### Modified Capabilities
- `cli-output`: adds unreadable sources, commands that need a repository, and help.

## Impact

- Adds `dkt init`, `dkt inspect`, the code that calls OpenSpec and their tests, and the shipped files embedded in dkt (placeholder content for now).
- Adds an OpenSpec job under `.github/workflows/`.
- Dependencies: still only the Go standard library, the system `git` and the OpenSpec CLI.
