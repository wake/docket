## MODIFIED Requirements

### Requirement: Index form
The index SHALL hold one JSON object whose `projects` is an array of entries, each with all four of `path` (an absolute path, held by no other entry), `origin` (a string or null), `added` (an RFC 3339 time in UTC) and `ignored` (a boolean). An entry MAY also hold `mode` (`sync` or `independent`) and `synced` (an object whose values are strings), as `user-workflow` defines them; dkt SHALL keep both when it writes the index.

#### Scenario: Entry written
- **WHEN** a project is added at `2026-11-08T17:10:11+08:00`
- **THEN** its entry holds `path`, `origin`, `added` `2026-11-08T09:10:11Z` and `ignored` false

#### Scenario: Mode kept on another write
- **WHEN** one entry has `mode` `sync` and `synced`, and a user runs `dkt projects add` in another Docket project
- **THEN** the first entry still holds the same `mode` and `synced`

#### Scenario: Unknown mode in the index
- **WHEN** an entry has `mode` `always` and a user runs `dkt projects ls --json`
- **THEN** `unreadable` lists the index and dkt exits with 3
