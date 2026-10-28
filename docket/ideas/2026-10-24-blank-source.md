---
title: Refuse a --source made of spaces
---

Found while reviewing `add-init-and-new`, 2026-10-24. Pick it up in the
change that touches the same code, or fix it directly when it changes no
spec sentence.

`dkt new --source " "` is accepted; only an empty `--source` is refused.
