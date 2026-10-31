# Evidence: install-schema-per-machine

| Field | Value |
|---|---|
| Date | 2026-11-01 |
| Docket | dev |
| OpenSpec | 1.14.3 |
| Repository | git@github.com:wake/docket.git |
| HEAD | b5db90f7cbb3cffbcc7f587fca1078cd1d381b0a |
| Title | Install the shipped schema once per machine |
| Kind | change-request |
| Source | docket/ideas/2026-10-31-global-schema.md (order 5) |
| Base | 1c6ca8d4a951ee2a2e2e1d028c0298ce8cf15e9b |

## Task commits

| Task | Commit | Subject |
|---|---|---|
| 1 | c6197140f68b64f243431c4f6fb27c523139621f | refactor(userdirs): find the home outside the workflow package |
| 1 | cabf77437304f94fcc9c14b9563c3bebdc8b3820 | feat(setup): the per-machine schema directory and shipped schema hashes |
| 2 | 53e1c7f5c14c1df471cef16105aa4bacc37f07f1 | feat(init, inspect): the schema per machine and its name in config.yaml |
| 2 | d0eaedc6135ab75adca7e5bab65d6312bc9c7cc2 | fix(init): cross-check the schema key with the YAML parser |
| 3 | b82157b39b98537b3d5c39fdd3935b0b6ba5eef5 | feat(init, inspect): remove or report a project copy of the schema |
| 3 | 38a3c8e051e981c6259b83b78cf0494563ec3bf2 | fix(init): check the project copy again after renaming it aside |
| 4 | c84281f4b41ffc1ec4099c1dc5c3e3cfdb06acc1 | feat(new, archive): find the schema as OpenSpec resolves it |
| 4 | 5a05e8e7a9430d505d1fa19fac021759bae6b3b3 | fix(new): read a schema copy with links instead of failing |
| 4 | cd92f8f902107e84f046a40bf5a0d204c60ef466 | docs: exit-4 wording and the doctor on project-copy removal |
| 5 | d6e38dec481ff4104e6752ea947e3512804663f5 | refactor: ship the schema from schema/docket |
| 5 | 3ea60a4e066164c9a4bd8be47b53923130dba65f | docs: the schema is installed per machine |
| 5 | c4417444d109a878b449b98da69476511a1a38f0 | docs: the per-machine schema in the Chinese README, and rewrapping |
| 6 | a635e670b81b1c7489c0febdd206a65bb22c3323 | docs(doctor): the per-machine schema findings |

## Checks

- `openspec validate install-schema-per-machine --strict`: Change 'install-schema-per-machine' is valid
- Base check: 13 requirements checked, none changed since the base
