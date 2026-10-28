---
title: Reject a spec file directly under specs/
---

Found while reviewing `add-conflicts`, 2026-10-24.

`delta.CapabilityPath` accepts `specs/spec.md` as the capability `spec.md`
(lint, archive and conflicts alike); OpenSpec ignores a file directly under
`specs/`.
