---
title: Cut the routing section without stopping inside a code fence
---

Found while reviewing `add-init-and-new`, 2026-10-24. Pick it up in the
change that touches the same code, or fix it directly when it changes no
spec sentence.

The routing section is cut out of AGENTS.md by heading, so a `## ` line
inside a code fence would end it early.
