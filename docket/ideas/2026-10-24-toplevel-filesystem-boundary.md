---
title: Name the filesystem boundary when git stops at a mount point
---

Found while reviewing `add-init-and-new`, 2026-10-24. Pick it up in the
change that touches the same code, or fix it directly when it changes no
spec sentence.

`gitx.Toplevel` does not model filesystem boundaries
(`GIT_DISCOVERY_ACROSS_FILESYSTEM`): a `.git` above a mount point keeps
git's own error instead of "not inside a git work tree". Both exit 4.
