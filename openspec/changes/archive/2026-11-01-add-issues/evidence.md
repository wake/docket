# Evidence: add-issues

| Field | Value |
|---|---|
| Date | 2026-11-01 |
| Docket | dev |
| OpenSpec | 1.14.3 |
| Repository | git@github.com:wake/docket.git |
| HEAD | f3d5cd8f9c1934f118e1039a80c8d8e8e9c8f8d5 |
| Title | Issues, apart from ideas |
| Kind | feature |
| Source | docket/ideas/2026-11-01-issues.md |
| Base | e52c6977830a1f41ad948ead45a9ca925bf92881 |

## Task commits

| Task | Commit | Subject |
|---|---|---|
| 1 | dd181a79a5b9451c45ab49116b830c637d959402 | refactor(record): share the parser, the writer and the section checks |
| 1 | 65b6bb9771bb6af26bccf0e8bf4bf040d06d4114 | test(change): DecodeMapping rejects empty and scalar input |
| 2 | 2d7393d7c618d621aa915fc1309ae707db142e31 | refactor(scan): read dated files from a list of folders |
| 2 | 1a07d465362de1d8574a682f55969cbd70be3314 | feat(scan): find file issues |
| 3 | 97dd96ba07db62c400b6458f0249288dc70aa6da | feat(issue): dkt issue writes a file issue from the shipped template |
| 4 | 05d17fe0311c65a04983ffc04d5578653991b0f7 | feat(lint): check file issues and the sources that name them |
| 4 | ec5704d27f206c5683674b2d45140f96aaf19eaa | fix(lint): count only regular files as issue and idea sources |
| 5 | 150eee06f8eef7db7ced07274ce6245fa3c115c7 | feat(status): list file issues with the work that names them |
| 5 | bb9c9e2f9d97f4f9f4a167a531beec7cd5bd990f | fix(status): read change metadata once per source, one handler per change slug |
| 5 | b2e8838385923419de066add13074da158eb6eec | fix(status): count only the tip's copies of a change the tip holds |
| 6 | dee122a06f69e2e639d2d4f361c1f1a13e2e2bef | docs(skill): send settled work to an issue |
| 6 | 8718499da09c02d070ceda77d09f53efed408053 | docs: issues in the README and the architecture notes |
| 6 | b39f17290bdf05799cb5edf8f2f760f4b9fb3272 | docs: what lint and status do with issues |

## Checks

- `openspec validate add-issues --strict`: Change 'add-issues' is valid
- Base check: 16 requirements checked, none changed since the base
