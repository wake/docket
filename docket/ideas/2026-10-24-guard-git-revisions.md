---
title: Guard git revisions that start with a dash
---

Found while reviewing `add-conflicts`, 2026-10-24.

`gitx` methods that take a revision (`ListFiles`, `ListDirs`,
`ShowIfExists`, `Show`) pass it where git still parses options, so a
revision starting with `-` would be read as an option. Every caller passes a
SHA from the scan today; guard them all in one fix.
