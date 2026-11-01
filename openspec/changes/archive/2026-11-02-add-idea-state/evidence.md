# Evidence: add-idea-state

| Field | Value |
|---|---|
| Date | 2026-11-02 |
| Docket | v0.14.0 |
| OpenSpec | 1.14.3 |
| Repository | git@github.com:wake/docket.git |
| HEAD | 17351834db8858a13762131ea8dbd8e5fec8e4b4 |
| Schema | docket |
| Skipped steps | none |
| Title | Ideas with derived state, and the order checked |
| Kind | feature |
| Source | docket/ideas/2026-10-29-ideas-and-order.md |
| Base | dc05392968629bb60dae3fed3de03c0fe835ff89 |

## Task commits

| Task | Commit | Subject |
|---|---|---|
| 1 | b8f0c2bc82a3173e6b2e6d7ef56772aeef59f232 | feat(change): a source names the path in its first field |
| 2 | ad337ab08ba4f5478ffe49e5af2411e2ef61d15d | refactor(lint): rename checkIssueSource to checkSources |
| 2 | 5257a352808e68d51015fef46c206851c024a740 | feat(lint): check the order and the ideas |
| 2 | c75087926e50a2fd92812c3ce3eafa6a4df7eb5e | fix(lint): read no ideas through a linked folder and check every order file |
| 3 | 034d9418f8248ae7ef1f2dbae1c5216c30b02075 | refactor(status): find handlers for any set of paths |
| 3 | 9b5cb3d6223b95b8b915445e1b3c8a989a2117c7 | feat(status): report the order, the next work and the ideas |
| 3 | 71a26a154e423077a6ed650740dc3df790fdf0f7 | docs(status): say what LoadBacklog loads |
| 4 | 71b3e59528871f9c2bf833eb7797ec53b5b745e7 | docs(skill): ideas and the order have one writer |

## Checks

- `openspec validate add-idea-state --strict`: Change 'add-idea-state' is valid
- Base check: 17 requirements checked, none changed since the base
