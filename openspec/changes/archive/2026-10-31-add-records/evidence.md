# Evidence: add-records

| Field | Value |
|---|---|
| Date | 2026-10-31 |
| Docket | dev |
| OpenSpec | 1.14.3 |
| Repository | git@github.com:wake/docket.git |
| HEAD | 942ca96ed081bd8497d7ffedbe28411fa4707f81 |
| Title | Records for work that opens no change |
| Kind | feature |
| Source | docket/ideas/2026-10-29-records.md |
| Base | 68294e24d0388daa7191c3a8adb5aebd94e36a2d |

## Task commits

| Task | Commit | Subject |
|---|---|---|
| 1 | 8648e65c26fe51efdbd02bce11e65c64615dcf04 | refactor(new): build the used-slug places in one function |
| 1 | 04c3c0c33c7ae4d611439cc4f7c3ec2d8fec3a94 | feat(new): take a free suffix for a used slug; drop the kind fix |
| 2 | eedde36a9eb1b42852d37901f7debe7be36ad786 | feat(scan): list the records of every source |
| 2 | 0f70bf4cc229d96081234204df81e0bba96b37ea | test(scan): pin record names and folder depth |
| 3 | 297af5ca512c875f07904ee88b9c290b2e49220f | feat(record): dkt record writes a record from the shipped template |
| 3 | 0d5f1b454a64b862ebef9541056e1c239956eef5 | test(record): pin the suffix limit, a taken name and the reported path |
| 3 | d7e90793a7ef4581be26a528499f47ab2472d5a3 | fix(record): create the records folder only when writing |
| 4 | 40d0b59ea28c2e38b8ceb8588dd6c4100d64bdcb | refactor(lint): share the task heading and placeholder checks |
| 4 | ff8699a3f7bd5ecb8891ce72afeac1172e67c148 | feat(lint): check the records the branch adds |
| 4 | 0b5fcab00871387c6b11c7d0fec818919172875c | test(lint): pin record word counting, sections, tasks and template text |
| 5 | 0cb0a059b8140375c11c85e9d723e2866db019fe | docs(skill): keep a record on the light review path |
| 6 | 2af25217e06ea08ac5eb9b16720c6580938fce35 | docs: records and dkt record |
| 6 | 092c7910b933f1f3188d1ccb76668e3ff87f45e2 | docs: place the records note for both light-review paths |
| 7 | 00a7b8ed69f6dd77f17e785c9d9b64ec13518eea | docs(skill): trailers in the last paragraph; the record's full slug |

## Checks

- `openspec validate add-records --strict`: Change 'add-records' is valid
- Base check: 12 requirements checked, none changed since the base
