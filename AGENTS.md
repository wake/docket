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
with `go install ./cmd/dkt`. Upcoming work is ordered in `docket/order.md`.

Before writing a proposal or a plan, read `docs/ARCHITECTURE.md` for module
boundaries and shared capabilities. Ideas are in `docket/ideas/`, one per
file; `docket/order.md` lists the ones queued, in order, and the first
unticked line is the next piece of work.

## Change workflow (superpowers × OpenSpec)

- Triage first. When several paths apply, take the heaviest:
  - The work changes a sentence in `openspec/specs/`: open a change (below).
    So does any edit that sets an expectation no spec states and no
    existing behavior shows, such as a bug fix with no specified answer or
    a test that asserts new behavior: it defines behavior, and the change
    adds the sentence.
  - A bug fix that restores the specified behavior, or a restructuring that
    changes no behavior (moving code is one): no change. For a fix, write
    a regression test and see it fail before the fix. For a restructuring,
    the tests that cover the code pass before and after, and mutating the
    changed lines makes one fail; when none fails, first add the tests as
    a test-only edit. Then have the whole diff reviewed once before
    merging. A comment that changes how the code builds or runs, such as a
    compiler directive (`//go:build`) or an annotation, is code.
  - Prompt text, tests that check behavior already specified or already
    shown, and check settings such as CI configuration: no change, and one
    review of the whole diff before merging.
    Prompt text is the text agents act on: files agents load by convention
    (`AGENTS.md`, `CLAUDE.md`, skill and command files) and text a program
    hands to an agent (embedded or printed instructions, templates). List
    these paths in AGENTS.md, and update the list in the pull request that
    adds, moves or removes one. Check prompts mechanically, not with tests
    that look for strings: they still load, and every copy says the same.
    Check tests by mutating the code they cover: they must fail. Check
    settings are the files that decide which checks run; check them by
    running them, since the pull request's own CI run uses the new
    settings, and have the review look for a check that got weaker.
  - Docs for people (moves and typos in them included), edits that only
    change comments or whitespace in code or test files (these are not
    edits to code or tests, so the paths above do not apply), version bumps,
    and dependency upgrades that touch only the dependency manifests (such
    as `go.mod` and `go.sum`): no change and no review beyond the automated
    checks. Sort an edit by what it touches: a typo in a prompt is prompt
    text, and moving code is a restructuring. When the edit touches a code
    or test file, have a fresh agent, given only the diff and these
    definitions, confirm before merging that every changed line is a
    comment or whitespace and that no comment changes how the code builds
    or runs. If it does not confirm, triage again.

  An idea that is not ready goes to `docket/ideas/`, one file per idea.
- To open a change, in a git worktree, run
  `dkt new <slug> --kind feature|change-request|fix|baseline --source "<where the request came from>" --title "<title>"`.
  A change's title, kind, source and base are in its `docket.yaml`; a new
  capability's delta folder has a `docket.yaml` with its title.
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
`/opsx:*` commands are not used. This block does not override the user's
own instructions.
<!-- docket:end -->
