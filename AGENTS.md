# AGENTS.md

Docket is a development workflow framework: it combines superpowers and OpenSpec for spec-driven development. It provides `dkt`, a CLI written in Go, to help run the workflow.

This is a public repository: write everything in English, and never commit host-specific details, names, personal paths or sensitive information.

## During the rebuild

main is being rebuilt from scratch. The previous version is on the `legacy` branch and the tag `v0.15.0`, for reference only; do not merge it back.

During the rebuild, do not use dkt; use OpenSpec's built-in workflow:

- If there is no `openspec/` yet, run `openspec init --tools <coding agent>` (for example `claude` or `codex`) and commit the files it creates.
- Start a change with `openspec new change <slug>`, which uses the default spec-driven schema.
- Each change lives in `openspec/changes/<slug>/`: `proposal.md`, `specs/`, `design.md` when needed, and `tasks.md`. Before writing each one, run `openspec instructions <artifact> --change <slug>` for its template and rules.
- superpowers' brainstorming and writing-plans do not write to their default location, `docs/superpowers/`: the design goes into the change folder, and the task list into `tasks.md`.
- Implement from `tasks.md` with superpowers' test-driven-development and subagent-driven-development.
- When done, archive with OpenSpec.

Once `dkt init`, `dkt new`, `dkt sync` and `dkt lint` exist, run `dkt init` on this repository, switch to Docket's own workflow, and update this section.

## Development

- The Go module goes at the repository root and is created by the first rebuild change. Once `go.mod` exists, check with `go vet ./...` and `go test ./...`.
- Requires OpenSpec 1.7.0 or later.

## Prompt files

Text agents act on: `AGENTS.md`, `CLAUDE.md`, `skills/`. When you add, move or remove such a file, update this list in the same pull request.
