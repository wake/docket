# Evidence: add-conflicts

| Field | Value |
|---|---|
| Date | 2026-10-24 |
| Docket | v0.1.0 |
| OpenSpec | 1.14.3 |
| Repository | git@github.com:wake/docket.git |
| HEAD | 4fcc4f4d80b6d728fc0ee0e6ccfc4ac796851985 |
| x-base | 4a3efff2613e3de2d09c55b0a53e50390c28cc66 |
| x-kind | feature |
| x-source | self: milestone 1 design discussion, 2026-10-24 |

## Task commits

| Task | Commit | Subject |
|---|---|---|
| 1 | 654483f16c5a856dce5d9773eeb1f8ccd6b26e95 | feat(scan): read every file below a folder from any scanned source |
| 1 | 52e7e3c205164c3ac4d7df2109812368ce05e1eb | fix(scan): report a dir that is not a folder as unreadable from every source |
| 2 | 616afd852b6bab987c3dba79f54ca2e812bcdbf8 | refactor: one rule for the capability path of a delta spec |
| 3 | 96fa73ddce1c2cc638ede0fce14a61b9d6808d73 | feat(conflicts): find requirements that more than one change touches |
| 3 | 555ce4afe0812dc56f0471fcf5652f6a53699fe4 | fix(conflicts): pin each entry's sources and never leave them nil |
| 3 | 69b0dde1e6932326269f7cd2872ee038f7b24c6e | fix(conflicts): give each entry its own copy of its change's sources |
| 4 | d85a2dabba5c499a2e1f8e3080461a0c7c96a802 | feat(conflicts): load every active change's deltas from its first source |
| 5 | 6acb2272832eba4483da47e14286983943bc3839 | feat: dkt conflicts lists requirements touched by more than one change |
| 5 | 2c92a64cdced56ba5c4dfa7ed593264f1175eea0 | test(cli): make dkt conflicts' no-write and usage tests fail without the command |
| 5 | 5a657a13ea7e3ef029ec158b39c30f8d4f73883a | fix(cli): keep the scan result when a valid slug is not an active change |
| 6 | 29bfe91dd328525e1f541f96c547f33112651ac1 | docs: map the conflicts package and close the first milestone |
| 6 | 63ea4435f103e1df7e1cb11e46ea7d25c01e3857 | docs: record the add-conflicts review follow-ups |

## Checks

- `openspec validate add-conflicts --strict`: Change 'add-conflicts' is valid
- Base check: 4 requirements checked, none changed since x-base
