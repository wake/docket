# Spec Delta

## MODIFIED Requirements

### Requirement: Files init writes
`dkt init` SHALL write these files shipped with the running dkt release:
- the schema under `openspec/schemas/docket/`: `schema.yaml` and the templates under `templates/`;
- the `schema` line and the `operations` section of `openspec/config.yaml`;
- `.claude/skills/docket/SKILL.md` and `.agents/skills/docket/SKILL.md`, which are the same;
- `docket/workflow.md`, the workflow template, only when it is missing, as workflow-file says.

#### Scenario: Fresh repository gets the files
- **WHEN** a user runs `dkt init --tools claude` in a repository without `openspec/`
- **THEN** each file listed above matches the shipped one, and the two `SKILL.md` files are byte-for-byte identical

### Requirement: Existing files
For each shipped file that already exists, apart from `openspec/config.yaml` and `docket/workflow.md`, `dkt init` SHALL:
- leave it alone and list it as `same` when it equals this release's file;
- replace it with this release's file and list it as `updated` when it equals what an earlier release, v0.16.0 or later, shipped;
- otherwise keep it, list it as `kept`, report `schema-differs` or `skill-differs`, and exit with 1; with `--force`, replace it instead and list it as `overwritten`.

#### Scenario: Re-run with nothing changed
- **WHEN** a user runs `dkt init`, changes nothing, and runs it again
- **THEN** the second run lists every file as `same`, apart from `docket/workflow.md`, which it lists as `kept`, writes nothing, and exits with 0

#### Scenario: File from an earlier release
- **WHEN** `openspec/schemas/docket/templates/plan.md` equals the file an earlier release shipped
- **THEN** it is replaced with this release's file and listed as `updated`, and dkt exits with 0

#### Scenario: Edited skill kept
- **WHEN** `.claude/skills/docket/SKILL.md` differs from every shipped version and `--force` is absent
- **THEN** the file is unchanged and listed as `kept`, the finding `skill-differs` names it, and dkt exits with 1

#### Scenario: Edited skill forced
- **WHEN** the same file differs and a user runs `dkt init --force`
- **THEN** the file matches the shipped one, is listed as `overwritten`, and dkt exits with 0

### Requirement: Init result
The `result` of `dkt init` SHALL hold `files` and `openspec_init`. Each entry of `files` has a `path` relative to the repository root, a `kind` (`schema`, `config`, `skill`, `workflow`, or `openspec` for a file `openspec init` created or changed) and a `status` (`created`, `updated`, `overwritten`, `same` or `kept`). `openspec_init` is the `--tools` value passed to `openspec init`, or null when it did not run. Without `--json`, the output SHALL list each file with its status.

#### Scenario: Files listed
- **WHEN** a user runs `dkt init --json --tools claude` in a repository without `openspec/`
- **THEN** `result.openspec_init` is `claude`, and `result.files` lists the files `openspec init` created with kind `openspec`, and Docket's files with status `created`

#### Scenario: Nothing to do
- **WHEN** a user runs `dkt init --json` in a repository already set up and unchanged since
- **THEN** `result.openspec_init` is null and every entry of `result.files` has status `same`, apart from `docket/workflow.md`, which has status `kept`
