---
title: Report delta specs that are links
---

Found while reviewing `add-conflicts`, 2026-10-24.

Delta specs that are links are skipped without being reported; OpenSpec may
follow them, so an overlap through a linked delta spec is not seen. Decide
whether to report them.
