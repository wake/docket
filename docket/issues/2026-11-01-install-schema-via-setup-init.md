---
title: Build test fixtures with setup.Init
source: review of add-init-and-new, 2026-10-24; moved from docket/ideas/2026-10-24-install-schema-helper.md
---

## Current

`testutil.InstallSchema` and its siblings (internal/testutil/openspec.go:104-139) repeat the walk-and-write loop of `setup.Init` (internal/setup/setup.go:151).

## Expected

`project-setup` defines what `dkt init` installs. Fixtures should come from the same code, so they cannot drift from it.

## Unchanged

The files fixtures end up with; test results; the old-schema fixture.

## Found

Review of `add-init-and-new`, 2026-10-24. Still present on main.
