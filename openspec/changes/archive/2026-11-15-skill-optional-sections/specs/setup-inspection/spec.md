## MODIFIED Requirements

### Requirement: Installed file findings
For each file that is not `same`, `dkt inspect` SHALL report a finding named after its kind and status: `schema-missing`, `skill-missing`, `schema-outdated`, `skill-outdated`, `schema-differs` or `skill-differs`, naming the file.

#### Scenario: Before init
- **WHEN** a user runs `dkt inspect --json` in a repository without `openspec/`
- **THEN** the findings hold `schema-missing` and `skill-missing`, and dkt exits with 1

#### Scenario: Edited skill
- **WHEN** `.claude/skills/docket/SKILL.md` differs from every shipped version
- **THEN** `result.files` lists it as `differs` and the finding `skill-differs` names it

#### Scenario: File from an earlier release
- **WHEN** `openspec/schemas/docket/templates/plan.md` equals the file an earlier release shipped
- **THEN** `result.files` lists it as `outdated` and the finding `schema-outdated` names it

#### Scenario: Skill folder from an earlier release inspected
- **WHEN** both skill folders hold only an earlier release's `SKILL.md`
- **THEN** `skill-outdated` names each `SKILL.md`, `skill-missing` names each other skill file in both folders, and dkt exits with 1
