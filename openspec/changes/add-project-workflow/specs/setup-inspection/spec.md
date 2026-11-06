# Spec Delta

## MODIFIED Requirements

### Requirement: Installed files inspected
`dkt inspect` SHALL check every file `dkt init` writes, by the rules `dkt init` uses, and list each in `result.files` with its `path`, `kind` and a `status`: `same` (equal to this release's file), `missing` (`dkt init` writes it), `outdated` (equal to an earlier release's file; `dkt init` updates it) or `differs` (`dkt init` keeps it; `dkt init --force` replaces it). `docket/workflow.md` is listed with the statuses workflow-file defines instead.

#### Scenario: Statuses listed
- **WHEN** a user runs `dkt inspect --json` in a repository set up by this release's `dkt init`
- **THEN** `result.files` lists every file `dkt init` writes, each with status `same`
