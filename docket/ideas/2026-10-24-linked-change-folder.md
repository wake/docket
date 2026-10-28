---
title: Report a change folder that is a link
---

Found while reviewing `add-init-and-new`, 2026-10-24. Pick it up in the
change that touches the same code, or fix it directly when it changes no
spec sentence.

On disk, a change folder that is a symbolic link is not listed
(`DirEntry.IsDir` is false for it). Decide whether it should be reported.
