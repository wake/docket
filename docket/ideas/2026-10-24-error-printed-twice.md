---
title: Print a text-mode error once
---

Found while reviewing `add-init-and-new`, 2026-10-24. Pick it up in the
change that touches the same code, or fix it directly when it changes no
spec sentence.

In text mode an error is printed on standard error and again as a finding on
standard output.
