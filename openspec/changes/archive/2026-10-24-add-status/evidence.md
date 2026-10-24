# Evidence: add-status

| Field | Value |
|---|---|
| Date | 2026-10-24 |
| Docket | v0.2.0 |
| OpenSpec | 1.14.3 |
| Repository | git@github.com:wake/docket.git |
| HEAD | 771448b9271217b3d108ee33f1a78dd6d5753192 |
| x-base | 152bff55cdfcc1490984aa9f3e22d39995a7007a |
| x-kind | feature |
| x-source | docs/ideas/2026-10-24-web-workbench.md, Order item 4: the status feed the workbench reads |

## Task commits

| Task | Commit | Subject |
|---|---|---|
| 1 | 4b6b8a0bc333d19f9551e59a37595c92df27e099 | feat(gitx): commit times, commits limited to a folder, uncommitted files under one |
| 1 | 7d033bd3427f93a7543818c056cfcfc822e5aba3 | refactor(gitx): share the status record loop and put the new tests in their named files |
| 1 | 5309dd888bd6540200e6efe2b536b37c20da8bc4 | fix(gitx): read commit times from epoch seconds so an odd zone cannot fail a range |
| 2 | ce91f083aa831a0592d1f815f61a99dff2864926 | refactor: one reading of Task trailers for sync and status |
| 3 | a9ba6986abc7b32a4790067e9e63e3acbaed89c5 | feat(status): phase and task states of a change |
| 3 | d640aefcee4965a01b5dc5884d53d2f3d39e9486 | fix(status): leave out cancelled bullets without a task number; cancelled-only plan is design |
| 4 | cce2db9ad483308dddda41349d8c9f7c476b6528 | feat(status): load every active change from its first source |
| 4 | 21e1946daec115210b75d4402baa2731040a5a59 | docs(status): say why Load drops a bad .openspec.yaml and unknown trailers |
| 4 | fb22b238fec98fdbbef2d48e075a0f6e4b30435d | test(status): pin Load on an unusable .openspec.yaml |
| 5 | 048f8ebbbca4114e21d6667aed1e536f1e10eb75 | feat: dkt status lists every active change with its phase, tasks and activity |
| 5 | 8c12c94002d562a66182e500bb6ff74dc8b0a95c | test(status): cover idle days and active-and-archived slug; inline scanUnreadableSource |
| 5 | be3677cb9893b9ee61e6723b09b979c9eb7d9188 | test(cli): date dkt status tests away from any real day |
| 5 | 732e5fce22ab8856beb71a67c09fe761c4ec118a | fix(cli): do not call a slug inactive while a whole source is unreadable |
| 6 | 1b1b36a627eeeb61f060a4ddaaceda9faeb21905 | docs: map the status package and record its follow-ups |
| 6 | db0221066ac03b139c06005179b453b1045720e5 | docs(plan): tick Task 6 steps |
| 6 | b866d5aa4efb783c66cfc03de0ed8f222f89ea17 | docs: the first seven commands completed the first milestone |

## Checks

- `openspec validate add-status --strict`: Change 'add-status' is valid
- Base check: 6 requirements checked, none changed since x-base
