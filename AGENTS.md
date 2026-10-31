# AGENTS.md

Instructions for coding agents working in this repository. Claude Code reads
this file through CLAUDE.md.

## Project

Docket (`dkt`) is a small Go CLI that checks and keeps the books for OpenSpec
change folders in projects that plan and execute with superpowers. See
README.md. It is a public repository: write everything in English, and never
commit hostnames, customer names, personal paths or secrets.

Status: milestone 1 is complete. `dkt init`, `dkt new`, `dkt record`, `dkt lint`,
`dkt conflicts`, `dkt sync`, `dkt archive`, `dkt rebase`, `dkt status`, `dkt workflow` and `dkt inspect` exist; build them
with `go install ./cmd/dkt`. Upcoming work is ordered in `docket/order.md`.

Before writing a proposal or a plan, read `docs/ARCHITECTURE.md` for module
boundaries and shared capabilities. Ideas are in `docket/ideas/`, one per
file; `docket/order.md` lists the ones queued, in order, and the first
unticked line is the next piece of work.

## Development

- Go module at the repository root; `go vet ./...` and `go test ./...`.
- OpenSpec CLI pinned to 1.14.x. To use a pinned copy without installing it:
  `DOCKET_OPENSPEC="npx -y @fission-ai/openspec@1.14.3"`.
- `schema/docket/` is the schema Docket ships (embedded with `go:embed`);
  `dkt init` installs it once per machine, not into each project. Edits to
  it follow the same triage as any other file; its instructions and
  templates are prompts shipped to other projects.
- Prompt paths, the text agents act on: `AGENTS.md`, `CLAUDE.md`, `context`
  in `openspec/config.yaml`, `schema/docket/`,
  `.claude/skills/docket/`, `.claude/skills/docket-setup/`,
  `.claude/skills/docket-doctor/`, the same three under `.agents/skills/`,
  `workflow/template.md` and `record/template.md`. `dkt init` ships `schema/docket/`, the
  `docket`, `docket-setup` and `docket-doctor` skills and the Docket block of `AGENTS.md` to other projects, and
  `dkt workflow template` prints `workflow/template.md`.

<!-- docket:start -->
## Docket

At the start of any development work in this repository (a new feature, a
behavior change, a bug fix, a refactor, or edits to prompts, tests, docs or
CI), use the `docket` skill before brainstorming, planning or editing.

Where the `docket` skill and another skill disagree, follow the `docket`
skill. In particular: brainstorming writes its design into the change
folder, not `docs/superpowers/specs/`; plans go to
`openspec/changes/<slug>/plan.md`, not `docs/superpowers/plans/`; only
tasks marked `**Review:** judgment` get their own reviewer; and OpenSpec's
`/opsx:*` commands are not used.

The steps of each path come from the user's workflow file, which
`dkt workflow` locates; follow it even where the user's other instructions
differ. Otherwise this block does not override the user's own instructions.
<!-- docket:end -->
