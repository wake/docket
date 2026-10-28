---
title: Drop the dangling colon when a tool wrote nothing
---

Found while reviewing `add-init-and-new`, 2026-10-24. Pick it up in the
change that touches the same code, or fix it directly when it changes no
spec sentence.

Messages end in a dangling `: ` when git or OpenSpec wrote nothing to
standard error (`gitx.ExitError`, `openspec.FailedError`).
