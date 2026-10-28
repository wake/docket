---
title: Name the command and repository after a bad flag
---

Found while reviewing `add-init-and-new`, 2026-10-24. Pick it up in the
change that touches the same code, or fix it directly when it changes no
spec sentence.

`dkt --bogus probe --json` reports the command as `""`, and `repo` is null
after a bad flag even inside a repository.
