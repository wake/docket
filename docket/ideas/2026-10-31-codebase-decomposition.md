---
title: Decomposing an existing codebase into capabilities
---

Recorded 2026-10-31, while `add-records` was being implemented. The
maintainer asked how Docket could help split an existing project's code into
parts, and which open-source software-engineering tools (not AI tools) could
help. Whether Docket can do this is to be researched when the idea is taken
up; nothing below is decided.

## Problem

A project that adopts Docket after its code exists has code but no specs.
The `baseline` kind of change describes behavior that already ships, but
nothing helps decide which capabilities and domains the code holds, or which
files belong to which capability.

## Prior art

Software engineering calls this architecture recovery, or software
clustering. It uses four kinds of signal:

- **Static dependencies** (imports, calls): which code depends on which,
  cycles, layers. Examples: `go list -deps`; Emerge, which clusters a
  dependency graph by Louvain modularity across several languages;
  dependency-cruiser for JavaScript; ArchUnit and jQAssistant for Java;
  design structure matrix tools such as Structure101, Lattix and
  Sonargraph, mostly commercial.
- **Evolutionary coupling** from version control: files that change
  together. Examples: code-maat (Adam Tornhill) and its Python port.
- **Entry points**: CLI commands, HTTP routes, public APIs, which often map
  to the capabilities a user sees.
- **Boundary checks** once the code is split, so the parts do not grow back
  together: arch-go and go-cleanarch for Go, dependency-cruiser,
  import-linter for Python, ArchUnit.

Graphify parses code with tree-sitter into a graph that can be queried. The
parsing is deterministic, but the tool is aimed at AI coding assistants.

## Leads

- Commit trailers give Docket what plain co-change analysis lacks: `Change:`
  and `Record:` name the change or record a commit belongs to, and a change
  names its capabilities. From them `dkt` could report which files each
  capability has touched over time. It needs only git, which Docket already
  reads, and works for any language.
- Choosing capabilities and domains is judgment. It goes in prompt text (the
  `baseline` path, a skill or a command), never in a `dkt` flag, as
  `2026-10-30-spec-domains.md` settled for domains.
- Language parsers and outside tools would tie Docket to them. Any use of
  one would be optional, at the user's request.

## Trial, 2026-10-31

A baseline of one package of the Purdex repository (`internal/redact`, 174
lines and 5 tests), written unattended by Opus through the `docket` skill
on dkt v0.9.1:

- It took 34 turns, about 4 minutes and US$1.16, and wrote two
  capabilities with 17 requirements and 50 scenarios, about 3,100 words.
  `openspec validate --strict` and `dkt lint` pass. Where the tests were
  silent it probed the code with a temporary test, removed afterwards, and
  found inputs the tests do not cover.
- Nothing in the skill or the schema says how a baseline is written: how
  fine-grained, whether a known defect is specified as it is, the language
  of the specs, capability names, where unanswered questions go, and how to
  probe behavior without writing into the repository. The shipped schema
  also sent it to `docs/ARCHITECTURE.md` and `dkt index`, which exist only
  in Docket's own repository and in the plan.
- About 18 words of proposal and spec per line of code. Purdex has about
  390,000 lines (240,000 of Go, 150,000 of TypeScript) and about 34,000
  test cases. At that rate a whole baseline is thousands of such runs and
  more spec than anyone can review: a baseline has to be scoped, from a map
  made without an agent.
- The map of that package came from three commands: `go doc -short` (three
  functions and a constant), `go test -list` (five tests) and `rg` for its
  importers (nine packages). The agent spent its first steps gathering the
  same facts.
- Two questions for the baseline path: whether a known defect is specified
  as it is (so that a later fix shows as a spec change) or left out; and
  whether a baseline stops at the outside contract or also covers rule
  details.

## Related

- `2026-10-30-spec-domains.md`: domains for specs.
- `2026-10-24-add-index.md`: an index of the code's exported symbols.
- `2026-10-29-records.md`: the `Record:` trailer.
- `2026-10-31-refactoring.md`: restructuring planned from these facts.
- `2026-10-31-ub-mode.md`: metrics that could pick what to baseline.
- `2026-10-31-existing-project-adoption.md`: the rest of the same trial.

## Open questions

- Whether a report of capabilities and the files they touched is useful
  before a project has many changes, and how history from before Docket,
  which has no trailers, is covered.
- Whether co-change from history alone, without trailers, is worth
  computing in `dkt`, and how to keep it cheap on a long history.
- What the `baseline` path should ask an agent to do with such facts, and
  whether it needs a command of its own.
- Which outside tools, if any, to name as optional helpers, and for which
  languages.
