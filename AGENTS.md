# AGENTS.md

Instructions for coding agents working in this repository. Claude Code reads
this file through CLAUDE.md.

## Project

Docket (`dkt`) is a small Go CLI that checks and keeps the books for OpenSpec
change folders in projects that plan and execute with superpowers. See
README.md. It is a public repository: write everything in English, and never
commit hostnames, customer names, personal paths or secrets.

Status: milestone 1 is complete. `dkt init`, `dkt new`, `dkt lint`,
`dkt conflicts`, `dkt sync`, `dkt archive`, `dkt rebase` and `dkt status` exist; build them
with `go install ./cmd/dkt`. Upcoming work is ordered in `docs/ideas/`.

Before writing a proposal or a plan, read `docs/ARCHITECTURE.md` for module
boundaries and shared capabilities. Ideas that are not ready, and the order of
upcoming work, are in `docs/ideas/`.

## Change workflow (superpowers × OpenSpec)

- Triage first. When several paths apply, take the heaviest:
  - The work changes a sentence in `openspec/specs/`: open a change (below).
    So does a bug fix whose correct behavior no spec states and no existing
    behavior shows: it defines behavior, and the change adds that sentence.
  - A bug fix that restores the specified behavior, or a restructuring that
    changes no behavior (edits to comments alone are not code): no change.
    Write a regression test and see it fail before the fix, then have the
    whole diff reviewed once before merging.
  - Prompt text, the text agents act on (list the project's prompt paths in
    AGENTS.md), or tests alone: no change, and one review of the whole diff
    before merging. Check prompts mechanically, not with tests that look for
    strings: they still load, and every copy says the same. Check tests by
    mutating the code they cover: they must fail.
  - Docs for people, version bumps and pure moves: no change and no review
    beyond the automated checks.

  An idea that is not ready goes to `docs/ideas/`.
- To open a change, in a git worktree, run
  `dkt new <slug> --kind feature|change-request|fix|baseline --source "<where the request came from>"`.
- superpowers:brainstorming writes its design into the change folder, not
  `docs/superpowers/specs/`: `proposal.md`, delta specs under `specs/`, and
  `design.md` when needed. Before writing each one, run
  `openspec instructions <artifact> --change <slug> --json` for its template
  and rules.
- superpowers:writing-plans writes `openspec/changes/<slug>/plan.md`, not
  `docs/superpowers/plans/`, following
  `openspec instructions plan --change <slug>`. There is no tasks.md. A task
  is a slice that can be verified on its own; tightly coupled tasks are one
  task. A task that needs a reviewer's judgment (an error path,
  environment-dependent behavior, a public interface or output format) says
  `**Review:** judgment`; the others say `**Review:** checks`.
- Reuse is decided in the plan: every capability the change needs gets a row
  in the plan's Reuse table, with the search behind the decision. The
  proposal's Builds On names what already exists.
- Review costs most where it is repeated, so it sits at the two ends of a
  change. Before any code, the plan is reviewed once together with the
  proposal and deltas, preferably by a different model from the one that
  wrote it.
- Execute with superpowers:subagent-driven-development, with one exception:
  only a `**Review:** judgment` task gets its own reviewer. Every task that
  changes code gets the mechanical checks instead: its new tests are seen
  failing before the code and passing after, and mutating the lines it
  changed makes a test fail. A task that changes no code, such as one that
  only edits docs, is checked by its own verification steps. Each
  implementer gets the Reuse rows for its task and reports every exported
  symbol it added, with why no existing one fit. Every commit made for a
  task carries the trailers `Change: <slug>` and `Task: <N>`; structural
  changes and behavior changes go in separate commits.
- Before merging, the whole change is reviewed once, by two reviewers in
  parallel: one on a different model, asked to attack (wrong results, edge
  cases, partial failure, the same defect elsewhere in the code), and one in a
  fresh context, checking the code against the proposal and deltas. After
  fixes, only a large fix is reviewed again.
- A change is one pull request; its commits show its layers. Split it only
  when it mixes unrelated concerns, when a structural refactor is mixed with
  behavior (the refactor goes first, in its own pull request), or when product
  code and tests together run far past about 1000 lines. Do not stack pull
  requests.
- When sessions work in parallel, each change belongs to one session, and
  the session that implements a change does not review it. Run
  `dkt conflicts` and compare the plans' `**Files:**` blocks before starting;
  overlapping changes run one after the other. A change that depends on
  another waits for it to merge. Merge one change at a time: rebase it on the
  default branch, run the full test suite, and review again only where the
  rebase had conflicts.
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
  `go:embed`). Edits to it follow the same triage as any other file; its
  instructions and templates are prompts shipped to other projects.
- Prompt paths, the text agents act on: `AGENTS.md`, `CLAUDE.md`, `context`
  in `openspec/config.yaml`, and `openspec/schemas/docket/`. `dkt init`
  ships the "Change workflow" section of `AGENTS.md` and
  `openspec/schemas/docket/` to other projects.
