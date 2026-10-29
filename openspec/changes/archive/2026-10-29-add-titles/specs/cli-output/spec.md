# Spec Delta

## MODIFIED Requirements

### Requirement: JSON output
Every command SHALL accept `--json`. With `--json`, the command SHALL write exactly one JSON object to standard output and nothing else, on success and on failure alike, and SHALL write diagnostics to standard error. Without `--json`, output SHALL be human-readable text.

#### Scenario: JSON on success
- **WHEN** a user runs `dkt init --json` in a repository where it succeeds
- **THEN** standard output parses as a single JSON object whose `ok` is true

#### Scenario: JSON on failure
- **WHEN** a user runs `dkt new Bad_Slug --kind feature --source x --title X --json`
- **THEN** standard output parses as a single JSON object whose `ok` is false and whose findings name the invalid slug
