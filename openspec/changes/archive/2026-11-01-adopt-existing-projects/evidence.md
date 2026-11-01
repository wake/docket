# Evidence: adopt-existing-projects

| Field | Value |
|---|---|
| Date | 2026-11-01 |
| Docket | v0.13.0 |
| OpenSpec | 1.14.3 |
| Repository | git@github.com:wake/docket.git |
| HEAD | d422a6d0c85da61118d44da94c7b2c109246d468 |
| Title | Adopt Docket in a project that already has its own files and OpenSpec changes |
| Kind | feature |
| Source | docket/ideas/2026-10-31-existing-project-adoption.md |
| Base | cd949431b7149a4cea8a9601c07df55ed42d669f |

## Task commits

| Task | Commit | Subject |
|---|---|---|
| 1 | 19907f6a3e57aaf208baf93cd7cb1efaf46b8a8d | feat(change): read a change's schema and count checkboxes |
| 1 | 1255a22c3716d3d17242d9c5cadfcb5399e275a7 | test(change): cover an empty schema value in ReadSchema |
| 1 | a446a1326ab84f3cb6ec8a7db01c756c526aaec8 | style(change): gofmt metadata_test.go |
| 2 | c354f340654ee99b898c9ea2d3bac16ac19752bc | feat(setup): report installed files that git ignores |
| 3 | 46fd8c8856e090ae766cc46fd9a8ec23cca44d1e | refactor(lint): split the title checks |
| 3 | aa3822cca9ec5b5acc39dce6b1a4f1ac6c1e604b | feat(lint): lint a change of another schema |
| 4 | a0e4b51e63325fd892f3f6107f20bf96e9c78f7f | feat(inspect): report main-spec titles and changes of another schema |
| 4 | 9cf4fb762dde9631a9616cddc0c2068200a04ad9 | fix(inspect): skip the spec scans only for a linked openspec |
| 4 | 24fd02cb3adbcc480b810af20a974fa2e700646f | test(inspect): a linked openspec holding specs and changes is not read |
| 5 | 22b50a4306db0c98538a12438a6aa8161f83ca4d | feat(status): show the schema of each change |
| 6 | 3aa58a95da9fb230cb47491045cc6091dcd00789 | feat(archive): archive a change of another schema, refuse sync and rebase |
| 6 | f89fbf60074d4a9ef5423b07014d0ef63edf1430 | fix(archive): write none for the base of a change of another schema |
| 7 | f1abf32da8f9781cf68a5eb91d95b6a6bfe27af2 | feat(doctor): adoption steps and the conversion guide |
| 7 | d195a716ee5be5a1675f793c1810c8308208e550 | fix(doctor): match the other-schema message when proposing conversion |
| 8 | ed82259aecf4cac75654878aa07a07079cce8cd9 | docs: adopting Docket in an existing project |

## Checks

- `openspec validate adopt-existing-projects --strict`: Change 'adopt-existing-projects' is valid
- Base check: 14 requirements checked, none changed since the base
