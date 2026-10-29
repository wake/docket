# Evidence: add-titles

| Field | Value |
|---|---|
| Date | 2026-10-29 |
| Docket | v0.4.3-0.20261029113230-8edde0eefcfc |
| OpenSpec | 1.14.3 |
| Repository | git@github.com:wake/docket.git |
| HEAD | 8edde0eefcfcfb982390fb7886f67a01f615771c |
| Title | Titles for changes and specs |
| Kind | feature |
| Source | docket/ideas/2026-10-29-document-titles.md, item 1 of docket/order.md |
| Base | 5df2ba2f4d187386ada6fc9e556edbb32f9597d7 |

## Task commits

| Task | Commit | Subject |
|---|---|---|
| 1 | 48f95fcd8b2f5ff82064aa6df85a6fc8b7c97b9c | docs(specs): titles for the main specs and for add-titles |
| 2 | f2a5d6e42c3b9a41b8cf64157a68733cfd4eda08 | refactor(change): SetBase becomes SetKey with a key parameter |
| 2 | 70fe77fcb6f43ec2274876fc2b47769223f4da21 | feat(change): read docket.yaml, with .openspec.yaml as the fallback |
| 3 | d1c7449b917615712212394d2728a8634288a1f1 | feat(new): dkt new requires --title and writes docket.yaml |
| 4 | 9c4cd68f928e4aaf22006335dde93f85c2229dfd | feat(lint): report missing titles and broken docket.yaml |
| 4 | 584e17bc66e3aecd257ecb2ccf2a7825556b2aae | test(cli): give the real-OpenSpec archive test a full docket.yaml |
| 5 | dc505f006e2d069f830590569e294524f719c342 | feat(archive): read the base from docket.yaml in archive, rebase and sync |
| 6 | d825e3427ddf909fbf64364fe300e6ce02cf0843 | feat(archive): carry spec titles to the main specs |
| 7 | 8806c46ce2886f92c0e69786306facec71dc9ef8 | feat(status): show each change title |
| 7 | 48dc28cb686ac9e8ac099c456873dd1cdc7a40a8 | refactor(change): unexport parseMetadata |
| 8 | 37699d268b13c39bd209db38c16ba53f8041e31a | docs: document docket.yaml and titles |
| 8 | 0a5970dcd5084d43b4e8680a911c1673ab5336f1 | docs: name the files archive and rebase write |

## Checks

- `openspec validate add-titles --strict`: Change 'add-titles' is valid
- Base check: 22 requirements checked, none changed since the base
