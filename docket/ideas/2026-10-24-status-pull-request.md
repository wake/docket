---
title: Show a change's pull request
---

Left out of `dkt status` on purpose by `add-status`, 2026-10-24.

Finding a pull request needs the hosting service's API, and `dkt status`
never fetches or calls out. A workbench can join pull requests onto the
`slug` it reads from `status --json`.
