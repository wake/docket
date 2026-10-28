---
title: Revisit a worktree shadowed by an unmerged branch
---

Found while reviewing `add-init-and-new`, 2026-10-24. Pick it up in the
change that touches the same code, or fix it directly when it changes no
spec sentence.

A worktree whose HEAD the default branch contains, with uncommitted edits to
a change, is shadowed by an unmerged branch holding the same slug: that is
the precedence order in the `workspace-scan` spec. Revisit now that `status`
shows where content comes from.
