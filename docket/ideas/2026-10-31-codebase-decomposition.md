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

## Related

- `2026-10-30-spec-domains.md`: domains for specs.
- `2026-10-24-add-index.md`: an index of the code's exported symbols.
- `2026-10-29-records.md`: the `Record:` trailer.
- `2026-10-31-refactoring.md`: restructuring planned from these facts.

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
