# Evidence: add-framework-skill

| Field | Value |
|---|---|
| Date | 2026-10-30 |
| Docket | v0.5.1-0.20261029182501-039f40c1e752 |
| OpenSpec | 1.14.3 |
| Repository | git@github.com:wake/docket.git |
| HEAD | 7f350b961d1926985c2a70bd4bc3e10298454c57 |
| Title | Ship the framework as a skill |
| Kind | feature |
| Source | docket/ideas/2026-10-29-framework-skill.md |
| Base | 7c5adcfe1144ce47f90dca24c2d2bee6e7968cd6 |

## Task commits

| Task | Commit | Subject |
|---|---|---|
| 1 | 34c38d5cf26e6406f242aec692478c636579c1e4 | feat(skill): add the docket skill, the AGENTS.md block and their embeds |
| 1 | 848a4ab1a39ac149dfced2e1917ff4867c104433 | docs(skill): the pre-merge reviewers fall back like any review |
| 1 | 48533b075ed81758271c05a1efa2419bdcf0680d | docs(skill): where a change and a new capability keep their titles |
| 2 | cb301fc7aa981ac13d49ffa536e20afcd9c4388a | feat(setup): find and place the Docket block in a file |
| 2 | 7739c14d1aa35834d2c6d0186b5da8d320458465 | refactor(schema): Block uses setup.FindBlock |
| 2 | 8a5ee2d8181fc6be35249aff3334e0f40d5e78c2 | test(setup): cover each duplicate-marker check alone |
| 2 | 9537beb76c853cab6232ecbe9a4a65aa00f967a9 | refactor(setup): wrap errMalformed with fmt.Errorf |
| 3 | 713bf30e801c3eaeebeec6a7c6af6e36229b54eb | feat(init): install the skill, checking every path before writing |
| 4 | 325794e520142c44cb891ce2a07134c92152353f | feat(init): keep the Docket block in AGENTS.md and CLAUDE.md |
| 4 | 8a8ad47414fd6f762c8cc2d7d2a59983f0399afa | fix(init): a CLAUDE.md that imports AGENTS.md still needs well-formed markers |
| 4 | a612470e6fd3f600b3d44a22c4d1e99c965a4711 | test(init): cover a linked AGENTS.md, a directory there and a forced CLAUDE.md block |
| 4 | d639c7b74c899cc667d462489a43ae4208a06a22 | fix(setup): rewrite an existing file through a temporary file |
| 4 | 8c9a3a54117c19fc55f0c28d7adef346422340de | test(setup): pin the permission copy and the temporary file cleanup |
| 4 | 2499c66317699c045c1706d709240155dd4388f2 | style(setup): gofmt rewrite_test.go |
| 4 | 778b5b8bbc6a78571c2eddbfff96202fb16589c2 | fix(setup): create a new file through a temporary file too |
| 5 | 21f87e420031d836af2058e2fe19a7eb5f3624d6 | test: install the docket skill in this repository and pin it to the shipped copy |
| 5 | dfe81e8fe09e865f3a620e9431474cf3752e0ef6 | docs: describe the docket skill and Docket block in AGENTS.md, READMEs and ideas |
| 6 | none | |

## Checks

- `openspec validate add-framework-skill --strict`: Change 'add-framework-skill' is valid
- Base check: 10 requirements checked, none changed since the base
