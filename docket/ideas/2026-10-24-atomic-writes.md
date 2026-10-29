---
title: Write metadata files through a temporary file
---

Found while reviewing `add-init-and-new`, 2026-10-24. Pick it up in the
change that touches the same code, or fix it directly when it changes no
spec sentence.

`dkt init` now writes existing files through a temporary file and a rename.
What remains is the metadata writes (`.openspec.yaml` and `docket.yaml`),
which are still rewritten in place, not written to a temporary file and
renamed.
