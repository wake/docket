---
title: Keep the comment on the x-base line
---

Found while reviewing `add-init-and-new`, 2026-10-24. Pick it up in the
change that touches the same code, or fix it directly when it changes no
spec sentence.

`change.SetBase` drops a trailing comment on the `x-base:` line.
