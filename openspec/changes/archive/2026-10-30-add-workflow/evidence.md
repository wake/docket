# Evidence: add-workflow

| Field | Value |
|---|---|
| Date | 2026-10-30 |
| Docket | dev |
| OpenSpec | 1.14.3 |
| Repository | git@github.com:wake/docket.git |
| HEAD | 8e30e643a31e61906f5d9edd36163e0e2d892560 |
| Title | A materialized workflow for each user |
| Kind | feature |
| Source | docket/ideas/2026-10-29-framework-skill.md, second of its three changes |
| Base | 14434891350493fe2b771e2ce5801097567873f1 |

## Task commits

| Task | Commit | Subject |
|---|---|---|
| 1 | 145c778d4e80a99a5c1c78f80fa2d4987ab1c03d | feat(schema): ship the workflow template |
| 2 | 10bc0a71303c5cd91d09448b35b0ecb6b0614482 | feat(workflow): locate, read and inspect the workflow file |
| 2 | 0f7aa3179fb49282d9d55afda19b8ed28cf61e05 | feat(cli): dkt workflow reports the workflow file and prints the template |
| 2 | b090ee0e3e4bda48db8edbea8c79aec362d02e47 | test(cli): the stale finding names the workflow file |
| 2 | c8f958e058eefb0f74fd433c949be81327eda250 | fix(workflow): a dangling link anywhere in the path is unreadable |
| 3 | 8176efe62e0b7ef22f75d985757a2f24acec956c | fix(init): report a linked skill directory once |
| 3 | 5e4d1c9278eba20fa178d6a4e593878c5eca77df | feat(skill): take each path's steps from the workflow file and add setup |
| 3 | d505b861a49dd41e99c401571a53c661f9e4e595 | fix(skill): setup brings an incomplete, older workflow up to date |
| 4 | ac6227cdf2130ce8631f54c5935217731b1b4112 | docs: describe the workflow file and dkt workflow |
| 4 | 5d7a801689889664bdba1eeb845a4b742b428337 | docs: upgrade note and the framework idea after add-workflow |
| 5 | e3ff563c0dc45c04acc87acfbc559690f3625248 | fix(skill): setup shows the whole file and reads no user file unasked |

## Checks

- `openspec validate add-workflow --strict`: Change 'add-workflow' is valid
- Base check: 8 requirements checked, none changed since the base
