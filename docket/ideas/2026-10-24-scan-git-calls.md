---
title: Start fewer git processes per ref in the scan
---

Found while reviewing `add-init-and-new`, 2026-10-24. Pick it up in the
change that touches the same code, or fix it directly when it changes no
spec sentence.

The scan starts several git processes per ref (`merge-base --is-ancestor`
and two `ls-tree`); `git for-each-ref --no-merged` could drop merged refs in
one call. `Toplevel` runs git twice.
