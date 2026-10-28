---
title: Make TestOriginHEADNotASource able to fail
---

Found while reviewing `add-init-and-new`, 2026-10-24. Pick it up in the
change that touches the same code, or fix it directly when it changes no
spec sentence.

`TestOriginHEADNotASource` cannot fail on its own, because `origin/HEAD`
points at the default tip; the guard is tested in `internal/gitx`.
