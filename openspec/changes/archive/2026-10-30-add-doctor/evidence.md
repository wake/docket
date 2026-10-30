# Evidence: add-doctor

| Field | Value |
|---|---|
| Date | 2026-10-30 |
| Docket | dev |
| OpenSpec | 1.14.3 |
| Repository | git@github.com:wake/docket.git |
| HEAD | 56cde04977b71e413cb637f2f20b4c580cf59526 |
| Title | Check a project's Docket setup |
| Kind | feature |
| Source | docket/ideas/2026-10-29-framework-skill.md |
| Base | 7df035726138ca1ce188421a5d2c287489e51ddb |

## Task commits

| Task | Commit | Subject |
|---|---|---|
| 1 | 9ee421c3d6931222a82434b12866d699900b36ae | refactor(setup): split init's decisions from its writes |
| 1 | 7e55418527984c10ab381fea5f36fb4d61bca95d | feat(inspect): report the installed files |
| 1 | b3f98a83adccdcee91ca8a01cafef326708e8446 | fix(inspect): check the installed files when openspec is a link |
| 2 | a61d0340c529d25c3ef8cf680cb01e2bcc124bd3 | refactor(workflow): share the workflow report |
| 2 | 2246600d6702b1b048a684897d026a798591465d | feat(inspect): report OpenSpec and the workflow file |
| 2 | e2a270502b4f50d7bad890972a164b6a42dad771 | test(inspect): text output of inspect and workflow |
| 2 | 7bcecc9390be4002d1b8162a05b56b6dc667369d | refactor(inspect): name the finding sets |
| 3 | a642e1b903e4ba5cde980b379d1841d380ff82a2 | feat(inspect): check the trailers of changes on the default branch |
| 3 | d4951ca212b5e515eea96e71242ff70ca3276e81 | test(inspect): metadata errors and the skipped line |
| 4 | 81149b03d52cc3c8e3fa4f367effb1bd6248fd49 | refactor(setup): take the skills by name |
| 4 | 28bd0d69974659cea124a93c6389c6cb755db030 | feat(skill): setup and doctor skills, installed by dkt init |
| 4 | cdc638a03b449a6636e7e93032d54b8c67441308 | fix(skill): doctor fixes installed files together; no home directory |
| 5 | b79673b84a2ed3354cdcf8af3721640a2a2d8580 | docs: dkt inspect, the setup and doctor skills |
| 5 | cb498a735611d1bb976c15ffe3b3d79ea17b50d9 | docs: the skills in the README; inspect comment |
| 6 | none | |

## Checks

- `openspec validate add-doctor --strict`: Change 'add-doctor' is valid
- Base check: 11 requirements checked, none changed since the base
