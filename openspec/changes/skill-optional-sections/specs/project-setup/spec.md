## MODIFIED Requirements

### Requirement: Files init writes
`dkt init` SHALL write these files shipped with the running dkt release:
- the schema under `openspec/schemas/docket/`: `schema.yaml` and the templates under `templates/`;
- the `schema` line and the `operations` section of `openspec/config.yaml`;
- every file of the docket skill, `SKILL.md` and the files next to it, in both `.claude/skills/docket/` and `.agents/skills/docket/`, the same in both;
- `docket/workflow.md`, the workflow template, only when it is missing, as workflow-file says.

#### Scenario: Fresh repository gets the files
- **WHEN** a user runs `dkt init --tools claude` in a repository without `openspec/`
- **THEN** each file listed above matches the shipped one, and each skill file is byte-for-byte identical in the two skill folders

#### Scenario: Skill folder from an earlier release
- **WHEN** both skill folders hold only an earlier release's `SKILL.md` and a user runs `dkt init`
- **THEN** each `SKILL.md` is listed as `updated`, each other skill file as `created`, every skill file matches the shipped one, and dkt exits with 0
