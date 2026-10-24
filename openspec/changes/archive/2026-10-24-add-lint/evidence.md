# Evidence: add-lint

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
| 1 | 82d9cfefda8b149fbbca9b161e302597a09fb9f5 | feat(markdown): read lines, fences, headings and tables |
| 1 | fd842bac2c426e0abca09535896ad600735a21ba | fix(markdown): one fence-aware comment rule for proposals and deltas |
| 1 | ee8676a05ef9331bdcb054ad940e24f12a12ed73 | fix(markdown): match JavaScript whitespace as OpenSpec does |
| 2 | 602eb5ab1820c8ed4bec272283457abe165a38ae | feat(delta): read deltas and main specs as OpenSpec 1.14 does |
| 2 | d0bbeb0d0cb1bf4be09b1399132b197d6d201f41 | fix(delta): close comments on --!> as OpenSpec does |
| 2 | edefb268b4f66116da92642fa407b7f6b13db93c | fix(delta): match JavaScript whitespace as OpenSpec does |
| 2 | a759aa41d02e15602b6089a150bf42fe17806d8e | fix(delta): trim as JavaScript does, like OpenSpec |
| 3 | db6443dfc7ac3046c72735b3aa5506da21a6a1a5 | test(delta): compare the delta reader with OpenSpec 1.14.3 |
| 4 | a7af88af07c9be80a8606f2a2a8051863554ce4c | feat(change): read proposal.md and plan.md |
| 4 | 1c1504d53982fb158f8232c0d0b3277d8b29eb7a | fix(change): read plan markers on every unfenced line |
| 4 | 5488acee3bed82cf418fed255473bdbb97f1b0aa | fix(change): check every top-level cancelled bullet and reject overflowing numbers |
| 4 | 7ce253d93861a38b00b3181d4d36abfb4c51374a | fix(change): an empty Scenarios value is not none |
| 5 | 16a6a029f8ab048ab51809b5bc686112f26c97f2 | feat(lint): lint proposals and leftover template text |
| 5 | f320d4ccfb8cd6dbd0a510fdbfe6e8db2121be7e | fix(lint): read the templates once and test the real template comment |
| 5 | 56d411d0d4dff6ad202525ad6f8c9ab29c6ced82 | fix(lint): treat linked change folders alike and check the first contract section |
| 6 | 8b271e09585cc80090de1c644977ba9399fede84 | feat(lint): check task headings, the Reuse table and cancelled tasks |
| 6 | 6d3eccba07d514deaac1ab92ef8e472a48482899 | fix(lint): accept a Reuse path that ends in a slash |
| 7 | 11a2cc7d5bf3478e5c911cff490b258df20a1993 | feat(lint): check scenario claims, slugs and delta structure |
| 7 | 8fe67008570bd31f0983b286c59977dbfc4842ee | fix(lint): read each main spec once and keep plan tests strict |
| 7 | 848bab4879b6f45aba30c7e971548c3bbd72ef69 | fix(lint): a renamed requirement leaves no scenarios in the main spec |
| 7 | d9b31982a3a5a43e06dfc5d5cdaf5e3782fb4d5f | fix(lint): check empty slugs and order delta scenarios by line |
| 7 | 0b2ecb620e356802f23c80611ce2845e394b512d | fix(lint): a pure rename keeps the requirement's scenarios |
| 8 | 09c7fcffadffa2aafc1b02d9ce79b7cbc3553305 | feat: dkt lint checks change folders and where designs were written |
| 8 | 81d21850e07b855ff2bf3bdc860783b2a5aa6401 | fix(gitx): a staged addition deleted from the worktree is not listed |
| 8 | 21222717563f6e813f3823bdb7d169bfec4cced5 | fix(lint): one exported list of routed directories |
| 9 | c7b119f34a5e6649106ec72fb26d9591d72539c9 | docs: map the lint packages and run dkt lint in CI |

## Checks

- `openspec validate add-lint --strict`: Change 'add-lint' is valid
- Base check: 13 requirements checked, none changed since x-base
