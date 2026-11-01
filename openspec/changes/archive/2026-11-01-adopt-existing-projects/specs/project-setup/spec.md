## ADDED Requirements

### Requirement: Ignored install paths
After writing, `dkt init` SHALL report `install-ignored` for each file it installs or keeps in the repository that git ignores and that is not tracked, naming the file and the rule that ignores it (the ignore file, its line and its pattern), and SHALL exit with 1. It SHALL still write such a file, and SHALL NOT change any ignore file.

#### Scenario: Ignored skill reported
- **WHEN** `.gitignore` holds `.claude/` on its third line and a user runs `dkt init`
- **THEN** each file under `.claude/skills/` is written, an `install-ignored` finding names each one with `.gitignore`, line 3 and `.claude/`, `.gitignore` is byte-for-byte unchanged, and the command exits with 1

#### Scenario: Tracked file not reported
- **WHEN** `.gitignore` holds `AGENTS.md` and `AGENTS.md` is already tracked by git
- **THEN** no `install-ignored` finding names `AGENTS.md`

#### Scenario: Nothing ignored
- **WHEN** no installed file is ignored and nothing else differs
- **THEN** `dkt init` reports no `install-ignored` finding and exits with 0
