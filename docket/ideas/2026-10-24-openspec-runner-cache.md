---
title: Cache a failed OpenSpec version check safely
---

Found while reviewing `add-init-and-new`, 2026-10-24. Pick it up in the
change that touches the same code, or fix it directly when it changes no
spec sentence.

`openspec.Runner` does not cache a failed version check, and its cache is
not safe for concurrent use.
