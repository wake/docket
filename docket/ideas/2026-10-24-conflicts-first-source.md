---
title: Say that dkt conflicts reads each change's first source
---

Found while reviewing `add-conflicts`, 2026-10-24.

`dkt conflicts` compares the content of each change's first source only; say
so in the command's help now that `status` shows where content comes from.
