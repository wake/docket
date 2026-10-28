---
title: Build test fixtures with setup.Init
---

Found while reviewing `add-init-and-new`, 2026-10-24. Pick it up in the
change that touches the same code, or fix it directly when it changes no
spec sentence.

`testutil.InstallSchema` repeats the walk-and-write loop of `setup.Init`; a
thin wrapper over `setup.Init` would keep test fixtures identical to what
`dkt init` installs.
