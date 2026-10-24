# Evidence: add-archive

| Field | Value |
|---|---|
| Date | 2026-10-24 |
| Docket | v0.0.0-20261024110645-b71f1f05053e |
| OpenSpec | 1.14.3 |
| Repository | git@github.com:wake/docket.git |
| HEAD | b71f1f05053edb536c033cb0b17f8e153a33595c |
| x-base | 538f530999cd099103939e45b07660c36356e829 |
| x-kind | feature |
| x-source | self: milestone 1 design discussion, 2026-10-24 |

## Task commits

| Task | Commit | Subject |
|---|---|---|
| 1 | 6c87fbd98c28e5bdd3f317d81b938f978de68d99 | feat(change): read .openspec.yaml and tick plan steps in place |
| 2 | c0bb1880eb3b48ad66bb57abea914bf94eac44fe | feat(gitx): list commits with trailers, uncommitted paths and optional files |
| 3 | 9f07007a8fb62cc46c46f4082d7992065f9b2054 | feat: dkt sync ticks plan.md from commit trailers |
| 3 | d7bd159a51acc93e0fa00d2407d4a249692e0e93 | test(cli): pin dkt sync's usage messages |
| 4 | 337c3e76c06901150b969823c5012c57d641023b | feat: dkt rebase moves x-base and lists what changed underneath |
| 4 | 83d9ac96d1cffec83796e1ba9c8d1e2dfc40b0db | test(archive): pin the worktree side of the base check and the rebase write guard |
| 4 | e98e664520f0fdc74d6a175d09a2306090798ae8 | refactor: share the HEAD and default-branch lookups of sync, rebase and new |
| 5 | de7e2827cd1d7ec8b783ff1ce1fef12f25cbecc8 | feat(archive): compare Docket's reading of a change with OpenSpec's |
| 6 | 6dc5bc42972761ff67f6b2ad9e917b2bd04a130d | feat(archive): check, archive and record evidence for each change |
| 6 | b793ca942220f4a71c5fe47c6454020e020839d0 | fix(archive): keep credentials and local paths out of evidence.md |
| 6 | 54757690a6657cad27259b59c0a368228fc197e6 | fix(archive): report a change OpenSpec archived under an unexpected name |
| 7 | f1b276403d65ae60f1c1a895ddad5144e1ef82e0 | feat: dkt archive checks a change before OpenSpec merges it |
| 7 | 56aa72c6800c8a872ced26394360d42238cbc638 | test(cli): pin dkt archive's usage error, version-first order and human lines |
| 7 | a8072c9c97785cc35748410d7ea6d55f2f0d61ce | fix(cli): keep dkt archive's result on errors and refuse repeated slugs |
| 8 | 451bc7f658c0ca2011a4c4786250b11b6ca2cc48 | docs: map the archive package and say which commands exist |
| 8 | f83766f478f9f7b1ada762a424eee36bc87f7d35 | docs: tick the add-archive plan with dkt sync |
| 8 | 3834a3af612295d0a892e96a92bcf86e733edd69 | docs: record two follow-ups from the add-archive review |

## Checks

- `openspec validate add-archive --strict`: Change 'add-archive' is valid
- Base check: 12 requirements checked, none changed since x-base
