---
title: Exit 3 for an internal error with unreadable sources
---

Found while reviewing `add-init-and-new`, 2026-10-24. Pick it up in the
change that touches the same code, or fix it directly when it changes no
spec sentence.

An internal error together with unreadable sources exits 1, not 3.
