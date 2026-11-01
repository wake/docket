## ADDED Requirements

### Requirement: Path a source names
The path that the `source` of a change or a record names SHALL be the source, without leading and trailing whitespace, up to its first whitespace, without one trailing comma. Rules that look for an issue or an idea by its path SHALL use this path, so a note may follow it.

#### Scenario: Path alone
- **WHEN** the default branch tip holds the idea `docket/ideas/2026-11-02-x.md` and the record `2026-11-02-fix-x`, whose source is `docket/ideas/2026-11-02-x.md`
- **THEN** the record is a handler of that idea

#### Scenario: Path with a note
- **WHEN** the default branch tip holds the idea `docket/ideas/2026-11-02-x.md`, a record whose source is `docket/ideas/2026-11-02-x.md, item 1 of docket/order.md`, and an archived change whose source is `docket/ideas/2026-11-02-x.md (order 5)`
- **THEN** both are handlers of that idea
