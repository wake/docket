# Evidence: add-record-tracking

| Field | Value |
|---|---|
| Date | 2026-11-01 |
| Docket | v0.10.1-0.20261101021001-a8a89cbc7fd2 |
| OpenSpec | 1.14.3 |
| Repository | git@github.com:wake/docket.git |
| HEAD | a8a89cbc7fd2370fd5b6421596df5091ccbe411f |
| Title | Track records in the scan, sync and status |
| Kind | feature |
| Source | docket/ideas/2026-10-29-records.md (order item 7), with its follow-ups from the review of add-records |
| Base | 19de473f38aab3be3c8940b3c170815efa54021c |

## Task commits

| Task | Commit | Subject |
|---|---|---|
| 1 | 416a087b1f7f42ca022ad4a3645a7c5edbd812cf | refactor(change): match Task trailers over commits already chosen |
| 1 | bc7dd641af7c249974bda6392e41a809f6943c1e | feat(gitx, record): read Record trailers |
| 2 | 33e5e5cada494767b6a4ed1e08c7c25462042f86 | feat(sync): tick a record's tasks from its commits |
| 2 | 1827272bfe8955165f7a75d5168f5c694da78509 | fix(sync): check a record slug before looking for the default branch |
| 2 | 5c1cbca9b5085fae1c4c3beca8aa6f3508e01421 | refactor(archive): one tick-and-write tail for plans and records |
| 3 | 70d35ad0f807665e95c9bb0c36dbb767c84cff06 | feat(status): list open records and named ones |
| 4 | c9c7140f402adde7d4493660bc29a48f16023559 | feat(status): merged records with their merge, tag and filters |
| 4 | 46dbe8f6ace4acd95c039baa2025beb575cdc6a6 | fix(status): refuse a --since that starts with a dash |
| 4 | dc01ce97fb5ec25f8fb5a8893df1bd8e088ef392 | fix(status): keep the merge commit's own time zone |
| 5 | 962457a16a8c64b361a435eff78cfe8fc7b16739 | feat(lint): one look at the branch, and Record trailers that name no record |
| 6 | 048a4480347a1a228734b74fa2ae9bb411ee5ebd | fix(record): count the prolonged sound mark as half a word |
| 7 | 1e1fd287fd611e2b8ebc41beebd9855653138085 | fix(scan): list a worktree unreadable as a whole once |
| 8 | 62b38c024ecdf87610bcf6c4a47d639a82020f51 | fix(record): leave no folder after a failed write |
| 8 | 273c052b0387559e60d400352e9ad97206113540 | fix(record): clean up after any failure once folders are made |
| 8 | f006d765179b0620d2f6932fa9e42ce2e1028a25 | fix(record): own only the folders this run made |
| 9 | 8bc6dd2f194a20beb3974ed460eef50de211fb6a | docs(skill): sync a record's tasks before lint |
| 10 | 58ad09a3c4710e71dc8912b0236afab9862138d9 | docs: records in status and sync |

## Checks

- `openspec validate add-record-tracking --strict`: Change 'add-record-tracking' is valid
- Base check: 17 requirements checked, none changed since the base
