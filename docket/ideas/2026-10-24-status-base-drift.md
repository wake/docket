---
title: Say when a change's base is behind the default branch
---

Left out of `dkt status` on purpose by `add-status`, 2026-10-24.

A field saying the change's recorded base is behind the default branch could
reuse the base check of `dkt archive` instead of a second implementation.
