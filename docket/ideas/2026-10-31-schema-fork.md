---
title: Let a project fork the shipped schema
---

Recorded 2026-10-31, split out of `2026-10-31-global-schema.md`: that
change installs the shipped schema once per machine, and the user left
forks for later.

## Decided

- The headings `dkt lint` reads are fixed English keywords, as OpenSpec's
  delta markers are (`## ADDED Requirements`, `### Requirement:`,
  `#### Scenario:`). The text under them can be in any language. A fork
  may change the rest of the schema, but not those headings.
- Headings in other languages were considered and dropped: a table of
  aliases per language, as Gherkin's `# language:` header does, or hidden
  markers in place of headings. An alias table could still be added later
  without changing the file format.

## What is known

- `openspec schema fork <source> [name]` copies a schema into the
  project's `openspec/schemas/` under a new name, `<source>-custom` by
  default, and does not record what it was forked from.
- A project would name its fork in `openspec/config.yaml`, and `dkt new`
  would use the schema `config.yaml` names instead of passing
  `--schema docket`.
- A project's copy of `docket` that equals no shipped version has been
  edited in place; `docket-doctor` could help turn it into a fork.

## Related

- `2026-10-31-global-schema.md`: the shipped schema installed once per
  machine.

## Open questions

- How `dkt` tells that a fork is Docket's, since the fork does not say
  what it came from.
- How a project learns that the shipped schema changed after it forked.
- Whether the record and workflow templates, already in the binary, get a
  project override too.
