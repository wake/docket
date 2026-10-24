# AGENTS.md

Instructions for coding agents working in this repository. Claude Code reads
this file through CLAUDE.md.

## Project

Docket (`dkt`) is a small Go CLI that checks and keeps the books for OpenSpec
change folders in projects that plan and execute with superpowers. See
README.md. It is a public repository: write everything in English, and never
commit hostnames, customer names, personal paths or secrets.

Status: milestone 1 is being built. `dkt init`, `dkt new`, `dkt lint`,
`dkt sync`, `dkt archive` and `dkt rebase` exist; build them with
`go install ./cmd/dkt`. `dkt conflicts` comes with `add-conflicts` (see the
Order in `docs/ideas/`).

Before writing a proposal or a plan, read `docs/ARCHITECTURE.md` for module
boundaries and shared capabilities. Ideas that are not ready, and the order of
upcoming work, are in `docs/ideas/`.

## Change workflow (superpowers × OpenSpec)

- Triage first. If the work changes no sentence in `openspec/specs/`, do not
  open a change: fix it directly (a bug fix comes with a regression test), or
  edit the docs. An idea that is not ready goes to `docs/ideas/`.
- Otherwise, in a git worktree, open a change:
  `dkt new <slug> --kind feature|change-request|fix|baseline --source "<where the request came from>"`.
- superpowers:brainstorming writes its design into the change folder, not
  `docs/superpowers/specs/`: `proposal.md`, delta specs under `specs/`, and
  `design.md` when needed. Before writing each one, run
  `openspec instructions <artifact> --change <slug> --json` for its template
  and rules.
- superpowers:writing-plans writes `openspec/changes/<slug>/plan.md`, not
  `docs/superpowers/plans/`, following
  `openspec instructions plan --change <slug>`. There is no tasks.md.
- Reuse is decided in the plan: every capability the change needs gets a row
  in the plan's Reuse table, with the search behind the decision. The
  proposal's Builds On names what already exists.
- Execute with superpowers:subagent-driven-development. Each implementer gets
  the Reuse rows for its task and reports every exported symbol it added,
  with why no existing one fit. Every commit made for a task carries the
  trailers `Change: <slug>` and `Task: <N>`.
- Do not use OpenSpec's `/opsx:*` commands or skills.
- When requirements change mid-way, follow the plan rules: cancelled tasks,
  compensating tasks, a Revision section, a Change Log row.
- Before opening the pull request, run `dkt sync <slug>` and
  `dkt lint <slug>`, and commit plan.md.
- Pull requests merge with a merge commit, never squash: `dkt sync` reads the
  trailers of each commit.
- Archive only with `dkt archive` on a branch made from the default branch,
  in the release step. Never run `openspec archive` directly.

## Development

- Go module at the repository root; `go vet ./...` and `go test ./...`.
- OpenSpec CLI pinned to 1.14.x. To use a pinned copy without installing it:
  `DOCKET_OPENSPEC="npx -y @fission-ai/openspec@1.14.3"`.
- `openspec/schemas/docket/` is the schema Docket ships (embedded with
  `go:embed`). Change it only through a change, like any other behavior.
