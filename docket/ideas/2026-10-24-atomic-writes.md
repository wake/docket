---
title: Write schema files and metadata through a temporary file
---

Found while reviewing `add-init-and-new`, 2026-10-24. Pick it up in the
change that touches the same code, or fix it directly when it changes no
spec sentence.

Schema files and `.openspec.yaml` are rewritten in place, not written to a
temporary file and renamed.
