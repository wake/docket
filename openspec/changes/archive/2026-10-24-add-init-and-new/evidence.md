# Evidence: add-init-and-new

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
| 1 | 3a42475dad61e8ecd94c384bc30294eddb2e5e77 | feat: embed the docket schema and the routing section |
| 2 | e2a15168af322d397ad7e66aae9bf68437477c2c | feat(gitx): run git with a given environment |
| 2 | bc5fd0c234382ba13b3fd7031a16e5ff93ba2f1b | fix(gitx): detect a missing work tree without parsing messages |
| 2 | e0ce078b0b27240771944f846478a449048e121b | fix(testutil): build the test environment from an allowlist |
| 2 | 5a2709371650996510007df0da5acf99396cf2c7 | fix(gitx): resolve PATH against Dir and keep real git failures |
| 2 | 1805c247c1c948512b000b30259e4f456d8e3764 | fix(gitx): read worktree paths NUL-separated |
| 2 | b6342aff2ad262e3ec5d455d76e3aa9b12d8b132 | fix(gitx): stop the .git scan where git's discovery stops |
| 3 | a170bfa93d1424c53ee0fb8d18a626e7c9db492c | feat(cli): result envelope, exit codes and dispatch |
| 3 | 9b7e320409533132e94a1b89640b234508539b99 | fix(cli): one JSON envelope for every --json run |
| 3 | cf6d4ce95a804243504b211cbfed8b24e2a4964f | fix(cli): classify late probe failures, honor --, keep JSON on marshal failure |
| 3 | 7866dad0737b07b6d3db797816bbd57c655031d8 | fix(cli): tell a -- flag value from the terminator |
| 4 | 357962deed8da2ad2fb85936f15629fbe345496f | feat(openspec): runner with a version gate |
| 4 | 0e5e0c581983fd07cc12ee0ab8b7e2fcb797444b | fix(openspec): resolve the program to an absolute path against Dir |
| 4 | 6e24c8ff245ad0138992eaff42f72f69c0559ba4 | fix(openspec): treat an unusable CLI as unavailable and name DOCKET_OPENSPEC |
| 5 | e745da71d84166a5f46398f390a1b882a40e4f46 | feat(scan): find changes in worktrees and branches |
| 5 | be8ec8edf52c33452eefd637ce5acf8ae94764f9 | fix(scan): rank sources by whether the default branch contains them |
| 5 | 600382d5f50acd950be5cb3f9992155c1ba63e06 | fix(scan): fall back to main or master when origin/HEAD is dangling |
| 5 | e2ccaa1c8068814bfd904f7f1846cbc769230e05 | fix(scan): report a worktree whose ancestry check fails as unreadable |
| 5 | 867074ceb0c28f7f43563d4bdbdc831b87340dbc | fix(scan): keep ReadFile inside the repository |
| 6 | 5e7bb2bf2f7f56dad78db243a6cb121c83726efb | feat: dkt init installs the docket schema |
| 6 | b45a4d0d4f1241858d60bb1515de312dcc5c1c77 | fix(cli): report a failing openspec init as openspec-failed |
| 6 | e3c61d14e8085820ccdfab555b7a2e468841c5f1 | fix(setup): refuse to write the schema through symbolic links |
| 6 | 6bd71a30e8c368c62fdf24d6ee708a1ecc08b6cd | fix(setup): compare schema files without opening them for writing |
| 7 | afd5cdca7495b7657aa1a9cbda472b194b3c9b69 | feat: dkt new opens a change with its kind, source and base |
| 7 | 5b4479bc02df3afd3c589483fb08db8e96882736 | chore: mark yaml dependency as direct |
| 7 | b46ee67df87954ce4a4c6c1b16dd58df5fd6f5d3 | fix(cli): reserve archive and keep folders that existed before |
| 7 | b945064b4028828cadbddcaf5fb73165362af36e | fix(cli): serialize dkt new and refuse symlinked change paths |
| 7 | da6a41e05b4e92da890706bfa30159b0cc5d09a4 | fix(cli): let dkt new remove only what it created under the lock |
| 7 | cac2e798db847ecc9440194db215af8797efb3af | fix(cli): do not clean up through a link that appeared during OpenSpec |
| 7 | e70d334a7bc514c89b0bff461681e8771ed64638 | fix(cli): check for links before any cleanup in dkt new |
| 8 | 44707bb72348d3087de8f8ddbc28868ab0cb3781 | docs: architecture map, CI and install note |
| 8 | 0b5c238be5819b2d56c47803f7289c09098de946 | ci: use the current action majors |
| 8 | c079069930f2d7f7a89c013e2fedc50b1933cf87 | docs: add internal/safepath to the architecture map |
| 8 | 24920a6c8064fb3dbb01ebf634af2b3bd6316776 | ci: give the workflow a read-only token |
| 8 | d60cc352625db03ed837952e099a4cd20e918a46 | docs: say which dkt commands exist and record deferred follow-ups |

## Checks

- `openspec validate add-init-and-new --strict`: Change 'add-init-and-new' is valid
- Base check: 21 requirements checked, none changed since x-base
