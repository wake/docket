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

## Added by hand at release

Ten commits made for this change's tasks wrote `Change:` and `Task:` as
separate paragraphs of the message, so git does not read them as trailers
and the Task commits table above leaves them out. Each one names its task
on a `Task:` line of its body:

| Task | Commit | Subject |
|---|---|---|
| 1 | 324222914878f869b33edafe741ed15c6ef8491c | fix(change): reject an empty or non-mapping .openspec.yaml |
| 2 | 35fef260383ee405d9feeb715f26103541b66987 | fix(gitx): frame commit fields with NUL so control characters cannot corrupt them |
| 3 | e96570b67ced3a796e37e1f86481f0e9b3a27f83 | fix(archive): let the loader skip delta specs so sync ignores unreadable ones |
| 3 | a823dd2abacc7c73be44b231d380932ac9ffb150 | fix(archive): replace plan.md through a temporary file and rename |
| 3 | 6a01a66c04340fe695641660ce063e82d1ea4000 | fix(archive): refuse a read-only target and keep its mode when replacing a file |
| 4 | 18e91211497987effda03d6e09b8f35e995678fa | fix(archive): rebase loads deltas, writes through the shared helper, and the slug check is tested |
| 5 | e7b293474bbad5df4cea58f4c373eb69a236c879 | fix(archive): quote cross-check lines and reject malformed OpenSpec output |
| 6 | f087454717abc4d70a7aa891935b83a3dbc389be | fix(archive): drop query and fragment from the evidence origin, keep readings on a cross-check error |
| 6 | 084b555dc495ba3c327583bab34f529c18daf188 | fix(archive): create evidence.md with a link so an existing file is never replaced |
| 8 | af3644719b1910eb521fbd3875b2432fd1ebc4ae | docs: drop the in-place rewrite follow-up that sync and rebase now fix |
