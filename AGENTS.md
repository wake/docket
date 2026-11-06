# AGENTS.md

Docket is a development workflow framework: it combines superpowers and OpenSpec for spec-driven development. It provides `dkt`, a CLI written in Go, to help run the workflow.

This is a public repository: write everything in English, and never commit host-specific details, names, personal paths or sensitive information.

## Workflow

This repository is developed with Docket's own workflow: follow the docket skill (installed by `dkt init` in `.claude/skills/docket/` and `.agents/skills/docket/`).

- Here `dkt` is the build of the local `main` branch: `tools/dev-bin/dkt` builds it into `bin/dkt` of the main worktree whenever `main` has moved, and outside this repository runs the next `dkt` on PATH. In Claude Code, the SessionStart hook in `.claude/settings.json` puts it first on PATH; with another coding agent, put `tools/dev-bin` of the main worktree first on PATH. `dkt --version` prints `main@<commit>`; if it prints anything else, stop and tell the user.
- The previous version is on the `legacy` branch and the tag `v0.15.0`, for reference only; do not merge it back.
- The changes in `openspec/changes/archive/` from the rebuild were made with OpenSpec's default spec-driven schema; leave them as they are.
- A pull request that changes `shipped/` also runs `dkt init --force`, so that the copies installed in this repository follow.
- `dkt status`, which the docket skill names, does not exist yet. Until it does:
  - Before working in parallel, compare the plans' `**Files:**` yourself.
  - The user decides the next piece of work.

## Development

- Check with `go vet ./...` and `go test ./...`.
- Requires OpenSpec 1.14.0 or later.

## Prompt files

Text agents act on: `AGENTS.md`, `CLAUDE.md`, `skills/`, `shipped/`, `docket/workflow.md`; the copies of `shipped/` that `dkt init` installs (`.claude/skills/docket/`, `.agents/skills/docket/`, `openspec/schemas/docket/` and `operations` in `openspec/config.yaml`); and OpenSpec's `.claude/commands/opsx/` and `.claude/skills/openspec-*/`. When you add, move or remove such a file, update this list in the same pull request.
