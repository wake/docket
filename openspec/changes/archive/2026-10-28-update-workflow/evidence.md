# Evidence: update-workflow

| Field | Value |
|---|---|
| Date | 2026-10-28 |
| Docket | v0.3.0 |
| OpenSpec | 1.14.3 |
| Repository | git@github.com:wake/docket.git |
| HEAD | aa13793ff08a7d2559aef428b83d8f513db5af62 |
| x-base | e967bbbe7a35168f8169fd29521ff8790ce54342 |
| x-kind | change-request |
| x-source | Development process adopted 2026-10-28 (review at both ends, three review modes, one pull request per change) |

## Task commits

| Task | Commit | Subject |
|---|---|---|
| 1 | a504e56cea855add06f10b19c67f716fadca6684 | docs(schema): review at both ends in the shipped workflow |
| 1 | c1e86e177ab8c7821dc2288896f79c4ead530f95 | docs(schema): review the plan before code; checks only for code tasks |
| 2 | b42a3e45852483bd06363222db956c85a77f9552 | docs: describe review at both ends in the READMEs |
| 2 | caab7a037ee127109020edd15ec56219e9da6b85 | docs: mechanical checks apply to tasks that change code |

## Checks

- `openspec validate update-workflow --strict`: Change 'update-workflow' is valid
- Base check: 0 requirements checked, none changed since x-base
