# Evidence: add-docket-init

| Field | Value |
|---|---|
| Date | 2026-11-01 |
| Docket | v0.12.0 |
| OpenSpec | 1.14.3 |
| Repository | git@github.com:wake/docket.git |
| HEAD | 7f1975923506b6c5a5e685fac7f489b5bd343661 |
| Title | A user-level docket-init command that sets a project up for Docket |
| Kind | feature |
| Source | docket/ideas/2026-11-01-docket-init-command.md |
| Base | 124ec08a66da9233d7947df00653ec10efa92d95 |

## Task commits

| Task | Commit | Subject |
|---|---|---|
| 1 | 14d4c906395993b7f532068e8896a910669427c5 | feat(user-skills): add the docket-init skill |
| 1 | a5db63a476ba4840f7f044f8082bbeffce84004d | fix(user-skills): note existing skill folders before docket-init runs |
| 1 | 7241f3365231c54b5227d6cf3810921a7860b2c5 | fix(user-skills): stop docket-init when dkt inspect fails |
| 2 | 590023f3250f6de9a3227a2b1fb9618cf7710d56 | refactor(setup): pass destinations as Dirs |
| 2 | b49ccc8a2e4d0dcb74e704867f7a53c0db3a1a20 | feat(setup): install the user-level docket-init skill |
| 2 | 5b789fcf5e158d06137c9941fb6048ac6beb1c01 | docs(setup): refer to Dirs.Schema in comments |
| 2 | 856f35f9cdbab6a15b295948240761051d7c73bb | docs(setup): rewrap the Inspect doc comment |
| 2 | 9c20deea59790d367b2f416c77b803550d53a6c2 | docs(setup): wrap the Inspect doc comment at the file's width |
| 3 | a094b60184c4bd03a3b9c0b28765160ba9d9f525 | feat(inspect): run without a default branch |
| 3 | f12103f3603d00172169da20ea72cf28a45c4563 | fix(inspect): print the default branch error in the skipped line |
| 4 | c22a29b9b143aff7f9eb33333eb6ae86ed2ae763 | test(cli): give init and inspect one home directory |
| 4 | eb60228929f9f59fe25984007c47340dbdef7e7e | feat(init): install and inspect the docket-init command |
| 5 | 471ed4f463ab873f6439e0dcb9a2f378baa3782a | docs: the docket-init command |
| 5 | 30f12f6f88dd6e23d376d4886b9cf46fd0720aaf | docs: add docket-init to the zh-TW file list |

## Checks

- `openspec validate add-docket-init --strict`: Change 'add-docket-init' is valid
- Base check: 9 requirements checked, none changed since the base
